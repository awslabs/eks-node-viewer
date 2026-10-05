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
	"sort"

	"github.com/facette/natsort"
	v1 "k8s.io/api/core/v1"
)

// UngroupedValue names the group holding nodes without the label.
const UngroupedValue = "<none>"

// GroupStats is usage and cost summed over the nodes sharing a label value.
type GroupStats struct {
	Name                 string
	NumNodes             int
	NumPods              int
	AllocatableResources v1.ResourceList
	UsedResources        v1.ResourceList
	TotalPrice           float64
	// PricedNodes is how many nodes contributed to TotalPrice
	PricedNodes int
}

// PercentUsed is the group's usage as a whole, so weighted by node size rather
// than the mean of each node's own percentage.
func (g GroupStats) PercentUsed(resource v1.ResourceName) float64 {
	allocatable := g.AllocatableResources[resource]
	if allocatable.AsApproximateFloat64() == 0 {
		return 0
	}
	used := g.UsedResources[resource]
	return 100 * (used.AsApproximateFloat64() / allocatable.AsApproximateFloat64())
}

// FullyPriced tells whether TotalPrice is the real cost or a lower bound.
func (g GroupStats) FullyPriced() bool {
	return g.PricedNodes == g.NumNodes
}

// GroupNodes aggregates nodes by their value for the label, sorted by name with
// the nodes missing the label last.
func GroupNodes(nodes []*Node, label string) []GroupStats {
	byValue := map[string]*GroupStats{}
	for _, n := range nodes {
		value := n.LabelValue(label)
		group, ok := byValue[value]
		if !ok {
			group = &GroupStats{
				Name:                 value,
				AllocatableResources: v1.ResourceList{},
				UsedResources:        v1.ResourceList{},
			}
			byValue[value] = group
		}
		group.NumNodes++
		group.NumPods += n.NumPods()
		n.AddAllocatableTo(group.AllocatableResources)
		n.AddUsedTo(group.UsedResources)
		if n.HasPrice() {
			group.TotalPrice += n.Price
			group.PricedNodes++
		}
	}

	groups := make([]GroupStats, 0, len(byValue))
	for _, g := range byValue {
		groups = append(groups, *g)
	}
	sort.Slice(groups, func(a, b int) bool {
		if (groups[a].Name == UngroupedValue) != (groups[b].Name == UngroupedValue) {
			return groups[b].Name == UngroupedValue
		}
		return natsort.Compare(groups[a].Name, groups[b].Name)
	})
	return groups
}
