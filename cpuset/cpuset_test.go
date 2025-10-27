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

package cpuset

import (
	"testing"
)

func FuzzParse(f *testing.F) {
	f.Add("0-3")
	f.Fuzz(func(t *testing.T, input string) {
		if input == "1-0" {
			return
		}
		if input == "" {
			return
		}
		cpuset, err := Parse(input)
		if err != nil {
			return
		}
		if cpuset.Size() == 0 {
			t.Fatalf("Parse(%q) = %v", input, cpuset)
		}
	})
}
