// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"encoding/json"
	"fmt"
	"strings"

	authV1 "k8s.io/api/authentication/v1"
	"k8s.io/api/authentication/v1beta1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/chaos-controller/o11y/tags"
)

const (
	annotationUserInfoKey = "UserInfo"
)

var (
	// ErrNoUserInfo is the error returned when the annotation does not contains expected user info key
	ErrNoUserInfo = fmt.Errorf("user info not found in annotations")
)

// UserInfo returns extracted user info informations or an error if not available or invalid
func (d *Disruption) UserInfo() (authV1.UserInfo, error) {
	return getUserInfo(d.GetObjectMeta())
}

// SetUserInfo store provided userInfo into expected disruption annotation
func (d *Disruption) SetUserInfo(userInfo authV1.UserInfo) error {
	return setUserInfo(userInfo, d)
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

func filterUserGroupsByPermitted(userInfo authV1.UserInfo, permittedGroups map[string]struct{}) []string {
	var filteredGroups []string

	for _, group := range userInfo.Groups {
		if _, ok := permittedGroups[group]; ok {
			filteredGroups = append(filteredGroups, group)
		}
	}

	return filteredGroups
}

func getUserInfoFromObject(object metaV1.Object) (authV1.UserInfo, error) {
	switch d := object.(type) {
	case *Disruption:
		return d.UserInfo()
	case *DisruptionCron:
		return d.UserInfo()
	}

	return authV1.UserInfo{}, fmt.Errorf("unable to extract user info from object")
}

// validateUserInfoImmutable checks that no changes have been made to the oldDisruption's UserInfo in the latest update
func validateUserInfoImmutable(oldObject, newObject client.Object) error {
	oldUserInfo, err := getUserInfoFromObject(oldObject)
	if err != nil {
		return nil
	}

	emptyUserInfo := fmt.Sprintf("%v", v1beta1.UserInfo{})
	if fmt.Sprintf("%v", oldUserInfo) == emptyUserInfo {
		return nil
	}

	userInfo, err := getUserInfoFromObject(newObject)
	if err != nil {
		return err
	}

	if fmt.Sprintf("%v", userInfo) != fmt.Sprintf("%v", oldUserInfo) {
		return fmt.Errorf("the user info annotation is immutable")
	}

	return nil
}

// validateUserInfoGroup checks that if permittedGroups is set, which is controlled in controller.safeMode.permittedUserGroups in the configmap,
// then we will return an error if the user in r.UserInfo does not belong to any of the permitted. If permittedGroups is unset, or if the user belongs to one of those
// groups, then we will return nil
func validateUserInfoGroup(object client.Object, permittedGroups map[string]struct{}, permittedGroupsString string) error {
	if len(permittedGroups) == 0 {
		return nil
	}

	userInfo, err := getUserInfoFromObject(object)
	if err != nil {
		return err
	}

	objectKind := object.GetObjectKind().GroupVersionKind().Kind

	if allowedGroups := filterUserGroupsByPermitted(userInfo, permittedGroups); len(allowedGroups) > 0 {
		logger.Debugw(fmt.Sprintf("permitting user %s creation, due to group membership", objectKind),
			tags.GroupsKey, strings.Join(allowedGroups, ", "),
		)

		return nil
	}

	logger.Warnw(fmt.Sprintf("rejecting user from creating this %s", objectKind),
		tags.PermittedUserGroupsKey, permittedGroupsString,
		tags.UserGroupsKey, userInfo.Groups,
	)

	return fmt.Errorf(
		"lacking sufficient authorization to create %s. your user groups are %s, but you must be in one of the following groups: %s",
		objectKind,
		strings.Join(userInfo.Groups, ", "),
		permittedGroupsString,
	)
}
