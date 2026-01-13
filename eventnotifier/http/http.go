// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

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

	"github.com/DataDog/jsonapi"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/eventnotifier/utils"
	cLog "github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/o11y/tags"
)

type DisruptionConfig struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
}

type DisruptionCronConfig struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
}

type Config struct {
	Headers         []string             `yaml:"headers"`
	HeadersFilepath string               `yaml:"headersFilepath"`
	AuthURL         string               `yaml:"authURL"`
	AuthHeaders     []string             `yaml:"authHeaders"`
	AuthTokenPath   string               `yaml:"authTokenPath"`
	Disruption      DisruptionConfig     `yaml:"disruption"`
	DisruptionCron  DisruptionCronConfig `yaml:"disruptioncron"`
}

func (c Config) IsEnabled() bool {
	return c.DisruptionCron.Enabled || c.Disruption.Enabled
}

// Notifier describes a HTTP notifier
type Notifier struct {
	common               types.NotifiersCommonConfig
	client               *http.Client
	headers              map[string]string
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
func New(commonConfig types.NotifiersCommonConfig, httpConfig Config) (*Notifier, error) {
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

		authTokenProvider = NewBearerAuthTokenProvider(httpClient, httpConfig.AuthURL, authHeaders, httpConfig.AuthTokenPath)
	}

	return &Notifier{
		common:               commonConfig,
		client:               httpClient,
		headers:              parsedHeaders,
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
func (n *Notifier) Notify(ctx context.Context, obj client.Object, event corev1.Event, notifType types.NotificationType) error {
	switch d := obj.(type) {
	case nil:
		return nil
	case *v1beta1.Disruption:
		return n.notifyDisruption(ctx, *d, event, notifType)
	case *v1beta1.DisruptionCron:
		return n.notifyDisruptionCron(ctx, *d, event, notifType)
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
func (n *Notifier) getUserDetails(ctx context.Context, uInfo authv1.UserInfo) (username, emailAddr, userGroups string) {
	username = uInfo.Username
	emailAddr, err := n.extractEmail(username)

	logger := cLog.FromContext(ctx).With(tags.UsernameKey, username)

	if err != nil {
		logger.Infow("http notifier: user info username is not a valid email address", tags.ErrorKey, err)
	}

	userGroups, err = n.marshalUserGroups(uInfo.Groups)
	if err != nil {
		logger.Warnw("http notifier: couldn't marshal user groups", tags.ErrorKey, err)
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

func (n *Notifier) notifyDisruption(ctx context.Context, dis v1beta1.Disruption, event corev1.Event, notifType types.NotificationType) error {
	logger := cLog.FromContext(ctx).With(
		tags.DisruptionNameKey, dis.Name,
		tags.DisruptionNamespaceKey, dis.Namespace,
		tags.EventTypeKey, event.Type,
		tags.EventKey, event,
	)
	ctx = cLog.WithLogger(ctx, logger)

	if !n.disruptionConfig.Enabled {
		return nil
	}

	userInfo, err := dis.UserInfo()
	if err != nil {
		logger.Warnw("http notifier: no user info in disruption", tags.ErrorKey, err)
	}

	notifierEvent := n.buildEvent(ctx, &dis, userInfo, event, notifType)

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
		logger.Warnw("http notifier: couldn't marshal notification",
			tags.NotifierDisruptionEventKey, notifierDisruptionEvent,
		)

		return fmt.Errorf("http notifier: couldn't marshal notification: %w", err)
	}

	logger = logger.With(tags.MessageKey, notifierDisruptionEvent.EventMessage)
	ctx = cLog.WithLogger(ctx, logger)

	return n.emitEvent(ctx, n.disruptionConfig.URL, jsonNotif)
}

func (n *Notifier) notifyDisruptionCron(ctx context.Context, disruptionCron v1beta1.DisruptionCron, event corev1.Event, notifType types.NotificationType) error {
	logger := cLog.FromContext(ctx).With(
		tags.DisruptionCronNameKey, disruptionCron.Name,
		tags.DisruptionCronNamespaceKey, disruptionCron.Namespace,
		tags.EventKey, event,
		tags.EventTypeKey, event.Type,
	)
	ctx = cLog.WithLogger(ctx, logger)

	if !n.disruptionCronConfig.Enabled {
		return nil
	}

	userInfo, err := disruptionCron.UserInfo()
	if err != nil {
		logger.Warnw("http notifier: no user info in disruptionCron", tags.ErrorKey, err)
	}

	notifierEvent := n.buildEvent(ctx, &disruptionCron, userInfo, event, notifType)

	logger = logger.With(tags.MessageKey, notifierEvent.EventMessage)
	ctx = cLog.WithLogger(ctx, logger)

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
		logger.Warnw("http notifier: couldn't marshal notification", tags.NotifierDisruptionCronEventKey, notifierDisruptionCronEvent)
		return fmt.Errorf("http notifier: couldn't marshal notification: %w", err)
	}

	return n.emitEvent(ctx, n.disruptionCronConfig.URL, jsonNotification)
}

func (n *Notifier) buildEvent(ctx context.Context, obj client.Object, uInfo authv1.UserInfo, event corev1.Event, notifType types.NotificationType) NotifierEvent {
	username, userEmail, userGroups := n.getUserDetails(ctx, uInfo)

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

func (n *Notifier) emitEvent(ctx context.Context, url string, jsonNotif []byte) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonNotif))
	if err != nil {
		return fmt.Errorf("http notifier: couldn't send notification: %w", err)
	}

	for headerKey, headerValue := range n.headers {
		req.Header.Add(headerKey, headerValue)
	}

	if n.authTokenProvider != nil {
		token, err := n.authTokenProvider.AuthToken(ctx)
		if err != nil {
			return fmt.Errorf("http notifier: unable to retrieve auth token through helper: %w", err)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	res, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("http notifier: error when sending notification: %w", err)
	}

	if res.StatusCode >= http.StatusMultipleChoices || res.StatusCode < http.StatusOK {
		return fmt.Errorf("http notifier: receiving %d status code from sent notification", res.StatusCode)
	}

	defer func() {
		if err = res.Body.Close(); err != nil {
			cLog.FromContext(ctx).Warnw("http notifier: error closing body", tags.ErrorKey, err)
		}
	}()

	return nil
}
