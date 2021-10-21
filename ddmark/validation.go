// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package ddmark

import (
	"fmt"
	"reflect"

	k8smarkers "sigs.k8s.io/controller-tools/pkg/markers"
)

// AllDefinitions contains all marker definitions for this package.
var AllDefinitions []*k8smarkers.Definition
var rulePrefix string = "ddmark:validation:"

func init() {
	addDefinition(Maximum(0), k8smarkers.DescribesField)
	addDefinition(Minimum(0), k8smarkers.DescribesField)
	addDefinition(Enum(nil), k8smarkers.DescribesField)
	addDefinition(Required(true), k8smarkers.DescribesField)

	addDefinition(ExclusiveFields(nil), k8smarkers.DescribesType)
	addDefinition(LinkedFields(nil), k8smarkers.DescribesType)
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

// LinkedFields can be applied to structs, and asserts the fields in the list are either all 'nil' or all non-'nil'
type LinkedFields []string

// AtLeastOneOf can be applied to structs, and asserts at least one of the following fields is non-'nil'
type AtLeastOneOf []string

func (m Maximum) ApplyRule(fieldvalue reflect.Value) error {
	fieldvalue = reflect.Indirect(fieldvalue)
	fieldInt, ok := parseIntOrUInt(fieldvalue)

	if !ok {
		return fmt.Errorf("%v: marker applied to wrong type: currently %v, can only be %v", ruleName(m), fieldvalue.Type(), "int or uint")
	}

	if int(m) < fieldInt {
		return fmt.Errorf("%v: field has value %v, max is %v (included)", ruleName(m), fieldInt, m)
	}

	return nil
}

func (m Minimum) ApplyRule(fieldvalue reflect.Value) error {
	fieldvalue = reflect.Indirect(fieldvalue)
	fieldInt, ok := parseIntOrUInt(fieldvalue)

	if !ok {
		return fmt.Errorf("%v: marker applied to wrong type: currently %v, can only be %v", ruleName(m), fieldvalue.Type(), "int or uint")
	}

	if int(m) > fieldInt {
		return fmt.Errorf("%v: field has value %v, min is %v (included)", ruleName(m), fieldInt, m)
	}

	return nil
}

func (e ExclusiveFields) ApplyRule(fieldvalue reflect.Value) error {
	fieldvalue = reflect.Indirect(fieldvalue)

	matchCount := 0

	structMap, ok := structValueToMap(fieldvalue)
	if !ok {
		return fmt.Errorf("%v: marker applied to wrong type: currently %v, can only be %v", ruleName(e), fieldvalue.Type(), "struct")
	}

	if structMap[e[0]] != nil {
		for _, item := range e[1:] {
			if structMap[item] != nil {
				matchCount++
			}
		}
	}

	if matchCount >= 1 {
		return fmt.Errorf("%v: some fields are incompatible, %s can't be set alongside any of %v", ruleName(e), e[0], e[1:])
	}

	return nil
}

func (e Enum) ApplyRule(fieldvalue reflect.Value) error {
	fieldvalue = reflect.Indirect(fieldvalue)
	fieldInterface := fieldvalue.Interface()

	for _, markerInterface := range e {
		if reflect.ValueOf(markerInterface).Type().ConvertibleTo(fieldvalue.Type()) {
			markerInterface = reflect.ValueOf(markerInterface).Convert(fieldvalue.Type()).Interface()
		} else {
			return fmt.Errorf("%v: Type Error - field needs to be one of %v, currently \"%v\"", ruleName(e), e, fieldvalue)
		}

		if fieldInterface == markerInterface || reflect.ValueOf(fieldInterface).IsZero() {
			return nil
		}
	}

	return fmt.Errorf("%v: field needs to be one of %v, currently \"%v\"", ruleName(e), e, fieldvalue)
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
		return fmt.Errorf("%v: field is required: currently missing", ruleName(r))
	}

	return nil
}

func (l LinkedFields) ApplyRule(fieldvalue reflect.Value) error {
	fieldvalue = reflect.Indirect(fieldvalue)

	var matchCount = 0

	structMap, ok := structValueToMap(fieldvalue)
	if !ok {
		return fmt.Errorf("%v: marker applied to wrong type: currently %v, can only be %v", ruleName(l), fieldvalue.Type(), "struct")
	}

	for _, item := range l {
		if structMap[item] != nil {
			matchCount++
		}
	}

	if matchCount != 0 && matchCount != len(l) {
		return fmt.Errorf("%v: all of the following fields need to be either nil of non-nil (currently unmatched): %v", ruleName(l), l)
	}

	return nil
}

func (r AtLeastOneOf) ApplyRule(fieldvalue reflect.Value) error {
	fieldvalue = reflect.Indirect(fieldvalue)

	structMap, ok := structValueToMap(fieldvalue)
	if !ok {
		return fmt.Errorf("%v: marker applied to wrong type: currently %v, can only be %v", ruleName(r), fieldvalue.Type(), "struct")
	}

	for _, item := range r {
		if structMap[item] != nil {
			return nil
		}
	}

	return fmt.Errorf("%v: at least one of the following fields need to be non-nil (currently all nil): %v", ruleName(r), r)
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
func addDefinition(obj interface{}, targetType k8smarkers.TargetType) {
	name := rulePrefix + reflect.TypeOf(obj).Name()
	def, err := k8smarkers.MakeDefinition(name, targetType, obj)

	if err != nil {
		panic(err)
	}

	AllDefinitions = append(AllDefinitions, def)
}

// ruleName takes a marker's object and returns its complete name
func ruleName(i interface{}) string {
	return fmt.Sprintf("%v%v", rulePrefix, reflect.TypeOf(i).Name())
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
