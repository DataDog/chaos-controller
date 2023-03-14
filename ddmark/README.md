# DDMARK

## What is it ?
DDMark is a validation module embedded in the chaos-controller, used to apply validation rules to the Kubernetes Custom Resource Definition of the chaos-controller, Disruptions. 
It allows for defining rules that put constraints on fields, which will be applied when they're unmarshalled from a file, or read.

## Why does it exist ?

DDMark emerged as an idea when using kubebuilder, which uses markers to put constraints on CRD (Custom Resource Definition) fields.
Those constraints could later only be applied when parsed by the kubernetes api, which isn't very practical for other uses.
It was decided to use the same [markers](https://pkg.go.dev/sigs.k8s.io/controller-tools/pkg/markers) in a customized way to be more flexible and usable.

DDMarkers can be validated through code, which allows us to validate through a CLI and any valid go runner - no need for kubernetes anymore. 
It also allows us to define custom rules to apply to our structures, and focus the code validation within the chaos-controller to environment-specific validation (*does this service exist in this cluster ? Is my cluster in a supported version ?*) and not structural config validation (*is this field under 100 like we need it to be* ?), which would end up being very annoying, cluttered and messy code.

## How to use DDMark ?

1. **Setup `ddmark` fields in your library:**

* Define new rules/read the existing rules within the `ddmark/validation/validation.go` file (or in this doc)
* Examples can be found in the `ddmark/teststruct.go` file
* Add the desired rules to the appropriate struct fields in the package you wish to add validation to (care for type checking - e.g. Maximum rule can only be applied to int/uint fields). Correct format is `// +ddmark:validation:<rulename>=<value>`
* The analyzed library has to contain a self-packaging exported [`Embed.FS`](https://pkg.go.dev/embed) field. This field will then be used by `ddmark` to import the versioned files into any executable.

2. **Call the validation function from your code:**

* Requirement: code needs to run on a **go 1.18+ environment**
* DDMark works through a client interface (which is linked to internal disk resources). Each `client` instance created with `ddmark.NewClient(<embed.FS>)` works independently and can be used concurrently. Use `client.CleanupLibraries()` to remove all files related to a specific client, or `ddmark.CleanupAllLibraries()` to clean up all the clients' files.
* Call the `ValidateStruct()` function with the unmarshalled struct, and the current file/location for debugging purposes (this is a validation tool !). Check `cli/chaosli/cmd/validate.go` for a functioning example.

## Markers Documentation
### Field Markers (within structs)
- Enum:
  - `// +ddmark:validation:Enum={<any>,<any>,...}`
  - Applies to: `<any>` field
  - Asserts field value is one of the given values of the Enum list
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

### Type Markers (outside structs)
- ExclusiveFields:
  - `// +ddmark:validation:ExclusiveFields={<fieldName1>,<fieldName2>,...}`
  - Applies to: any `<struct>` type
  - Asserts that `<fieldname1>` can only be non-nil iff all of the other fields are `nil`
- LinkedFields:
  - `// +ddmark:validation:LinkedFields={<fieldName1>,<fieldName2>,...}`
  - Applies to: any `<struct>` type
  - Asserts the fields in the list are either all `nil` or all non-`nil`
- AtLeastOneOf:
  - `// +ddmark:validation:AtLeastOneOf={<fieldName1>,<fieldName2>,...}`
  - Applies to: any `<struct>` type
  - Asserts at least one of the fields in the list is non-`nil`.
    - Note: if all the sub-fields of the `<struct>` can be/are `nil`, the parent field will be `nil` and this marker will be ignored. In this case, please consider using the `Required` marker on the parent field.
