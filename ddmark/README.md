# DDMARK

## What is it ?
DDMark is a validation module embedded in the chaos-controller, used to apply validation rules to the Kubernetes Custom Resource Definition of the chaos-controller, Disruptions. 
It allows for defining rules that put constraints on fields, which will be applied when they're unmarshalled from a file, or read.

## Why does it exist ?

DDMark emerged as an idea when using kubebuilder, which uses markers to put constraints on CRD (Custom Resource Definition) fields.
Those constraints could later only be applied when parsed by the kubernetes api, which isn't very practical for other uses.
It was decided to use the same [markers](https://pkg.go.dev/sigs.k8s.io/controller-tools/pkg/markers) in a customised way to be more flexible and usable.

DDMarkers can be validated through code, which allows us to validate through a CLI and any valid go runner - no need for kubernetes anymore. 
It also allows us to define custom rules to apply to our structures, and focus the code validation within the chaos-controller to environment-specific validation (*does this service exist in this cluster ? Is my cluster in a supported version ?*) and not structural config validation (*is this field under 100 like we need it to be* ?), which would end up being very annoying, cluttered and messy code.

## How to use DDMark ?

* Define new rules/read the existing rules within the `ddmark/validation/validation.go` file (or in this doc)
* Examples can be found in the `ddmark/teststruct.go` file
* Add the desired rules to the appropriate struct fields in the package you wish to add validation to (care for type checking - e.g. Maximum rule can only be applied to int/uint fields)
* Make sure to use the format `// +ddmark:validation:<rulename>=<value>`
* Call the `ValidateStruct` function, with the unmarshalled struct, the file path, and the full path of the packages containing the included structs definition (with markers embedded). Check `cli/chaosli/cmd/validate.go` for a functioning example.

## Markers Documentation

- Enum:
  - `// +ddmark:validation:Enum={<any>,<any>,...}`
  - Applies to: `<any>` field
  - Asserts field value is one of the given values of the Enum list
- ExclusiveFields:
  - `// +ddmark:validation:ExclusiveFields={<fieldName1>,<fieldName2>,...}`
  - Applies to: `<struct>` type
  - Asserts only one of the given fieldnames isn’t `nil`
- Maximum:
  - `// +ddmark:validation:Maximum=<int/uint>`
  - Applies to: `int/uint` field
  - Asserts a value is equal or inferior to a given int
- Minimum:
  - `// +ddmark:validation:Minimum=<int/uint>`
  - Applies to: `int/uint` field
  - Asserts a value is equal or superior to the given int
- Required:
  - `// +ddmark:validation:Required=<bool>`
  - Applies to: `<any>` field
  - Asserts the concerned field isn’t `nil` (or 0, “”, or other null value)
