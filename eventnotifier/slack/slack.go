// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package slack

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/slack-go/slack"
	corev1 "k8s.io/api/core/v1"
)

// Notifier describes a Slack notifier
type Notifier struct {
	client slack.Client
}

// New Slack Notifier
func New(tokenFilePath string) (*Notifier, error) {
	not := &Notifier{}
	tokenfile, err := os.Open(filepath.Clean(tokenFilePath))

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

	_, err = not.client.AuthTest()
	if err != nil {
		return nil, fmt.Errorf("slack auth failed: %w", err)
	}

	return not, nil
}

// GetNotifierName returns the driver's name
func (n *Notifier) GetNotifierName() string {
	return string(types.NotifierDriverSlack)
}

// NotifyWarning generates a notification for generic k8s Warning events
func (n *Notifier) NotifyWarning(dis v1beta1.Disruption, event corev1.Event) error {
	headerText := "Disruption `" + dis.Name + "` encountered an issue."
	bodyText := "> Disruption `" + dis.Name + "` emitted the event " + event.Reason + ": " + event.Message

	blocks := []slack.Block{
		slack.NewHeaderBlock(slack.NewTextBlockObject("plain_text", headerText, false, false)),
		slack.NewDividerBlock(),
		slack.NewSectionBlock(nil, []*slack.TextBlockObject{
			slack.NewTextBlockObject("mrkdwn", "*Kind:*\n"+dis.Kind, false, false),
			slack.NewTextBlockObject("mrkdwn", "*Name:*\n"+dis.Name, false, false),
			slack.NewTextBlockObject("mrkdwn", "*Cluster:*\n"+dis.ClusterName, false, false),
			slack.NewTextBlockObject("mrkdwn", "*Namespace:*\n"+dis.Namespace, false, false),
			slack.NewTextBlockObject("mrkdwn", "*Targets:*\n"+fmt.Sprint(len(dis.Status.Targets)), false, false),
		}, nil),
		slack.NewDividerBlock(),
		slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", bodyText, false, false), nil, nil),
	}

	err := n.notifySlack("emitted a warning", dis, blocks...)

	return err
}

// helper for Slack notifier
func (n *Notifier) notifySlack(notificationText string, dis v1beta1.Disruption, blocks ...slack.Block) error {
	p1, err := n.client.GetUserByEmail(dis.Status.UserInfo.Username)
	if err != nil {
		return err
	}

	_, _, err = n.client.PostMessage(p1.ID,
		slack.MsgOptionText("Disruption "+dis.Name+" "+notificationText, false),
		slack.MsgOptionUsername("Disruption Status Bot"),
		slack.MsgOptionIconURL("https://upload.wikimedia.org/wikipedia/commons/3/39/LogoChaosMonkeysNetflix.png"),
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionAsUser(true),
	)

	return err
}
