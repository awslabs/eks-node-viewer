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
	"k8s.io/apimachinery/pkg/api/resource"
	"reflect"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/awslabs/eks-node-viewer/pkg/model"
)

func testNode(name string) *v1.Node {
	n := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: v1.NodeStatus{
			Phase: v1.NodePending,
		},
	}
	return n
}

func TestNewNode(t *testing.T) {
	n := testNode("mynode")
	node := model.NewNode(n)
	if exp, got := "mynode", node.Name(); exp != got {
		t.Errorf("expeted Name == %s, got %s", exp, got)
	}
}

func TestNodeTypeUnknown(t *testing.T) {
	n := testNode("mynode")
	node := model.NewNode(n)
	if node.IsOnDemand() {
		t.Errorf("exepcted to not be on-demand")
	}
	if node.IsSpot() {
		t.Errorf("exepcted to not be spot")
	}
}

func TestNodeTypeOnDemand(t *testing.T) {
	for label, value := range map[string]string{
		"karpenter.sh/capacity-type":     "on-demand",
		"eks.amazonaws.com/capacityType": "ON_DEMAND",
	} {
		n := testNode("mynode")
		n.Labels = map[string]string{
			label: value,
		}
		node := model.NewNode(n)
		if !node.IsOnDemand() {
			t.Errorf("exepcted on-demand")
		}
		if node.IsSpot() {
			t.Errorf("exepcted to not be spot")
		}
		if node.IsFargate() {
			t.Errorf("exepcted to not be fargate")
		}
	}
}

func TestNodeTypeSpot(t *testing.T) {
	for label, value := range map[string]string{
		"karpenter.sh/capacity-type":     "spot",
		"eks.amazonaws.com/capacityType": "SPOT",
	} {
		n := testNode("mynode")
		n.Labels = map[string]string{
			label: value,
		}
		node := model.NewNode(n)
		if node.IsOnDemand() {
			t.Errorf("exepcted to not be on-demand")
		}
		if !node.IsSpot() {
			t.Errorf("exepcted to be spot")
		}
		if node.IsFargate() {
			t.Errorf("exepcted to not be fargate")
		}
	}
}

func TestNodeTypeFargate(t *testing.T) {
	for label, value := range map[string]string{
		"eks.amazonaws.com/compute-type": "fargate",
	} {
		n := testNode("mynode")
		n.Labels = map[string]string{
			label: value,
		}
		node := model.NewNode(n)
		if node.IsOnDemand() {
			t.Errorf("exepcted to not be on-demand")
		}
		if node.IsSpot() {
			t.Errorf("exepcted to not be spot")
		}
		if !node.IsFargate() {
			t.Errorf("exepcted to be fargate")
		}
	}
}

func TestNodeTypeAuto(t *testing.T) {
	for label, value := range map[string]string{
		"eks.amazonaws.com/compute-type": "auto",
	} {
		n := testNode("mynode")
		n.Labels = map[string]string{
			label: value,
		}
		node := model.NewNode(n)
		if node.IsOnDemand() {
			t.Errorf("exepcted to not be on-demand")
		}
		if node.IsSpot() {
			t.Errorf("exepcted to not be spot")
		}
		if node.IsFargate() {
			t.Errorf("exepcted to not be fargate")
		}
		if !node.IsAuto() {
			t.Errorf("exepcted to be auto")
		}
	}
}

func TestNodeNotReadyFalse(t *testing.T) {
	for _, status := range []v1.ConditionStatus{v1.ConditionFalse, v1.ConditionUnknown} {
		t.Run(string(status), func(t *testing.T) {
			n := testNode("mynode")
			n.Status.Phase = v1.NodeRunning
			notReadyTime := time.Now().Add(-1 * time.Hour)

			n.Status.Conditions = append(n.Status.Conditions, v1.NodeCondition{
				Type:   v1.NodeReady,
				Status: status,
				LastTransitionTime: metav1.Time{
					Time: notReadyTime,
				},
			})
			node := model.NewNode(n)
			if node.Ready() {
				t.Fatalf("expected node to be not ready")
			}

			if node.NotReadyTime() != notReadyTime {
				t.Errorf("expected not ready time = %s, got %s", notReadyTime, node.NotReadyTime())
			}
		})
	}
}

func TestNodeNotReadyNoCondition(t *testing.T) {
	for _, status := range []v1.ConditionStatus{v1.ConditionFalse, v1.ConditionUnknown} {
		t.Run(string(status), func(t *testing.T) {
			n := testNode("mynode")
			n.Status.Phase = v1.NodeRunning
			notReadyTime := time.Now().Add(-1 * time.Hour)
			n.CreationTimestamp = metav1.NewTime(notReadyTime)

			node := model.NewNode(n)
			if node.Ready() {
				t.Fatalf("expected node to be not ready")
			}

			if node.NotReadyTime() != notReadyTime {
				t.Errorf("expected not ready time = %s, got %s", notReadyTime, node.NotReadyTime())
			}
		})
	}
}

func TestNode_UsedPct(t *testing.T) {
	type args struct {
		res                  v1.ResourceName
		normalizedAllocation bool
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "cpu used",
			args: args{
				res: v1.ResourceCPU,
			},
			want: 0.25,
		},
		{
			name: "memory used",
			args: args{
				res: v1.ResourceMemory,
			},
			want: 0.50,
		},
		{
			name: "cpu used normalized",
			args: args{
				res:                  v1.ResourceCPU,
				normalizedAllocation: true,
			},
			want: 0.50,
		},
		{
			name: "memory used normalized",
			args: args{
				res:                  v1.ResourceMemory,
				normalizedAllocation: true,
			},
			want: 0.50,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := testNode("mynode")
			n.Status.Allocatable = v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("8"),
				v1.ResourceMemory: resource.MustParse("4Gi"),
			}
			node := model.NewNode(n)

			p := testPod("default", "mypod")
			p.Spec.NodeName = n.Name
			pod := model.NewPod(p)
			node.BindPod(pod)

			if got := node.UsedPct(tt.args.res, tt.args.normalizedAllocation); got != tt.want {
				t.Errorf("UsedPct() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNode_UsedNormalized(t *testing.T) {
	type args struct {
		normalizedAllocation bool
	}
	tests := []struct {
		name string
		args args
		want v1.ResourceList
	}{
		{
			name: "not normalized",
			args: args{},
			want: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("2"),
				v1.ResourceMemory: resource.MustParse("2Gi"),
				v1.ResourcePods:   resource.MustParse("1"),
			},
		},
		{
			name: "normalized",
			args: args{
				normalizedAllocation: true,
			},
			want: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("4000m"),
				v1.ResourceMemory: resource.MustParse("2Gi"),
				v1.ResourcePods:   resource.MustParse("1"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := testNode("mynode")
			n.Status.Allocatable = v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("8"),
				v1.ResourceMemory: resource.MustParse("4Gi"),
			}
			node := model.NewNode(n)

			p := testPod("default", "mypod")
			p.Spec.NodeName = n.Name
			pod := model.NewPod(p)
			node.BindPod(pod)

			// remove the string notation from the resource so
			// reflect.DeepEqual can work
			want := v1.ResourceList{}
			for k, v := range tt.want {
				v.Add(resource.MustParse("0"))
				want[k] = v
			}

			if got := node.UsedNormalized(tt.args.normalizedAllocation); !reflect.DeepEqual(got, want) {
				t.Errorf("UsedNormalized() = %v, want %v", got, tt.want)
			}
		})
	}
}
