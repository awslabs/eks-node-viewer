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

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const mebibyte = 1 << 20

// formatQuantity renders a quantity for display. Byte valued resources go to Mi:
// kubelet reports memory in Ki and a quantity keeps the scale it was parsed with,
// so a summed cluster reads 64371224Ki. The unit is fixed rather than scaled per
// value to keep the rows of a column comparable.
func formatQuantity(name v1.ResourceName, q resource.Quantity) string {
	if !isByteValued(name, q) {
		return q.String()
	}
	return fmt.Sprintf("%.0fMi", q.AsApproximateFloat64()/mebibyte)
}

func isByteValued(name v1.ResourceName, q resource.Quantity) bool {
	switch name {
	case v1.ResourceMemory, v1.ResourceEphemeralStorage, v1.ResourceStorage:
		return true
	}
	if strings.HasPrefix(string(name), v1.ResourceHugePagesPrefix) {
		return true
	}
	// a custom resource measured in bytes
	return q.Format == resource.BinarySI
}
