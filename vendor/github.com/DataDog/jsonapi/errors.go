package jsonapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var (
	// ErrMarshalInvalidPrimaryField indicates that the id (primary) fields was invalid.
	ErrMarshalInvalidPrimaryField = errors.New("primary/id field must be a string or implement fmt.Stringer or in a struct which implements MarshalIdentifier")

	// ErrUnmarshalInvalidPrimaryField indicates that the id (primary) fields was invalid.
	ErrUnmarshalInvalidPrimaryField = errors.New("primary/id field must be a string or in a struct which implements UnmarshalIdentifer")

	// ErrUnmarshalDuplicatePrimaryField indicates that the id (primary) field is duplicated in a struct.
	ErrUnmarshalDuplicatePrimaryField = errors.New("there must be only one `jsonapi:\"primary\"` field to Unmarshal")

	// ErrMissingPrimaryField indicates that the id (primary) field is not identified.
	ErrMissingPrimaryField = errors.New("primary/id field must labeled with `jsonapi:\"primary,{type}\"`")

	// ErrEmptyPrimaryField indicates that the id (primary) field is identified but empty.
	ErrEmptyPrimaryField = errors.New("the `jsonapi:\"primary\"` field value must not be empty")

	// ErrMissingLinkFields indicates that a LinkObject is not valid.
	ErrMissingLinkFields = errors.New("at least one of Links.Self or Links.Related must be set to a nonempty string or *LinkObject")

	// ErrEmptyDataObject indicates that a primary or relationship data member is incorrectly represented by an empty JSON object {}
	ErrEmptyDataObject = errors.New("resource \"data\" members may not be represented by an empty object {}")

	// ErrDocumentMissingRequiredMembers indicates that a document does not have at least one required top-level member
	ErrDocumentMissingRequiredMembers = errors.New("document is missing required top-level members; must have one of: \"data\", \"meta\", \"errors\"")

	// ErrRelationshipMissingRequiredMembers indicates that a relationship does not have at least one required member
	ErrRelationshipMissingRequiredMembers = errors.New("relationship is missing required top-level members; must have one of: \"data\", \"meta\", \"links\"")

	// ErrErrorUnmarshalingNotImplemented indicates that an attempt was made to unmarshal an error document
	ErrErrorUnmarshalingNotImplemented = errors.New("error unmarshaling is not implemented")
)

// TypeError indicates that an unexpected type was encountered.
type TypeError struct {
	Actual   string
	Expected []string
}

// Error implements the error interface.
func (e *TypeError) Error() string {
	if len(e.Expected) > 0 {
		return fmt.Sprintf("got type %q expected one of %q", e.Actual, strings.Join(e.Expected, ","))
	}
	return fmt.Sprintf("got type %q expected %q", e.Actual, e.Expected[0])
}

// TagError indicates that an invalid struct tag was encountered.
type TagError struct {
	TagName string
	Field   string
	Reason  string
}

// Error implements the error interface.
func (e *TagError) Error() string {
	return fmt.Sprintf("invalid %q tag on field %q: %s", e.TagName, e.Field, e.Reason)
}

// PartialLinkageError indicates that an incomplete relationship chain was encountered.
type PartialLinkageError struct {
	invalidResources []string
}

// Error implements the error interface.
func (e *PartialLinkageError) Error() string {
	sort.Strings(e.invalidResources)
	return fmt.Sprintf(
		"the following resources have no chain of relationships from primary data: %q",
		strings.Join(e.invalidResources, ","),
	)
}

// MemberNameValidationError indicates that a document member name failed a validation step.
type MemberNameValidationError struct {
	MemberName string
}

// Error implements the error interface.
func (e *MemberNameValidationError) Error() string {
	return fmt.Sprintf("invalid member name: %s", e.MemberName)
}

// ErrorLink represents a JSON:API error links object as defined by https://jsonapi.org/format/1.0/#error-objects.
type ErrorLink struct {
	About any `json:"about,omitempty"`
}

// ErrorSource represents a JSON:API Error.Source as defined by https://jsonapi.org/format/1.0/#error-objects.
type ErrorSource struct {
	Pointer   string `json:"pointer,omitempty"`
	Parameter string `json:"parameter,omitempty"`
}

// Status provides a helper for setting an Error.Status value.
func Status(s int) *int {
	return &s
}

// Error represents a JSON:API error object as defined by https://jsonapi.org/format/1.0/#error-objects.
type Error struct {
	ID     string       `json:"id,omitempty"`
	Links  *ErrorLink   `json:"links,omitempty"`
	Status *int         `json:"status,omitempty"`
	Code   string       `json:"code,omitempty"`
	Title  string       `json:"title,omitempty"`
	Detail string       `json:"detail,omitempty"`
	Source *ErrorSource `json:"source,omitempty"`
	Meta   any          `json:"meta,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface.
func (e *Error) MarshalJSON() ([]byte, error) {
	var status string
	if e.Status != nil {
		status = fmt.Sprintf("%d", *e.Status)
	}

	type alias Error
	return json.Marshal(&struct {
		Status string `json:"status,omitempty"`
		*alias
	}{
		Status: status,
		alias:  (*alias)(e),
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (e *Error) UnmarshalJSON(data []byte) error {
	type alias Error

	aux := &struct {
		*alias
		Status string `json:"status"`
	}{
		alias: (*alias)(e),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Status != "" {
		status, err := strconv.Atoi(aux.Status)
		if err != nil {
			return err
		}

		e.Status = &status
	}

	return nil
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Title, e.Detail)
}
