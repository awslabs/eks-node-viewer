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
	"time"

	v1 "k8s.io/api/core/v1"
)

type objectKey struct {
	namespace string
	name      string
}
type Node struct {
	mu      sync.RWMutex
	visible bool
	node    v1.Node
	pods    map[objectKey]*Pod
	used    v1.ResourceList
	Price   float64
}

func NewNode(n *v1.Node) *Node {
	node := &Node{
		node: *n,
		pods: map[objectKey]*Pod{},
		used: v1.ResourceList{},
	}

	return node
}
func (n *Node) IsOnDemand() bool {
	return n.node.Labels["karpenter.sh/capacity-type"] == "on-demand" ||
		n.node.Labels["eks.amazonaws.com/capacityType"] == "ON_DEMAND"
}

func (n *Node) IsSpot() bool {
	return n.node.Labels["karpenter.sh/capacity-type"] == "spot" ||
		n.node.Labels["eks.amazonaws.com/capacityType"] == "SPOT"
}

func (n *Node) Update(node *v1.Node) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.node = *node
}

func (n *Node) Name() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.node.Name
}

func (n *Node) BindPod(pod *Pod) {
	n.mu.Lock()
	defer n.mu.Unlock()
	key := objectKey{
		namespace: pod.Namespace(),
		name:      pod.Name(),
	}
	_, alreadyBound := n.pods[key]
	n.pods[key] = pod

	if !alreadyBound {
		for rn, q := range pod.Requested() {
			existing := n.used[rn]
			existing.Add(q)
			n.used[rn] = existing
		}
	}
}

func (n *Node) DeletePod(namespace string, name string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	key := objectKey{namespace: namespace, name: name}
	if p, ok := n.pods[key]; ok {
		// subtract the pod requests
		for rn, q := range p.Requested() {
			existing := n.used[rn]
			existing.Sub(q)
			n.used[rn] = existing
		}
		delete(n.pods, key)
	}
}

func (n *Node) Allocatable() v1.ResourceList {
	n.mu.RLock()
	defer n.mu.RUnlock()
	// shouldn't be modified so it's safe to return
	return n.node.Status.Allocatable
}

func (n *Node) Used() v1.ResourceList {
	n.mu.RLock()
	defer n.mu.RUnlock()
	used := v1.ResourceList{}
	for rn, q := range n.used {
		used[rn] = q.DeepCopy()
	}
	return used
}

func (n *Node) Cordoned() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.node.Spec.Unschedulable
}

func (n *Node) Ready() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, c := range n.node.Status.Conditions {
		if c.Status == v1.ConditionTrue && c.Type == v1.NodeReady {
			return true
		}
	}
	return false
}

func (n *Node) Created() time.Time {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.node.CreationTimestamp.Time
}

func (n *Node) InstanceType() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.node.Labels[v1.LabelInstanceTypeStable]
}

func (n *Node) Zone() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.node.Labels[v1.LabelTopologyZone]
}

func (n *Node) NumPods() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.pods)
}

func (n *Node) Hide() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.visible = false
}
func (n *Node) Visible() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.visible
}

func (n *Node) Show() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.visible = true
}

func (n *Node) Deleting() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return !n.node.DeletionTimestamp.IsZero()
}
