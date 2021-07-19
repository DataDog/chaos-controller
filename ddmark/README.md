# DDMARK

## What is it ?
DDmark is a validation Modules embedded in the chaos-controller, used to apply validation rules to the Custom Kubernetes API of the controller, Disruptions. 
It allows to define rules that are to put constraints on fields, which will be applied when they're unmarshalled from a file, or read.

## Why does it exist ?

DDMark emerged as an idea when using kubebuilder, which uses markers to put constraints on CRD (Custom Ressource Defintion) fields.
Those constraints could later only be applied when parsed kubernetes, which isn't very practical.
It was decided to use the same [markers](https://pkg.go.dev/sigs.k8s.io/controller-tools/pkg/markers) in a customised way to be more flexible and usable.

DDMarkers can be validated through code, which allows us to validate through a CLI and any valid go runner - no need for kubernetes anymore. 
It also allows us to define custom rules to apply to our structures, and focus the code validation within the chaos-controller to environment-specific validation (*does this service exist in this cluster ? Is my cluster in a supported version ?*) and not structural config validation (*is this field under 100 like we need it to be* ?), which would end up being very annoying, cluttered and messy code.

## How to use DDMark ?

* Define new rules/read the existing rules within the ddmark/validation/validation.go file
* Add the appropriate rules to the appropriate struct fields in the package you wish to add validation to (care for type checking - eg. Maximum rule can only be applied to int/uint fields)
* Make sure to use the format `// +ddmark:validation:<rulename>=<value>`
* Call the `ValidateStruct` function, with the unmarshalled struct, the file path and the full path of the packages containing the included structs definition (with markers embedded). Check `cli/chaosli/cmd/validate.go` for a functionning example.

## To Do
- [ ] Add formal documentation on each existing marker: 
    - [ ] specs
    - [ ] expected behavior
    - [ ] usage example