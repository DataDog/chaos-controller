// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package ddmark

import (
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"fmt"
	"os"
	"reflect"

	"github.com/hashicorp/go-multierror"
	"golang.org/x/tools/go/packages"
	k8sloader "sigs.k8s.io/controller-tools/pkg/loader"
	k8smarkers "sigs.k8s.io/controller-tools/pkg/markers"
)

// EmbeddedDDMarkAPI includes the teststruct so it can be statically exported for ddmark testing
//
//go:embed validation_teststruct.go
var EmbeddedDDMarkAPI embed.FS

// Client interface to manage validation of struct fields
//
// Client is the expected way to use DDMark. Create a client with ddmark.NewClient(embed.FS).
// The client interfaces with local-disk copies of the given files.
// DDMark clients can be used simultaneously, as they manage independent/separate disk resources.
//
// Client creates local files in the GOPATH. Make sure to use Client.CleanupLibraries (or ddmark.CleanupAllLibraries) to remove them.
type Client interface {
	// ValidateStruct applies struct markers found in structPkgs struct definitions to a marshalledStruct object.
	// It allows to enforce ddmark rules onto that object, according to the constraints defined in struct file.
	ValidateStruct(marshalledStruct interface{}, filePath string) []error
	// ValidateStructMultierror is the parent function of ValidateStruct.
	// It allows users to leverage the multierror package for error management.
	ValidateStructMultierror(marshalledStruct interface{}, filePath string) (retErr *multierror.Error)
	// CleanupLibraries removes the disk files related to the client.
	CleanupLibraries() error
}

// client struct implementing DDMark interface
type client struct {
	markedLibs []markedLib
}

// markedLib is a struct describing a library containing DDMarkers to be used in validation.
// It includes the embedded library FS and a path naming option for consistency.
type markedLib struct {
	EmbeddedFS embed.FS
	APIName    string
}

// NewClient create an new instance of DDMark
func NewClient(embeddedFS ...embed.FS) (Client, error) {
	var err error

	c := client{
		markedLibs: []markedLib{},
	}

	for _, lib := range embeddedFS {
		randomSha, err := generateRandomSha()
		if err != nil {
			return nil, err
		}

		c.markedLibs = append(c.markedLibs, markedLib{lib, randomSha})

		err = c.initLibrary(lib, randomSha)
		if err != nil {
			return nil, err
		}
	}

	return c, err
}

// initializeMarkers creates and sets up the controller-tools Collector instance for ddmark.
func initializeMarkers() *k8smarkers.Collector {
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
// It allows to enforce ddmark rules onto that object, according to the constraints defined in struct file.
func (c client) ValidateStruct(marshalledStruct interface{}, filePath string) []error {
	return c.ValidateStructMultierror(marshalledStruct, filePath).Errors
}

// ValidateStructMultierror is the parent function of ValidateStruct.
// It allows users to leverage the multierror package for error management.
func (c client) ValidateStructMultierror(marshalledStruct interface{}, filePath string) (retErr *multierror.Error) {
	col := initializeMarkers()

	var err error

	var pkgs []*k8sloader.Package

	localStructPkgs := []string{}

	for _, pkg := range c.markedLibs {
		localStructPkgs = append(localStructPkgs, thisLibPath(pkg.APIName))
	}

	pkgs, err = k8sloader.LoadRootsWithConfig(&packages.Config{Env: append(os.Environ(), "GO111MODULE=off")}, localStructPkgs...)

	if err != nil {
		return multierror.Append(retErr, fmt.Errorf("error loading markers from crd validation: \n\t%v", err))
	}

	typesMap := getAllPackageTypes(pkgs, col)
	if len(typesMap) == 0 {
		retErr = multierror.Append(retErr, fmt.Errorf("%v: loaded classes are empty or not found", filePath))
	}

	retErr = multierror.Append(retErr, validateStruct(marshalledStruct, typesMap, nil, filePath, col))

	return retErr
}

// validateStruct is an internal recursive function that recursively applies markers rules to types and fields.
func validateStruct(marshalledStruct interface{}, typesMap map[string]*k8smarkers.TypeInfo, markerValues k8smarkers.MarkerValues, fieldName string, col *k8smarkers.Collector) (retErr error) {
	value := reflect.ValueOf(marshalledStruct)
	unpointedValue := reflect.Indirect(value) // dereferences pointer value if there is one

	if unpointedValue.IsValid() && !unpointedValue.IsZero() {
		markerType := typesMap[unpointedValue.Type().Name()]
		if markerType != nil {
			// apply the markers on the type level (if there is any)
			retErr = multierror.Append(retErr, applyMarkers(value, markerType.Markers, fieldName, k8smarkers.DescribesType, col))

			// apply this function to each subsequent fields - on structs only
			for _, field := range markerType.Fields {
				if fieldValue := unpointedValue.FieldByName(field.Name); fieldValue.IsValid() {
					retErr = multierror.Append(retErr, validateStruct(
						fieldValue.Interface(),
						typesMap,
						field.Markers,
						fieldName+">"+field.Name,
						col,
					))
				}
			}
		}

		// apply markers to slice/array values
		if unpointedValue.Kind() == reflect.Slice || unpointedValue.Kind() == reflect.Array {
			for i := 0; i < unpointedValue.Len(); i++ {
				retErr = multierror.Append(retErr, validateStruct(unpointedValue.Index(i).Interface(), typesMap, nil, fieldName+">"+unpointedValue.Type().Name(), col))
			}
		}
	}

	retErr = multierror.Append(retErr, applyMarkers(value, markerValues, fieldName, k8smarkers.DescribesField, col))

	return retErr
}

// applyMarkers applies all markers found in the markers arg to a given type/field.
func applyMarkers(value reflect.Value, markers k8smarkers.MarkerValues, fieldName string, targetType k8smarkers.TargetType, col *k8smarkers.Collector) (retErr error) {
	// if value is Invalid, field is most likely absent -- needs to add an error if Required is found true
	if !reflect.Indirect(value).IsValid() {
		isRequired := markers.Get("ddmark:validation:Required")
		if isRequired != nil {
			typedIsRequired, ok := isRequired.(Required)
			if !ok {
				retErr = multierror.Append(retErr, fmt.Errorf("%v: required marker needs to be a bool, check struct definition", fieldName))
			}

			boolIsRequired := bool(typedIsRequired)

			if boolIsRequired {
				retErr = multierror.Append(retErr, fmt.Errorf("%v is required", fieldName))
				return retErr
			}
		}

		return retErr
	}

	// run all existing markers for that field
	for markerName, markerValueList := range markers {
		// fetch the marker definition in order to type-check the corresponding field
		thisdef := col.Lookup(fmt.Sprintf("+%s", markerName), targetType)
		if thisdef == nil {
			retErr = multierror.Append(retErr, fmt.Errorf("could not find marker definition for %v - check target type", markerName))
			continue
		}

		// if a marker is used multiple times on a single type/field, a single marker will have multiple values
		// that need to be iterated on (eg. ExclusiveFields, where multiple pairs can be concurrently restricted)
		for _, markerValueInterface := range markerValueList {
			// type-check the marker value to fit the DDValidationMarker interface
			markerValue := reflect.ValueOf(markerValueInterface)
			isok := markerValue.Type().ConvertibleTo(thisdef.Output)

			if !isok {
				retErr = multierror.Append(retErr,
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
				retErr = multierror.Append(retErr, fmt.Errorf("cannot convert %v to DDmarker, please check the interface definition", thisdef.Output))
			} else {
				// conversions are done, proceed to validation
				err := ddmarker.ApplyRule(value)
				if err != nil {
					retErr = multierror.Append(retErr, fmt.Errorf("%v - %v", fieldName, err))
				}
			}
		}
	}

	return retErr
}

// getAllPackageTypes extracts all marker rules found in packages and keeps them in a map, ordered by type names.
func getAllPackageTypes(packages []*k8sloader.Package, col *k8smarkers.Collector) map[string]*k8smarkers.TypeInfo {
	typesMap := map[string]*k8smarkers.TypeInfo{}

	for _, pkg := range packages {
		isEmpty := true

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

// generateRandomSha generates a 64-chars sha as a string
func generateRandomSha() (string, error) {
	var err error

	data := make([]byte, 10)

	if _, err = rand.Read(data); err == nil {
		return fmt.Sprintf("%x", sha256.Sum256(data)), err
	}

	return "", err
}

// HELPERS

// GetErrorList returns a list of errors as a string
func GetErrorList(errorList []error) string {
	return multierror.ListFormatFunc(errorList)
}
