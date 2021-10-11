// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

// This struct cannot be within the ddmark_test package in order to be properly loaded
// by the loader package, inherent to the markers
package ddmark

type Teststruct struct {
	MinMaxTest          MinMaxTestStruct
	RequiredTest        RequiredTestStruct
	EnumTest            EnumTestStruct
	ExclusiveFieldsTest ExclusiveFieldsTestStruct
	LinkedFieldsTest    LinkedFieldsTestStruct
	RequireOneOfTest    RequireOneOfTestStruct
}

// +ddmark:validation:ExclusiveFields={PIntField,PStrField}
// +ddmark:validation:ExclusiveFields={PIntField,BField,CField}
// +ddmark:validation:ExclusiveFields={IntField,StrField}
type ExclusiveFieldsTestStruct struct {
	IntField  int
	PIntField *int
	StrField  string
	PStrField *string
	BField    int
	CField    int
}

type MinMaxTestStruct struct {
	// +ddmark:validation:Minimum=5
	// +ddmark:validation:Maximum=10
	IntField int
	// +ddmark:validation:Minimum=5
	// +ddmark:validation:Maximum=10
	PIntField *int
}

type RequiredTestStruct struct {
	// +ddmark:validation:Required=true
	IntField int
	// +ddmark:validation:Required=true
	PIntField *int
	// +ddmark:validation:Required=true
	StrField string
	// +ddmark:validation:Required=true
	PStrField *string
	// +ddmark:validation:Required=true
	StructField struct {
		A int
	}
	// +ddmark:validation:Required=true
	PStructField *struct {
		A int
	}
}

type EnumTestStruct struct {
	// +ddmark:validation:Enum={aa,bb,11}
	StrField string
	// +ddmark:validation:Enum={aa,bb,11}
	PStrField *string
	// +ddmark:validation:Enum={1,2,3}
	IntField int
	// +ddmark:validation:Enum={1,2,3}
	PIntField *int
}

// +ddmark:validation:LinkedFields={StrField,IntField}
// +ddmark:validation:LinkedFields={PStrField,PIntField,AIntField}
type LinkedFieldsTestStruct struct {
	RandomIntField int // allows to actually check all-empty structs
	StrField       string
	PStrField      *string
	IntField       int
	PIntField      *int
	AIntField      []int
}

// +ddmark:validation:RequireOneOf={StrField,IntField}
// +ddmark:validation:RequireOneOf={PStrField,PIntField,AIntField}
type RequireOneOfTestStruct struct {
	RandomIntField int // allows to actually check all-empty structs
	StrField       string
	PStrField      *string
	IntField       int
	PIntField      *int
	AIntField      []int
}
