// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/eventnotifier/utils"
	"github.com/DataDog/jsonapi"
	"go.uber.org/zap"
	v1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DisruptionConfig struct {
	Enabled bool
	URL     string
}

type DisruptionCronConfig struct {
	Enabled bool
	URL     string
}

type Config struct {
	Headers         []string
	HeadersFilepath string
	AuthURL         string
	AuthHeaders     []string
	AuthTokenPath   string
	Disruption      DisruptionConfig
	DisruptionCron  DisruptionCronConfig
}

func (c Config) IsEnabled() bool {
	return c.DisruptionCron.Enabled || c.Disruption.Enabled
}

// Notifier describes a HTTP notifier
type Notifier struct {
	common               types.NotifiersCommonConfig
	client               *http.Client
	headers              map[string]string
	logger               *zap.SugaredLogger
	authTokenProvider    BearerAuthTokenProvider
	disruptionConfig     DisruptionConfig
	disruptionCronConfig DisruptionCronConfig
}

// NotifierEvent represents a notification event
type NotifierEvent struct {
	ID                 string                 `jsonapi:"primary,http_notifier_events"`
	Name               string                 `jsonapi:"attribute" json:"name"`
	NotificationTitle  string                 `jsonapi:"attribute" json:"notification_title"`
	NotificationType   types.NotificationType `jsonapi:"attribute" json:"notification_type"`
	EventReason        v1beta1.EventReason    `jsonapi:"attribute" json:"event_reason"`
	EventMessage       string                 `jsonapi:"attribute" json:"event_message"`
	InvolvedObjectKind string                 `jsonapi:"attribute" json:"involved_object_kind"`
	// Deprecated: DisruptionName exists for historical compatibility
	// and should not be used. Use Name instead.
	DisruptionName string `jsonapi:"attribute" json:"disruption_name"`
	// Deprecated: Disruption exists for historical compatibility
	// and should not be used. Use DisruptionEvent.Manifest or DisruptionCronEvent.Manifest instead.
	Disruption string `jsonapi:"attribute" json:"disruption"`
	// Deprecated: TargetsCount exists for historical compatibility
	// and should not be used. Use DisruptionEvent.TargetsCount instead.
	TargetsCount int    `jsonapi:"attribute" json:"targets_count"`
	Timestamp    int64  `jsonapi:"attribute" json:"timestamp"`
	Cluster      string `jsonapi:"attribute" json:"cluster"`
	Namespace    string `jsonapi:"attribute" json:"namespace"`
	Username     string `jsonapi:"attribute" json:"username,omitempty"`
	UserEmail    string `jsonapi:"attribute" json:"user_email,omitempty"`
	UserGroups   string `jsonapi:"attribute" json:"user_groups,omitempty"`
}

type DisruptionEvent struct {
	NotifierEvent
	// Manifest is the JSON representation of the Disruption object
	Manifest string `jsonapi:"attribute" json:"manifest"`
	// TargetsCount is the number of targets in the Disruption object
	TargetsCount int `jsonapi:"attribute" json:"targets_count"`
}

type DisruptionCronEvent struct {
	NotifierEvent
	// Manifest is the JSON representation of the DisruptionCron object
	Manifest string `jsonapi:"attribute" json:"manifest"`
}

// New HTTP Notifier
func New(commonConfig types.NotifiersCommonConfig, httpConfig Config, logger *zap.SugaredLogger) (*Notifier, error) {
	headers := []string{}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	if httpConfig.Disruption.Enabled && httpConfig.Disruption.URL == "" {
		return nil, fmt.Errorf("http notifier: missing URL for disruption notifications")
	}

	if httpConfig.DisruptionCron.Enabled && httpConfig.DisruptionCron.URL == "" {
		return nil, fmt.Errorf("http notifier: missing URL URL for disruption cron notifications")
	}

	if httpConfig.HeadersFilepath != "" {
		headersFile, err := os.Open(filepath.Clean(httpConfig.HeadersFilepath))
		if err != nil {
			return nil, fmt.Errorf("headers file not found: %w", err)
		}

		readHeaders, err := io.ReadAll(headersFile)
		if err != nil {
			return nil, fmt.Errorf("headers file could not be read: %w", err)
		}

		sHeaders := string(readHeaders)
		if sHeaders != "" {
			headers = strings.Split(sHeaders, "\n")
		}
	}

	headers = append(headers, httpConfig.Headers...)

	parsedHeaders, err := splitHeaders(headers)
	if err != nil {
		return nil, fmt.Errorf("notifier http: invalid headers in headers file: %w", err)
	}

	var authTokenProvider BearerAuthTokenProvider

	if httpConfig.AuthURL != "" {
		authHeaders, err := splitHeaders(httpConfig.AuthHeaders)
		if err != nil {
			return nil, fmt.Errorf("notifier http: invalid headers for auth: %w", err)
		}

		authTokenProvider = NewBearerAuthTokenProvider(logger, httpClient, httpConfig.AuthURL, authHeaders, httpConfig.AuthTokenPath)
	}

	return &Notifier{
		common:               commonConfig,
		client:               httpClient,
		headers:              parsedHeaders,
		logger:               logger,
		authTokenProvider:    authTokenProvider,
		disruptionConfig:     httpConfig.Disruption,
		disruptionCronConfig: httpConfig.DisruptionCron,
	}, nil
}

// GetNotifierName returns the driver's name
func (n *Notifier) GetNotifierName() string {
	return string(types.NotifierDriverHTTP)
}

// Notify generates a notification for generic k8s Warning events
func (n *Notifier) Notify(obj client.Object, event corev1.Event, notifType types.NotificationType) error {
	switch d := obj.(type) {
	case *v1beta1.Disruption:
		return n.notifyDisruption(*d, event, notifType)
	case *v1beta1.DisruptionCron:
		return n.notifyDisruptionCron(*d, event, notifType)
	default:
		return nil
	}
}

// splitHeaders parses the headers from a slice of strings in the format key:value.
// It returns a map of headers or an error if the headers are not in the correct format.
func splitHeaders(headers []string) (map[string]string, error) {
	parsedHeaders := make(map[string]string)

	for _, header := range headers {
		if header == "" {
			continue
		}

		splittedHeader := strings.Split(header, ":")

		if len(splittedHeader) == 2 {
			key := splittedHeader[0]
			val := splittedHeader[1]
			parsedHeaders[key] = val
		} else {
			return nil, fmt.Errorf("invalid headers: Must be in the format: key:value, found %s", header)
		}
	}

	return parsedHeaders, nil
}

// getUserDetails retrieves the user details associated with a given disruption.
// It returns the username, the email address, and a JSON string of user groups.
// On error, it logs a warning and returns empty values for the fields.
func (n *Notifier) getUserDetails(uInfo v1.UserInfo, logger *zap.SugaredLogger) (username, emailAddr, userGroups string) {
	username = uInfo.Username
	emailAddr, err := n.extractEmail(username)

	if err != nil {
		logger.Warnw("http notifier: user info username is not a valid email address", "error", err, "username", username)
	}

	userGroups, err = n.marshalUserGroups(uInfo.Groups)
	if err != nil {
		logger.Warnw("http notifier: couldn't marshal user groups", "error", err)
	}

	return username, emailAddr, userGroups
}

// extractEmail tries to parse the provided username as an email address and returns it if valid.
func (n *Notifier) extractEmail(username string) (string, error) {
	emailAddr, err := mail.ParseAddress(username)
	if err != nil {
		return "", err
	}

	return emailAddr.Address, nil
}

// marshalUserGroups converts user groups to a JSON string.
func (n *Notifier) marshalUserGroups(groups []string) (string, error) {
	groupsBytes, err := json.Marshal(groups)
	if err != nil {
		return "", err
	}

	return string(groupsBytes), nil
}

func (n *Notifier) notifyDisruption(dis v1beta1.Disruption, event corev1.Event, notifType types.NotificationType) error {
	logger := n.logger

	logger.With("disruptionName", dis.Name, "disruptionNamespace", dis.Namespace)

	if !n.disruptionConfig.Enabled {
		logger.Debug("http notifier: disruption notifications are disabled")
		return nil
	}

	userInfo, err := dis.UserInfo()
	if err != nil {
		n.logger.Warnw("http notifier: no user info in disruption", "error", err)
	}

	notifierEvent := n.buildEvent(&dis, userInfo, event, notifType, logger)

	disruptionStr, err := json.Marshal(dis)
	if err != nil {
		return fmt.Errorf("http notifier: couldn't marshal disruption: %w", err)
	}

	// Support deprecated fields
	notifierEvent.DisruptionName = dis.Name
	notifierEvent.Disruption = string(disruptionStr)
	notifierEvent.TargetsCount = len(dis.Status.TargetInjections)

	notifierDisruptionEvent := &DisruptionEvent{
		TargetsCount:  len(dis.Status.TargetInjections),
		Manifest:      string(disruptionStr),
		NotifierEvent: notifierEvent,
	}

	jsonNotif, err := jsonapi.Marshal(&notifierDisruptionEvent)
	if err != nil {
		logger.Warnw("http notifier: couldn't marshal notification", "notifierDisruptionEvent", notifierDisruptionEvent, "event", event)

		return fmt.Errorf("http notifier: couldn't marshal notification: %w", err)
	}

	logger.With("eventType", event.Type, "message", notifierDisruptionEvent.EventMessage)

	return n.emitEvent(n.disruptionConfig.URL, jsonNotif, logger)
}

func (n *Notifier) notifyDisruptionCron(disruptionCron v1beta1.DisruptionCron, event corev1.Event, notifType types.NotificationType) error {
	logger := n.logger

	logger.With("disruptionCronName", disruptionCron.Name, "disruptionCronNamespace", disruptionCron.Namespace)

	if !n.disruptionCronConfig.Enabled {
		logger.Debug("http notifier: disruption cron notifications are disabled")
		return nil
	}

	userInfo, err := disruptionCron.UserInfo()
	if err != nil {
		n.logger.Warnw("http notifier: no user info in disruptionCron", "error", err)
	}

	notifierEvent := n.buildEvent(&disruptionCron, userInfo, event, notifType, logger)

	disruptionCronStr, err := json.Marshal(disruptionCron)
	if err != nil {
		return fmt.Errorf("http notifier: couldn't marshal disruptionCron: %w", err)
	}

	notifierDisruptionCronEvent := &DisruptionCronEvent{
		Manifest:      string(disruptionCronStr),
		NotifierEvent: notifierEvent,
	}

	jsonNotification, err := jsonapi.Marshal(&notifierDisruptionCronEvent)
	if err != nil {
		logger.Warnw("http notifier: couldn't marshal notification", "notifierDisruptionCronEvent", notifierDisruptionCronEvent, "event", event)
		return fmt.Errorf("http notifier: couldn't marshal notification: %w", err)
	}

	logger.With("eventType", event.Type, "message", notifierEvent.EventMessage)

	return n.emitEvent(n.disruptionCronConfig.URL, jsonNotification, logger)
}

func (n *Notifier) buildEvent(obj client.Object, uInfo v1.UserInfo, event corev1.Event, notifType types.NotificationType, logger *zap.SugaredLogger) NotifierEvent {
	username, userEmail, userGroups := n.getUserDetails(uInfo, logger)

	return NotifierEvent{
		ID:                 string(obj.GetUID()),
		Name:               obj.GetName(),
		NotificationTitle:  utils.BuildHeaderMessageFromObjectEvent(obj, event, notifType),
		NotificationType:   notifType,
		EventReason:        v1beta1.GetEventReason(event),
		EventMessage:       utils.BuildBodyMessageFromObjectEvent(obj, event, false),
		InvolvedObjectKind: obj.GetObjectKind().GroupVersionKind().Kind,
		Timestamp:          event.FirstTimestamp.UnixNano(),
		Cluster:            n.common.ClusterName,
		Namespace:          obj.GetNamespace(),
		Username:           username,
		UserEmail:          userEmail,
		UserGroups:         userGroups,
	}
}

func (n *Notifier) emitEvent(url string, jsonNotif []byte, logger *zap.SugaredLogger) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonNotif))
	if err != nil {
		return fmt.Errorf("http notifier: couldn't send notification: %w", err)
	}

	for headerKey, headerValue := range n.headers {
		req.Header.Add(headerKey, headerValue)
	}

	if n.authTokenProvider != nil {
		token, err := n.authTokenProvider.AuthToken(context.Background())
		if err != nil {
			return fmt.Errorf("http notifier: unable to retrieve auth token through helper: %w", err)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	logger.Debug("http notifier: sending notifier event to http")

	res, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("http notifier: error when sending notification: %w", err)
	}

	logger.Debugw("http notifier: received response from http", "status", res.StatusCode)

	if res.StatusCode >= http.StatusMultipleChoices || res.StatusCode < http.StatusOK {
		return fmt.Errorf("http notifier: receiving %d status code from sent notification", res.StatusCode)
	}

	defer func() {
		if err = res.Body.Close(); err != nil {
			logger.Warnw("http notifier: error closing body", "error", err)
		}
	}()

	return nil
}
