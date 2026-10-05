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

// formatQuantity is unexported, so this is an internal test
package model

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestFormatQuantity(t *testing.T) {
	for _, tc := range []struct {
		name     v1.ResourceName
		quantity string
		expected string
	}{
		// kubelet reports memory in Ki, which is what we don't want to display
		{v1.ResourceMemory, "1048576Ki", "1024Mi"},
		{v1.ResourceMemory, "64371224Ki", "62863Mi"},
		{v1.ResourceMemory, "2Gi", "2048Mi"},
		{v1.ResourceMemory, "512Mi", "512Mi"},
		{v1.ResourceMemory, "0", "0Mi"},
		// bytes without a suffix are still bytes
		{v1.ResourceEphemeralStorage, "2147483648", "2048Mi"},
		{"hugepages-2Mi", "4Mi", "4Mi"},
		// CPU and pod counts keep their canonical form
		{v1.ResourceCPU, "15890m", "15890m"},
		{v1.ResourceCPU, "16", "16"},
		{v1.ResourcePods, "234", "234"},
	} {
		q := resource.MustParse(tc.quantity)
		if got := formatQuantity(tc.name, q); got != tc.expected {
			t.Errorf("formatQuantity(%s, %s) = %s, expected %s", tc.name, tc.quantity, got, tc.expected)
		}
	}
}

// summing across nodes is what makes the Ki scale unreadable
func TestFormatQuantitySummed(t *testing.T) {
	total := v1.ResourceList{}
	for i := 0; i < 6; i++ {
		addResources(total, v1.ResourceList{
			v1.ResourceMemory: resource.MustParse("64371224Ki"),
			v1.ResourceCPU:    resource.MustParse("15890m"),
		})
	}

	memory := total[v1.ResourceMemory]
	if got, exp := memory.String(), "386227344Ki"; got != exp {
		t.Errorf("expected the raw sum to be %s, got %s", exp, got)
	}
	if got, exp := formatQuantity(v1.ResourceMemory, memory), "377175Mi"; got != exp {
		t.Errorf("expected %s, got %s", exp, got)
	}

	cpu := total[v1.ResourceCPU]
	if got, exp := formatQuantity(v1.ResourceCPU, cpu), "95340m"; got != exp {
		t.Errorf("expected %s, got %s", exp, got)
	}
}
