package validation

import "reflect"

type DDValidationMarker interface {
	ApplyRule(reflect.Value) error
}
