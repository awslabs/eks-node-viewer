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
package model_test

import (
	"fmt"
	"math"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/awslabs/eks-node-viewer/pkg/model"
)

const nodePoolLabel = "karpenter.sh/nodepool"

func groupTestNode(name, nodepool, cpu string, price float64) *model.Node {
	n := testNode(name)
	n.Spec.ProviderID = "aws:///us-west-2a/i-" + name
	if nodepool != "" {
		n.Labels = map[string]string{nodePoolLabel: nodepool}
	}
	n.Status.Allocatable = v1.ResourceList{
		v1.ResourceCPU:  resource.MustParse(cpu),
		v1.ResourcePods: resource.MustParse("110"),
	}
	node := model.NewNode(n)
	node.Price = price
	node.Show()
	return node
}

// bindPods attaches n pods, each requesting 2 CPU, to the named node
func bindPods(c *model.Cluster, nodeName string, count int) {
	for i := 0; i < count; i++ {
		p := testPod("default", fmt.Sprintf("pod-%s-%d", nodeName, i))
		p.Spec.NodeName = nodeName
		p.Status.Phase = v1.PodRunning
		c.AddPod(model.NewPod(p))
	}
}

func TestGroupNodes(t *testing.T) {
	c := model.NewCluster()
	c.AddNode(groupTestNode("a", "default", "4", 0.10))
	c.AddNode(groupTestNode("b", "default", "4", 0.10))
	c.AddNode(groupTestNode("c", "gpu", "8", 1.00))
	// no nodepool label and an unknown price
	c.AddNode(groupTestNode("d", "", "2", math.NaN()))

	bindPods(c, "a", 1)
	bindPods(c, "c", 2)

	groups := model.GroupNodes(c.Stats().Nodes, nodePoolLabel)
	if got := len(groups); got != 3 {
		t.Fatalf("expected 3 groups, got %d", got)
	}

	// nodes without the label sort last
	for i, exp := range []string{"default", "gpu", model.UngroupedValue} {
		if got := groups[i].Name; got != exp {
			t.Errorf("expected group %d to be %s, got %s", i, exp, got)
		}
	}

	def := groups[0]
	if got := def.NumNodes; got != 2 {
		t.Errorf("expected 2 nodes in default, got %d", got)
	}
	if got := def.NumPods; got != 1 {
		t.Errorf("expected 1 pod in default, got %d", got)
	}
	if got := def.AllocatableResources[v1.ResourceCPU]; got.Cmp(resource.MustParse("8")) != 0 {
		t.Errorf("expected 8 CPU allocatable in default, got %s", got.String())
	}
	if got := def.UsedResources[v1.ResourceCPU]; got.Cmp(resource.MustParse("2")) != 0 {
		t.Errorf("expected 2 CPU used in default, got %s", got.String())
	}
	// weighted over the group, not the mean of each node
	if got := def.PercentUsed(v1.ResourceCPU); got != 25 {
		t.Errorf("expected default at 25%% CPU, got %f", got)
	}
	if got := def.TotalPrice; got != 0.20 {
		t.Errorf("expected default to cost 0.20, got %f", got)
	}
	if !def.FullyPriced() {
		t.Error("expected default to be fully priced")
	}

	if got := groups[1].PercentUsed(v1.ResourceCPU); got != 50 {
		t.Errorf("expected gpu at 50%% CPU, got %f", got)
	}

	// a node with no known price contributes nothing and marks the group
	ungrouped := groups[2]
	if got := ungrouped.TotalPrice; got != 0 {
		t.Errorf("expected ungrouped to cost 0, got %f", got)
	}
	if ungrouped.FullyPriced() {
		t.Error("expected ungrouped to not be fully priced")
	}
	// a resource the group doesn't have must not divide by zero
	if got := ungrouped.PercentUsed(v1.ResourceMemory); got != 0 {
		t.Errorf("expected 0%% for a missing resource, got %f", got)
	}
}

func TestGroupNodesSortsNaturally(t *testing.T) {
	c := model.NewCluster()
	for _, pool := range []string{"pool-10", "pool-2", "pool-1"} {
		c.AddNode(groupTestNode(pool, pool, "4", 0.10))
	}

	groups := model.GroupNodes(c.Stats().Nodes, nodePoolLabel)
	for i, exp := range []string{"pool-1", "pool-2", "pool-10"} {
		if got := groups[i].Name; got != exp {
			t.Errorf("expected group %d to be %s, got %s", i, exp, got)
		}
	}
}

func TestGroupNodesByMissingLabel(t *testing.T) {
	c := model.NewCluster()
	c.AddNode(groupTestNode("a", "default", "4", 0.10))
	c.AddNode(groupTestNode("b", "gpu", "4", 0.10))

	// nothing carries the label, so everything lands in one group
	groups := model.GroupNodes(c.Stats().Nodes, "company.com/team")
	if got := len(groups); got != 1 {
		t.Fatalf("expected 1 group, got %d", got)
	}
	if got := groups[0].Name; got != model.UngroupedValue {
		t.Errorf("expected group %s, got %s", model.UngroupedValue, got)
	}
	if got := groups[0].NumNodes; got != 2 {
		t.Errorf("expected 2 nodes, got %d", got)
	}
}
