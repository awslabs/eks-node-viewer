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

	"github.com/aws/aws-sdk-go-v2/config"
	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

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

	pprov := aws.NewStaticPricingProvider()
	style, err := model.ParseStyle(flags.Style)
	if err != nil {
		log.Fatalf("creating style, %s", err)
	}
	m := model.NewUIModel(strings.Split(flags.ExtraLabels, ","), flags.NodeSort, style)
	m.DisablePricing = flags.DisablePricing
	m.SetResources(strings.FieldsFunc(flags.Resources, func(r rune) bool { return r == ',' }))
	if flags.GroupBy != "" {
		groupBy, err := parseLabelFilter(flags.GroupBy)
		if err != nil {
			log.Fatalf("parsing group-by: %s", err)
		}
		m.SetGrouping(groupBy.label, flags.GroupsOnly)
	}

	nodeSelector, err := buildNodeSelector(flags.NodeSelector, flags.GroupBy)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if !flags.DisablePricing {
		// Use AWS SDK Go v2 for configuration
		cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(""))
		if err != nil {
			log.Fatalf("unable to load AWS SDK config: %s", err)
		}
		pprov = aws.NewPricingProvider(ctx, cfg)
	}
	controller := client.NewController(cs, nodeClaimClient, m, nodeSelector, pprov)

	controller.Start(ctx)

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		log.Fatalf("error running tea: %s", err)
	}
	cancel()
}

// labelFilter is a --group-by value: 'label' or 'label=a,b,c'.
type labelFilter struct {
	label  string
	values []string
}

func parseLabelFilter(s string) (labelFilter, error) {
	label, rawValues, _ := strings.Cut(s, "=")
	filter := labelFilter{label: strings.TrimSpace(label)}
	if filter.label == "" {
		return labelFilter{}, fmt.Errorf("parsing %q: no label name given", s)
	}
	for _, v := range strings.Split(rawValues, ",") {
		if v = strings.TrimSpace(v); v != "" {
			filter.values = append(filter.values, v)
		}
	}
	return filter, nil
}

// buildNodeSelector combines --node-selector with the values listed in --group-by.
// A --group-by without values only says how to aggregate and leaves the selection
// alone, keeping nodes without the label visible in their own group.
func buildNodeSelector(nodeSelector string, groupBy string) (labels.Selector, error) {
	selector, err := labels.Parse(nodeSelector)
	if err != nil {
		return nil, fmt.Errorf("parsing node selector: %w", err)
	}
	if strings.TrimSpace(groupBy) == "" {
		return selector, nil
	}

	filter, err := parseLabelFilter(groupBy)
	if err != nil {
		return nil, err
	}
	if len(filter.values) == 0 {
		return selector, nil
	}

	req, err := labels.NewRequirement(filter.label, selection.In, filter.values)
	if err != nil {
		return nil, fmt.Errorf("building selector for label %q: %w", filter.label, err)
	}
	return selector.Add(*req), nil
}
