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
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	awsSdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/awslabs/eks-node-viewer/pkg/aws"
	"github.com/awslabs/eks-node-viewer/pkg/client"
	"github.com/awslabs/eks-node-viewer/pkg/model"
)

//go:generate cp -r ../../ATTRIBUTION.md ./
//go:embed ATTRIBUTION.md
var attribution string

func main() {
	flags, err := ParseFlags()
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		log.Fatalf("cannot parse flags: %v", err)
	}

	if flags.ShowAttribution {
		fmt.Println(attribution)
		os.Exit(0)
	}

	if flags.Version {
		fmt.Printf("eks-node-viewer version %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("built at: %s\n", date)
		fmt.Printf("built by: %s\n", builtBy)
		os.Exit(0)
	}

	cs, err := client.NewKubernetes(flags.Kubeconfig, flags.Context)
	if err != nil {
		log.Fatalf("creating client, %s", err)
	}
	nodeClaimClient, err := client.NewNodeClaims(flags.Kubeconfig, flags.Context)
	if err != nil {
		log.Fatalf("creating node claim client, %s", err)
	}
	ctx, cancel := context.WithCancel(context.Background())

	region, profile := client.GetAWSRegionAndProfile(flags.Kubeconfig, flags.Context)

	pprov := aws.NewStaticPricingProvider(region)
	style, err := model.ParseStyle(flags.Style)
	if err != nil {
		log.Fatalf("creating style, %s", err)
	}
	m := model.NewUIModel(strings.Split(flags.ExtraLabels, ","), flags.NodeSort, style)
	m.DisablePricing = flags.DisablePricing
	m.SetResources(strings.FieldsFunc(flags.Resources, func(r rune) bool { return r == ',' }))

	var nodeSelector labels.Selector
	if ns, err := labels.Parse(flags.NodeSelector); err != nil {
		log.Fatalf("parsing node selector: %s", err)
	} else {
		nodeSelector = ns
	}

	if !flags.DisablePricing {
		sess := session.Must(session.NewSessionWithOptions(
			session.Options{
				Config:            awsSdk.Config{Region: &region},
				Profile:           profile,
				SharedConfigState: session.SharedConfigEnable,
			},
		))
		pprov = aws.NewPricingProvider(ctx, sess)
	}
	controller := client.NewController(cs, nodeClaimClient, m, nodeSelector, pprov)

	controller.Start(ctx)

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		log.Fatalf("error running tea: %s", err)
	}
	cancel()
}
