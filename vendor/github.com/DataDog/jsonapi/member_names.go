package jsonapi

import (
	"encoding/json"
	"fmt"
	"regexp"
)

var (
	defaultNameRegex *regexp.Regexp
	strictNameRegex  *regexp.Regexp
)

func init() {
	// properties of the default name regex:
	// - at least one character, no case requirements
	// - dash, space, and underscore allowed anywhere except the ends of the string
	// - no characters below unicode 0080 allowed
	defaultNameRegex = regexp.MustCompile(`^([a-zA-Z\d]|[^\x{0000}-\x{0080}])(([a-zA-Z\d]|[^\x{0000}-\x{0080}])|[-_ ]([a-zA-Z\d]|[^\x{0000}-\x{0080}]))*$`)

	// properties of the strict name regex:
	// - at least one lower case letter
	// - camel case, and must end with a lower case letter
	// - may have digits inside the word
	strictNameRegex = regexp.MustCompile(`^[a-z]+(|(\d|([A-Z\d][a-z\d]+))*([A-Z\d][a-z\d]*[a-z]))$`)
}

// MemberNameValidationMode controls how document member names are checked for correctness.
type MemberNameValidationMode int

const (
	// DefaultValidation verifies that member names are valid according to the spec in
	// https://jsonapi.org/format/#document-member-names.
	//
	// Note that this validation mode allows for non-URL-safe member names.
	DefaultValidation MemberNameValidationMode = iota

	// DisableValidation turns off member name validation for convenience or performance-saving
	// reasons.
	//
	// Note that this validation mode allows member names that do NOT conform to the JSON:API spec.
	DisableValidation

	// StrictValidation verifies that member names are valid according to the spec in
	// https://jsonapi.org/format/#document-member-names, and follow recommendations from
	// https://jsonapi.org/recommendations/#naming.
	//
	// Note that these names are always URL-safe.
	StrictValidation
)

func isValidMemberName(name string, mode MemberNameValidationMode) bool {
	switch mode {
	case DisableValidation:
		return true
	case StrictValidation:
		return strictNameRegex.MatchString(name)
	default:
		return defaultNameRegex.MatchString(name)
	}
}

func validateMapMemberNames(m map[string]any, mode MemberNameValidationMode) error {
	for member, val := range m {
		if !isValidMemberName(member, mode) {
			return &MemberNameValidationError{member}
		}
		switch nested := val.(type) {
		case map[string]any:
			if err := validateMapMemberNames(nested, mode); err != nil {
				return err
			}
		case []any:
			for _, entry := range nested {
				if subMap, ok := entry.(map[string]any); ok {
					if err := validateMapMemberNames(subMap, mode); err != nil {
						return err
					}
				}
			}
		default:
			continue
		}
	}
	return nil
}

func validateJSONMemberNames(b []byte, mode MemberNameValidationMode) error {
	// do not unmarshal if validation is disabled
	if mode == DisableValidation {
		return nil
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("unexpected unmarshal failure: %w", err)
	}
	return validateMapMemberNames(m, mode)
}
