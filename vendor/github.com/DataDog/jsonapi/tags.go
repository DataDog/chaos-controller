package jsonapi

import (
	"reflect"
	"strings"
)

type directive int

const (
	primary directive = iota
	attribute
	meta
	relationship
	invalid
)

func parseDirective(v string) (directive, bool) {
	switch v {
	case "primary":
		return primary, true
	case "attribute", "attr":
		return attribute, true
	case "meta":
		return meta, true
	case "relationship", "rel":
		return relationship, true
	}
	return invalid, false
}

type tag struct {
	directive    directive
	resourceType string // only valid for primary
	omitEmpty    bool
}

func parseJSONTag(f reflect.StructField) (string, bool, bool) {
	t := f.Tag.Get("json")
	if t == "" {
		if f.IsExported() {
			return f.Name, true, false
		}
		return t, false, false
	}
	ts := strings.Split(t, ",")

	var omit bool
	if len(ts) > 1 {
		omit = ts[1] == "omitempty"
	}

	return ts[0], f.IsExported(), omit
}

func parseJSONAPITag(f reflect.StructField) (*tag, error) {
	t := f.Tag.Get("jsonapi")
	ts := strings.Split(t, ",")

	var omitEmpty bool
	switch len(ts) {
	case 3:
		if ts[2] == "omitempty" {
			omitEmpty = true
			break // good
		}
	case 2:
		break // good
	case 1:
		// this is a missing tag
		if ts[0] == "" {
			return nil, nil
		}
	default:
		return nil, &TagError{
			TagName: "jsonapi",
			Field:   f.Name,
			Reason:  "expected format {directive},{optional:type},{optional:omitempty}",
		}
	}

	d, ok := parseDirective(ts[0])
	if !ok {
		return nil, &TagError{TagName: "jsonapi", Field: f.Name, Reason: "invalid directive"}
	}

	tag := &tag{directive: d, omitEmpty: omitEmpty}
	if d == primary {
		if len(ts) < 2 {
			return nil, &TagError{
				TagName: "jsonapi",
				Field:   f.Name,
				Reason:  "missing type in primary directive",
			}
		}
		tag.resourceType = ts[1]
	}

	return tag, nil
}
