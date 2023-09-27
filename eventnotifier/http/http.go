// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

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
	"sort"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/eventnotifier/utils"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	coretypes "k8s.io/apimachinery/pkg/types"
)

type NotifierHTTPConfig struct {
	Enabled         bool
	URL             string
	Headers         []string
	HeadersFilepath string
	HasDetails      bool
	FilteredReasons []string
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
	hasDetails        bool
	filteredReasons   map[string]struct{}
	authTokenProvider BearerAuthTokenProvider
}

type HTTPNotifierEvent struct {
	NotificationTitle  string                    `json:"notification-title"`
	NotificationType   types.NotificationType    `json:"notification-type"`
	EventMessage       string                    `json:"event-message"`
	InvolvedObjectKind string                    `json:"involved-object-kind"`
	DisruptionName     string                    `json:"disruption-name"`
	Cluster            string                    `json:"cluster"`
	Namespace          string                    `json:"namespace"`
	TargetsCount       int                       `json:"targets-count"`
	Username           string                    `json:"username,omitempty"`
	UserEmail          string                    `json:"user-email,omitempty"`
	Details            *HTTPNotifierEventDetails `json:"details,omitempty"`
}

// HTTPNotifierEventDetails contains detailed informations we might or might not keep in the long term
// they are isolated behind a feature flag and won't be provided by default
type HTTPNotifierEventDetails struct {
	Disruption v1beta1.Disruption        `json:"disruption,omitempty"`
	Targets    []HTTPNotifierEventTarget `json:"targets,omitempty"`
}

// HTTPNotifierEventTarget contains detailed informations about a target of a disruption and associated chaos pod
type HTTPNotifierEventTarget struct {
	Name              string            `json:"name,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	DisruptionPodName string            `json:"disruption-pod-name,omitempty"`
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

	mapFilteredReasons := make(map[string]struct{}, len(httpConfig.FilteredReasons))
	for _, reason := range httpConfig.FilteredReasons {
		mapFilteredReasons[reason] = struct{}{}
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
		hasDetails:        httpConfig.HasDetails,
		filteredReasons:   mapFilteredReasons,
		authTokenProvider: authTokenProvider,
	}, nil
}

// GetNotifierName returns the driver's name
func (n *Notifier) GetNotifierName() string {
	return string(types.NotifierDriverHTTP)
}

// NotifyWarning generates a notification for generic k8s Warning events
func (n *Notifier) Notify(dis v1beta1.Disruption, event corev1.Event, notifType types.NotificationType) error {
	if len(n.filteredReasons) != 0 {
		if _, ok := n.filteredReasons[event.Reason]; !ok {
			return nil
		}
	}

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

	if n.hasDetails {
		notif.Details = &HTTPNotifierEventDetails{
			Disruption: dis,
			Targets:    make([]HTTPNotifierEventTarget, 0, len(dis.Status.TargetInjections)),
		}

		for targetName, targetInjection := range dis.Status.TargetInjections {
			target := HTTPNotifierEventTarget{
				Name:              targetName,
				DisruptionPodName: targetInjection.InjectorPodName,
			}

			if n.common.Client != nil {
				if dis.Spec.Level == chaostypes.DisruptionLevelNode {
					node := corev1.Node{}
					if err := n.common.Client.Get(context.Background(), coretypes.NamespacedName{Namespace: dis.Namespace, Name: targetName}, &node); err != nil {
						if apierrors.IsNotFound(err) {
							continue
						}

						return err
					}

					target.Labels = node.Labels
				} else {
					pod := corev1.Pod{}
					if err := n.common.Client.Get(context.Background(), coretypes.NamespacedName{Namespace: dis.Namespace, Name: targetName}, &pod); err != nil {
						if apierrors.IsNotFound(err) {
							continue
						}

						return err
					}

					target.Labels = pod.Labels
				}
			}

			notif.Details.Targets = append(notif.Details.Targets, target)
		}

		// Ensure we return consistent ordered results (looping over a map does not provide a consistent ordering)
		sort.Slice(notif.Details.Targets, func(i, j int) bool {
			return notif.Details.Targets[i].Name < notif.Details.Targets[j].Name
		})
	}

	body, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("http notifier: couldn't send notification: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, n.url, bytes.NewReader(body))
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

	res, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("http notifier: error when sending notification: %w", err)
	}

	defer func() {
		if err = res.Body.Close(); err != nil {
			n.logger.Warnw("http notifier: error closing body", "error", err)
		}
	}()

	// ensure underlying roundTripper is able to reuse connection if possible
	_, err = io.Copy(io.Discard, res.Body)
	if err != nil {
		return fmt.Errorf("http notifier: error reading body: %w", err)
	}

	n.logger.Debugw("notifier: sending notifier event to http", "disruption", dis.Name, "eventType", event.Type, "message", notif.EventMessage)

	if res.StatusCode >= 300 || res.StatusCode < 200 {
		return fmt.Errorf("http notifier: receiving %d status code from sent notification", res.StatusCode)
	}

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
