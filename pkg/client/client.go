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
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // pull auth
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	karpv1apis "sigs.k8s.io/karpenter/pkg/apis"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
)

// NewKubernetes returns the name of the kubeconfig context in use along with a
// clientset for it.
func NewKubernetes(kubeconfig, context string) (string, *kubernetes.Clientset, error) {
	contextName, config, err := getConfig(kubeconfig, context)
	if err != nil {
		return "", nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", nil, err
	}
	return contextName, clientset, nil
}

func NewNodeClaims(kubeconfig, context string) (*rest.RESTClient, error) {
	_, c, err := getConfig(kubeconfig, context)
	if err != nil {
		return nil, err
	}

	gv := schema.GroupVersion{Group: karpv1apis.Group, Version: "v1"}
	scheme.Scheme.AddKnownTypes(gv,
		&karpv1.NodeClaim{},
		&karpv1.NodeClaimList{})

	config := *c
	config.ContentConfig.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	return rest.RESTClientFor(&config)
}

// getConfig returns the name of the kubeconfig context in use along with its
// rest config. If context is empty, the kubeconfig's current context is used.
func getConfig(kubeconfig, context string) (string, *rest.Config, error) {
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{Precedence: strings.Split(kubeconfig, ":")},
		&clientcmd.ConfigOverrides{CurrentContext: context})

	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return "", nil, err
	}
	contextName := rawConfig.CurrentContext
	if context != "" {
		contextName = context
	}

	config, err := clientConfig.ClientConfig()
	if err != nil {
		return "", nil, err
	}
	return contextName, config, nil
}
