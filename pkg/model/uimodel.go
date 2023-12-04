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
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/facette/natsort"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/duration"

	"github.com/awslabs/eks-node-viewer/pkg/text"

	clipboard "golang.design/x/clipboard"
)

var (
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render
	// white / black
	activeDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	// black / white
	inactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")

	selectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#FFFFFF")).Bold(true).Render
	deselectedStyle = lipgloss.NewStyle().Render
)

type editorFinishedMsg struct{ err error }

type UIModel struct {
	progress       progress.Model
	cluster        *Cluster
	extraLabels    []string
	paginator      paginator.Model
	height         int
	nodeSorter     func(lhs, rhs *Node) bool
	style          *Style
	cursor         int
	selected       string
	start          int
	end            int
	err            error
	copyInstanceID bool
}

func NewUIModel(extraLabels []string, nodeSort string, style *Style, copyInstanceID bool) *UIModel {
	pager := paginator.New()
	pager.Type = paginator.Dots
	pager.ActiveDot = activeDot
	pager.InactiveDot = inactiveDot
	return &UIModel{
		// red to green
		progress:       progress.New(style.gradient),
		cluster:        NewCluster(),
		extraLabels:    extraLabels,
		paginator:      pager,
		nodeSorter:     makeNodeSorter(nodeSort),
		style:          style,
		cursor:         0,
		selected:       "",
		start:          0,
		end:            0,
		copyInstanceID: copyInstanceID,
	}
}

func (u *UIModel) Cluster() *Cluster {
	return u.cluster
}

func (u *UIModel) Init() tea.Cmd {
	return nil
}

func (u *UIModel) View() string {
	b := strings.Builder{}

	stats := u.cluster.Stats()

	sort.Slice(stats.Nodes, func(a, b int) bool {
		return u.nodeSorter(stats.Nodes[a], stats.Nodes[b])
	})

	ctw := text.NewColorTabWriter(&b, 0, 8, 1)
	u.writeClusterSummary(u.cluster.resources, stats, ctw)
	ctw.Flush()
	u.progress.ShowPercentage = true
	// message printer formats numbers nicely with commas
	enPrinter := message.NewPrinter(language.English)
	enPrinter.Fprintf(&b, "%d pods (%d pending %d running %d bound)\n", stats.TotalPods,
		stats.PodsByPhase[v1.PodPending], stats.PodsByPhase[v1.PodRunning], stats.BoundPodCount)

	if stats.NumNodes == 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "Waiting for update or no nodes found...")
		fmt.Fprintln(&b, u.paginator.View())
		fmt.Fprintln(&b, helpStyle("←/→ page • q: quit"))
		return b.String()
	}

	fmt.Fprintln(&b)
	u.paginator.PerPage = u.computeItemsPerPage(stats.Nodes, &b)
	u.paginator.SetTotalPages(stats.NumNodes)
	// check if we're on a page that is outside of the NumNode upper bound
	if u.paginator.Page*u.paginator.PerPage > stats.NumNodes {
		// set the page to the last page
		u.paginator.Page = u.paginator.TotalPages - 1
	}
	u.start, u.end = u.paginator.GetSliceBounds(stats.NumNodes)

	if u.cursor > u.end-u.start {
		u.cursor = (u.end - u.start) - 1
	}

	if u.start >= 0 && u.end >= u.start {
		for i, n := range stats.Nodes[u.start:u.end] {
			u.writeNodeInfo(n, ctw, u.cluster.resources, i)
		}
	}
	ctw.Flush()

	fmt.Fprintln(&b, u.paginator.View())
	fmt.Fprintln(&b, helpStyle("←/→ page • q: quit"))

	return b.String()
}

func (u *UIModel) writeNodeInfo(n *Node, w io.Writer, resources []v1.ResourceName, nodeIndex int) {
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
			priceLabel := fmt.Sprintf("/$%0.4f", n.Price)
			if !n.HasPrice() {
				priceLabel = ""
			}

			style := deselectedStyle(n.Name())
			if nodeIndex == u.cursor {
				style = selectedStyle(n.Name())
			}

			fmt.Fprintf(w, style)
			fmt.Fprintf(w, "\t%s\t%s\t(%d pods)\t%s%s", res, u.progress.ViewAs(pct), n.NumPods(), n.InstanceType(), priceLabel)

			// node compute type
			if n.IsOnDemand() {
				fmt.Fprintf(w, "\tOn-Demand")
			} else if n.IsSpot() {
				fmt.Fprintf(w, "\tSpot")
			} else if n.IsFargate() {
				fmt.Fprintf(w, "\tFargate")
			} else {
				fmt.Fprintf(w, "\t-")
			}

			// node status
			if n.Cordoned() && n.Deleting() {
				fmt.Fprintf(w, "\tCordoned/Deleting")
			} else if n.Deleting() {
				fmt.Fprintf(w, "\tDeleting")
			} else if n.Cordoned() {
				fmt.Fprintf(w, "\tCordoned")
			} else {
				fmt.Fprintf(w, "\t-")
			}

			// node readiness or time we've been waiting for it to be ready
			if n.Ready() {
				fmt.Fprintf(w, "\tReady")
			} else {
				fmt.Fprintf(w, "\tNotReady/%s", duration.HumanDuration(time.Since(n.NotReadyTime())))
			}

			for _, label := range u.extraLabels {
				labelValue, ok := n.node.Labels[label]
				if !ok {
					// support computed label values
					labelValue = n.ComputeLabel(label)
				}
				fmt.Fprintf(w, "\t%s", labelValue)
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
			pctUsedStr = u.style.green(pctUsedStr)
		} else if pctUsed > 60 {
			pctUsedStr = u.style.yellow(pctUsedStr)
		} else {
			pctUsedStr = u.style.red(pctUsedStr)
		}

		u.progress.ShowPercentage = false
		monthlyPrice := stats.TotalPrice * (365 * 24) / 12 // average hours per month
		// message printer formats numbers nicely with commas
		enPrinter := message.NewPrinter(language.English)
		clusterPrice := enPrinter.Sprintf("$%0.3f/hour | $%0.3f/month", stats.TotalPrice, monthlyPrice)
		if firstLine {
			enPrinter.Fprintf(w, "%d nodes\t(%s/%s)\t%s\t%s\t%s\t%s\n",
				stats.NumNodes, used.String(), allocatable.String(), pctUsedStr, res, u.progress.ViewAs(pctUsed/100.0), clusterPrice)
		} else {
			enPrinter.Fprintf(w, " \t%s/%s\t%s\t%s\t%s\t\n",
				used.String(), allocatable.String(), pctUsedStr, res, u.progress.ViewAs(pctUsed/100.0))
		}
		firstLine = false
	}
}

// computeItemsPerPage dynamically calculates the number of lines we can fit per page
// taking into account header and footer text
func (u *UIModel) computeItemsPerPage(nodes []*Node, b *strings.Builder) int {
	var buf bytes.Buffer
	u.writeNodeInfo(nodes[0], &buf, u.cluster.resources, 0)
	headerLines := strings.Count(b.String(), "\n") + 2
	nodeLines := strings.Count(buf.String(), "\n")
	if nodeLines == 0 {
		nodeLines = 1
	}
	return ((u.height - headerLines) / nodeLines) - 1
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (u *UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		u.height = msg.Height
		return u, tickCmd()
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if u.cursor > 0 {
				u.cursor--
			}
		case "down":
			if u.cursor < (u.end-u.start)-1 {
				u.cursor++
			}
		case "q", "esc", "ctrl+c":
			return u, tea.Quit
		case "enter":
			stats := u.cluster.Stats()
			sort.Slice(stats.Nodes, func(a, b int) bool {
				return u.nodeSorter(stats.Nodes[a], stats.Nodes[b])
			})

			if u.copyInstanceID {
				u.selected = stats.Nodes[u.start:u.end][u.cursor].InstanceID()
			} else {
				u.selected = stats.Nodes[u.start:u.end][u.cursor].Name()
			}

			return u, openNode(u, msg)
		}
	case editorFinishedMsg:
		if msg.err != nil {
			u.err = msg.err
			return u, tea.Quit
		}
	case tickMsg:
		return u, tickCmd()
	}
	var cmd tea.Cmd
	u.paginator, cmd = u.paginator.Update(msg)
	return u, cmd
}

func openNode(u *UIModel, msg tea.Msg) tea.Cmd {
	nodeName := u.selected
	nodeExec := os.Getenv("NODE_EXEC")
	if nodeExec == "" || nodeName == "" || strings.HasPrefix(nodeName, "fargate") {
		// copy only actions
		err := clipboard.Init()
		if err != nil {
			panic(err)
		}

		clipboard.Write(clipboard.FmtText, []byte(nodeName))

		var cmd tea.Cmd
		_, cmd = u.paginator.Update(msg)
		return cmd
	}

	nodeExec = fmt.Sprintf(nodeExec, nodeName)
	c := exec.Command("/bin/sh", "-c", nodeExec)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err}
	})
}

func (u *UIModel) SetResources(resources []string) {
	u.cluster.resources = nil
	for _, r := range resources {
		u.cluster.resources = append(u.cluster.resources, v1.ResourceName(r))
	}
}

func makeNodeSorter(nodeSort string) func(lhs *Node, rhs *Node) bool {
	sortOrder := func(b bool) bool { return b }
	if strings.HasSuffix(nodeSort, "=asc") {
		nodeSort = nodeSort[:len(nodeSort)-4]
	}
	if strings.HasSuffix(nodeSort, "=dsc") {
		sortOrder = func(b bool) bool { return !b }
		nodeSort = nodeSort[:len(nodeSort)-4]
	}

	if nodeSort == "creation" {
		return func(lhs *Node, rhs *Node) bool {
			if lhs.Created() == rhs.Created() {
				return sortOrder(natsort.Compare(lhs.Name(), rhs.Name()))
			}
			return sortOrder(rhs.Created().Before(lhs.Created()))
		}
	}

	return func(lhs *Node, rhs *Node) bool {
		lhsLabel, ok := lhs.node.Labels[nodeSort]
		if !ok {
			lhsLabel = lhs.ComputeLabel(nodeSort)
		}
		rhsLabel, ok := rhs.node.Labels[nodeSort]
		if !ok {
			rhsLabel = rhs.ComputeLabel(nodeSort)
		}
		if lhsLabel == rhsLabel {
			return sortOrder(natsort.Compare(lhs.InstanceID(), rhs.InstanceID()))
		}
		return sortOrder(natsort.Compare(lhsLabel, rhsLabel))
	}
}
