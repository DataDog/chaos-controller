// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package slack

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/slack-go/slack"
)

// Notifier describes a Slack notifier
type Notifier struct {
	client slack.Client
}

// New Slack Notifier
func New() *Notifier {

	not := &Notifier{}
	not.client = *slack.New("")

	return not
}

// Close returns nil
func (n *Notifier) Clean() error {
	return nil
}

// GetNotifierName returns Slack
func (n *Notifier) GetNotifierName() string {
	return string(types.NotifierDriverSlack)
}

// NotifyNotInjected signals a disruption was injected successfully
func (n *Notifier) NotifyNotInjected(dis v1beta1.Disruption) error {
	headerText := "New Disruption: " + dis.Name
	bodyText := "> You started the disruption `" + dis.Name + "`. Waiting for " + fmt.Sprint(len(dis.Status.Targets)) + " injection(s)."

	blocks := []slack.Block{
		slack.NewHeaderBlock(slack.NewTextBlockObject("plain_text", headerText, false, false)),
		slack.NewDividerBlock(),
		slack.NewSectionBlock(nil, []*slack.TextBlockObject{
			slack.NewTextBlockObject("mrkdwn", "*Kind:*\n"+dis.Kind, false, false),
			slack.NewTextBlockObject("mrkdwn", "*Name:*\n"+dis.Name, false, false),
			slack.NewTextBlockObject("mrkdwn", "*Cluster:*\nminikube", false, false),
			slack.NewTextBlockObject("mrkdwn", "*Namespace:*\n"+dis.Namespace, false, false),
			slack.NewTextBlockObject("mrkdwn", "*Targets:*\n"+fmt.Sprint(len(dis.Status.Targets)), false, false),
		}, nil),
		slack.NewDividerBlock(),
		slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", bodyText, false, false), nil, nil),
	}

	err := n.notifySlack("isn't injected", dis, blocks...)

	return err
}

// NotifyInjected signals a disruption was injected successfully
func (n *Notifier) NotifyInjected(dis v1beta1.Disruption) error {
	headerText := "> Disruption `" + dis.Name + "` has been fully injected. Waiting for deletion."

	blocks := []slack.Block{
		slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", headerText, false, false), nil, nil),
	}

	err := n.notifySlack("is injected", dis, blocks...)

	return err
}

// NotifyCleanedUp signals a disruption's been cleaned up successfully
func (n *Notifier) NotifyCleanedUp(dis v1beta1.Disruption) error {
	headerText := "> Disruption `" + dis.Name + "` has been cleaned up."

	blocks := []slack.Block{
		slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", headerText, false, false), nil, nil),
		slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", "from #chaos-engineering", false, false)),
	}

	err := n.notifySlack("has been cleaned up", dis, blocks...)

	return err
}

// NotifyNoTarget signals a disruption's been cleaned up successfully
func (n *Notifier) NotifyNoTarget(dis v1beta1.Disruption) error {
	n.notifySlack("has no target", dis)

	return nil
}

// NotifyStuckOnRemoval signals a disruption's been cleaned up successfully
func (n *Notifier) NotifyStuckOnRemoval(dis v1beta1.Disruption) error {
	n.notifySlack("is stuck on removal. Please check the logs !", dis)

	return nil
}

// helper for Slack notifier
func (n *Notifier) notifySlack(notificationName string, dis v1beta1.Disruption, blocks ...slack.Block) error {
	fmt.Printf("SLACK: %s for disruption %s - user %s\n", notificationName, dis.Name, dis.Status.UserInfo.Username)

	p1, err := n.client.GetUserByEmail("nathan.tournant@datadoghq.com")
	if err != nil {
		return err
	}

	_, _, err = n.client.PostMessage(p1.ID,
		slack.MsgOptionText("Disruption "+dis.Name+" "+notificationName, false),
		slack.MsgOptionUsername("Disruption Status Bot"),
		slack.MsgOptionIconURL("https://upload.wikimedia.org/wikipedia/commons/3/39/LogoChaosMonkeysNetflix.png"),
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionAsUser(false),
	)

	return err
}
