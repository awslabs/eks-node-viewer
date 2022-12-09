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
package main

import (
	"context"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	tea "github.com/charmbracelet/bubbletea"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/awslabs/eks-node-viewer/pkg/client"
	"github.com/awslabs/eks-node-viewer/pkg/model"
	"github.com/awslabs/eks-node-viewer/pkg/pricing"
)

func main() {
	nodeSelectorFlag := flag.String("nodeSelector", "", "Node label selector used to filter nodes, if empty all nodes are selected ")
	resources := flag.String("resources", "cpu", "List of comma separated resources to monitor")
	disablePricing := flag.Bool("disable-pricing", false, "Disable pricing lookups")

	flag.Parse()
	cs, err := client.Create()
	if err != nil {
		log.Fatalf("creating client, %s", err)
	}
	ctx, cancel := context.WithCancel(context.Background())

	defaults.SharedCredentialsFilename()
	pprov := pricing.NewStaticProvider()
	if !*disablePricing {
		sess := session.Must(session.NewSession(nil))
		pprov = pricing.NewProvider(ctx, sess)
	}
	m := model.NewUIModel()

	m.SetResources(strings.FieldsFunc(*resources, func(r rune) bool { return r == ',' }))

	var nodeSelector labels.Selector
	if ns, err := labels.Parse(*nodeSelectorFlag); err != nil {
		log.Fatalf("parsing node selector: %s", err)
	} else {
		nodeSelector = ns
	}

	startMonitor(ctx, cs, m, nodeSelector, pprov)
	if err := tea.NewProgram(m, tea.WithAltScreen()).Start(); err != nil {
		log.Fatalf("error running tea: %s", err)
	}
	cancel()
}

func startMonitor(ctx context.Context, clientset *kubernetes.Clientset, m *model.UIModel, nodeSelector labels.Selector, pprov *pricing.Provider) {
	podWatchList := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods",
		v1.NamespaceAll, fields.Everything())

	cluster := m.Cluster()
	_, podController := cache.NewInformer(
		podWatchList,
		&v1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				cluster.AddPod(model.NewPod(obj.(*v1.Pod)), pprov)
			},
			DeleteFunc: func(obj interface{}) {
				p := obj.(*v1.Pod)
				cluster.DeletePod(p.Namespace, p.Name)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				p := newObj.(*v1.Pod)
				if !p.DeletionTimestamp.IsZero() {
					cluster.DeletePod(p.Namespace, p.Name)
				} else {
					pod, ok := cluster.GetPod(p.Namespace, p.Name)
					if !ok {
						cluster.AddPod(model.NewPod(p), pprov)
					} else {
						pod.Update(p)
						cluster.AddPod(pod, pprov)
					}
				}
			},
		},
	)
	go podController.Run(ctx.Done())

	nodeWatchList := cache.NewFilteredListWatchFromClient(clientset.CoreV1().RESTClient(), "nodes",
		v1.NamespaceAll, func(options *metav1.ListOptions) {
			options.LabelSelector = nodeSelector.String()
		})
	_, controller := cache.NewInformer(
		nodeWatchList,
		&v1.Node{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				n := cluster.AddNode(model.NewNode(obj.(*v1.Node), pprov))
				n.Show()
			},
			DeleteFunc: func(obj interface{}) {
				cluster.DeleteNode(obj.(*v1.Node).Name)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				n := newObj.(*v1.Node)
				if !n.DeletionTimestamp.IsZero() {
					cluster.DeleteNode(n.Name)
				} else {
					node, ok := cluster.GetNode(n.Name)
					if !ok {
						log.Println("unable to find node", n.Name)
					} else {
						node.Update(n)
					}
					node.Show()
				}
			},
		},
	)
	go controller.Run(ctx.Done())

}
