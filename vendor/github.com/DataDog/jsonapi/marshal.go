package jsonapi

import (
	"encoding"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strings"
)

var fieldsQueryRegex *regexp.Regexp

func init() {
	fieldsQueryRegex = regexp.MustCompile(`^fields\[(\w+)\]$`)
}

// Marshaler is configured internally via MarshalOption's passed to Marshal.
// It's used to configure the Marshaling by including optional fields like Meta or JSONAPI.
type Marshaler struct {
	meta                     any
	includeJSONAPI           bool
	jsonAPImeta              any
	included                 []any
	link                     *Link
	clientMode               bool
	memberNameValidationMode MemberNameValidationMode

	// fields support sparse fieldsets https://jsonapi.org/format/#fetching-sparse-fieldsets
	fields map[string][]string
}

// MarshalOption allows for configuration of Marshaling.
type MarshalOption func(m *Marshaler)

// MarshalMeta includes the given meta (must be a map or struct) as Document.Meta when marshaling.
func MarshalMeta(meta any) MarshalOption {
	return func(m *Marshaler) {
		m.meta = meta
	}
}

// MarshalJSONAPI includes the given meta (must be a map or struct) as Document.JSONAPI.Meta when marshaling.
// This also enables writing Document.JSONAPI.Version.
func MarshalJSONAPI(meta any) MarshalOption {
	return func(m *Marshaler) {
		m.includeJSONAPI = true
		m.jsonAPImeta = meta
	}
}

// MarshalInclude includes the json:api encoding of v within Document.Included creating a compound document as defined by https://jsonapi.org/format/#document-compound-documents.
func MarshalInclude(v ...any) MarshalOption {
	return func(m *Marshaler) {
		m.included = v
	}
}

// MarshalFields supports sparse fieldsets as defined by https://jsonapi.org/format/1.0/#fetching-sparse-fieldsets.
// The input is a url.Values and if given only the fields included in `fields[type]=a,b` are included in the response.
func MarshalFields(query url.Values) MarshalOption {
	return func(m *Marshaler) {
		m.fields = make(map[string][]string)
		for name, params := range query {
			matches := fieldsQueryRegex.FindStringSubmatch(name)
			if len(matches) > 1 {
				// first match is "fields[type]", second is just "type", aka the capture group
				m.fields[matches[1]] = strings.Split(params[0], ",")
			}
		}
	}
}

// MarshalLinks includes the given links as Document.Links when marshaling.
func MarshalLinks(l *Link) MarshalOption {
	return func(m *Marshaler) {
		m.link = l
	}
}

// MarshalClientMode enables client mode which skips validation only relevant for servers writing JSON:API responses.
func MarshalClientMode() MarshalOption {
	return func(m *Marshaler) {
		m.clientMode = true
	}
}

// MarshalSetNameValidation enables a given level of document member name validation.
func MarshalSetNameValidation(mode MemberNameValidationMode) MarshalOption {
	return func(m *Marshaler) {
		m.memberNameValidationMode = mode
	}
}

// relationshipMarshaler creates a new marshaler from a parent one for the sake of marshaling
// relationship documents, by copying over relevant fields.
func (m *Marshaler) relationshipMarshaler(link *Link) *Marshaler {
	rm := new(Marshaler)

	rm.memberNameValidationMode = m.memberNameValidationMode
	rm.link = link
	rm.clientMode = m.clientMode
	return rm
}

// Marshal returns the json:api encoding of v. If v is type *Error or []*Error only the errors will be marshaled.
func Marshal(v any, opts ...MarshalOption) (b []byte, err error) {
	defer func() {
		// because we make use of reflect we must recover any panics
		if rvr := recover(); rvr != nil {
			err = recoverError(rvr)
			return
		}
	}()

	m := new(Marshaler)
	for _, opt := range opts {
		opt(m)
	}

	// marshal first constructs a jsonapi.Document
	// the given "v" is the resource document (either one or many) of any type
	var d *document
	d, err = makeDocument(v, m, false)
	if err != nil {
		return
	}

	// now that we have a document, just marshal it as normal json
	b, err = json.Marshal(d)
	if err != nil {
		return
	}

	err = validateJSONMemberNames(b, m.memberNameValidationMode)

	return
}

func makeDocument(v any, m *Marshaler, isRelationship bool) (*document, error) {
	// first attempt to make errors
	// if we got errors the document will be non-nil and since data+errors cannot
	// both exist in the same document, just return before any other work
	d, err := makeDocumentErrors(v, m)
	if err != nil {
		return nil, err
	}
	if d != nil {
		return d, nil
	}

	// at this point we have no errors, so lets make the document
	d = newDocument()
	d.isRelationship = isRelationship

	// the given "v" is the resource object (or a slice of them)
	//
	// besides nil, only a struct or slice of struct are valid here because
	// we'll be parsing the jsonapi struct tags from the struct to make the
	// resource object
	vt := reflect.TypeOf(v)
	switch {
	case vt == nil:
		// if v is nil we want `{"data":null}` so continue with an empty document
		break
	case derefType(vt).Kind() == reflect.Slice:
		// if we get a slice we make a resource object for each item
		d.hasMany = true
		// if v is an empty slice we want `{"data":[]}`
		if reflect.ValueOf(v).IsZero() {
			break
		}
		rv := derefValue(reflect.ValueOf(v))
		for i := 0; i < rv.Len(); i++ {
			iv := rv.Index(i).Interface()
			ro, err := d.makeResourceObject(iv, reflect.TypeOf(iv), m)
			if err != nil {
				return nil, err
			}
			if ro != nil {
				d.DataMany = append(d.DataMany, ro)
			}
		}
	case derefType(vt).Kind() == reflect.Struct:
		if reflect.ValueOf(v).IsZero() {
			break
		}
		// if we get a struct we just make a single resource object
		ro, err := d.makeResourceObject(v, vt, m)
		if err != nil {
			return nil, err
		}
		d.DataOne = ro
	default:
		return nil, &TypeError{Actual: fmt.Sprintf("%T", v), Expected: []string{"struct", "slice"}}
	}

	// if we got any included data, build the resource object/s and include them
	for _, v := range m.included {
		ro, err := d.makeResourceObject(v, reflect.TypeOf(v), m)
		if err != nil {
			return nil, err
		}
		d.Included = append(d.Included, ro)
	}

	// if we got any included data, verify full-linkage of this compound document.
	if err := d.verifyFullLinkage(false); err != nil {
		return nil, err
	}

	filterDocumentFieldsets(d, m)

	if err := addOptionalDocumentFields(d, m); err != nil {
		return nil, err
	}

	return d, nil
}

// filterDocumentFieldsets supports Sparse Fieldsets by filtering out any of the attributes or
// relationships in the document's resource objects that were not chosen in MarshalFields.
func filterDocumentFieldsets(d *document, m *Marshaler) {
	if len(m.fields) == 0 {
		return
	}

	// retain only the attributes or relationships specified in MarshalFields for some type
	filterResourceObject := func(ro *resourceObject) {
		fields, ok := m.fields[ro.Type]
		if !ok {
			// this type has no fieldset filters
			return
		}

		filteredAttributes := make(map[string]any)
		filteredRelationships := make(map[string]*document)

		for _, field := range fields {
			if v, ok := ro.Attributes[field]; ok {
				filteredAttributes[field] = v
			} else if v, ok := ro.Relationships[field]; ok {
				filteredRelationships[field] = v
			}
		}
		ro.Attributes = filteredAttributes
		ro.Relationships = filteredRelationships
	}

	// filter fields in primary data and then included data
	if d.hasMany {
		for _, ro := range d.DataMany {
			filterResourceObject(ro)
		}
	} else {
		filterResourceObject(d.DataOne)
	}

	for _, ro := range d.Included {
		filterResourceObject(ro)
	}
}

func makeDocumentErrors(v any, m *Marshaler) (*document, error) {
	var errorObjects []*Error

	// support marshaling a single error and non-pointer types
	switch errorObject := v.(type) {
	case Error:
		errorObjects = append(errorObjects, &errorObject)
	case *Error:
		errorObjects = append(errorObjects, errorObject)
	case []Error:
		for _, e := range errorObject {
			e := e
			errorObjects = append(errorObjects, &e)
		}
	case []*Error:
		errorObjects = errorObject
	}

	// if no error objects have been collected, move on to normal document creation
	if len(errorObjects) == 0 {
		return nil, nil
	}

	// check for valid error links and meta fields if present
	for _, eo := range errorObjects {
		if eo.Links != nil {
			if _, err := checkLinkValue(eo.Links.About); err != nil {
				return nil, err
			}
		}
		if err := checkMeta(eo.Meta); err != nil {
			return nil, err
		}
	}

	d := newDocument()
	d.Errors = errorObjects

	if err := addOptionalDocumentFields(d, m); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *document) makeResourceObject(v any, vt reflect.Type, m *Marshaler) (*resourceObject, error) {
	// the given "v" here is a single resource object

	// first, it must be a struct since we'll be parsing the jsonapi struct tags
	if derefType(vt).Kind() != reflect.Struct {
		return nil, &TypeError{Actual: vt.String(), Expected: []string{"struct"}}
	}

	ro := &resourceObject{
		Attributes:    make(map[string]any, 0),
		Relationships: make(map[string]*document, 0),
	}

	// get fields from embedded structs
	fields := getFlattenedFields(v)

	var foundPrimary bool
	for _, field := range fields {
		// for each field in the struct we'll parse the jsonapi struct tag
		// this will determine where it goes in the resource object (e.g. id,type,attributes,...)

		f := field.v
		ft := field.f

		tag, err := parseJSONAPITag(ft)
		if err != nil {
			return nil, err
		}
		if tag == nil {
			// this field is not tagged w/ jsonapi and will be ignored
			continue
		}

		switch tag.directive {
		case primary:
			ro.Type = tag.resourceType
			if !isValidMemberName(ro.Type, m.memberNameValidationMode) {
				// type names count as member names
				return nil, &MemberNameValidationError{ro.Type}
			}

			// to marshal the id we follow these rules
			//     1. Use MarshalIdentifier if it is implemented
			//     2. Use the value directly if it is a string
			//     3. Use fmt.Stringer if it is implemented
			//     4. Use encoding.TextMarshaler if it is implemented
			//     5. Fail

			if vm, ok := v.(MarshalIdentifier); ok {
				ro.ID = vm.MarshalID()
				foundPrimary = true
				continue
			}

			fv := f.Interface()

			if vs, ok := fv.(string); ok {
				ro.ID = vs
				foundPrimary = true
				continue
			}

			if _, ok := fv.(fmt.Stringer); ok {
				ro.ID = fmt.Sprintf("%s", fv)
				foundPrimary = true
				continue
			}

			if fvm, ok := fv.(encoding.TextMarshaler); ok {
				vb, err := fvm.MarshalText()
				if err != nil {
					return nil, err
				}
				ro.ID = string(vb)
				foundPrimary = true
				continue
			}

			return nil, ErrMarshalInvalidPrimaryField
		case attribute:
			if d.isRelationship {
				// relationships must only be resource identifier objects so skip attributes
				continue
			}
			fieldName, ok, omit := parseJSONTag(ft)
			if !ok {
				continue
			}
			if f.IsZero() && omit {
				continue
			}
			ro.Attributes[fieldName] = f.Interface()
		case meta:
			metaObject := f.Interface()
			if err := checkMeta(metaObject); err != nil {
				return nil, err
			}

			if f.IsZero() {
				// ensure json omitempty works correctly with meta any type
				metaObject = nil
			}

			if d.isRelationship {
				// let meta become document-level for relationships (treated as nested documents)
				m.meta = metaObject
			} else {
				ro.Meta = metaObject
			}
		case relationship:
			if d.isRelationship {
				// relationship nesting must occur in include data, not the relationship fields
				continue
			}
			fieldName, ok, omit := parseJSONTag(ft)
			if !ok {
				continue
			}
			if f.IsZero() && omit {
				continue
			}

			// if LinkableRelation is implemented include Document.Links for the related resource
			var link *Link
			if lv, ok := v.(LinkableRelation); ok {
				link = lv.LinkRelation(fieldName)
				if err := link.check(); err != nil {
					return nil, err
				}
			}

			rm := m.relationshipMarshaler(link)
			d, err := makeDocument(f.Interface(), rm, true)
			if err != nil {
				return nil, err
			}

			ro.Relationships[fieldName] = d
		}
	}

	// primary is the only required jsonapi struct tag as it defines the id/type
	if !foundPrimary {
		return nil, ErrMissingPrimaryField
	}

	// id (e.g. the primary field) must not be empty
	if ro.ID == "" && !m.clientMode {
		return nil, ErrEmptyPrimaryField
	}

	// if Linkable is implemented include ResourceObject.Links
	if lv, ok := v.(Linkable); ok {
		link := lv.Link()
		if err := link.check(); err != nil {
			return nil, err
		}
		ro.Links = link
	}

	return ro, nil
}

func getFlattenedFields(iface interface{}) []struct {
	v reflect.Value
	f reflect.StructField
} {
	rv := derefValue(reflect.ValueOf(iface))
	rt := reflect.TypeOf(rv.Interface())

	fields := make([]struct {
		v reflect.Value
		f reflect.StructField
	}, 0)

	for i := 0; i < rv.NumField(); i++ {
		v := rv.Field(i)
		f := rt.Field(i)

		if f.Anonymous && (v.Kind() == reflect.Struct || v.Kind() == reflect.Pointer) {
			fields = append(fields, getFlattenedFields(v.Interface())...)
		} else {
			fields = append(fields, struct {
				v reflect.Value
				f reflect.StructField
			}{v, f})
		}
	}

	return fields
}

func addOptionalDocumentFields(d *document, m *Marshaler) error {
	// optionally include Document.meta (may be nil, which will be omitted)
	if err := checkMeta(m.meta); err != nil {
		return err
	}
	d.Meta = m.meta

	// optionally include the Document.jsonapi (may be nil, which will be omitted)
	if m.includeJSONAPI {
		d.JSONAPI = &jsonAPI{Version: "1.0"}
		if err := checkMeta(m.jsonAPImeta); err != nil {
			return err
		}
		d.JSONAPI.Meta = m.jsonAPImeta
	}

	// optionally include Document.links (may be nil, which will be omitted)
	d.Links = m.link

	return nil
}
