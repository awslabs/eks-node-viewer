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

// naturalLess is unexported, so this is an internal test
package model

import "testing"

func TestNaturalLess(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		less bool
	}{
		// digit runs compare by value, not by length
		{"pool-2", "pool-10", true},
		{"pool-10", "pool-2", false},
		{"2", "10", true},
		{"10", "2", false},
		{"node-9", "node-10", true},
		// real node names
		{"ip-10-0-1-3.ec2.internal", "ip-10-0-1-23.ec2.internal", true},
		{"ip-10-0-1-23.ec2.internal", "ip-10-0-1-3.ec2.internal", false},
		{"ip-10-0-2-7.ec2.internal", "ip-10-0-12-7.ec2.internal", true},
		// plain text
		{"a", "ab", true},
		{"ab", "a", false},
		{"gpu", "spot", true},
		// nothing sorts before itself
		{"pool-2", "pool-2", false},
		{"", "", false},
		{"", "a", true},
	} {
		if got := naturalLess(tc.a, tc.b); got != tc.less {
			t.Errorf("naturalLess(%q, %q) = %v, expected %v", tc.a, tc.b, got, tc.less)
		}
	}
}

// the comparison has to be a strict ordering or sort.Slice can misbehave
func TestNaturalLessIsAStrictOrdering(t *testing.T) {
	values := []string{
		"pool-1", "pool-2", "pool-10", "pool-02", "pool", "", "9", "10",
		"ip-10-0-1-3.ec2.internal", "ip-10-0-1-23.ec2.internal", "a-1-b", "a-1-c",
	}
	for _, a := range values {
		if naturalLess(a, a) {
			t.Errorf("naturalLess(%q, %q) must be false", a, a)
		}
		for _, b := range values {
			if a == b {
				continue
			}
			// exactly one direction holds
			if naturalLess(a, b) == naturalLess(b, a) {
				t.Errorf("%q and %q compare equal in both directions", a, b)
			}
		}
	}
}
