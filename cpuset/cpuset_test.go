/*
Copyright 2017 The Kubernetes Authors.
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

/*
This file was copied directly from k8s.io/kubernetes v1.20.2. It is not importable in a normal way, as kubernetes/pkg/kubelet isn't meant to be imported,
 doing so requires `replace` statements that make it difficult to import the chaos-controller from other modules.
*/

package cpuset

import (
	"testing"
)

func FuzzParse(f *testing.F) {
	f.Add("0-3")
	f.Add("1,2,3")
	f.Add("0")
	f.Fuzz(func(t *testing.T, input string) {
		// Parse should either succeed or return an error, but never panic
		cpuset, err := Parse(input)

		// If parsing succeeded, verify basic invariants
		if err == nil {
			// Empty string should produce empty set
			if input == "" && cpuset.Size() != 0 {
				t.Errorf("Parse(%q) produced non-empty set: %v", input, cpuset)
			}
			// Non-empty valid input should produce non-empty set
			// (though we can't easily determine if input is "valid" without re-parsing)
		}
		// If err != nil, that's fine - invalid input should return errors
	})
}
