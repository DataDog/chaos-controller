// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
package v1beta1

import (
	"encoding/json"
	"fmt"

	authV1 "k8s.io/api/authentication/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	annotationUserInfoKey = "UserInfo"
)

var (
	// ErrNoUserInfo is the error returned when the annotation does not contains expected user info key
	ErrNoUserInfo = fmt.Errorf("user info not found in annotations")
)

// UserInfo returns extracted user info informations or an error if not available or invalid
func (r *Disruption) UserInfo() (authV1.UserInfo, error) {
	return getUserInfo(r.GetObjectMeta())
}

// SetUserInfo store provided userInfo into expected disruption annotation
func (r *Disruption) SetUserInfo(userInfo authV1.UserInfo) error {
	return setUserInfo(userInfo, r)
}

// UserInfo returns extracted user info informations or an error if not available or invalid
func (d *DisruptionCron) UserInfo() (authV1.UserInfo, error) {
	return getUserInfo(d.GetObjectMeta())
}

// SetUserInfo store provided userInfo into expected disruption annotation
func (d *DisruptionCron) SetUserInfo(userInfo authV1.UserInfo) error {
	return setUserInfo(userInfo, d)
}

func getUserInfo(object metaV1.Object) (authV1.UserInfo, error) {
	var uInfo authV1.UserInfo
	if userInfo, ok := object.GetAnnotations()[annotationUserInfoKey]; ok {
		err := json.Unmarshal([]byte(userInfo), &uInfo)
		if err != nil {
			return uInfo, fmt.Errorf("unable to unmarshal user info from uInfo: %w", err)
		}

		return uInfo, nil
	}

	return uInfo, ErrNoUserInfo
}

func setUserInfo(userInfo authV1.UserInfo, obj metaV1.Object) error {
	marshaledUserInfo, err := json.Marshal(userInfo)
	if err != nil {
		return fmt.Errorf("unable to marshal user info: %w", err)
	}

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[annotationUserInfoKey] = string(marshaledUserInfo)

	obj.SetAnnotations(annotations)

	return nil
}
