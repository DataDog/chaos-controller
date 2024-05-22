package jsonapi

import (
	"encoding"
	"encoding/json"
	"reflect"
)

// Unmarshaler is configured internally via UnmarshalOption's passed to Unmarshal.
// It's used to configure the Unmarshaling by decoding optional fields like Meta.
type Unmarshaler struct {
	unmarshalMeta            bool
	meta                     any
	memberNameValidationMode MemberNameValidationMode
}

// UnmarshalOption allows for configuration of Unmarshaling.
type UnmarshalOption func(m *Unmarshaler)

// UnmarshalMeta decodes Document.Meta into the given interface when unmarshaling.
func UnmarshalMeta(meta any) UnmarshalOption {
	return func(m *Unmarshaler) {
		m.unmarshalMeta = true
		m.meta = meta
	}
}

// UnmarshalSetNameValidation enables a given level of document member name validation.
func UnmarshalSetNameValidation(mode MemberNameValidationMode) UnmarshalOption {
	return func(m *Unmarshaler) {
		m.memberNameValidationMode = mode
	}
}

// relationshipUnmarshaler creates a new marshaler from a parent one for the sake of unmarshaling
// relationship documents, by copying over relevant fields.
func (m *Unmarshaler) relationshipUnmarshaler() *Unmarshaler {
	rm := new(Unmarshaler)

	rm.memberNameValidationMode = m.memberNameValidationMode
	return rm
}

// Unmarshal parses the json:api encoded data and stores the result in the value pointed to by v.
// If v is nil or not a pointer, Unmarshal returns an error.
func Unmarshal(data []byte, v any, opts ...UnmarshalOption) (err error) {
	defer func() {
		// because we make use of reflect we must recover any panics
		if rvr := recover(); rvr != nil {
			err = recoverError(rvr)
			return
		}
	}()

	m := new(Unmarshaler)
	for _, opt := range opts {
		opt(m)
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		err = &TypeError{Actual: rv.Kind().String(), Expected: []string{"non-nil pointer"}}
		return
	}

	var d document
	if err = json.Unmarshal(data, &d); err != nil {
		return
	}

	if err = validateJSONMemberNames(data, m.memberNameValidationMode); err != nil {
		return
	}

	err = d.unmarshal(v, m)

	return
}

func (d *document) unmarshal(v any, m *Unmarshaler) (err error) {
	// verify full-linkage in-case this is a compound document
	if err = d.verifyFullLinkage(true); err != nil {
		return
	}

	if d.hasMany {
		err = unmarshalResourceObjects(d.DataMany, v, m)
		if err != nil {
			return
		}
	} else if d.DataOne != nil {
		err = d.DataOne.unmarshal(v, m)
		if err != nil {
			return
		}
	} else if d.Errors != nil {
		// TODO(#36): Support unmarshaling of errors
		return ErrErrorUnmarshalingNotImplemented
	}

	err = d.unmarshalOptionalFields(m)

	return

}

func (d *document) unmarshalOptionalFields(m *Unmarshaler) error {
	if m == nil {
		// this is possible during recursive document unmarshaling
		return nil
	}
	if m.unmarshalMeta {
		b, err := json.Marshal(d.Meta)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, m.meta); err != nil {
			return err
		}
		if err := validateJSONMemberNames(b, m.memberNameValidationMode); err != nil {
			return err
		}
	}
	return nil
}

func unmarshalResourceObjects(ros []*resourceObject, v any, m *Unmarshaler) error {
	outType := derefType(reflect.TypeOf(v))
	outValue := derefValue(reflect.ValueOf(v))

	// first, it must be a struct since we'll be parsing the jsonapi struct tags
	if outType.Kind() != reflect.Slice {
		return &TypeError{Actual: outType.String(), Expected: []string{"slice"}}
	}

	// allocate an empty slice of the outType if there are no resource objects to unmarshal,
	// because the main loop cannot construct one.
	if len(ros) == 0 {
		outValue = reflect.MakeSlice(outType, 0, 0)
	}

	for _, ro := range ros {
		// unmarshal the resource object into an empty value of the slices element type
		outElem := reflect.New(derefType(outType.Elem())).Interface()
		if err := ro.unmarshal(outElem, m); err != nil {
			return err
		}

		// reflect.New creates a pointer, so if our slices underlying type
		// is not a pointer we must dereference the value before appending it
		outElemValue := reflect.ValueOf(outElem)
		if outType.Elem().Kind() != reflect.Pointer {
			outElemValue = derefValue(outElemValue)
		}

		// append the unmarshaled resource object to the result slice
		outValue = reflect.Append(outValue, outElemValue)
	}

	// set the value of the passed in object to our result
	reflect.ValueOf(v).Elem().Set(outValue)

	return nil
}

func (ro *resourceObject) unmarshal(v any, m *Unmarshaler) error {
	// first, it must be a struct since we'll be parsing the jsonapi struct tags
	vt := reflect.TypeOf(v)
	if derefType(vt).Kind() != reflect.Struct {
		return &TypeError{Actual: vt.String(), Expected: []string{"struct"}}
	}

	rv := derefValue(reflect.ValueOf(v))
	rt := reflect.TypeOf(rv.Interface())
	if err := ro.unmarshalFields(v, rv, rt, m); err != nil {
		return err
	}

	return ro.unmarshalAttributes(v)
}

// unmarshalFields unmarshals a resource object into all non-attribute struct fields
func (ro *resourceObject) unmarshalFields(v any, rv reflect.Value, rt reflect.Type, m *Unmarshaler) error {
	setPrimary := false

	for i := 0; i < rv.NumField(); i++ {
		fv := rv.Field(i)
		ft := rt.Field(i)

		jsonapiTag, err := parseJSONAPITag(ft)
		if err != nil {
			return err
		}
		if jsonapiTag == nil {
			if ft.Anonymous && fv.Kind() == reflect.Struct {
				if err := ro.unmarshalFields(v, fv, reflect.TypeOf(fv.Interface()), m); err != nil {
					return err
				}
			}
			continue
		}

		switch jsonapiTag.directive {
		case primary:
			if setPrimary {
				return ErrUnmarshalDuplicatePrimaryField
			}
			if ro.Type != jsonapiTag.resourceType {
				return &TypeError{Actual: ro.Type, Expected: []string{jsonapiTag.resourceType}}
			}
			if !isValidMemberName(ro.Type, m.memberNameValidationMode) {
				// type names count as member names
				return &MemberNameValidationError{ro.Type}
			}

			// if omitempty is allowed, skip if this is an empty id
			if jsonapiTag.omitEmpty && ro.ID == "" {
				continue
			}

			// to unmarshal the id we follow these rules
			//     1. Use UnmarshalIdentifier if it is implemented
			//     2. Use encoding.TextUnmarshaler if it is implemented
			//     3. Use the value directly if it is a string
			//     4. Fail
			if vu, ok := v.(UnmarshalIdentifier); ok {
				if err := vu.UnmarshalID(ro.ID); err != nil {
					return err
				}
				setPrimary = true
				continue
			}

			// get the underlying fields interface
			var fvi any
			switch fv.CanAddr() {
			case true:
				fvi = fv.Addr().Interface()
			default:
				fvi = fv.Interface()
			}

			if fviu, ok := fvi.(encoding.TextUnmarshaler); ok {
				if err := fviu.UnmarshalText([]byte(ro.ID)); err != nil {
					return err
				}
				setPrimary = true
				continue
			}

			if fv.Kind() == reflect.String {
				fv.SetString(ro.ID)
				setPrimary = true
				continue
			}

			return ErrUnmarshalInvalidPrimaryField
		case relationship:
			name, exported, _ := parseJSONTag(ft)
			if !exported {
				continue
			}
			relDocument, ok := ro.Relationships[name]
			if !ok {
				continue
			}
			if !relDocument.hasMany && relDocument.isEmpty() {
				// ensure struct field is nil for data:null cases only (we want empty slice for data:[])
				continue
			}

			rm := m.relationshipUnmarshaler()
			rel := reflect.New(derefType(ft.Type)).Interface()
			if err := relDocument.unmarshal(rel, rm); err != nil {
				return err
			}
			setFieldValue(fv, rel)
		case meta:
			if ro.Meta == nil {
				continue
			}
			b, err := json.Marshal(ro.Meta)
			if err != nil {
				return err
			}

			meta := reflect.New(derefType(ft.Type)).Interface()
			if err = json.Unmarshal(b, meta); err != nil {
				return err
			}
			setFieldValue(fv, meta)
		default:
			continue
		}
	}

	return nil
}

func (ro *resourceObject) unmarshalAttributes(v any) error {
	if len(ro.Attributes) == 0 {
		return nil
	}

	b, err := json.Marshal(ro.Attributes)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
