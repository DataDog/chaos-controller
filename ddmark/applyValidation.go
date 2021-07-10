// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package ddmark

import (
	"fmt"
	"reflect"

	ddvalidation "github.com/DataDog/chaos-controller/ddmark/validation"
	k8sloader "sigs.k8s.io/controller-tools/pkg/loader"
	k8smarkers "sigs.k8s.io/controller-tools/pkg/markers"
)

var pkgs []*k8sloader.Package
var col *k8smarkers.Collector
var reg *k8smarkers.Registry

func init() {
	col = &k8smarkers.Collector{}
	reg = &k8smarkers.Registry{}

	err := ddvalidation.Register(reg)
	if err != nil {
		fmt.Printf("\nerror loading markers from crd validation: %v", err)
		return
	}

	col.Registry = reg
}

func ValidateStruct(marshalledStruct interface{}, filePath string, structPkg ...string) {
	var err error
	pkgs, err = k8sloader.LoadRoots(structPkg...)

	if err != nil {
		fmt.Println("error loading markers from crd validation")
		return
	}

	var errorList []error = make([]error, 0)

	typesMap := getAllPackageTypes(pkgs)
	if len(typesMap) == 0 {
		errorList = append(errorList, fmt.Errorf("%v: loaded classes are empty or not found", filePath))
	}

	validateStruct(marshalledStruct, typesMap, nil, &errorList, filePath)
	printErrorList(errorList)
}

func validateStruct(marshalledStruct interface{}, typesMap map[string]*k8smarkers.TypeInfo, markerValues k8smarkers.MarkerValues, errorList *[]error, fieldName string) {
	value := reflect.Indirect(reflect.ValueOf(marshalledStruct)) // dereferences pointer value if there is one
	if value.IsValid() && !value.IsZero() {
		markerType := typesMap[value.Type().Name()]
		if markerType != nil {
			applyMarkers(value, markerType.Markers, errorList, fieldName, k8smarkers.DescribesType)

			// apply markers to each fields
			for _, field := range markerType.Fields {
				if fieldValue := value.FieldByName(field.Name); fieldValue.IsValid() {
					validateStruct(
						fieldValue.Interface(),
						typesMap,
						field.Markers,
						errorList,
						fieldName+">"+field.Name,
					)
				}
			}
		}

		if value.Kind() == reflect.Slice {
			for i := 0; i < value.Len(); i++ {
				validateStruct(value.Index(i).Interface(), typesMap, nil, errorList, fieldName+">"+value.Type().Name())
			}
		}
	}

	applyMarkers(value, markerValues, errorList, fieldName, k8smarkers.DescribesField)
}

func applyMarkers(value reflect.Value, markers k8smarkers.MarkerValues, errorList *[]error, fieldName string, targetType k8smarkers.TargetType) {
	if !value.IsValid() {
		isRequired := markers.Get("ddmark:validation:Required")
		if isRequired != nil {
			typedIsRequired, ok := isRequired.(ddvalidation.Required)
			if !ok {
				*errorList = append(*errorList, fmt.Errorf("%v: required marker needs to be a bool, check struct definition", fieldName))
			}

			boolIsRequired := bool(typedIsRequired)

			if boolIsRequired {
				*errorList = append(*errorList, fmt.Errorf("%v is required", fieldName))
				return
			}
		}
	}

	for markerName, marker := range markers {
		thisdef := col.Lookup(fmt.Sprintf("+%s", markerName), targetType)
		if thisdef == nil {
			panic(fmt.Errorf("could not find marker definition - check target type"))
		}

		markerValue := reflect.ValueOf(marker[0])
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

		markerType := markerValue.Convert(thisdef.Output)
		ddmarker, ok := markerType.Interface().(ddvalidation.DDValidationMarker)

		if !ok {
			*errorList = append(*errorList, fmt.Errorf("cannot convert %v to DDmarker, please check the interface definition", thisdef.Output))
		} else if value.IsValid() {
			err := ddmarker.ApplyRule(value)
			if err != nil {
				*errorList = append(*errorList, fmt.Errorf("%v - %v", fieldName, err))
			}
		}
	}
}

func getAllPackageTypes(packages []*k8sloader.Package) map[string]*k8smarkers.TypeInfo {
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
func printErrorList(errorList []error) {
	switch a := len(errorList); {
	case a == 0:
		fmt.Println("file is valid !")
	default:
		fmt.Println("errors found:")

		for _, err := range errorList {
			fmt.Println(err)
		}
	}
}
