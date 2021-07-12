// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package validation

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
}

// Maximum can applied to an int field and provides a (non-strict) maximum value for that field
type Maximum int

// Minimum can applied to an int field and provides a (non-strict) minimum value for that field
type Minimum int

// Enum can be applied to a string (or string-able) field and provides a restricted amount of possible values for that field
type Enum []string

// Required can be applied to any field, and asserts this field will return an error if not provided
type Required bool

// ExclusiveFields can be applied to structs, and asserts that at most one of the given field names is not null
type ExclusiveFields []string

func (m Maximum) ApplyRule(fieldvalue reflect.Value) error {
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
	var matchCount int = 0

	structMap, ok := structValueToMap(fieldvalue)
	if !ok {
		return fmt.Errorf("%v: marker applied to wrong type: currently %v, can only be %v", ruleName(e), fieldvalue.Type(), "struct")
	}

	for _, item := range e {
		if structMap[item] != nil {
			matchCount++
		}
	}

	if matchCount > 1 {
		return fmt.Errorf("%v: some fields are incompatible, there can only be one of %v", ruleName(e), e)
	}

	return nil
}

func (e Enum) ApplyRule(fieldvalue reflect.Value) error {
	fieldString := fieldvalue.String()

	for _, str := range e {
		if fieldString == str || fieldString == "" {
			return nil
		}
	}

	return fmt.Errorf("%v: field needs to be one of %v, currently \"%v\"", ruleName(e), e, fieldvalue)
}

func (r Required) ApplyRule(fieldvalue reflect.Value) error {
	if bool(r) && (!fieldvalue.IsValid() || fieldvalue.IsZero()) {
		return fmt.Errorf("%v: field is required: currently %v", ruleName(r), "missing")
	}

	return nil
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

// parstIntOrUInt allows to factorize rules for ints and uints -- will need to be replaced if large uints are expected
func parseIntOrUInt(value reflect.Value) (int, bool) {
	fieldInt, ok := value.Interface().(int) // convert from int
	if !ok {                                // convert from uint
		var fieldUInt uint
		fieldUInt, ok = value.Interface().(uint)
		fieldInt = int(fieldUInt)
	}

	return fieldInt, ok
}
