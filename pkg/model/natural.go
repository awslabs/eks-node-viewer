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

package model

// naturalLess compares a and b with their digit runs ordered by value, so node-2
// sorts before node-10. It walks both strings in place: the natural sort helpers
// split into chunks with a regexp, which dominates a frame full of nodes.
func naturalLess(a, b string) bool {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if isDigit(a[i]) && isDigit(b[j]) {
			// skip leading zeros so 007 and 7 are the same value
			for i < len(a) && a[i] == '0' {
				i++
			}
			for j < len(b) && b[j] == '0' {
				j++
			}
			endA, endB := i, j
			for endA < len(a) && isDigit(a[endA]) {
				endA++
			}
			for endB < len(b) && isDigit(b[endB]) {
				endB++
			}
			// the longer run is the bigger number, equal lengths compare byte wise
			if endA-i != endB-j {
				return endA-i < endB-j
			}
			if numA, numB := a[i:endA], b[j:endB]; numA != numB {
				return numA < numB
			}
			i, j = endA, endB
			continue
		}
		if a[i] != b[j] {
			return a[i] < b[j]
		}
		i++
		j++
	}
	if len(a)-i != len(b)-j {
		return len(a)-i < len(b)-j
	}
	// equal values, so order on the raw bytes to stay deterministic
	return a < b
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
