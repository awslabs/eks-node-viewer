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

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // pull auth
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	karpv1apis "sigs.k8s.io/karpenter/pkg/apis"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
)

func NewKubernetes(kubeconfig, context string) (*kubernetes.Clientset, error) {
	config, err := getConfig(kubeconfig, context)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, err
}

func NewNodeClaims(kubeconfig, context string) (*rest.RESTClient, error) {
	c, err := getConfig(kubeconfig, context)
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

func GetAWSRegionAndProfile(kubeconfig, context string) (region, profile string) {
	config := getClientConfig(kubeconfig, context)
	raw, err := config.RawConfig()
	if err != nil {
		return "", ""
	}

	if context == "" {
		context = raw.CurrentContext
	}
	kubeContext := raw.Contexts[context]
	if kubeContext == nil {
		return "", ""
	}
	auth := raw.AuthInfos[kubeContext.AuthInfo]
	if auth == nil || auth.Exec == nil {
		return "", ""
	}

	// use a flagset to parse the args from the exec config
	//
	flagSet := pflag.NewFlagSet("aws", pflag.ContinueOnError)
	flagSet.ParseErrorsWhitelist.UnknownFlags = true
	regionPtr := flagSet.String("region", "", "")
	_ = flagSet.Parse(auth.Exec.Args)

	for _, env := range auth.Exec.Env {
		if env.Name == "AWS_PROFILE" {
			profile = env.Value
		}
	}

	return *regionPtr, profile
}

func getClientConfig(kubeconfig, context string) clientcmd.ClientConfig {
	// use the current context in kubeconfig
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{Precedence: strings.Split(kubeconfig, ":")},
		&clientcmd.ConfigOverrides{CurrentContext: context})
}

func getConfig(kubeconfig, context string) (*rest.Config, error) {
	return getClientConfig(kubeconfig, context).ClientConfig()
}
