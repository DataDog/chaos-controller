// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package datadog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/eventnotifier/utils"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

type NotifierHTTPConfig struct {
	Enabled bool
	URL     string
	Headers []string
}

// Notifier describes a HTTP notifier
type Notifier struct {
	common  types.NotifiersCommonConfig
	client  *http.Client
	url     string
	headers map[string][]string
	logger  *zap.SugaredLogger
}

type HTTPNotifierEvent struct {
	NotificationTitle string `json:"notification-title"`
	NotificationType  string `json:"notification-type"`
	EventMessage      string `json:"message"`
	DisruptionKind    string `json:"disruption-kind"`
	DisruptionName    string `json:"disruption-name"`
	Cluster           string `json:"cluster"`
	Namespace         string `json:"namespace"`
	TargetsCount      int    `json:"targets-count"`
	Username          string `json:"username,omitempty"`
	UserEmail         string `json:"user-email,omitempty"`
}

// New HTTP Notifier
func New(commonConfig types.NotifiersCommonConfig, httpConfig NotifierHTTPConfig, logger *zap.SugaredLogger) (*Notifier, error) {
	client := &http.Client{
		Timeout: 1 * time.Minute,
	}

	parsedHeaders := make(map[string][]string)

	// header is of format: key:value, we need to parse it
	for _, header := range httpConfig.Headers {
		splittedHeader := strings.Split(header, ":")
		if len(splittedHeader) == 2 {
			if parsedHeaders[splittedHeader[0]] == nil {
				parsedHeaders[splittedHeader[0]] = []string{}
			}

			parsedHeaders[splittedHeader[0]] = append(parsedHeaders[splittedHeader[0]], splittedHeader[1])
		} else {
			return nil, fmt.Errorf("notifier http: invalid headers in conf. Must be of format: key:value. %s", header)
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

func (n *Notifier) buildAndSendRequest(dis v1beta1.Disruption, event corev1.Event, notificationType string) error {
	emailAddr, err := utils.GetUserInfoFromDisruption(dis)
	if err != nil {
		n.logger.Warnf("http notifier: no userinfo in disruption %s: %v", dis.Name, err)

		emailAddr = &mail.Address{}
	}

	notif := HTTPNotifierEvent{
		NotificationTitle: "Disruption '" + dis.Name + "' encountered an issue.",
		NotificationType:  notificationType,
		EventMessage:      "Disruption " + dis.Name + " emitted the event " + event.Reason + ": " + event.Message,
		DisruptionKind:    dis.Kind,
		DisruptionName:    dis.Name,
		Cluster:           n.common.ClusterName,
		Namespace:         dis.Namespace,
		TargetsCount:      len(dis.Status.Targets),
		Username:          emailAddr.Name,
		UserEmail:         emailAddr.Address,
	}

	body, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("http notifier: couldn't send notification: %s", err.Error())
	}

	req, err := http.NewRequest(http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("http notifier: couldn't send notification: %s", err.Error())
	}

	for headerKey, headerValues := range n.headers {
		for _, headerValue := range headerValues {
			req.Header.Add(headerKey, headerValue)
		}
	}

	res, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("http notifier: couldn't send notification: %s", err.Error())
	}

	if res.StatusCode >= 300 || res.StatusCode < 200 {
		return fmt.Errorf("http notifier: receiving %d status code from sent notification", res.StatusCode)
	}

	return nil
}

// NotifyWarning generates a notification for generic k8s Warning events
func (n *Notifier) NotifyWarning(dis v1beta1.Disruption, event corev1.Event) error {
	return n.buildAndSendRequest(dis, event, event.Type)
}

// NotifyWarning generates a notification for generic k8s Warning events
func (n *Notifier) NotifyRecovery(dis v1beta1.Disruption, event corev1.Event) error {
	return n.buildAndSendRequest(dis, event, event.Type)
}
