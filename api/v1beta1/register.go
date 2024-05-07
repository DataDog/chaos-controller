// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// ClientSchemeBuilder is exported for client-go purposes
	ClientSchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&Disruption{},
		&DisruptionList{},
		&DisruptionCron{},
		&DisruptionCronList{},
	)

	scheme.AddKnownTypes(schema.GroupVersion{Group: GroupName, Version: runtime.APIVersionInternal},
		&Disruption{},
		&DisruptionList{},
		&DisruptionCron{},
		&DisruptionCronList{},
	)

	metav1.AddToGroupVersion(scheme, GroupVersion)

	return nil
}
