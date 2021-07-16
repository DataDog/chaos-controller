// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package validation

type Teststruct struct {
	MinMaxTest      MinMaxTestStruct
	RequiredTest    RequiredTestStruct
	EnumTest        EnumTestStruct
	ExclusiveFields ExclusiveFieldsTestStruct
}

// +ddmark:validation:ExclusiveFields={Subfield1,Subfield2}
type ExclusiveFieldsTestStruct struct {
	Subfield2  int
	PSubfield2 *int
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
	StrField1 string
	// +ddmark:validation:Enum={aa,bb,11}
	PStrField1 *string
}
