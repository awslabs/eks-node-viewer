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
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/awslabs/eks-node-viewer/pkg/model"
)

func TestClusterAddNode(t *testing.T) {
	cluster := model.NewCluster()

	if got := len(cluster.Stats().Nodes); got != 0 {
		t.Errorf("expected 0 nodes, got %d", got)
	}
	nodeCount := 0
	cluster.ForEachNode(func(n *model.Node) {
		nodeCount++
	})
	if got := nodeCount; got != 0 {
		t.Errorf("expected to iterate over 0 nodes, had %d", got)
	}

	n := testNode("mynode")
	node := model.NewNode(n)
	cluster.AddNode(node)

	// doesn't show not-visible node
	if got := len(cluster.Stats().Nodes); got != 0 {
		t.Errorf("expected 0 nodes, got %d", got)
	}

	// but is iterable
	cluster.ForEachNode(func(n *model.Node) {
		nodeCount++
	})
	if got := nodeCount; got != 1 {
		t.Errorf("expected to iterate over 1 node, had %d", got)
	}

	// making the node visible causes it to appear in stats
	node.Show()
	if got := len(cluster.Stats().Nodes); got != 1 {
		t.Errorf("expected 1 nodes, got %d", got)

	}

}

func TestClusterGetNodeByProviderID(t *testing.T) {
	cluster := model.NewCluster()

	_, ok := cluster.GetNode("mynode-id")
	if ok {
		t.Errorf("expected to not find node")
	}
	n := testNode("mynode")
	n.Spec.ProviderID = "mynode-id"
	node := model.NewNode(n)
	cluster.AddNode(node)

	_, ok = cluster.GetNode("mynode-id")
	if !ok {
		t.Errorf("expected to find node by provider id")
	}

	// delete and we should fail to find it
	cluster.DeleteNode("mynode-id")
	_, ok = cluster.GetNode("mynode-id")
	if ok {
		t.Errorf("expected to not find node after deletion")
	}
}

func TestClusterGetNodeByName(t *testing.T) {
	cluster := model.NewCluster()

	_, ok := cluster.GetNodeByName("mynode")
	if ok {
		t.Errorf("expected to not find node")
	}
	n := testNode("mynode")
	node := model.NewNode(n)
	cluster.AddNode(node)

	_, ok = cluster.GetNodeByName("mynode")
	if !ok {
		t.Errorf("expected to find node by name")
	}
}

func TestClusterUpdateNode(t *testing.T) {
	cluster := model.NewCluster()

	n1 := testNode("mynode")
	n1.Status.Allocatable = v1.ResourceList{
		"cpu": resource.MustParse("1"),
	}
	node1 := model.NewNode(n1)
	node1.Show()
	cluster.AddNode(node1)

	if got := cluster.Stats().AllocatableResources["cpu"]; got.Cmp(resource.MustParse("1")) != 0 {
		t.Errorf("expected total CPU = 1, got %s", got.String())
	}

	// simulate a node update
	n2 := testNode("mynode")
	n2.Status.Allocatable = v1.ResourceList{
		"cpu": resource.MustParse("2"),
	}
	node2 := model.NewNode(n2)
	node2.Show()
	cluster.AddNode(node2)

	if got := cluster.Stats().AllocatableResources["cpu"]; got.Cmp(resource.MustParse("2")) != 0 {
		t.Errorf("expected total CPU = 2, got %s", got.String())
	}

}

func TestClusterAddPod(t *testing.T) {
	cluster := model.NewCluster()

	n := testNode("mynode")
	n.Spec.ProviderID = "mynode-id"
	node := model.NewNode(n)
	node.Show()
	cluster.AddNode(node)

	if got := cluster.Stats().TotalPods; got != 0 {
		t.Errorf("expected 0 pods, got %d", got)
	}
	if got := cluster.Stats().UsedResources["cpu"]; got.Cmp(resource.MustParse("0")) != 0 {
		t.Errorf("expected 0 CPU used, got %s", got.String())
	}

	p := testPod("default", "mypod")
	p.Spec.NodeName = n.Name
	pod := model.NewPod(p)
	cluster.AddPod(pod)

	if got := cluster.Stats().TotalPods; got != 1 {
		t.Errorf("expected 0 pods, got %d", got)
	}

	if got := cluster.Stats().UsedResources["cpu"]; got.Cmp(resource.MustParse("2")) != 0 {
		t.Errorf("expected 2 CPU used, got %s", got.String())
	}

	// deleting the pod should remove the usage
	cluster.DeletePod("default", "mypod")
	if got := cluster.Stats().TotalPods; got != 0 {
		t.Errorf("expected 0 pods, got %d", got)
	}
	if got := cluster.Stats().UsedResources["cpu"]; got.Cmp(resource.MustParse("0")) != 0 {
		t.Errorf("expected 0 CPU used, got %s", got.String())
	}

}

func TestClusterDeleteNodeDeletesPods(t *testing.T) {
	cluster := model.NewCluster()

	// add a node and pod bound to that node
	n := testNode("mynode")
	n.Spec.ProviderID = "mynode-id"
	node := model.NewNode(n)
	node.Show()
	cluster.AddNode(node)

	p := testPod("default", "mypod")
	p.Spec.NodeName = n.Name
	pod := model.NewPod(p)
	cluster.AddPod(pod)

	// verify we are tracking usage
	if got := cluster.Stats().TotalPods; got != 1 {
		t.Errorf("expected 0 pods, got %d", got)
	}

	if got := cluster.Stats().UsedResources["cpu"]; got.Cmp(resource.MustParse("2")) != 0 {
		t.Errorf("expected 2 CPU used, got %s", got.String())
	}

	// deleting the node should clear all of the usage of pods that were bound to the node
	cluster.DeleteNode("mynode-id")

	if got := cluster.Stats().TotalPods; got != 0 {
		t.Errorf("expected 0 pods, got %d", got)
	}
	if got := cluster.Stats().UsedResources["cpu"]; got.Cmp(resource.MustParse("0")) != 0 {
		t.Errorf("expected 0 CPU used, got %s", got.String())
	}

}

// pods are watched cluster wide while nodes can be filtered, so the counts have to
// leave out the pods running somewhere we aren't displaying
func TestClusterStatsCountsOnlyPodsOnVisibleNodes(t *testing.T) {
	cluster := model.NewCluster()

	for _, name := range []string{"node-a", "node-b"} {
		n := testNode(name)
		n.Spec.ProviderID = "aws:///us-west-2a/i-" + name
		node := model.NewNode(n)
		node.Show()
		cluster.AddNode(node)
	}

	addPod := func(name, nodeName string, phase v1.PodPhase) {
		p := testPod("default", name)
		p.Spec.NodeName = nodeName
		p.Status.Phase = phase
		cluster.AddPod(model.NewPod(p))
	}

	// on the nodes we are displaying
	addPod("mine-1", "node-a", v1.PodRunning)
	addPod("mine-2", "node-a", v1.PodRunning)
	addPod("mine-3", "node-b", v1.PodRunning)
	// on a node that the node selector filtered out, so we never saw the node
	for i := 0; i < 5; i++ {
		addPod(fmt.Sprintf("other-%d", i), "node-not-watched", v1.PodRunning)
	}
	// not scheduled anywhere yet, so they belong to no node
	addPod("pending-1", "", v1.PodPending)
	addPod("pending-2", "", v1.PodPending)

	st := cluster.Stats()
	if got := st.BoundPodCount; got != 3 {
		t.Errorf("expected 3 bound pods, got %d", got)
	}
	// 3 on our nodes plus the 2 unscheduled, never the 5 elsewhere
	if got := st.TotalPods; got != 5 {
		t.Errorf("expected 5 pods, got %d", got)
	}
	if got := st.PodsByPhase[v1.PodRunning]; got != 3 {
		t.Errorf("expected 3 running, got %d", got)
	}
	if got := st.PodsByPhase[v1.PodPending]; got != 2 {
		t.Errorf("expected 2 pending, got %d", got)
	}

	// hiding a node drops its pods from the counts too
	node, ok := cluster.GetNodeByName("node-a")
	if !ok {
		t.Fatal("expected to find node-a")
	}
	node.Hide()

	st = cluster.Stats()
	if got := st.NumNodes; got != 1 {
		t.Errorf("expected 1 visible node, got %d", got)
	}
	if got := st.BoundPodCount; got != 1 {
		t.Errorf("expected 1 bound pod, got %d", got)
	}
	if got := st.TotalPods; got != 3 {
		t.Errorf("expected 3 pods, got %d", got)
	}
}
