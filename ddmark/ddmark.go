// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package ddmark

import (
	"fmt"
	"reflect"
	"strings"

	k8sloader "sigs.k8s.io/controller-tools/pkg/loader"
	k8smarkers "sigs.k8s.io/controller-tools/pkg/markers"
)

func InitializeMarkers() *k8smarkers.Collector {
	col := &k8smarkers.Collector{}
	reg := &k8smarkers.Registry{}

	// takes all the markers definition found in the ddmark/validation package, prior to analyzing the packages
	err := Register(reg)
	if err != nil {
		fmt.Printf("\nerror loading markers from crd validation: %v", err)
		return col
	}

	col.Registry = reg

	return col
}

// ValidateStruct applies struct markers found in structPkgs struct definitions to a marshalledStruct object.
// It allows to enforce markers rule onto that object, according to the constraints defined in structPkgs
func ValidateStruct(marshalledStruct interface{}, filePath string, structPkgs ...string) []error {
	col := InitializeMarkers()

	var err error

	var errorList []error = make([]error, 0)

	var pkgs []*k8sloader.Package

	pkgs, err = k8sloader.LoadRoots(structPkgs...)

	if err != nil {
		return append(errorList, fmt.Errorf("error loading markers from crd validation: \n\t%v", err))
	}

	typesMap := getAllPackageTypes(pkgs, col)
	if len(typesMap) == 0 {
		errorList = append(errorList, fmt.Errorf("%v: loaded classes are empty or not found", filePath))
	}

	validateStruct(marshalledStruct, typesMap, nil, &errorList, filePath, col)

	return errorList
}

// validateStruct is an internal recursive function that recursively applies markers rules to types and fields
func validateStruct(marshalledStruct interface{}, typesMap map[string]*k8smarkers.TypeInfo, markerValues k8smarkers.MarkerValues, errorList *[]error, fieldName string, col *k8smarkers.Collector) {
	value := reflect.ValueOf(marshalledStruct)
	unpointedValue := reflect.Indirect(value) // dereferences pointer value if there is one

	if unpointedValue.IsValid() && !unpointedValue.IsZero() {
		markerType := typesMap[unpointedValue.Type().Name()]
		if markerType != nil {
			// apply the markers on the type level (if there is any)
			applyMarkers(value, markerType.Markers, errorList, fieldName, k8smarkers.DescribesType, col)

			// apply this function to each subsequent fields - on structs only
			for _, field := range markerType.Fields {
				if fieldValue := unpointedValue.FieldByName(field.Name); fieldValue.IsValid() {
					validateStruct(
						fieldValue.Interface(),
						typesMap,
						field.Markers,
						errorList,
						fieldName+">"+field.Name,
						col,
					)
				}
			}
		}

		// apply markers to slice/array values
		if unpointedValue.Kind() == reflect.Slice || unpointedValue.Kind() == reflect.Array {
			for i := 0; i < unpointedValue.Len(); i++ {
				validateStruct(unpointedValue.Index(i).Interface(), typesMap, nil, errorList, fieldName+">"+unpointedValue.Type().Name(), col)
			}
		}
	}

	applyMarkers(value, markerValues, errorList, fieldName, k8smarkers.DescribesField, col)
}

// applyMarkers applies all markers found in the markers arg to a given type/field
func applyMarkers(value reflect.Value, markers k8smarkers.MarkerValues, errorList *[]error, fieldName string, targetType k8smarkers.TargetType, col *k8smarkers.Collector) {
	// if value is Invalid, field is most likely absent -- needs to add an error if Required is found true
	if !reflect.Indirect(value).IsValid() {
		isRequired := markers.Get("ddmark:validation:Required")
		if isRequired != nil {
			typedIsRequired, ok := isRequired.(Required)
			if !ok {
				*errorList = append(*errorList, fmt.Errorf("%v: required marker needs to be a bool, check struct definition", fieldName))
			}

			boolIsRequired := bool(typedIsRequired)

			if boolIsRequired {
				*errorList = append(*errorList, fmt.Errorf("%v is required", fieldName))
				return
			}
		}

		return
	}

	// run all existing markers for that field
	for markerName, markerValueList := range markers {
		// fetch the marker definition in order to type-check the corresponding field
		thisdef := col.Lookup(fmt.Sprintf("+%s", markerName), targetType)
		if thisdef == nil {
			*errorList = append(*errorList, fmt.Errorf("could not find marker definition for %v - check target type", markerName))
			continue
		}

		// if a marker is used multiple times on a single type/field, a single marker will have multiple values
		// that need to be iterated on (eg. ExclusiveFields, where multiple pairs can be concurrently restricted)
		for _, markerValueInterface := range markerValueList {
			// type-check the marker value to fit the DDValidationMarker interface
			markerValue := reflect.ValueOf(markerValueInterface)
			isok := markerValue.Type().ConvertibleTo(thisdef.Output)

			if !isok {
				*errorList = append(*errorList,
					fmt.Errorf("%v - %v: this marker is of kind %v - cannot be converted to %v",
						fieldName,
						markerName,
						markerValue.Type().Kind(),
						thisdef.Output.Kind()))

				continue
			}

			// convert to the DDValidationMarker interface in order to apply validation
			markerType := markerValue.Convert(thisdef.Output)
			ddmarker, ok := markerType.Interface().(DDValidationMarker)

			if !ok {
				*errorList = append(*errorList, fmt.Errorf("cannot convert %v to DDmarker, please check the interface definition", thisdef.Output))
			} else {
				// conversions are done, proceed to validation
				err := ddmarker.ApplyRule(value)
				if err != nil {
					*errorList = append(*errorList, fmt.Errorf("%v - %v", fieldName, err))
				}
			}
		}
	}
}

// getAllPackageTypes extracts all marker rules found in packages and keeps them in a map, ordered by type names
func getAllPackageTypes(packages []*k8sloader.Package, col *k8smarkers.Collector) map[string]*k8smarkers.TypeInfo {
	var typesMap = map[string]*k8smarkers.TypeInfo{}

	for _, pkg := range packages {
		var isEmpty bool = true

		err := k8smarkers.EachType(col, pkg, func(info *k8smarkers.TypeInfo) {
			isEmpty = false
			typesMap[info.Name] = info
		})

		if err != nil {
			fmt.Println(pkg, "marker loader:", err)
		}

		if isEmpty {
			fmt.Printf("marker loader: package %s is not found or contains no structure\n", pkg)
		}
	}

	return typesMap
}

// HELPERS

// GetErrorList returns a list of errors as a string
func GetErrorList(errorList []error) string {
	var res strings.Builder

	switch a := len(errorList); {
	case a == 0:
		res.WriteString("file is valid !")
	default:
		res.WriteString("errors:")

		for _, err := range errorList {
			res.WriteString(fmt.Sprintf("\n - %v", err))
		}
	}

	return res.String()
}
