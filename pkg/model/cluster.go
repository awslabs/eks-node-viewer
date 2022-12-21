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
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/awslabs/eks-node-viewer/pkg/pricing"
)

type Cluster struct {
	mu        sync.RWMutex
	nodes     map[string]*Node
	pods      map[objectKey]*Pod
	resources []v1.ResourceName
}

func NewCluster() *Cluster {
	return &Cluster{
		nodes:     map[string]*Node{},
		pods:      map[objectKey]*Pod{},
		resources: []v1.ResourceName{v1.ResourceCPU},
	}
}
func (c *Cluster) AddNode(node *Node) *Node {
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.nodes[node.Name()]; ok {
		existing.Update(&node.node)
		return existing
	}

	c.nodes[node.Name()] = node
	return node
}

func (c *Cluster) DeleteNode(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.nodes, name)
	var podsToDelete []objectKey
	for k, p := range c.pods {
		if p.NodeName() == name {
			podsToDelete = append(podsToDelete, k)
		}
	}
	for _, k := range podsToDelete {
		delete(c.pods, k)
	}
}

func (c *Cluster) GetNode(name string) (*Node, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	n, ok := c.nodes[name]
	return n, ok
}

func (c *Cluster) AddPod(pod *Pod, pprov *pricing.Provider) (totalPods int) {
	c.mu.Lock()
	c.pods[objectKey{namespace: pod.Namespace(), name: pod.Name()}] = pod
	totalPods = len(c.pods)
	c.mu.Unlock()

	if !pod.IsScheduled() || pod.IsCompleted() {
		return
	}

	n, ok := c.GetNode(pod.NodeName())
	if !ok {
		// node doesn't exist so we need to create it first to have somewhere to record the pod, it will be updated
		// when we are notified about the node by the API Server
		n = c.AddNode(NewNode(&v1.Node{ObjectMeta: metav1.ObjectMeta{Name: pod.NodeName()}}))
		n.Hide()
	}
	n.BindPod(pod)
	return
}

func (c *Cluster) DeletePod(namespace, name string) (totalPods int) {
	p, ok := c.GetPod(namespace, name)
	if ok && p.IsScheduled() {
		n, ok := c.GetNode(p.NodeName())
		if ok {
			n.DeletePod(namespace, name)
		}
	}
	c.mu.Lock()
	delete(c.pods, objectKey{namespace: namespace, name: name})
	totalPods = len(c.pods)
	c.mu.Unlock()
	return
}

func (c *Cluster) GetPod(namespace string, name string) (*Pod, bool) {
	pod, ok := c.pods[objectKey{namespace: namespace, name: name}]
	return pod, ok
}

func (c *Cluster) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	st := Stats{
		AllocatableResources: v1.ResourceList{},
		UsedResources:        v1.ResourceList{},
		PercentUsedResoruces: map[v1.ResourceName]float64{},
		PodsByPhase:          map[v1.PodPhase]int{},
	}

	for _, p := range c.pods {
		// skip pods bound to non-visible nodes
		if n, ok := c.nodes[p.NodeName()]; ok && !n.Visible() {
			continue
		}

		st.TotalPods++
		st.PodsByPhase[p.Phase()]++
		if p.NodeName() != "" {
			st.BoundPodCount++
		}
	}

	for _, n := range c.nodes {
		if !n.Visible() {
			continue
		}
		st.TotalPrice += n.Price
		st.NumNodes++
		st.Nodes = append(st.Nodes, n)
		addResources(st.AllocatableResources, n.Allocatable())
		addResources(st.UsedResources, n.Used())
	}

	sort.Slice(st.Nodes, func(a, b int) bool {
		aCreated := st.Nodes[a].Created()
		bCreated := st.Nodes[b].Created()
		if aCreated == bCreated {
			return st.Nodes[a].Name() < st.Nodes[b].Name()
		}
		return st.Nodes[a].Created().Before(st.Nodes[b].Created())
	})
	return st
}

// addResources sets lhs = lhs + rhs
func addResources(lhs v1.ResourceList, rhs v1.ResourceList) {
	for rn, q := range rhs {
		existing := lhs[rn]
		existing.Add(q)
		lhs[rn] = existing
	}
}
