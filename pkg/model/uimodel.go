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
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	v1 "k8s.io/api/core/v1"
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

type UIModel struct {
	progress progress.Model
	cluster  *Cluster
}

func NewUIModel() *UIModel {
	return &UIModel{
		// red to green
		progress: progress.New(progress.WithGradient("#ff0000", "#04B575")),
		cluster:  NewCluster(),
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

	u.writeClusterSummary(u.cluster.resources, stats, &b)

	u.progress.ShowPercentage = true
	fmt.Fprintf(&b, "%d pods (%d pending %d running %d bound)\n", stats.TotalPods,
		stats.PodsByPhase[v1.PodPending], stats.PodsByPhase[v1.PodRunning], stats.BoundPodCount)

	fmt.Fprintln(&b)
	nodeNameLen := 0
	for _, n := range stats.Nodes {
		if len(n.Name()) > nodeNameLen {
			nodeNameLen = len(n.Name())
		}
	}

	for _, n := range stats.Nodes {
		u.writeNodeInfo(n, &b, u.cluster.resources, nodeNameLen)
	}

	fmt.Fprintln(&b)

	fmt.Fprintln(&b, helpStyle("Press any key to quit"))
	return b.String()
}

func (u *UIModel) writeNodeInfo(n *Node, b *strings.Builder, resources []v1.ResourceName, nodeNameLen int) {
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
		extra := ""
		if n.IsOnDemand() {
			extra = "On-Demand "
		} else {
			extra = "Spot      "
		}

		if n.Cordoned() {
			extra += " cordoned"
		}
		if n.Ready() {
			extra += " ready"
		} else {
			extra += time.Since(n.Created()).String()
		}

		price := ""
		if n.Price != 0 {
			price = fmt.Sprintf("$%0.3f", n.Price)
		}
		if firstLine {
			fmt.Fprintf(b, "%s %s %s (%3d pods) %s/%s %s\n", pad(n.Name(), nodeNameLen), pad(string(res), resNameLen), u.progress.ViewAs(pct), n.NumPods(),
				n.InstanceType(), price, extra)
		} else {
			fmt.Fprintf(b, "%s %s %s\n", pad("", nodeNameLen), pad(string(res), resNameLen), u.progress.ViewAs(pct))
		}
		firstLine = false
	}
}

func (u *UIModel) writeClusterSummary(resources []v1.ResourceName, stats Stats, b *strings.Builder) {
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
			pctUsedStr = green(pctUsedStr)
		} else if pctUsed > 60 {
			pctUsedStr = yellow(pctUsedStr)
		} else {
			pctUsedStr = red(pctUsedStr)
		}

		u.progress.ShowPercentage = false
		monthlyPrice := stats.TotalPrice * (365 * 24) / 12 // average hours per month
		descr := pad(fmt.Sprintf("%s/%s %s %s $%0.3f/hour $%0.3f/month", used.String(), allocatable.String(), pctUsedStr, res, stats.TotalPrice, monthlyPrice), 60)
		if firstLine {
			fmt.Fprintf(b, "%d nodes %s %s\n", stats.NumNodes, descr, u.progress.ViewAs(pctUsed/100.0))
		} else {
			fmt.Fprintf(b, "%s%s %s\n", pad("", len(fmt.Sprintf("%d nodes ", stats.NumNodes))),
				descr, u.progress.ViewAs(pctUsed/100.0))
		}
		firstLine = false
	}
}

func pad(s string, minLen int) any {
	if len(s) >= minLen {
		return s
	}
	var sb strings.Builder
	sb.WriteString(s)
	for sb.Len() < minLen {
		sb.WriteByte(' ')
	}
	return sb.String()
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
