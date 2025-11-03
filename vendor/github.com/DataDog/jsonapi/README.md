[![Go Reference](https://pkg.go.dev/badge/github.com/DataDog/jsonapi.svg)](https://pkg.go.dev/github.com/DataDog/jsonapi)
[![test](https://github.com/DataDog/jsonapi/actions/workflows/test.yml/badge.svg)](https://github.com/DataDog/jsonapi/actions/workflows/test.yml)
[![golangci-lint](https://github.com/DataDog/jsonapi/actions/workflows/lint.yml/badge.svg)](https://github.com/DataDog/jsonapi/actions/workflows/lint.yml)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/DataDog/jsonapi)

jsonapi
-----

Package jsonapi implements a marshaler/unmarshaler for [JSON:API v1.0](https://jsonapi.org/format/1.0).

# Version

This package is in production use at [DataDog](https://www.datadoghq.com/) and should be considered stable and production ready.

We follow [semver](https://semver.org/) and are reserving the `v1.0.0` release until:

- [JSON:API v1.1](https://jsonapi.org/format/1.1/) is released and we evaluate any breaking changes needed (unlikely)
- Community adoption and feedback are evaluated
- Continued internal use lets us gain more experience with the package

# Quickstart

Take a look at the [go reference](https://pkg.go.dev/github.com/DataDog/jsonapi) for more examples and detailed usage information.

## Marshaling

[jsonapi.Marshal](https://pkg.go.dev/github.com/DataDog/jsonapi#Marshal)

```go
type Article struct {
    ID    string `jsonapi:"primary,articles"`
    Title string `jsonapi:"attribute" json:"title"`
}

a := Article{ID: "1", Title:"Hello World"}

b, err := jsonapi.Marshal(&a)
if err != nil {
    // ...
}

fmt.Println("%s", string(b))
// {
//     "data": {
//         "id": "1",
//         "type": "articles",
//         "attributes": {
//             "title": "Hello World"
//         }
//     }
// }
```

## Unmarshaling

[jsonapi.Unmarshal](https://pkg.go.dev/github.com/DataDog/jsonapi#Marshal)

```go
body := `{"data":{"id":"1","type":"articles","attributes":{"title":"Hello World"}}}`

type Article struct {
    ID    string `jsonapi:"primary,articles"`
    Title string `jsonapi:"attribute" json:"title"`
}

var a Article
if err := jsonapi.Unmarshal([]byte(body), &a); err != nil {
    // ...
}

fmt.Prints("%s, %s", a.ID, a.Title)
// "1", "Hello World"
```

# Reference

The following information is well documented in the [go reference](https://pkg.go.dev/github.com/DataDog/jsonapi). This section is included for a high-level overview of the features available.

## Struct Tags

Like [encoding/json](https://pkg.go.dev/encoding/json) jsonapi is primarily controlled by the presence of a struct tag `jsonapi`. The standard `json` tag is still used for naming and `omitempty`.

| Tag | Usage | Description | Alias |
| --- | --- | --- | --- |
| primary | `jsonapi:"primary,{type},{omitempty}"` | Defines the [identification](https://jsonapi.org/format/1.0/#document-resource-object-identification) field. Including omitempty allows for empty IDs (used for server-side id generation) | N/A |
| attribute | `jsonapi:"attribute"` | Defines an [attribute](https://jsonapi.org/format/1.0/#document-resource-object-attributes). | attr |
| relationship | `jsonapi:"relationship"` | Defines a [relationship](https://jsonapi.org/format/1.0/#document-resource-object-relationships). | rel |
| meta | `jsonapi:"meta"` | Defines a [meta object](https://jsonapi.org/format/1.0/#document-meta). | N/A |

## Functional Options

Both [jsonapi.Marshal](https://pkg.go.dev/github.com/DataDog/jsonapi#Marshal) and [jsonapi.Unmarshal](https://pkg.go.dev/github.com/DataDog/jsonapi#Unmarshal) take functional options.

| Option | Supports |
| --- | --- |
| [jsonapi.MarshalOption](https://pkg.go.dev/github.com/DataDog/jsonapi#MarshalOption) | [meta](https://pkg.go.dev/github.com/DataDog/jsonapi#MarshalMeta), [json:api](https://pkg.go.dev/github.com/DataDog/jsonapi#MarshalJSONAPI), [includes](https://pkg.go.dev/github.com/DataDog/jsonapi#MarshalInclude), [document links](https://pkg.go.dev/github.com/DataDog/jsonapi#MarshalLinks), [sparse fieldsets](https://pkg.go.dev/github.com/DataDog/jsonapi#MarshalFields), [name validation](https://pkg.go.dev/github.com/DataDog/jsonapi#MarshalSetNameValidation) |
| [jsonapi.UnmarshalOption](https://pkg.go.dev/github.com/DataDog/jsonapi#UnmarshalOption) | [meta](https://pkg.go.dev/github.com/DataDog/jsonapi#UnmarshalMeta), [document links](https://pkg.go.dev/github.com/DataDog/jsonapi#UnmarshalLinks), [name validation](https://pkg.go.dev/github.com/DataDog/jsonapi#UnmarshalSetNameValidation) |

## Non-String Identifiers

[Identification](https://jsonapi.org/format/1.0/#document-resource-object-identification) MUST be represented as a `string` regardless of the actual type in Go. To support non-string types for the primary field you can implement optional interfaces.

You can implement the following on the parent types (that contain non-string fields):

| Context | Interface |
| --- | --- |
| Marshal | [jsonapi.MarshalIdentifier](https://pkg.go.dev/github.com/DataDog/jsonapi#MarshalIdentifier) |
| Unmarshal | [jsonapi.UnmarshalIdentifier](https://pkg.go.dev/github.com/DataDog/jsonapi#UnmarshalIdentifier) |

You can implement the following on the field types themselves if they are not already implemented.

| Context | Interface |
| --- | --- |
| Marshal | [fmt.Stringer](https://pkg.go.dev/fmt#Stringer) |
| Marshal | [encoding.TextMarshaler](https://pkg.go.dev/encoding#TextMarshaler) |
| Unmarshal | [encoding.TextUnmarshaler](https://pkg.go.dev/encoding#TextUnmarshaler) |

### Order of Operations

#### Marshaling

1. Use MarshalIdentifier if it is implemented on the parent type
2. Use the value directly if it is a string
3. Use fmt.Stringer if it is implemented
4. Use encoding.TextMarshaler if it is implemented
5. Fail

#### Unmarshaling

1. Use UnmarshalIdentifier if it is implemented on the parent type
2. Use encoding.TextUnmarshaler if it is implemented
3. Use the value directly if it is a string
4. Fail

## Links

[Links](https://jsonapi.org/format/1.0/#document-links) are supported via two interfaces and the [Link](https://pkg.go.dev/github.com/DataDog/jsonapi#Link) type. To include links you must implement one or both of the following interfaces.

| Type | Interface |
| --- | --- |
| [Resource Object Link](https://jsonapi.org/format/1.0/#document-resource-object-links) | [Linkable](https://pkg.go.dev/github.com/DataDog/jsonapi#Linkable) |
| [Resource Object Related Resource Link](https://jsonapi.org/format/1.0/#document-resource-object-related-resource-links) | [LinkableRelation](https://pkg.go.dev/github.com/DataDog/jsonapi#LinkableRelation) |

# Alternatives

## [google/jsonapi](https://github.com/google/jsonapi)

- exposes an API that looks/feels a lot different than encoding/json
- has quite a few bugs w/ complex types in attributes
- doesn't provide easy access to top-level fields like meta
- allows users to generate invalid JSON:API
- not actively maintained

## [manyminds/api2go/jsonapi](https://github.com/manyminds/api2go/tree/master/jsonapi)

- has required interfaces
- interfaces for includes/relationships are hard to understand and verbose to implement
- allows users to generate invalid JSON:API
- part of a broader api2go framework ecosystem
