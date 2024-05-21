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
	"github.com/google/jsonapi"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

type NotifierHTTPConfig struct {
	Enabled         bool
	URL             string
	Headers         []string
	HeadersFilepath string
	AuthURL         string
	AuthHeaders     []string
	AuthTokenPath   string
}

// Notifier describes a HTTP notifier
type Notifier struct {
	common            types.NotifiersCommonConfig
	client            *http.Client
	url               string
	headers           map[string]string
	logger            *zap.SugaredLogger
	authTokenProvider BearerAuthTokenProvider
}

type HTTPNotifierEvent struct {
	ID                 string                 `jsonapi:"primary,http_notifier_events"`
	NotificationTitle  string                 `jsonapi:"attr,notification_title"`
	NotificationType   types.NotificationType `jsonapi:"attr,notification_type"`
	EventMessage       string                 `jsonapi:"attr,event_message"`
	InvolvedObjectKind string                 `jsonapi:"attr,involved_object_kind"`
	DisruptionName     string                 `jsonapi:"attr,disruption_name"`
	Disruption         string                 `jsonapi:"attr,disruption"`
	Timestamp          int64                  `jsonapi:"attr,timestamp"`
	Cluster            string                 `jsonapi:"attr,cluster"`
	Namespace          string                 `jsonapi:"attr,namespace"`
	TargetsCount       int                    `jsonapi:"attr,targets_count"`
	Username           string                 `jsonapi:"attr,username,omitempty"`
	UserEmail          string                 `jsonapi:"attr,user_email,omitempty"`
}

// New HTTP Notifier
func New(commonConfig types.NotifiersCommonConfig, httpConfig NotifierHTTPConfig, logger *zap.SugaredLogger) (*Notifier, error) {
	headers := []string{}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	if httpConfig.URL == "" {
		return nil, fmt.Errorf("notifier http: missing URL")
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

		authTokenProvider = NewBearerAuthTokenProvider(logger, client, httpConfig.AuthURL, authHeaders, httpConfig.AuthTokenPath)
	}

	return &Notifier{
		common:            commonConfig,
		client:            client,
		url:               httpConfig.URL,
		headers:           parsedHeaders,
		logger:            logger,
		authTokenProvider: authTokenProvider,
	}, nil
}

// GetNotifierName returns the driver's name
func (n *Notifier) GetNotifierName() string {
	return string(types.NotifierDriverHTTP)
}

// Notify generates a notification for generic k8s Warning events
func (n *Notifier) Notify(dis v1beta1.Disruption, event corev1.Event, notifType types.NotificationType) error {
	emailAddr := &mail.Address{}

	if userInfo, err := dis.UserInfo(); err != nil {
		n.logger.Warnw("http notifier: no user info in disruption", "disruptionName", dis.Name, "disruptionNamespace", dis.Namespace, "error", err)
	} else {
		if userInfoEmailAddr, err := mail.ParseAddress(userInfo.Username); err != nil {
			n.logger.Warnw("http notifier: user info username is not a valid email address", "disruptionName", dis.Name, "disruptionNamespace", dis.Namespace, "error", err, "username", userInfo.Username)
		} else {
			emailAddr = userInfoEmailAddr
		}
	}

	disruptionStr, err := json.Marshal(dis)
	if err != nil {
		return fmt.Errorf("http notifier: couldn't marshal disruption: %w", err)
	}

	now := time.Now()

	notif := HTTPNotifierEvent{
		ID:                 string(dis.UID),
		NotificationTitle:  utils.BuildHeaderMessageFromDisruptionEvent(dis, notifType),
		NotificationType:   notifType,
		EventMessage:       utils.BuildBodyMessageFromDisruptionEvent(dis, event, false),
		InvolvedObjectKind: dis.Kind,
		DisruptionName:     dis.Name,
		Disruption:         string(disruptionStr),
		Timestamp:          now.UnixNano(),
		Cluster:            n.common.ClusterName,
		Namespace:          dis.Namespace,
		TargetsCount:       len(dis.Status.TargetInjections),
		Username:           emailAddr.Name,
		UserEmail:          emailAddr.Address,
	}

	body := bytes.NewBuffer(nil)
	if err := jsonapi.MarshalOnePayloadEmbedded(body, &notif); err != nil {
		return fmt.Errorf("http notifier: couldn't marshal notification: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, n.url, body)
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

	n.logger.Debugw("http notifier: sending notifier event to http", "disruption", dis.Name, "eventType", event.Type, "message", notif.EventMessage)

	res, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("http notifier: error when sending notification: %w", err)
	}

	n.logger.Debugw("http notifier: received response from http", "status", res.StatusCode)

	if res.StatusCode >= 300 || res.StatusCode < 200 {
		return fmt.Errorf("http notifier: receiving %d status code from sent notification", res.StatusCode)
	}

	defer func() {
		if err = res.Body.Close(); err != nil {
			n.logger.Warnw("http notifier: error closing body", "error", err)
		}
	}()

	return nil
}

func splitHeaders(headers []string) (map[string]string, error) {
	parsedHeaders := make(map[string]string)

	for _, header := range headers {
		if header == "" {
			continue
		}

		splittedHeader := strings.Split(header, ":")
		if len(splittedHeader) == 2 {
			parsedHeaders[splittedHeader[0]] = splittedHeader[1]
		} else {
			return nil, fmt.Errorf("invalid headers: Must be in the format: key:value, found %s", header)
		}
	}

	return parsedHeaders, nil
}
