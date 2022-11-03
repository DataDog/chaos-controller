// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package slack

import (
	"fmt"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/eventnotifier/utils"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	infoNotAvailable = "n/a"
)

//go:generate mockery --name=slackNotifier --inpackage --case=underscore --testonly
type slackNotifier interface {
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
	GetUserByEmail(email string) (*slack.User, error)
}

type NotifierSlackConfig struct {
	Enabled              bool
	TokenFilepath        string
	MirrorSlackChannelID string // To remove when we stop testing observer feature
}

// Notifier describes a Slack notifier
type Notifier struct {
	client slackNotifier
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

	defer func() {
		err := tokenfile.Close()
		if err != nil {
			not.logger.Warnw("unable to close token file", "error", err)
		}
	}()

	token, err := io.ReadAll(tokenfile)
	if err != nil {
		return nil, fmt.Errorf("slack token file could not be read: %w", err)
	}

	stoken := string(token)

	if stoken == "" {
		return nil, fmt.Errorf("slack token file is read, but seemingly empty")
	}

	stoken = strings.Fields(stoken)[0] // removes eventual \n at the end of the file
	slackClient := slack.New(stoken)

	if _, err = slackClient.AuthTest(); err != nil {
		return nil, fmt.Errorf("slack auth failed: %w", err)
	}

	not.client = slackClient

	not.logger.Info("notifier: slack notifier connected to workspace")

	return not, nil
}

// GetNotifierName returns the driver's name
func (n *Notifier) GetNotifierName() string {
	return string(types.NotifierDriverSlack)
}

func (n *Notifier) buildSlackBlocks(dis v1beta1.Disruption, notifType types.NotificationType) []*slack.TextBlockObject {
	if n.common.ClusterName == "" {
		if dis.ClusterName != "" {
			n.common.ClusterName = dis.ClusterName
		} else {
			n.common.ClusterName = infoNotAvailable
		}
	}

	return []*slack.TextBlockObject{
		slack.NewTextBlockObject("mrkdwn", "*Kind:*\n"+dis.Kind, false, false),
		slack.NewTextBlockObject("mrkdwn", "*Name:*\n"+dis.Name, false, false),
		slack.NewTextBlockObject("mrkdwn", "*Notification Type:*\n"+string(notifType), false, false),
		slack.NewTextBlockObject("mrkdwn", "*Cluster:*\n"+n.common.ClusterName, false, false),
		slack.NewTextBlockObject("mrkdwn", "*Namespace:*\n"+dis.Namespace, false, false),
		slack.NewTextBlockObject("mrkdwn", "*Targets:*\n"+fmt.Sprint(len(dis.Status.Targets)), false, false),
		slack.NewTextBlockObject("mrkdwn", "*DryRun:*\n"+strconv.FormatBool(dis.Spec.DryRun), false, false),
		slack.NewTextBlockObject("mrkdwn", "*Duration:*\n"+dis.Spec.Duration.Duration().String(), false, false),
	}
}

// Notify generates a notification for generic k8s events
func (n *Notifier) Notify(dis v1beta1.Disruption, event corev1.Event, notifType types.NotificationType) error {
	headerText := utils.BuildHeaderMessageFromDisruptionEvent(dis, notifType)
	headerBlock := slack.NewHeaderBlock(slack.NewTextBlockObject("plain_text", headerText, false, false))
	bodyText := utils.BuildBodyMessageFromDisruptionEvent(dis, event, true)
	bodyBlock := slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", bodyText, false, false), nil, nil)
	disruptionBlocks := n.buildSlackBlocks(dis, notifType)

	userInfo, err := dis.UserInfo()
	if err != nil {
		n.logger.Errorw("unable to retrieve disruption user info", "error", err, "disruption", dis.Name)
	}

	// Whenever a purpose is defined, we expect it to be available into all notifications sent messages
	if nil != dis.Spec.Reporting && dis.Spec.Reporting.Purpose != "" {
		disruptionBlocks = append(disruptionBlocks, slack.NewTextBlockObject("mrkdwn", "*Purpose:*\n"+dis.Spec.Reporting.Purpose, false, false))
	}

	if n.config.MirrorSlackChannelID != "" {
		n.sendMessageToChannel(userInfo, n.config.MirrorSlackChannelID, headerText, headerBlock, disruptionBlocks, bodyBlock)
	}

	if nil != dis.Spec.Reporting && dis.Spec.Reporting.SlackChannel != "" && dis.Spec.Reporting.MinNotificationType.Allows(notifType) {
		n.sendMessageToChannel(userInfo, dis.Spec.Reporting.SlackChannel, headerText, headerBlock, disruptionBlocks, bodyBlock)
	}

	// We expect notification equal to or above success to be sent to users
	if !types.NotificationSuccess.Allows(notifType) {
		n.logger.Debugw("slack notifier: not sending info notification type to not flood user", "disruptionName", dis.Name, "eventType", event.Type, "message", bodyText)

		return nil
	}

	emailAddr, err := mail.ParseAddress(userInfo.Username)
	if err != nil {
		return fmt.Errorf("slack notifier: invalid user info email in disruption %s: %w", dis.Name, err)
	}

	p1, err := n.client.GetUserByEmail(emailAddr.Address)
	if err != nil {
		n.logger.Warnw("slack notifier: user not found", "userAddress", emailAddr.Address, "error", err)
		return nil
	}

	_, _, err = n.client.PostMessage(p1.ID,
		slack.MsgOptionText(headerText, false),
		slack.MsgOptionUsername("Disruption Status Bot"),
		slack.MsgOptionIconURL("https://upload.wikimedia.org/wikipedia/commons/3/39/LogoChaosMonkeysNetflix.png"),
		slack.MsgOptionBlocks(
			headerBlock,
			slack.NewDividerBlock(),
			slack.NewSectionBlock(nil, disruptionBlocks, nil),
			slack.NewDividerBlock(),
			bodyBlock,
		),
		slack.MsgOptionAsUser(true),
	)
	if err != nil {
		return fmt.Errorf("slack notifier: %w", err)
	}

	n.logger.Debugw("notifier: sending notifier event to slack", "disruptionName", dis.Name, "eventType", event.Type, "message", bodyText)

	return nil
}

func (n *Notifier) sendMessageToChannel(userInfo authv1.UserInfo, slackChannel, headerText string, headerBlock *slack.HeaderBlock, disruptionBlocks []*slack.TextBlockObject, bodyBlock *slack.SectionBlock) {
	userName := infoNotAvailable
	if userInfo.Username != "" {
		userName = userInfo.Username
	}

	_, _, err := n.client.PostMessage(slackChannel,
		slack.MsgOptionText(headerText, false),
		slack.MsgOptionUsername("Disruption Status Bot"),
		slack.MsgOptionIconURL("https://upload.wikimedia.org/wikipedia/commons/3/39/LogoChaosMonkeysNetflix.png"),
		slack.MsgOptionBlocks(
			headerBlock,
			slack.NewDividerBlock(),
			slack.NewSectionBlock(nil, append(disruptionBlocks, slack.NewTextBlockObject("mrkdwn", "*Author:*\n"+userName, false, false)), nil),
			slack.NewDividerBlock(),
			bodyBlock,
		),
		slack.MsgOptionAsUser(true),
	)
	if err != nil {
		n.logger.Errorw("slack notifier: couldn't send a message to the channel", "slackChannel", slackChannel, "error", err)
	}
}
