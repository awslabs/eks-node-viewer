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
	"log"
	"regexp"
	"strconv"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Pod is our pod model used for internal storage and display
type Pod struct {
	mu  sync.RWMutex
	pod v1.Pod
}

// NewPod constructs a pod model based off of the K8s pod object
func NewPod(n *v1.Pod) *Pod {
	return &Pod{
		pod: *n,
	}
}

// Update updates the pod model, replacing it with a shallow copy of the provided pod
func (p *Pod) Update(pod *v1.Pod) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pod = *pod
}

// IsScheduled returns true if the pod has been scheduled to a node
func (p *Pod) IsScheduled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.pod.Spec.NodeName != ""
}

// NodeName returns the node that the pod is scheduled against, or an empty string
func (p *Pod) NodeName() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.pod.Spec.NodeName
}

// Namespace returns the namespace of the pod
func (p *Pod) Namespace() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.pod.Namespace
}

// Name returns the name of the pod
func (p *Pod) Name() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.pod.Name
}

// Phase returns the pod phase
func (p *Pod) Phase() v1.PodPhase {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.pod.Status.Phase
}

// Requested returns the sum of the resources requested by the pod.
// Also include resources for init containers that are sidecars as described in
// https://kubernetes.io/blog/2023/08/25/native-sidecar-containers .
func (p *Pod) Requested() v1.ResourceList {
	p.mu.RLock()
	defer p.mu.RUnlock()
	requested := v1.ResourceList{}
	for _, c := range p.pod.Spec.InitContainers {
		if c.RestartPolicy == nil || *c.RestartPolicy != v1.ContainerRestartPolicyAlways {
			continue
		}
		for rn, q := range c.Resources.Requests {
			existing := requested[rn]
			existing.Add(q)
			requested[rn] = existing
		}
	}
	for _, c := range p.pod.Spec.Containers {
		for rn, q := range c.Resources.Requests {
			existing := requested[rn]
			existing.Add(q)
			requested[rn] = existing
		}
	}
	requested[v1.ResourcePods] = resource.MustParse("1")
	return requested
}

var fargateCapacityRe = regexp.MustCompile("(.*?)vCPU (.*?)GB")

func (p *Pod) FargateCapacityProvisioned() (float64, float64, bool) {
	provisioned, ok := p.pod.Annotations["CapacityProvisioned"]
	if !ok {
		return 0, 0, false
	}

	match := fargateCapacityRe.FindStringSubmatch(provisioned)
	if len(match) != 3 {
		log.Printf("unable to parse %q for fargate provisioner capacity", provisioned)
	}
	cpu, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		log.Printf("unable to parse CPU from fargate capacity, %q, %s", provisioned, err)
		return 0, 0, false
	}
	mem, err := strconv.ParseFloat(match[2], 64)
	if err != nil {
		log.Printf("unable to parse memory from fargate capacity, %q, %s", provisioned, err)
		return 0, 0, false
	}
	return cpu, mem, true
}
