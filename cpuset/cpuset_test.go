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
