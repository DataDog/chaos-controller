// Package jsonapi implements encoding and decoding of JSON:API as defined in https://jsonapi.org/format/.
package jsonapi

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// ResourceObject is a JSON:API resource object as defined by https://jsonapi.org/format/1.0/#document-resource-objects
type resourceObject struct {
	ID            string               `json:"id,omitempty"`
	Type          string               `json:"type"`
	Attributes    map[string]any       `json:"attributes,omitempty"`
	Relationships map[string]*document `json:"relationships,omitempty"`
	Meta          any                  `json:"meta,omitempty"`
	Links         *Link                `json:"links,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (ro *resourceObject) UnmarshalJSON(data []byte) error {
	type alias resourceObject

	auxRaw := &struct {
		Rels map[string]json.RawMessage `json:"relationships,omitempty"`
		*alias
	}{
		alias: (*alias)(ro),
	}
	if err := json.Unmarshal(data, &auxRaw); err != nil {
		return err
	}

	ro.Relationships = make(map[string]*document, len(auxRaw.Rels))
	for name, raw := range auxRaw.Rels {
		// mark the created sub-documents as relationships so that the document Unmarshaler
		// can handle their different member requirements
		d := document{isRelationship: true}
		if err := json.Unmarshal(raw, &d); err != nil {
			return err
		}
		ro.Relationships[name] = &d
	}
	return nil
}

// JSONAPI is a JSON:API object as defined by https://jsonapi.org/format/1.0/#document-jsonapi-object.
type jsonAPI struct {
	Version string `json:"version"`
	Meta    any    `json:"meta,omitempty"`
}

// checkMeta returns a type error if the given meta value is not map-like
func checkMeta(m any) *TypeError {
	if m == nil {
		return nil
	}

	mt := derefType(reflect.TypeOf(m))
	if mt.Kind() == reflect.Struct || mt.Kind() == reflect.Map {
		return nil
	}

	return &TypeError{Actual: mt.String(), Expected: []string{"struct", "map"}}
}

// LinkObject is a links object as defined by https://jsonapi.org/format/1.0/#document-links
type LinkObject struct {
	Href string `json:"href,omitempty"`
	Meta any    `json:"meta,omitempty"`
}

// Link is the top-level links object as defined by https://jsonapi.org/format/1.0/#document-top-level.
// First|Last|Next|Previous are provided to support pagination as defined by https://jsonapi.org/format/1.0/#fetching-pagination.
type Link struct {
	Self    any `json:"self,omitempty"`
	Related any `json:"related,omitempty"`

	First    string `json:"first,omitempty"`
	Last     string `json:"last,omitempty"`
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
}

func checkLinkValue(linkValue any) (bool, *TypeError) {
	var isEmpty bool

	switch lv := linkValue.(type) {
	case *LinkObject:
		if err := checkMeta(lv.Meta); err != nil {
			return false, err
		}
		isEmpty = (lv.Href == "")
	case string:
		isEmpty = (lv == "")
	case nil:
		isEmpty = true
	default:
		return false, &TypeError{Actual: fmt.Sprintf("%T", lv), Expected: []string{"*LinkObject", "string"}}
	}

	return isEmpty, nil
}

func (l *Link) check() error {
	selfIsEmpty, err := checkLinkValue(l.Self)
	if err != nil {
		return err
	}

	relatedIsEmpty, err := checkLinkValue(l.Related)
	if err != nil {
		return err
	}

	// if both are empty then fail, and if one is empty, it must be set to nil to satisfy omitempty
	switch {
	case selfIsEmpty && relatedIsEmpty:
		return ErrMissingLinkFields
	case selfIsEmpty:
		l.Self = nil
	case relatedIsEmpty:
		l.Related = nil
	}

	return nil
}

// Document is a JSON:API document as defined by https://jsonapi.org/format/1.0/#document-top-level
type document struct {
	// Data is a ResourceObject as defined by https://jsonapi.org/format/1.0/#document-resource-objects.
	// DataOne/DataMany are translated to Data in document.MarshalJSON
	hasMany  bool
	DataOne  *resourceObject   `json:"-"`
	DataMany []*resourceObject `json:"-"`

	// isRelationship marks a document as a relationship sub-document (within primary data)
	isRelationship bool `json:"-"`

	// Meta is Meta Information as defined by https://jsonapi.org/format/1.0/#document-meta.
	Meta any `json:"meta,omitempty"`

	// JSONAPI is a JSON:API object as defined by https://jsonapi.org/format/1.0/#document-jsonapi-object.
	JSONAPI *jsonAPI `json:"jsonapi,omitempty"`

	// Errors is a list of JSON:API error objects as defined by https://jsonapi.org/format/1.0/#error-objects.
	Errors []*Error `json:"errors,omitempty"`

	// Links is the top-level links object as defined by https://jsonapi.org/format/1.0/#document-top-level.
	Links *Link `json:"links,omitempty"`

	// Includes contains ResourceObjects creating a compound document as defined by https://jsonapi.org/format/#document-compound-documents.
	Included []*resourceObject `json:"included,omitempty"`
}

func newDocument() *document {
	return &document{
		DataMany: make([]*resourceObject, 0),
		Errors:   make([]*Error, 0),
	}
}

// MarshalJSON implements the json.Marshaler interface.
func (d *document) MarshalJSON() ([]byte, error) {
	// if we get errors, force exclusion of the Data field
	if len(d.Errors) > 0 {
		type alias document
		return json.Marshal(&struct{ *alias }{alias: (*alias)(d)})
	}

	// if DataMany is populated Data is a []*resourceObject
	if d.hasMany {
		type alias document
		return json.Marshal(&struct {
			Data []*resourceObject `json:"data"`
			*alias
		}{
			Data:  d.DataMany,
			alias: (*alias)(d),
		})
	}

	type alias document
	return json.Marshal(&struct {
		Data *resourceObject `json:"data"`
		*alias
	}{
		Data:  d.DataOne,
		alias: (*alias)(d),
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *document) UnmarshalJSON(data []byte) error {
	type alias document

	auxRaw := &struct {
		Data json.RawMessage `json:"data,omitempty"`
		*alias
	}{
		alias: (*alias)(d),
	}
	if err := json.Unmarshal(data, &auxRaw); err != nil {
		return err
	}

	switch string(auxRaw.Data) {
	case "":
		// no "data" field -> check that other required members are present
		if d.isRelationship {
			if d.Meta == nil && d.Links == nil {
				return ErrRelationshipMissingRequiredMembers
			}
		} else if d.Meta == nil && d.Errors == nil {
			return ErrDocumentMissingRequiredMembers
		}
		return nil
	case "{}":
		// {"data":{}, ...} is invalid
		return ErrEmptyDataObject
	case "null":
		// {"data":null, ...} is valid
		return nil
	}

	if auxRaw.Data[0] == '[' {
		d.hasMany = true
		return json.Unmarshal(auxRaw.Data, &auxRaw.DataMany)
	}
	return json.Unmarshal(auxRaw.Data, &auxRaw.DataOne)
}

// isEmpty returns true if there is no primary data in the given document (i.e. null or []).
func (d *document) isEmpty() bool {
	return len(d.DataMany) == 0 && d.DataOne == nil
}

// verifyFullLinkage returns an error if the given compound document is not fully-linked as
// described by https://jsonapi.org/format/1.1/#document-compound-documents. That is, there must be
// a chain of relationships linking all included data to primary data transitively.
func (d *document) verifyFullLinkage(aliasRelationships bool) error {
	if len(d.Included) == 0 {
		return nil
	}

	getResourceObjectSlice := func(d *document) []*resourceObject {
		if d.hasMany {
			return d.DataMany
		}
		if d.DataOne == nil {
			return nil
		}
		return []*resourceObject{d.DataOne}
	}

	resourceIdentifier := func(ro *resourceObject) string {
		return fmt.Sprintf("{Type: %v, ID: %v}", ro.Type, ro.ID)
	}

	// a list of related resource identifiers, and a flag to mark nodes as visited
	type includeNode struct {
		included  *resourceObject
		relatedTo []*resourceObject
		visited   bool
	}

	// compute a graph of relationships between just the included resources
	includeGraph := make(map[string]*includeNode)
	for _, included := range d.Included {
		relatedTo := make([]*resourceObject, 0)

		for _, relationship := range included.Relationships {
			relatedTo = append(relatedTo, getResourceObjectSlice(relationship)...)
		}

		includeGraph[resourceIdentifier(included)] = &includeNode{included, relatedTo, false}
	}

	// helper to traverse the graph from a given key and mark nodes as visited
	var visit func(ro *resourceObject)
	visit = func(ro *resourceObject) {
		node, ok := includeGraph[resourceIdentifier(ro)]
		if !ok {
			return
		}
		if aliasRelationships {
			// fill the relationship document itself with included data
			*ro = *node.included
		}
		if node.visited {
			// cycle detected, don't visit adjacent nodes
			return
		}

		node.visited = true
		for _, related := range node.relatedTo {
			visit(related)
		}
	}

	// visit all include nodes that are accessible from the primary data
	primaryData := getResourceObjectSlice(d)
	for _, data := range primaryData {
		for _, relationship := range data.Relationships {
			for _, ro := range getResourceObjectSlice(relationship) {
				visit(ro)
			}
		}
	}

	invalidResources := make([]string, 0)
	for identifier, node := range includeGraph {
		if !node.visited {
			invalidResources = append(invalidResources, identifier)
		}
	}

	if len(invalidResources) > 0 {
		return &PartialLinkageError{invalidResources}
	}

	return nil
}

// Linkable can be implemented to marshal resource object links as defined by https://jsonapi.org/format/1.0/#document-resource-object-links.
type Linkable interface {
	Link() *Link
}

// LinkableRelation can be implemented to marshal resource object related resource links as defined by https://jsonapi.org/format/1.0/#document-resource-object-related-resource-links.
type LinkableRelation interface {
	LinkRelation(relation string) *Link
}

// MarshalIdentifier can be optionally implemented to control marshaling of the primary field to a string.
//
// The order of operations for marshaling the primary field is:
//
//  1. Use MarshalIdentifier if it is implemented
//  2. Use the value directly if it is a string
//  3. Use fmt.Stringer if it is implemented
//  4. Fail
type MarshalIdentifier interface {
	MarshalID() string
}

// UnmarshalIdentifier can be optionally implemented to control unmarshaling of the primary field from a string.
//
// The order of operations for unmarshaling the primary field is:
//
//  1. Use UnmarshalIdentifier if it is implemented
//  2. Use the value directly if it is a string
//  3. Fail
type UnmarshalIdentifier interface {
	UnmarshalID(id string) error
}
