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
	"math"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/duration"

	"github.com/awslabs/eks-node-viewer/pkg/text"
)

var (
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render
	// white / black
	activeDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	// black / white
	inactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
)

// avgHoursPerMonth turns an hourly price into a monthly one
const avgHoursPerMonth = (365 * 24) / 12

type UIModel struct {
	progress    progress.Model
	cluster     *Cluster
	extraLabels []string
	paginator   paginator.Model
	height      int
	nodeSorter  *nodeSorter
	style       *Style
	groupBy     string
	showGroups  bool
	showNodes   bool
	// printer formats numbers with commas, built once as the view is redrawn often
	printer        *message.Printer
	bars           map[barKey]string
	DisablePricing bool
}

type barKey struct {
	// fill is the level in thousandths, finer than the bar's own cells
	fill           int
	width          int
	showPercentage bool
}

// every fill at or above 100% draws the same full bar
const maxCachedBarFill = 2000

// progressBar renders the bar for a 0..1 fill level, cached because every cell of
// the gradient is a colour interpolated in Luv space and a frame draws dozens of
// bars. Rounding the level keeps the cache bounded and is finer than what the bar
// and its percentage can show. The cache needs no lock: only the view renders.
func (u *UIModel) progressBar(fill float64) string {
	if fill < 0 || math.IsNaN(fill) {
		fill = 0
	}
	rounded := int(math.Round(fill * 1000))
	if rounded > maxCachedBarFill {
		return u.progress.ViewAs(fill)
	}
	key := barKey{fill: rounded, width: u.progress.Width, showPercentage: u.progress.ShowPercentage}
	if bar, ok := u.bars[key]; ok {
		return bar
	}
	bar := u.progress.ViewAs(float64(rounded) / 1000)
	u.bars[key] = bar
	return bar
}

func NewUIModel(extraLabels []string, nodeSort string, style *Style) *UIModel {
	pager := paginator.New()
	pager.Type = paginator.Dots
	pager.ActiveDot = activeDot
	pager.InactiveDot = inactiveDot
	return &UIModel{
		// red to green
		progress:    progress.New(style.gradient),
		cluster:     NewCluster(),
		extraLabels: extraLabels,
		paginator:   pager,
		nodeSorter:  newNodeSorter(nodeSort),
		style:       style,
		showNodes:   true,
		printer:     message.NewPrinter(language.English),
		bars:        map[barKey]string{},
	}
}

func (u *UIModel) Cluster() *Cluster {
	return u.cluster
}

// SetGrouping summarizes nodes by their value for the label, empty to disable.
// groupsOnly starts with the node list hidden. Both sections can be toggled at
// runtime, the summary with 'g' and the node list with 'n'.
func (u *UIModel) SetGrouping(label string, groupsOnly bool) {
	u.groupBy = label
	u.showGroups = label != ""
	u.showNodes = label == "" || !groupsOnly
}

// toggleGroups and toggleNodes never leave both sections hidden, there would be
// nothing left to look at.
func (u *UIModel) toggleGroups() {
	if u.groupBy == "" || (u.showGroups && !u.showNodes) {
		return
	}
	u.showGroups = !u.showGroups
	u.paginator.Page = 0
}

func (u *UIModel) toggleNodes() {
	if u.showNodes && !u.showGroups {
		return
	}
	u.showNodes = !u.showNodes
	u.paginator.Page = 0
}

func (u *UIModel) Init() tea.Cmd {
	return nil
}

func (u *UIModel) View() string {
	b := strings.Builder{}

	stats := u.cluster.Stats()

	u.nodeSorter.sort(stats.Nodes)

	ctw := text.NewColorTabWriter(&b, 0, 8, 1)
	u.writeClusterSummary(u.cluster.resources, stats, ctw)
	ctw.Flush()
	u.printer.Fprintf(&b, "%d pods (%d pending %d running %d bound)\n", stats.TotalPods,
		stats.PodsByPhase[v1.PodPending], stats.PodsByPhase[v1.PodRunning], stats.BoundPodCount)

	if stats.NumNodes == 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "Waiting for update or no nodes found...")
		fmt.Fprintln(&b, u.paginator.View())
		fmt.Fprintln(&b, helpStyle(u.helpText()))
		return b.String()
	}

	if u.showGroups && u.groupBy != "" {
		groups := GroupNodes(stats.Nodes, u.groupBy)
		fmt.Fprintln(&b)
		if u.showNodes {
			u.writeGroupSummary(groups, ctw)
		} else {
			// with no node list, the groups are the paginated content
			u.paginator.PerPage = u.itemsPerPage(&b, len(u.cluster.resources))
			u.paginator.SetTotalPages(len(groups))
			u.clampPage(len(groups))
			if start, end := u.paginator.GetSliceBounds(len(groups)); start >= 0 && end >= start {
				u.writeGroupSummary(groups[start:end], ctw)
			}
		}
		ctw.Flush()
	}

	if !u.showNodes {
		fmt.Fprintln(&b, u.paginator.View())
		fmt.Fprintln(&b, helpStyle(u.helpText()))
		return b.String()
	}

	fmt.Fprintln(&b)
	u.progress.ShowPercentage = true
	u.paginator.PerPage = u.computeItemsPerPage(stats.Nodes, &b)
	u.paginator.SetTotalPages(stats.NumNodes)
	u.clampPage(stats.NumNodes)
	start, end := u.paginator.GetSliceBounds(stats.NumNodes)
	if start >= 0 && end >= start {
		for _, n := range stats.Nodes[start:end] {
			u.writeNodeInfo(n, ctw, u.cluster.resources)
		}
	}
	ctw.Flush()

	fmt.Fprintln(&b, u.paginator.View())
	fmt.Fprintln(&b, helpStyle(u.helpText()))
	return b.String()
}

// clampPage keeps the page in bounds as the item count shrinks underneath us.
func (u *UIModel) clampPage(totalItems int) {
	if u.paginator.PerPage > 0 && u.paginator.Page*u.paginator.PerPage > totalItems {
		u.paginator.Page = u.paginator.TotalPages - 1
	}
	if u.paginator.Page < 0 {
		u.paginator.Page = 0
	}
}

func (u *UIModel) helpText() string {
	if u.groupBy == "" {
		return "←/→ page • q: quit"
	}
	groups, nodes := "hide", "hide"
	if !u.showGroups {
		groups = "show"
	}
	if !u.showNodes {
		nodes = "show"
	}
	return fmt.Sprintf("←/→ page • g: %s groups • n: %s nodes • q: quit", groups, nodes)
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
			priceLabel := fmt.Sprintf("/$%0.4f", n.Price)
			if !n.HasPrice() || u.DisablePricing {
				priceLabel = ""
			}
			maxPods, _ := allocatable.Pods().AsInt64()
			fmt.Fprintf(w, "%s\t%s\t%s\t(%d/%d pods)\t%s%s", n.Name(), res, u.progressBar(pct), n.NumPods(), maxPods, n.InstanceType(), priceLabel)

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

			if n.IsAuto() {
				fmt.Fprintf(w, "/Auto")
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
			fmt.Fprintf(w, " \t%s\t%s\t\t\t\t\t", res, u.progressBar(pct))
			for range u.extraLabels {
				fmt.Fprintf(w, "\t")
			}
		}
		fmt.Fprintln(w)
		firstLine = false
	}
}

// writeGroupSummary writes a block per group: nodes, pods, usage and cost.
func (u *UIModel) writeGroupSummary(groups []GroupStats, w io.Writer) {
	u.progress.ShowPercentage = false

	for _, g := range groups {
		firstLine := true
		for _, res := range u.cluster.resources {
			allocatable := g.AllocatableResources[res]
			used := g.UsedResources[res]
			pctUsed := g.PercentUsed(res)
			pctUsedStr := u.colorizeUsage(pctUsed, fmt.Sprintf("%0.1f%%", pctUsed))

			if firstLine {
				groupPrice := ""
				if !u.DisablePricing && g.PricedNodes > 0 {
					groupPrice = u.printer.Sprintf("$%0.3f/hour | $%0.3f/month", g.TotalPrice, g.TotalPrice*avgHoursPerMonth)
					if !g.FullyPriced() {
						// an unknown price somewhere, so this is a lower bound
						groupPrice = ">= " + groupPrice
					}
				}
				u.printer.Fprintf(w, "%s\t%d nodes\t%d pods\t(%s/%s)\t%s\t%s\t%s\t%s\n",
					g.Name, g.NumNodes, g.NumPods, formatQuantity(res, used), formatQuantity(res, allocatable),
					pctUsedStr, res, u.progressBar(pctUsed/100.0), groupPrice)
			} else {
				u.printer.Fprintf(w, " \t\t\t(%s/%s)\t%s\t%s\t%s\t\n",
					formatQuantity(res, used), formatQuantity(res, allocatable), pctUsedStr, res, u.progressBar(pctUsed/100.0))
			}
			firstLine = false
		}
	}
}

// colorizeUsage styles by how full a resource is, fuller being better.
func (u *UIModel) colorizeUsage(pctUsed float64, label string) string {
	if pctUsed > 90 {
		return u.style.green(label)
	} else if pctUsed > 60 {
		return u.style.yellow(label)
	}
	return u.style.red(label)
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
		pctUsedStr := u.colorizeUsage(pctUsed, fmt.Sprintf("%0.1f%%", pctUsed))

		u.progress.ShowPercentage = false
		monthlyPrice := stats.TotalPrice * avgHoursPerMonth
		clusterPrice := u.printer.Sprintf("$%0.3f/hour | $%0.3f/month", stats.TotalPrice, monthlyPrice)
		if u.DisablePricing {
			clusterPrice = ""
		}
		if firstLine {
			u.printer.Fprintf(w, "%d nodes\t(%10s/%s)\t%s\t%s\t%s\t%s\n",
				stats.NumNodes, formatQuantity(res, used), formatQuantity(res, allocatable), pctUsedStr, res, u.progressBar(pctUsed/100.0), clusterPrice)
		} else {
			u.printer.Fprintf(w, " \t%s/%s\t%s\t%s\t%s\t\n",
				formatQuantity(res, used), formatQuantity(res, allocatable), pctUsedStr, res, u.progressBar(pctUsed/100.0))
		}
		firstLine = false
	}
}

// computeItemsPerPage dynamically calculates the number of nodes we can fit per page
// taking into account header and footer text
func (u *UIModel) computeItemsPerPage(nodes []*Node, b *strings.Builder) int {
	var buf bytes.Buffer
	u.writeNodeInfo(nodes[0], &buf, u.cluster.resources)
	return u.itemsPerPage(b, strings.Count(buf.String(), "\n"))
}

// itemsPerPage fits items of linesPerItem lines below the header written so far.
// Never returns zero: the paginator divides by it.
func (u *UIModel) itemsPerPage(header *strings.Builder, linesPerItem int) int {
	if linesPerItem < 1 {
		linesPerItem = 1
	}
	headerLines := strings.Count(header.String(), "\n") + 2
	perPage := ((u.height - headerLines) / linesPerItem) - 1
	if perPage < 1 {
		return 1
	}
	return perPage
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
		case "q", "esc", "ctrl+c":
			return u, tea.Quit
		case "g":
			u.toggleGroups()
			return u, nil
		case "n":
			u.toggleNodes()
			return u, nil
		}
	case tickMsg:
		return u, tickCmd()
	}
	var cmd tea.Cmd
	u.paginator, cmd = u.paginator.Update(msg)
	return u, cmd
}

func (u *UIModel) SetResources(resources []string) {
	u.cluster.resources = nil
	for _, r := range resources {
		u.cluster.resources = append(u.cluster.resources, v1.ResourceName(r))
	}
}

// nodeSorter orders the node list for display, either by creation time or by the
// value of a node label.
type nodeSorter struct {
	// label is the node label to sort on, empty to sort by creation time
	label      string
	descending bool
}

func newNodeSorter(nodeSort string) *nodeSorter {
	s := &nodeSorter{}
	if strings.HasSuffix(nodeSort, "=asc") {
		nodeSort = strings.TrimSuffix(nodeSort, "=asc")
	} else if strings.HasSuffix(nodeSort, "=dsc") {
		nodeSort = strings.TrimSuffix(nodeSort, "=dsc")
		s.descending = true
	}
	if nodeSort != "creation" {
		s.label = nodeSort
	}
	return s
}

type nodeSortKey struct {
	node       *Node
	created    time.Time
	value      string
	tieBreaker string
}

// sort orders the nodes in place. Keys are built up front, not inside the
// comparison: there are O(n) nodes but O(n log n) comparisons, and a key can be
// costly, a computed label has to walk the node's resources to produce one.
func (s *nodeSorter) sort(nodes []*Node) {
	keys := make([]nodeSortKey, len(nodes))
	for i, n := range nodes {
		keys[i] = nodeSortKey{node: n}
		if s.label == "" {
			keys[i].created = n.Created()
			keys[i].value = n.Name()
		} else {
			keys[i].value = n.LabelValue(s.label)
			keys[i].tieBreaker = n.InstanceID()
		}
	}

	order := func(less bool) bool { return less }
	if s.descending {
		order = func(less bool) bool { return !less }
	}

	sort.Slice(keys, func(a, b int) bool {
		lhs, rhs := &keys[a], &keys[b]
		if s.label == "" {
			if lhs.created.Equal(rhs.created) {
				return order(naturalLess(lhs.value, rhs.value))
			}
			return order(rhs.created.Before(lhs.created))
		}
		if lhs.value == rhs.value {
			return order(naturalLess(lhs.tieBreaker, rhs.tieBreaker))
		}
		return order(naturalLess(lhs.value, rhs.value))
	})

	for i := range keys {
		nodes[i] = keys[i].node
	}
}
