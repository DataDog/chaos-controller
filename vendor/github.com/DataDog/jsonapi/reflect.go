package jsonapi

import (
	"fmt"
	"reflect"
	"runtime/debug"
)

func derefValue(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Pointer:
		if v.Elem().Kind() == reflect.Invalid {
			v.Set(reflect.New(v.Type().Elem()))
			return v.Elem()
		}
		return derefValue(v.Elem())
	}
	return v
}

func derefType(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Pointer:
		return derefType(t.Elem())
	}
	return t
}

func recoverError(rvr any) error {
	var err error
	switch e := rvr.(type) {
	case error:
		err = fmt.Errorf("unknown error: %w %s", e, debug.Stack())
	default:
		err = fmt.Errorf("%v %s", e, debug.Stack())
	}
	return err
}

func setFieldValue(fv reflect.Value, v any) {
	vv := reflect.ValueOf(v)

	// if the field is not a pointer, dereference the value fully in case
	// it is a pointer (likely returned by reflect.New)
	if fv.Kind() != reflect.Pointer {
		vv = derefValue(vv)
	}
	fv.Set(vv)
}
