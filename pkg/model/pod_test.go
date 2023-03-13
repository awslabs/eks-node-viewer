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
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/awslabs/eks-node-viewer/pkg/model"
)

func testPod(namespace, name string) *v1.Pod {
	p := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Status: v1.PodStatus{
			Phase: v1.PodPending,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Image: "test-image",
					Name:  "container",
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("1"),
							v1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
				},
			},
		},
	}
	return p
}
func TestNewPod(t *testing.T) {
	pod := testPod("default", "mypod")
	pod.Spec.NodeName = "mynode"
	p := model.NewPod(pod)
	if exp, got := "default", p.Namespace(); exp != got {
		t.Errorf("expected Namespace = %s, got %s", exp, got)
	}
	if exp, got := "mypod", p.Name(); exp != got {
		t.Errorf("expected Name = %s, got %s", exp, got)
	}
	if exp, got := "mynode", p.NodeName(); exp != got {
		t.Errorf("expected NodeName = %s, got %s", exp, got)
	}
	if exp, got := true, p.IsScheduled(); exp != got {
		t.Errorf("expected IsScheduled = %v, got %v", exp, got)
	}
	if exp, got := v1.PodPending, p.Phase(); exp != got {
		t.Errorf("expected Phase = %v, got %v", exp, got)
	}

	if exp, got := resource.MustParse("1"), p.Requested()[v1.ResourceCPU]; exp.Cmp(got) != 0 {
		t.Errorf("expected CPU = %s, got %s", exp.String(), got.String())
	}
	if exp, got := resource.MustParse("1Gi"), p.Requested()[v1.ResourceMemory]; exp.Cmp(got) != 0 {
		t.Errorf("expected Memory = %s, got %s", exp.String(), got.String())
	}
}

func TestPodUpdate(t *testing.T) {
	p := model.NewPod(testPod("default", "mypod"))
	if exp, got := "", p.NodeName(); got != exp {
		t.Errorf("expeted NodeName == %s, got %s", exp, got)
	}
	replacement := testPod("default", "mypod")
	replacement.Spec.NodeName = "scheduled.node"
	p.Update(replacement)
	if exp, got := "scheduled.node", p.NodeName(); got != exp {
		t.Errorf("expeted NodeName == %s, got %s", exp, got)
	}
}

func TestFargateCapacity(t *testing.T) {
	tp := testPod("default", "mypod")
	tp.Annotations = map[string]string{
		"CapacityProvisioned": "0.25vCPU 0.5GB",
	}
	p := model.NewPod(tp)
	cpu, mem, ok := p.FargateCapacityProvisioned()
	if !ok {
		t.Errorf("expected to have a fargate capacity")
	}
	if cpu != 0.25 {
		t.Errorf("expected to have a cpu capacity of 0.25, got %f", cpu)
	}
	if mem != 0.5 {
		t.Errorf("expected to have a mem capacity of 0.5, got %f", mem)
	}
}
