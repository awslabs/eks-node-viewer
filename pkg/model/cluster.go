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
	"sync"

	v1 "k8s.io/api/core/v1"
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
	if existing, ok := c.nodes[node.ProviderID()]; ok {
		existing.Update(&node.node)
		return existing
	}

	c.nodes[node.ProviderID()] = node
	return node
}

func (c *Cluster) DeleteNode(providerID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n, ok := c.nodes[providerID]
	if !ok {
		return
	}
	var podsToDelete []objectKey
	for k, p := range c.pods {
		if p.NodeName() == n.node.Name {
			podsToDelete = append(podsToDelete, k)
		}
	}
	for _, k := range podsToDelete {
		delete(c.pods, k)
	}
	delete(c.nodes, providerID)
}

func (c *Cluster) ForEachNode(f func(n *Node)) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, n := range c.nodes {
		f(n)
	}
}

func (c *Cluster) GetNode(providerID string) (*Node, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	n, ok := c.nodes[providerID]
	return n, ok
}

func (c *Cluster) GetNodeByName(name string) (*Node, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, n := range c.nodes {
		if n.node.Name == name {
			return n, true
		}
	}
	return nil, false
}

func (c *Cluster) AddPod(pod *Pod) (totalPods int) {
	c.mu.Lock()
	c.pods[objectKey{namespace: pod.Namespace(), name: pod.Name()}] = pod
	totalPods = len(c.pods)
	c.mu.Unlock()

	if !pod.IsScheduled() {
		return
	}
	n, ok := c.GetNodeByName(pod.NodeName())
	if !ok {
		return
	}
	n.BindPod(pod)
	return
}

func (c *Cluster) DeletePod(namespace, name string) (totalPods int) {
	p, ok := c.GetPod(namespace, name)
	if ok && p.IsScheduled() {
		n, ok := c.GetNodeByName(p.NodeName())
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
	c.mu.Lock()
	pod, ok := c.pods[objectKey{namespace: namespace, name: name}]
	c.mu.Unlock()
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
		// only add the price if it's not NaN which is used to indicate an unknown
		// price
		if n.HasPrice() {
			st.TotalPrice += n.Price
		}
		st.NumNodes++
		st.Nodes = append(st.Nodes, n)
		addResources(st.AllocatableResources, n.Allocatable())
		addResources(st.UsedResources, n.Used())
	}
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
