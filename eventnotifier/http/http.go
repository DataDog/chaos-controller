// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package datadog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/eventnotifier/utils"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

type NotifierHTTPConfig struct {
	Enabled         bool
	URL             string
	Headers         []string
	HeadersFilepath string
}

// Notifier describes a HTTP notifier
type Notifier struct {
	common  types.NotifiersCommonConfig
	client  *http.Client
	url     string
	headers map[string]string
	logger  *zap.SugaredLogger
}

type HTTPNotifierEvent struct {
	NotificationTitle  string                 `json:"notification-title"`
	NotificationType   types.NotificationType `json:"notification-type"`
	EventMessage       string                 `json:"event-message"`
	InvolvedObjectKind string                 `json:"involved-object-kind"`
	DisruptionName     string                 `json:"disruption-name"`
	Cluster            string                 `json:"cluster"`
	Namespace          string                 `json:"namespace"`
	TargetsCount       int                    `json:"targets-count"`
	Username           string                 `json:"username,omitempty"`
	UserEmail          string                 `json:"user-email,omitempty"`
}

// New HTTP Notifier
func New(commonConfig types.NotifiersCommonConfig, httpConfig NotifierHTTPConfig, logger *zap.SugaredLogger) (*Notifier, error) {
	parsedHeaders := make(map[string]string)
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

		readHeaders, err := ioutil.ReadAll(headersFile)
		if err != nil {
			return nil, fmt.Errorf("headers file could not be read: %w", err)
		}

		sHeaders := string(readHeaders)
		if sHeaders != "" {
			headers = strings.Split(sHeaders, "\n")
		}
	}

	headers = append(headers, httpConfig.Headers...)

	for _, header := range headers {
		if header == "" {
			continue
		}

		splittedHeader := strings.Split(header, ":")
		if len(splittedHeader) == 2 {
			parsedHeaders[splittedHeader[0]] = splittedHeader[1]
		} else {
			return nil, fmt.Errorf("notifier http: invalid headers in headers file. Must be of format: key:value")
		}
	}

	return &Notifier{
		common:  commonConfig,
		client:  client,
		url:     httpConfig.URL,
		headers: parsedHeaders,
		logger:  logger,
	}, nil
}

// GetNotifierName returns the driver's name
func (n *Notifier) GetNotifierName() string {
	return string(types.NotifierDriverHTTP)
}

// NotifyWarning generates a notification for generic k8s Warning events
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

	notif := HTTPNotifierEvent{
		NotificationTitle:  utils.BuildHeaderMessageFromDisruptionEvent(dis, notifType),
		NotificationType:   notifType,
		EventMessage:       utils.BuildBodyMessageFromDisruptionEvent(dis, event, false),
		InvolvedObjectKind: dis.Kind,
		DisruptionName:     dis.Name,
		Cluster:            n.common.ClusterName,
		Namespace:          dis.Namespace,
		TargetsCount:       len(dis.Status.TargetInjections),
		Username:           emailAddr.Name,
		UserEmail:          emailAddr.Address,
	}

	body, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("http notifier: couldn't send notification: %s", err.Error())
	}

	req, err := http.NewRequest(http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("http notifier: couldn't send notification: %s", err.Error())
	}

	for headerKey, headerValue := range n.headers {
		req.Header.Add(headerKey, headerValue)
	}

	res, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("http notifier: error when sending notification: %s", err.Error())
	}

	n.logger.Debugw("notifier: sending notifier event to http", "disruption", dis.Name, "eventType", event.Type, "message", notif.EventMessage)

	if res.StatusCode >= 300 || res.StatusCode < 200 {
		return fmt.Errorf("http notifier: receiving %d status code from sent notification", res.StatusCode)
	}

	return nil
}
