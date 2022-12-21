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
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	v1 "k8s.io/api/core/v1"

	"github.com/awslabs/eks-node-viewer/pkg/text"
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

type UIModel struct {
	progress    progress.Model
	cluster     *Cluster
	extraLabels []string
}

func NewUIModel(extraLabels []string) *UIModel {
	return &UIModel{
		// red to green
		progress:    progress.New(progress.WithGradient("#04B575", "#ff0000")),
		cluster:     NewCluster(),
		extraLabels: extraLabels,
	}
}

func (u *UIModel) Cluster() *Cluster {
	return u.cluster
}

func (u *UIModel) Init() tea.Cmd {
	return tickCmd()
}

var green = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Render
var yellow = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Render
var red = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render

func (u *UIModel) View() string {
	b := strings.Builder{}

	stats := u.cluster.Stats()
	if stats.NumNodes == 0 {
		return "Waiting for update or no nodes found..."
	}

	ctw := text.NewColorTabWriter(&b, 0, 8, 1)
	u.writeClusterSummary(u.cluster.resources, stats, ctw)
	ctw.Flush()
	u.progress.ShowPercentage = true

	fmt.Fprintf(&b, "%d pods (%d pending %d running %d bound)\n", stats.TotalPods,
		stats.PodsByPhase[v1.PodPending], stats.PodsByPhase[v1.PodRunning], stats.BoundPodCount)

	fmt.Fprintln(&b)
	for _, n := range stats.Nodes {
		u.writeNodeInfo(n, ctw, u.cluster.resources)
	}
	ctw.Flush()

	fmt.Fprintln(&b, helpStyle("Press any key to quit"))
	return b.String()
}

func (u *UIModel) writeNodeInfo(n *Node, w io.Writer, resources []v1.ResourceName) {
	allocatable := n.Allocatable()
	used := n.Used()
	firstLine := true
	resNameLen := 0
	for _, res := range resources {
		if len(res) > resNameLen {
			resNameLen = len(res)
		}
	}
	for _, res := range resources {
		usedRes := used[res]
		allocatableRes := allocatable[res]
		pct := usedRes.AsApproximateFloat64() / allocatableRes.AsApproximateFloat64()
		if allocatableRes.AsApproximateFloat64() == 0 {
			pct = 0
		}

		if firstLine {
			fmt.Fprintf(w, "%s\t%s\t%s\t(%d pods)\t%s/$%0.3f", n.Name(), res, u.progress.ViewAs(pct), n.NumPods(), n.InstanceType(), n.Price)
			if n.IsOnDemand() {
				fmt.Fprintf(w, "\tOn-Demand")
			} else {
				fmt.Fprintf(w, "\tSpot")
			}

			if n.Cordoned() && n.Deleting() {
				fmt.Fprintf(w, "\tCordoned/Deleting")
			} else if n.Deleting() {
				fmt.Fprintf(w, "\tDeleting")
			} else if n.Cordoned() {
				fmt.Fprintf(w, "\tCordoned")
			} else {
				fmt.Fprintf(w, "\t")
			}

			if n.Ready() {
				fmt.Fprintf(w, "\tReady")
			} else {
				fmt.Fprintf(w, "\t%s", time.Since(n.Created()).String())
			}

			for _, label := range u.extraLabels {
				fmt.Fprintf(w, "\t%s", n.node.Labels[label])
			}

		} else {
			fmt.Fprintf(w, " \t%s\t%s\t\t\t\t\t", res, u.progress.ViewAs(pct))
			for range u.extraLabels {
				fmt.Fprintf(w, "\t")
			}
		}
		fmt.Fprintln(w)
		firstLine = false
	}
}

func (u *UIModel) writeClusterSummary(resources []v1.ResourceName, stats Stats, w io.Writer) {
	firstLine := true

	for _, res := range resources {
		allocatable := stats.AllocatableResources[res]
		used := stats.UsedResources[res]
		pctUsed := 0.0
		if allocatable.AsApproximateFloat64() != 0 {
			pctUsed = 100 * (used.AsApproximateFloat64() / allocatable.AsApproximateFloat64())
		}
		pctUsedStr := fmt.Sprintf("%0.1f%%", pctUsed)
		if pctUsed > 90 {
			pctUsedStr = red(pctUsedStr)
		} else if pctUsed > 60 {
			pctUsedStr = yellow(pctUsedStr)
		} else {
			pctUsedStr = green(pctUsedStr)
		}

		u.progress.ShowPercentage = false
		monthlyPrice := stats.TotalPrice * (365 * 24) / 12 // average hours per month
		clusterPrice := fmt.Sprintf("%0.3f/hour $%0.3f/month", stats.TotalPrice, monthlyPrice)
		if firstLine {
			fmt.Fprintf(w, "%d nodes\t%s/%s\t%s\t%s\t%s\t%s\n",
				stats.NumNodes, used.String(), allocatable.String(), pctUsedStr, res, u.progress.ViewAs(pctUsed/100.0), clusterPrice)
		} else {
			fmt.Fprintf(w, " \t%s/%s\t%s\t%s\t%s\t\n",
				used.String(), allocatable.String(), pctUsedStr, res, u.progress.ViewAs(pctUsed/100.0))
		}
		firstLine = false
	}
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (u *UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		return u, tea.Quit
	case tickMsg:
		return u, tickCmd()
	default:
		return u, nil
	}
}

func (u *UIModel) SetResources(resources []string) {
	u.cluster.resources = nil
	for _, r := range resources {
		u.cluster.resources = append(u.cluster.resources, v1.ResourceName(r))
	}
}
