package v1beta1

import (
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/authentication/v1"
)

const (
	annotationUserInfoKey = "UserInfo"
)

var (
	// ErrNoUserInfo is the error returned when the annotation does not contains expected user info key
	ErrNoUserInfo = fmt.Errorf("user info not found in disruption annotations")
)

// UserInfo returns extracted user info informations or an error if not available or invalid
func (r *Disruption) UserInfo() (v1.UserInfo, error) {
	var annotation v1.UserInfo
	if userInfo, ok := r.Annotations[annotationUserInfoKey]; ok {
		err := json.Unmarshal([]byte(userInfo), &annotation)
		if err != nil {
			return annotation, fmt.Errorf("unable to unmarshal user info from annotation: %w", err)
		}

		return annotation, nil
	}

	return annotation, ErrNoUserInfo
}

// SetUserInfo store provided userInfo into expected disruption annotation
func (r *Disruption) SetUserInfo(userInfo v1.UserInfo) error {
	marshaledUserInfo, err := json.Marshal(userInfo)
	if err != nil {
		return fmt.Errorf("unable to marshal user info: %w", err)
	}

	r.Annotations[annotationUserInfoKey] = string(marshaledUserInfo)

	return nil
}
