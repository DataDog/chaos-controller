// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

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
	cLog "github.com/DataDog/chaos-controller/log"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	infoNotAvailable = "n/a"
)

type slackNotifier interface {
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
	GetUserByEmail(email string) (*slack.User, error)
}

type slackMessage struct {
	HeaderText  string
	HeaderBlock slack.HeaderBlock
	UserName    string
	BodyText    string
	BodyBlock   slack.SectionBlock
	InfoBlocks  []*slack.TextBlockObject
	UserEmail   string
}

type NotifierSlackConfig struct {
	Enabled              bool   `yaml:"enabled"`
	TokenFilepath        string `yaml:"tokenFilepath"`
	MirrorSlackChannelID string `yaml:"mirrorSlackChannelId"`
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

// Notify generates a notification for generic k8s events
func (n *Notifier) Notify(obj client.Object, event corev1.Event, notifType types.NotificationType) error {
	switch d := obj.(type) {
	case *v1beta1.Disruption:
		return n.notifyForDisruption(d, event, notifType)
	case *v1beta1.DisruptionCron:
		return n.notifyForDisruptionCron(d, event, notifType)
	}

	return nil
}

func (n *Notifier) notifyForDisruption(dis *v1beta1.Disruption, event corev1.Event, notifType types.NotificationType) error {
	logger := n.logger.With(cLog.DisruptionNameKey, dis.Name, cLog.DisruptionNamespaceKey, dis.Namespace, "eventType", event.Type)

	slackMsg := n.buildSlackMessage(dis, event, notifType, dis.Spec.Reporting, logger)

	if n.config.MirrorSlackChannelID != "" {
		if err := n.sendMessageToChannel(n.config.MirrorSlackChannelID, slackMsg); err != nil {
			logger.Warnw("slack notifier: couldn't send a message to the mirror slack channel", "slackChannel", n.config.MirrorSlackChannelID, "error", err)
		}
	}

	if nil != dis.Spec.Reporting && dis.Spec.Reporting.SlackChannel != "" && dis.Spec.Reporting.MinNotificationType.Allows(notifType) {
		if err := n.sendMessageToChannel(dis.Spec.Reporting.SlackChannel, slackMsg); err != nil {
			logger.Warnw("slack notifier: couldn't send a message to the channel from the reporting", "slackChannel", dis.Spec.Reporting.SlackChannel, "error", err)
		}
	}

	// We expect notification equal to or above success to be sent to users
	if !types.NotificationSuccess.Allows(notifType) {
		logger.Debugw("slack notifier: not sending info notification type to not flood user", "message", slackMsg.BodyText)

		return nil
	}

	if err := n.sendMessageToUserChannel(slackMsg, logger); err != nil {
		return fmt.Errorf("slack notifier: %w", err)
	}

	logger.Debugw("notifier: sending notifier event to slack", "message", slackMsg.BodyText)

	return nil
}

func (n *Notifier) notifyForDisruptionCron(disruptionCron *v1beta1.DisruptionCron, event corev1.Event, notifType types.NotificationType) error {
	logger := n.logger.With(cLog.DisruptionCronNameKey, disruptionCron.Name, cLog.DisruptionCronNamespaceKey, disruptionCron.Namespace, "eventType", event.Type)

	slackMsg := n.buildSlackMessage(disruptionCron, event, notifType, disruptionCron.Spec.Reporting, logger)

	if n.config.MirrorSlackChannelID != "" {
		if err := n.sendMessageToChannel(n.config.MirrorSlackChannelID, slackMsg); err != nil {
			logger.Warnw("slack notifier: couldn't send a message to the mirror slack channel", "slackChannel", n.config.MirrorSlackChannelID, "error", err)
		}
	}

	if nil != disruptionCron.Spec.Reporting && disruptionCron.Spec.Reporting.SlackChannel != "" {
		if err := n.sendMessageToChannel(disruptionCron.Spec.Reporting.SlackChannel, slackMsg); err != nil {
			logger.Warnw("slack notifier: couldn't send a message to the channel from the reporting", "slackChannel", disruptionCron.Spec.Reporting.SlackChannel, "error", err)
		}
	}

	if err := n.sendMessageToUserChannel(slackMsg, logger); err != nil {
		return fmt.Errorf("slack notifier: %w", err)
	}

	n.logger.Debugw("notifier: sending notifier event to slack", "message", slackMsg.BodyText)

	return nil
}

func (n *Notifier) sendMessageToUserChannel(slackMsg slackMessage, logger *zap.SugaredLogger) error {
	// emailAddr, err := mail.ParseAddress(slackMsg.UserInfo.Username)
	// if err != nil {
	// 	logger.Infow("username could not be parsed as an email address", "err", err, "username", slackMsg.UserInfo.Username)

	// 	return nil
	// }
	// emailAddr = slackMsg.UserEmail

	p1, err := n.client.GetUserByEmail(slackMsg.UserEmail)
	if err != nil {
		logger.Warnw("user not found", "userAddress", slackMsg.UserEmail, "error", err)

		return nil
	}

	return n.sendMessageToChannel(p1.ID, slackMsg)
}

func (n *Notifier) sendMessageToChannel(slackChannel string, slackMsg slackMessage) error {
	userName := infoNotAvailable
	if slackMsg.UserEmail != "" {
		userName = slackMsg.UserEmail
	}

	_, _, err := n.client.PostMessage(slackChannel,
		slack.MsgOptionText(slackMsg.HeaderText, false),
		slack.MsgOptionUsername(slackMsg.UserName),
		slack.MsgOptionIconURL("https://upload.wikimedia.org/wikipedia/commons/3/39/LogoChaosMonkeysNetflix.png"),
		slack.MsgOptionBlocks(
			slackMsg.HeaderBlock,
			slack.NewDividerBlock(),
			slack.NewSectionBlock(nil, append(slackMsg.InfoBlocks, slack.NewTextBlockObject("mrkdwn", "*Author:*\n"+userName, false, false)), nil),
			slack.NewDividerBlock(),
			slackMsg.BodyBlock,
		),
		slack.MsgOptionAsUser(true),
	)

	return err
}

func (n *Notifier) buildSlackMessage(obj client.Object, event corev1.Event, notifType types.NotificationType, reporting *v1beta1.Reporting, logger *zap.SugaredLogger) slackMessage {
	headerText := utils.BuildHeaderMessageFromObjectEvent(obj, event, notifType)
	headerBlock := slack.NewHeaderBlock(slack.NewTextBlockObject("plain_text", headerText, false, false))
	bodyText := utils.BuildBodyMessageFromObjectEvent(obj, event, true)
	bodyBlock := slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", bodyText, false, false), nil, nil)
	infoBlocks := n.buildSlackBlocks(obj, notifType, reporting)

	var (
		userEmail string
		userInfo  authv1.UserInfo
		err       error
	)

	//two options: make this entire thing apart of a condition so if email doesn't exist then do this current
	//otherwise if email doees exist, some code logic and rather than user info I return slack message with just email
	//however its entirely possible I can't do that so I have to look at what slack message is expecting

	switch d := obj.(type) {
	case *v1beta1.Disruption:
		if d.Spec.Reporting.SlackUserEmail != "" {
			userEmail = d.Spec.Reporting.SlackUserEmail
		}

		userInfo, err = d.UserInfo()
	case *v1beta1.DisruptionCron:
		if d.Spec.Reporting.SlackUserEmail != "" {
			userEmail = d.Spec.Reporting.SlackUserEmail
		}
		userInfo, err = d.UserInfo()
	}

	if err != nil {
		logger.Warnw("unable to retrieve user info", "error", err)
	}

	if userEmail == "" {
		userEmailAddress, err := mail.ParseAddress(userInfo.Username)

		userEmail = userEmailAddress.Address

		if err != nil {
			logger.Warnw("unable to retrieve user info", "error", err)
		}
	}

	//do the parse regardless to make sure we have a valid email
	userEmailAddress, err := mail.ParseAddress(userEmail)

	userEmail = userEmailAddress.Address

	if err != nil {
		logger.Warnw("unable to retrieve user info", "error", err)
	}

	return slackMessage{
		HeaderText:  headerText,
		HeaderBlock: *headerBlock,
		BodyText:    bodyText,
		BodyBlock:   *bodyBlock,
		InfoBlocks:  infoBlocks,
		UserEmail:   userEmail,
		UserName:    fmt.Sprintf("%s Status Bot", obj.GetObjectKind().GroupVersionKind().Kind),
	}
}

func (n *Notifier) buildSlackBlocks(object client.Object, notifType types.NotificationType, reporting *v1beta1.Reporting) []*slack.TextBlockObject {
	textBlocks := []*slack.TextBlockObject{
		slack.NewTextBlockObject("mrkdwn", "*Kind:*\n"+object.GetObjectKind().GroupVersionKind().Kind, false, false),
		slack.NewTextBlockObject("mrkdwn", "*Name:*\n"+object.GetName(), false, false),
		slack.NewTextBlockObject("mrkdwn", "*Notification Type:*\n"+string(notifType), false, false),
		slack.NewTextBlockObject("mrkdwn", "*Cluster:*\n"+n.common.ClusterName, false, false),
		slack.NewTextBlockObject("mrkdwn", "*Namespace:*\n"+object.GetNamespace(), false, false),
	}

	d, ok := object.(*v1beta1.Disruption)
	if ok {
		slack.NewTextBlockObject("mrkdwn", "*Targets:*\n"+fmt.Sprint(len(d.Status.TargetInjections)), false, false)
		slack.NewTextBlockObject("mrkdwn", "*DryRun:*\n"+strconv.FormatBool(d.Spec.DryRun), false, false)
		slack.NewTextBlockObject("mrkdwn", "*Duration:*\n"+d.Spec.Duration.Duration().String(), false, false)
	}

	// Whenever a purpose is defined, we expect it to be available into all notifications sent messages
	if nil != reporting && reporting.Purpose != "" {
		textBlocks = append(textBlocks, slack.NewTextBlockObject("mrkdwn", "*Purpose:*\n"+reporting.Purpose, false, false))
	}

	return textBlocks
}

//to see why my changes aren't being found
