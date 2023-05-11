// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package ddmark

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	k8smarkers "sigs.k8s.io/controller-tools/pkg/markers"
)

// AllDefinitions contains all marker definitions for this package.
var AllDefinitions []*k8smarkers.Definition
var rulePrefix = "ddmark:validation:"

func init() {
	addDefinition(Maximum(0), k8smarkers.DescribesField)
	addDefinition(Minimum(0), k8smarkers.DescribesField)
	addDefinition(Enum(nil), k8smarkers.DescribesField)
	addDefinition(Required(true), k8smarkers.DescribesField)

	addDefinition(ExclusiveFields(nil), k8smarkers.DescribesType)
	addDefinition(LinkedFieldsValue(nil), k8smarkers.DescribesType)
	addDefinition(LinkedFieldsValueWithTrigger(nil), k8smarkers.DescribesType)
	addDefinition(AtLeastOneOf(nil), k8smarkers.DescribesType)
}

// Maximum can applied to an int field and provides a (non-strict) maximum value for that field
type Maximum int

// Minimum can applied to an int field and provides a (non-strict) minimum value for that field
type Minimum int

// Enum can be applied to any interface field and provides a restricted amount of possible values for that field.
// Values within the marker strictly need to fit the given field interface. Usage is recommended to simple types.
type Enum []interface{}

// Required can be applied to any field, and asserts this field will return an error if not provided
type Required bool

// ExclusiveFields can be applied to structs, and asserts that the first field can only be non-'nil' iff all of the other fields are 'nil'
type ExclusiveFields []string

// LinkedFieldsValue can be applied to structs, and asserts the fields in the list are either all 'nil' or all non-'nil'
type LinkedFieldsValue []string

// LinkedFieldsValueWithTrigger can be applied to structs, and asserts the following:
// - if first field exists (or has the indicated value), all the following fields need to exist (or have the indicated value)
// - fields in question can be int or strings
type LinkedFieldsValueWithTrigger []string

// AtLeastOneOf can be applied to structs, and asserts at least one of the following fields is non-'nil'
type AtLeastOneOf []string

func (m Maximum) ApplyRule(fieldvalue reflect.Value) error {
	fieldvalue = reflect.Indirect(fieldvalue)
	fieldInt, ok := parseIntOrUInt(fieldvalue)

	if !ok {
		return m.TypeCheckError(fieldvalue)
	}

	if int(m) < fieldInt {
		return m.ValueCheckError()
	}

	return nil
}

func (m Maximum) ValueCheckError() error {
	return fmt.Errorf("%s: max value for field is %d (included)", ruleName(m), m)
}

func (m Maximum) TypeCheckError(fieldValue reflect.Value) error {
	return GenericTypeCheckError(m, fieldValue, "int or uint")
}

func (m Minimum) ApplyRule(fieldvalue reflect.Value) error {
	fieldvalue = reflect.Indirect(fieldvalue)
	fieldInt, ok := parseIntOrUInt(fieldvalue)

	if !ok {
		return m.TypeCheckError(fieldvalue)
	}

	if int(m) > fieldInt {
		return m.ValueCheckError()
	}

	return nil
}

func (m Minimum) ValueCheckError() error {
	return fmt.Errorf("%s: min value for field is %d (included)", ruleName(m), m)
}

func (m Minimum) TypeCheckError(fieldValue reflect.Value) error {
	return GenericTypeCheckError(m, fieldValue, "int or uint")
}

func (e ExclusiveFields) ApplyRule(fieldvalue reflect.Value) error {
	fieldvalue = reflect.Indirect(fieldvalue)

	matchCount := 0

	structMap, ok := structValueToMap(fieldvalue)
	if !ok {
		return e.TypeCheckError(fieldvalue)
	}

	if structMap[e[0]] != nil {
		for _, item := range e[1:] {
			if structMap[item] != nil {
				matchCount++
			}
		}
	}

	if matchCount >= 1 {
		return e.ValueCheckError()
	}

	return nil
}

func (e ExclusiveFields) ValueCheckError() error {
	return fmt.Errorf("%s: some fields are incompatible, %s can't be set alongside any of %s", ruleName(e), e[0], e[1:])
}

func (e ExclusiveFields) TypeCheckError(fieldValue reflect.Value) error {
	return GenericTypeCheckError(e, fieldValue, "struct")
}

func (e Enum) ApplyRule(fieldvalue reflect.Value) error {
	fieldvalue = reflect.Indirect(fieldvalue)
	fieldInterface := fieldvalue.Interface()

	for _, markerInterface := range e {
		if !reflect.ValueOf(markerInterface).Type().ConvertibleTo(fieldvalue.Type()) {
			return e.TypeCheckError(fieldvalue)
		}

		markerInterface = reflect.ValueOf(markerInterface).Convert(fieldvalue.Type()).Interface()

		if fieldInterface == markerInterface || reflect.ValueOf(fieldInterface).IsZero() {
			return nil
		}
	}

	return e.ValueCheckError()
}

func (e Enum) ValueCheckError() error {
	return fmt.Errorf("%s: field needs to be one of %v", ruleName(e), e)
}

func (e Enum) TypeCheckError(fieldValue reflect.Value) error {
	return fmt.Errorf("%v: Type Error - field needs to be one of %v, currently \"%v\"", ruleName(e), e, fieldValue)
}

func (r Required) ApplyRule(fieldvalue reflect.Value) error {
	if !bool(r) {
		return nil
	}

	if fieldvalue.Kind() == reflect.Ptr && (!fieldvalue.IsNil() || !fieldvalue.IsZero()) {
		return nil
	}

	fieldvalue = reflect.Indirect(fieldvalue)
	if !fieldvalue.IsValid() || fieldvalue.IsZero() {
		return r.ValueCheckError()
	}

	return nil
}

func (r Required) ValueCheckError() error {
	return fmt.Errorf("%s: field is required: currently missing", ruleName(r))
}

func (r Required) TypeCheckError(fieldValue reflect.Value) error {
	return nil
}

func (l LinkedFieldsValue) ApplyRule(fieldvalue reflect.Value) error {
	fieldvalue = reflect.Indirect(fieldvalue)

	var matchCount = 0

	structMap, ok := structValueToMap(fieldvalue)
	if !ok {
		return l.TypeCheckError(fieldvalue)
	}

	for _, item := range l {
		res, err := checkValueExistsOrIsValid(item, structMap, ruleName(l))
		if err != nil {
			return err
		}

		if res {
			matchCount++
		}
	}

	if matchCount != 0 && matchCount != len(l) {
		return l.ValueCheckError()
	}

	return nil
}

func (l LinkedFieldsValue) ValueCheckError() error {
	template := "%v: all of the following fields need to be either nil/at the indicated value or non-nil/not at the indicated value; currently unmatched: %v"
	return fmt.Errorf(template, ruleName(l), l)
}

func (l LinkedFieldsValue) TypeCheckError(fieldValue reflect.Value) error {
	return GenericTypeCheckError(l, fieldValue, "struct")
}

func (l LinkedFieldsValueWithTrigger) ApplyRule(fieldvalue reflect.Value) error {
	fieldvalue = reflect.Indirect(fieldvalue)

	var matchCount = 0
	// room for logic to possibly expand the marker to accept multiple/combined trigger values (instead of 1)
	var c = 1

	if len(l) < 2 {
		return fmt.Errorf("%s: marker was wrongly defined in struct: less than 2 fields", ruleName(l))
	}

	structMap, ok := structValueToMap(fieldvalue)
	if !ok {
		return l.TypeCheckError(fieldvalue)
	}

	for _, markerString := range l[:c] {
		res, err := checkValueExistsOrIsValid(markerString, structMap, ruleName(l))
		if err != nil {
			return err
		}

		if res {
			matchCount++
		}
	}

	if matchCount != len(l[:c]) {
		return nil
	}

	for _, markerString := range l[c:] {
		res, err := checkValueExistsOrIsValid(markerString, structMap, ruleName(l))
		if err != nil {
			return err
		}

		if res {
			matchCount++
		}
	}

	if matchCount != 0 && matchCount != len(l) {
		return l.ValueCheckError()
	}

	return nil
}

func (l LinkedFieldsValueWithTrigger) ValueCheckError() error {
	template := "%v: all of the following fields need to be aligned; if %v is set, all the following need to either exist or have the indicated value: %v"
	return fmt.Errorf(template, ruleName(l), l[0], l[1:])
}

func (l LinkedFieldsValueWithTrigger) TypeCheckError(fieldValue reflect.Value) error {
	return GenericTypeCheckError(l, fieldValue, "struct")
}

func (a AtLeastOneOf) ApplyRule(fieldvalue reflect.Value) error {
	fieldvalue = reflect.Indirect(fieldvalue)

	structMap, ok := structValueToMap(fieldvalue)
	if !ok {
		return a.TypeCheckError(fieldvalue)
	}

	for _, item := range a {
		if structMap[item] != nil {
			return nil
		}
	}

	return a.ValueCheckError()
}

func (a AtLeastOneOf) ValueCheckError() error {
	template := "%v: at least one of the following fields need to be non-nil (currently all nil): %v"
	return fmt.Errorf(template, ruleName(a), a)
}

func (a AtLeastOneOf) TypeCheckError(fieldValue reflect.Value) error {
	return GenericTypeCheckError(a, fieldValue, "struct")
}

func Register(reg *k8smarkers.Registry) error {
	for _, def := range AllDefinitions {
		if err := reg.Register(def); err != nil {
			return err
		}
	}

	return nil
}

// HELPERS

// addDefinition creates and adds a definition to the package's AllDefinition object, containing all markers definitions
func addDefinition(obj DDValidationMarker, targetType k8smarkers.TargetType) {
	name := rulePrefix + reflect.TypeOf(obj).Name()
	def, err := k8smarkers.MakeDefinition(name, targetType, obj)

	if err != nil {
		panic(err)
	}

	AllDefinitions = append(AllDefinitions, def)
}

// ruleName takes a marker's object and returns its complete name
func ruleName(i interface{}) string {
	return fmt.Sprintf("%s%s", rulePrefix, reflect.TypeOf(i).Name())
}

// genericTypeError returns a generic error for wrong type marker attempt
func GenericTypeCheckError(i interface{}, fieldValue reflect.Value, expectedTypes string) error {
	return fmt.Errorf("%s: marker applied to wrong type: currently %T, can only be %s", ruleName(i), fieldValue, expectedTypes)
}

// structValueToMap takes a struct value and turns it into a map, allowing more flexible field and value parsing
func structValueToMap(value reflect.Value) (map[string]interface{}, bool) {
	m := make(map[string]interface{})

	if value.Kind() != reflect.Struct {
		return nil, false
	}

	relType := value.Type()

	for i := 0; i < relType.NumField(); i++ {
		if !value.Field(i).IsValid() || value.Field(i).IsZero() {
			m[relType.Field(i).Name] = nil
		} else {
			m[relType.Field(i).Name] = value.Field(i).Interface()
		}
	}

	return m, (len(m) > 0)
}

// parseIntOrUInt allows us to factorize rules for ints and uints -- this will need to be replaced if large uints are expected
func parseIntOrUInt(value reflect.Value) (int, bool) {
	fieldInt, ok := value.Interface().(int) // convert from int
	if !ok {                                // convert from uint
		var fieldUInt uint
		fieldUInt, ok = value.Interface().(uint)
		fieldInt = int(fieldUInt)
	}

	return fieldInt, ok
}

// checkValueExistOrIsValid checks if a given string marker item name value exist in a unmarshalled struct (converted to a map by structValueToMap)
// it returns true if the value is found and -if applicable- the required value is valid, false otherwise
func checkValueExistsOrIsValid(markerItem string, structMap map[string]interface{}, ruleName string) (bool, error) {
	// markerItem can either be fieldName, or fieldName=fieldValue
	markerSubfieldName, markerSubfieldValue, isValueField := strings.Cut(markerItem, "=")
	val, fieldExists := structMap[markerSubfieldName]

	if !fieldExists {
		return false, fmt.Errorf("%v: field name %v not found in struct for marker %v", ruleName, markerSubfieldName, markerItem)
	}

	// no given value to respect => check if item is not nil / not nil
	if !isValueField {
		if structMap[markerSubfieldName] != nil {
			return true, nil
		}

		return false, nil
	}

	// a value was required => check if item has described value

	// if field is found in the struct with a nil value, check if marker expected a nil value
	if val == nil {
		if markerSubfieldValue == "" {
			return true, nil
		}

		return false, nil
	}

	v := reflect.Indirect(reflect.ValueOf(val))
	vType := v.Type()
	stringType := reflect.TypeOf(markerSubfieldValue)

	// this marker uses string comparison so the underlying type has to be convertible to string
	convertibleToString := vType.ConvertibleTo(stringType)
	if !convertibleToString {
		return false, fmt.Errorf("%v: wrong type for value field %v; only int and string are allowed", ruleName, markerSubfieldName)
	}

	var vStr string

	switch vType.Kind() {
	case reflect.Int:
		vInt := v.Convert(vType).Interface().(int)
		vStr = strconv.Itoa(vInt)
	case reflect.String:
		vStr = v.Convert(vType).Interface().(string)
	default:
		return false, fmt.Errorf("%v: please do not apply this marker to anything else than int or string. Current type: %v", ruleName, v.Type().Name())
	}

	return strings.EqualFold(markerSubfieldValue, vStr), nil
}
