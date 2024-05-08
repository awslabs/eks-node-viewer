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
package client

import (
	"context"
	"log"
	"math"
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/karpenter/pkg/apis/v1beta1"

	"github.com/awslabs/eks-node-viewer/pkg/model"
	"github.com/awslabs/eks-node-viewer/pkg/pricing"
)

type Controller struct {
	kubeClient      *kubernetes.Clientset
	uiModel         *model.UIModel
	pricing         pricing.Provider
	nodeSelector    labels.Selector
	nodeClaimClient *rest.RESTClient
}

func NewController(kubeClient *kubernetes.Clientset, nodeClaimClient *rest.RESTClient, uiModel *model.UIModel, nodeSelector labels.Selector, pricing pricing.Provider) *Controller {
	c := &Controller{
		kubeClient:      kubeClient,
		uiModel:         uiModel,
		pricing:         pricing,
		nodeSelector:    nodeSelector,
		nodeClaimClient: nodeClaimClient,
	}
	pricing.OnUpdate(c.RefreshNodePrices)
	return c
}

func (m Controller) Start(ctx context.Context) {
	cluster := m.uiModel.Cluster()

	m.startPodWatch(ctx, cluster)
	m.startNodeWatch(ctx, cluster)

	// If a NodeClaims Get returns an error, then don't startup the nodeclaims controller since the CRD is not registered
	if err := m.nodeClaimClient.Get().Do(ctx).Error(); err == nil {
		m.startNodeClaimWatch(ctx, cluster)
	}
}

func (m Controller) startNodeClaimWatch(ctx context.Context, cluster *model.Cluster) {
	nodeClaimWatchList := cache.NewFilteredListWatchFromClient(m.nodeClaimClient, "nodeclaims",
		v1.NamespaceAll, func(options *metav1.ListOptions) {
			options.LabelSelector = m.nodeSelector.String()
		})
	_, nodeClaimController := cache.NewInformer(
		nodeClaimWatchList,
		&v1beta1.NodeClaim{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				nc := obj.(*v1beta1.NodeClaim)
				if nc.Status.ProviderID == "" {
					return
				}
				if _, ok := cluster.GetNode(nc.Status.ProviderID); ok {
					return
				}
				node := model.NewNodeFromNodeClaim(nc)
				m.updatePrice(node)
				n := cluster.AddNode(node)
				n.Show()
			},
			DeleteFunc: func(obj interface{}) {
				cluster.DeleteNode(ignoreDeletedFinalStateUnknown(obj).(*v1beta1.NodeClaim).Status.ProviderID)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				nc := newObj.(*v1beta1.NodeClaim)
				if nc.Status.ProviderID == "" {
					return
				}
				if _, ok := cluster.GetNode(nc.Status.ProviderID); ok {
					return
				}
				node := model.NewNodeFromNodeClaim(nc)
				m.updatePrice(node)
				n := cluster.AddNode(node)
				n.Show()
			},
		},
	)
	go nodeClaimController.Run(ctx.Done())
}

func (m Controller) startNodeWatch(ctx context.Context, cluster *model.Cluster) {
	nodeWatchList := cache.NewFilteredListWatchFromClient(m.kubeClient.CoreV1().RESTClient(), "nodes",
		v1.NamespaceAll, func(options *metav1.ListOptions) {
			options.LabelSelector = m.nodeSelector.String()
		})
	_, nodeController := cache.NewInformer(
		nodeWatchList,
		&v1.Node{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				node := model.NewNode(obj.(*v1.Node))
				m.updatePrice(node)
				n := cluster.AddNode(node)
				n.Show()
			},
			DeleteFunc: func(obj interface{}) {
				cluster.DeleteNode(ignoreDeletedFinalStateUnknown(obj).(*v1.Node).Spec.ProviderID)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				n := newObj.(*v1.Node)
				if !n.DeletionTimestamp.IsZero() && len(n.Finalizers) == 0 {
					cluster.DeleteNode(n.Spec.ProviderID)
				} else {
					node, ok := cluster.GetNode(n.Spec.ProviderID)
					if !ok {
						log.Println("unable to find node", n.Name)
					} else {
						node.Update(n)
						m.updatePrice(node)
					}
					node.Show()
				}
			},
		},
	)
	go nodeController.Run(ctx.Done())
}

func (m Controller) startPodWatch(ctx context.Context, cluster *model.Cluster) {
	podWatchList := cache.NewListWatchFromClient(m.kubeClient.CoreV1().RESTClient(), "pods",
		v1.NamespaceAll, fields.Everything())

	_, podController := cache.NewInformer(
		podWatchList,
		&v1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				p := obj.(*v1.Pod)
				if !isTerminalPod(p) {
					cluster.AddPod(model.NewPod(p))
					node, ok := cluster.GetNodeByName(p.Spec.NodeName)
					// need to potentially update node price as we need the fargate pod in order to figure out the cost
					if ok && node.IsFargate() && !node.HasPrice() {
						m.updatePrice(node)
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				p := ignoreDeletedFinalStateUnknown(obj).(*v1.Pod)
				cluster.DeletePod(p.Namespace, p.Name)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				p := newObj.(*v1.Pod)
				if isTerminalPod(p) {
					cluster.DeletePod(p.Namespace, p.Name)
				} else {
					pod, ok := cluster.GetPod(p.Namespace, p.Name)
					if !ok {
						cluster.AddPod(model.NewPod(p))
					} else {
						pod.Update(p)
						cluster.AddPod(pod)
					}
				}
			},
		},
	)
	go podController.Run(ctx.Done())
}

func (m Controller) updatePrice(node *model.Node) {
	// If the node has the instance-price override label, don't look up pricing
	// and use the value here.
	if val, ok := node.Labels()["eks-node-viewer/instance-price"]; ok {
		if price, err := strconv.ParseFloat(val, 64); err == nil {
			node.SetPrice(price)
			return
		}
	}
	// lookup our n price
	node.Price = math.NaN()
	if price, ok := m.pricing.NodePrice(node); ok {
		node.SetPrice(price)
	}

}

func (m Controller) RefreshNodePrices() {
	m.uiModel.Cluster().ForEachNode(func(n *model.Node) {
		m.updatePrice(n)
	})
}

// isTerminalPod returns true if the pod is deleting or in a terminal state
func isTerminalPod(p *v1.Pod) bool {
	if !p.DeletionTimestamp.IsZero() {
		return true
	}
	switch p.Status.Phase {
	case v1.PodSucceeded, v1.PodFailed:
		return true
	}
	return false
}

// ignoreDeletedFinalStateUnknown returns the object wrapped in
// DeletedFinalStateUnknown. Useful in OnDelete resource event handlers that do
// not need the additional context.
func ignoreDeletedFinalStateUnknown(obj interface{}) interface{} {
	if obj, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		return obj.Obj
	}
	return obj
}
