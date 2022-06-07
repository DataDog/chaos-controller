// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package slack

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/mail"
	"os"
	"path/filepath"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/eventnotifier/utils"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
	v1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
)

type NotifierSlackConfig struct {
	Enabled              bool
	TokenFilepath        string
	MirrorSlackChannelID string // To remove when we stop testing observer feature
}

// Notifier describes a Slack notifier
type Notifier struct {
	client slack.Client
	common types.NotifiersCommonConfig
	config NotifierSlackConfig
	logger *zap.SugaredLogger
}

// New Slack Notifier
func New(commonConfig types.NotifiersCommonConfig, slackConfig NotifierSlackConfig, logger *zap.SugaredLogger) (*Notifier, error) {
	not := &Notifier{
		common: commonConfig,
		config: slackConfig,
		logger: logger,
	}

	tokenfile, err := os.Open(filepath.Clean(not.config.TokenFilepath))

	if err != nil {
		return nil, fmt.Errorf("slack token file not found: %w", err)
	}

	token, err := ioutil.ReadAll(tokenfile)

	if err != nil {
		return nil, fmt.Errorf("slack token file could not be read: %w", err)
	}

	stoken := string(token)

	if stoken == "" {
		return nil, fmt.Errorf("slack token file is read, but seemingly empty")
	}

	stoken = strings.Fields(stoken)[0] // removes eventual \n at the end of the file
	not.client = *slack.New(stoken)

	if _, err = not.client.AuthTest(); err != nil {
		return nil, fmt.Errorf("slack auth failed: %w", err)
	}

	not.logger.Info("notifier: slack notifier connected to workspace")

	return not, nil
}

// GetNotifierName returns the driver's name
func (n *Notifier) GetNotifierName() string {
	return string(types.NotifierDriverSlack)
}

func (n *Notifier) buildSlackBlocks(dis v1beta1.Disruption, bodyText string, headerText string) []slack.Block {
	if n.common.ClusterName == "" {
		if dis.ClusterName != "" {
			n.common.ClusterName = dis.ClusterName
		} else {
			n.common.ClusterName = "n/a"
		}
	}

	return []slack.Block{
		slack.NewHeaderBlock(slack.NewTextBlockObject("plain_text", headerText, false, false)),
		slack.NewDividerBlock(),
		slack.NewSectionBlock(nil, []*slack.TextBlockObject{
			slack.NewTextBlockObject("mrkdwn", "*Kind:*\n"+dis.Kind, false, false),
			slack.NewTextBlockObject("mrkdwn", "*Name:*\n"+dis.Name, false, false),
			slack.NewTextBlockObject("mrkdwn", "*Cluster:*\n"+n.common.ClusterName, false, false),
			slack.NewTextBlockObject("mrkdwn", "*Namespace:*\n"+dis.Namespace, false, false),
			slack.NewTextBlockObject("mrkdwn", "*Targets:*\n"+fmt.Sprint(len(dis.Status.Targets)), false, false),
		}, nil),
		slack.NewDividerBlock(),
		slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", bodyText, false, false), nil, nil),
	}
}

// NotifyWarning generates a notification for generic k8s Warning events
func (n *Notifier) NotifyWarning(dis v1beta1.Disruption, event corev1.Event) error {
	headerText := utils.BuildHeaderMessageFromDisruptionEvent(dis, event)
	bodyText := utils.BuildBodyMessageFromDisruptionEvent(dis, event, true)
	blocks := n.buildSlackBlocks(dis, bodyText, headerText)

	n.logger.Debugw("notifier: sending notifier event to slack", "disruption", dis.Name, "eventType", event.Type, "message", bodyText)

	return n.notifySlack("emitted a warning", dis, blocks...)
}

// NotifyWarning generates a notification for generic k8s normal events
func (n *Notifier) NotifyRecovery(dis v1beta1.Disruption, event corev1.Event) error {
	headerText := utils.BuildHeaderMessageFromDisruptionEvent(dis, event)
	bodyText := utils.BuildBodyMessageFromDisruptionEvent(dis, event, true)
	blocks := n.buildSlackBlocks(dis, bodyText, headerText)

	n.logger.Debugw("notifier: sending notifier event to slack", "disruption", dis.Name, "eventType", event.Type, "message", bodyText)

	return n.notifySlack("emitted a notification", dis, blocks...)
}

// helper for Slack notifier
func (n *Notifier) notifySlack(notificationText string, dis v1beta1.Disruption, blocks ...slack.Block) error {
	var annotation v1.UserInfo

	err := json.Unmarshal([]byte(dis.Annotations["UserInfo"]), &annotation)
	if err != nil {
		return fmt.Errorf("slack notifier: no userinfo in disruption %s: %v", dis.Name, err)
	}

	if _, err := mail.ParseAddress(annotation.Username); err != nil {
		return nil
	}

	p1, err := n.client.GetUserByEmail(annotation.Username)
	if err != nil {
		n.logger.Warn(fmt.Errorf("slack notifier: user %s not found: %w", annotation.Username, err))
		return nil
	}

	_, _, err = n.client.PostMessage(p1.ID,
		slack.MsgOptionText("Disruption "+dis.Name+" "+notificationText, false),
		slack.MsgOptionUsername("Disruption Status Bot"),
		slack.MsgOptionIconURL("https://upload.wikimedia.org/wikipedia/commons/3/39/LogoChaosMonkeysNetflix.png"),
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionAsUser(true),
	)
	if err != nil {
		return fmt.Errorf("slack notifier: %w", err)
	}

	// To remove when we stop testing this feature
	if n.config.MirrorSlackChannelID != "" {
		_, _, err = n.client.PostMessage(n.config.MirrorSlackChannelID,
			slack.MsgOptionText("Disruption "+dis.Name+" "+notificationText, false),
			slack.MsgOptionUsername("Disruption Status Bot"),
			slack.MsgOptionIconURL("https://upload.wikimedia.org/wikipedia/commons/3/39/LogoChaosMonkeysNetflix.png"),
			slack.MsgOptionBlocks(blocks...),
			slack.MsgOptionAsUser(true),
		)
		if err != nil {
			n.logger.Errorw("slack notifier: couldn't send a message to the channel %s. %s", n.config.MirrorSlackChannelID, err.Error())
		}
	}

	return nil
}
