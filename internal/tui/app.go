package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gregoirelafitte/packman/internal/analyzer"
	"github.com/gregoirelafitte/packman/pkg/types"
)

// viewState tracks which view is active.
type viewState int

const (
	viewLoading viewState = iota
	viewTable
	viewDetail
	viewHelp
)

// Messages
type analysisProgressMsg struct{ step string }
type analysisDoneMsg struct{ reports []types.ProjectReport }
type analysisErrorMsg struct{ err error }

// Model is the top-level bubbletea model.
type Model struct {
	// State
	state       viewState
	projectRoot string

	// Data
	reports      []types.ProjectReport
	activeReport int
	allResults   []types.AnalysisResult // unfiltered
	filtered     []types.AnalysisResult // filtered + sorted
	cursor       int
	scrollOffset int

	// Sorting
	sortCol sortColumn
	sortAsc bool

	// Filtering
	filterActive bool
	filterText   string

	// Loading
	spinner      spinner.Model
	progressMsg  string
	progressDone []string // completed steps

	// Error
	errMsg string

	// Dimensions
	width  int
	height int
}

// New creates a new TUI model.
func New(projectRoot string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return Model{
		state:       viewLoading,
		projectRoot: projectRoot,
		spinner:     s,
		sortCol:     sortByHealth,
		sortAsc:     true,
		progressMsg: "Starting analysis...",
	}
}


func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case analysisProgressMsg:
		if m.progressMsg != "" && m.progressMsg != "Starting analysis..." {
			m.progressDone = append(m.progressDone, m.progressMsg)
		}
		m.progressMsg = msg.step
		return m, nil

	case analysisDoneMsg:
		m.reports = msg.reports
		if len(m.reports) > 0 {
			m.activeReport = 0
			m.loadReport()
		}
		m.state = viewTable
		return m, nil

	case analysisErrorMsg:
		m.errMsg = msg.err.Error()
		m.state = viewTable
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle filter input mode
	if m.filterActive {
		return m.handleFilterKey(msg)
	}

	switch {
	case key.Matches(msg, keys.Quit):
		if m.state == viewDetail || m.state == viewHelp {
			m.state = viewTable
			return m, nil
		}
		return m, tea.Quit

	case key.Matches(msg, keys.Escape):
		switch m.state {
		case viewDetail, viewHelp:
			m.state = viewTable
		}
		return m, nil

	case key.Matches(msg, keys.Help):
		if m.state == viewHelp {
			m.state = viewTable
		} else {
			m.state = viewHelp
		}
		return m, nil

	case key.Matches(msg, keys.Up):
		if m.state == viewTable && m.cursor > 0 {
			m.cursor--
			m.adjustScroll()
		}
		return m, nil

	case key.Matches(msg, keys.Down):
		if m.state == viewTable && m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.adjustScroll()
		}
		return m, nil

	case key.Matches(msg, keys.PageUp):
		if m.state == viewTable {
			m.cursor -= m.visibleRows()
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.adjustScroll()
		}
		return m, nil

	case key.Matches(msg, keys.PageDown):
		if m.state == viewTable {
			m.cursor += m.visibleRows()
			if m.cursor >= len(m.filtered) {
				m.cursor = len(m.filtered) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.adjustScroll()
		}
		return m, nil

	case key.Matches(msg, keys.Enter):
		if m.state == viewTable && m.cursor < len(m.filtered) {
			m.state = viewDetail
		}
		return m, nil

	case key.Matches(msg, keys.Filter):
		if m.state == viewTable {
			m.filterActive = true
			m.filterText = ""
		}
		return m, nil

	case key.Matches(msg, keys.Sort):
		if m.state == viewTable {
			m.sortCol = (m.sortCol + 1) % sortColumnCount
			m.applySort()
		}
		return m, nil

	case key.Matches(msg, keys.SortRev):
		if m.state == viewTable {
			m.sortAsc = !m.sortAsc
			m.applySort()
		}
		return m, nil

	case key.Matches(msg, keys.Tab):
		if m.state == viewTable && len(m.reports) > 1 {
			m.activeReport = (m.activeReport + 1) % len(m.reports)
			m.loadReport()
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.filterActive = false
		m.filterText = ""
		m.applyFilter()
		return m, nil
	case tea.KeyEnter:
		m.filterActive = false
		return m, nil
	case tea.KeyBackspace:
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			m.applyFilter()
		}
		return m, nil
	default:
		if len(msg.Runes) > 0 {
			m.filterText += string(msg.Runes)
			m.applyFilter()
		}
		return m, nil
	}
}

func (m *Model) loadReport() {
	if m.activeReport >= len(m.reports) {
		return
	}
	report := m.reports[m.activeReport]
	m.allResults = report.Dependencies
	m.cursor = 0
	m.scrollOffset = 0
	m.applySort()
}

func (m *Model) applySort() {
	m.filtered = filterResults(m.allResults, m.filterText)
	sortResults(m.filtered, m.sortCol, m.sortAsc)
}

func (m *Model) applyFilter() {
	m.filtered = filterResults(m.allResults, m.filterText)
	sortResults(m.filtered, m.sortCol, m.sortAsc)
	m.cursor = 0
	m.scrollOffset = 0
}

func (m *Model) adjustScroll() {
	visible := m.visibleRows()
	if visible <= 0 {
		visible = 10
	}
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+visible {
		m.scrollOffset = m.cursor - visible + 1
	}
}

func (m Model) visibleRows() int {
	// height minus header(3) + summary(2) + footer(2) + borders(2)
	rows := m.height - 9
	if rows < 1 {
		rows = 10
	}
	return rows
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.state {
	case viewLoading:
		return m.viewLoading()
	case viewDetail:
		return m.viewDetail()
	case viewHelp:
		return m.viewHelp()
	default:
		return m.viewTable()
	}
}

func (m Model) viewLoading() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + titleStyle.Render(" PackMan ") + "\n\n")

	// Show completed steps with checkmarks
	for _, step := range m.progressDone {
		b.WriteString("  " + cellGreen.Render("✓") + " " + cellDim.Render(step) + "\n")
	}

	// Current step with spinner
	b.WriteString("  " + m.spinner.View() + " " + lipgloss.NewStyle().Foreground(colorWhite).Render(m.progressMsg) + "\n")

	return b.String()
}

func (m Model) viewTable() string {
	if m.errMsg != "" {
		return cellRed.Render(fmt.Sprintf("\n  Error: %s\n\n  Press q to quit.", m.errMsg))
	}

	var b strings.Builder

	// Title bar
	ecosystem := ""
	if len(m.reports) > 0 {
		ecosystem = m.reports[m.activeReport].Ecosystem
	}

	title := titleStyle.Render(" PackMan ")
	ecosystemLabel := subtitleStyle.Render(fmt.Sprintf(" %s Analysis", ecosystem))

	// Tab indicators if multiple ecosystems
	tabs := ""
	if len(m.reports) > 1 {
		for i, r := range m.reports {
			if i == m.activeReport {
				tabs += titleStyle.Render(fmt.Sprintf(" %s ", r.Ecosystem))
			} else {
				tabs += cellDim.Render(fmt.Sprintf(" %s ", r.Ecosystem))
			}
			tabs += " "
		}
	}

	b.WriteString(title + ecosystemLabel)
	if tabs != "" {
		b.WriteString("  " + tabs)
	}
	b.WriteString("\n")

	// Summary bar
	if len(m.reports) > 0 {
		b.WriteString(renderSummary(m.reports[m.activeReport], m.width))
	}
	b.WriteString("\n\n")

	// Sort indicator
	sortInfo := cellDim.Render(fmt.Sprintf("Sort: %s", m.sortCol))
	if m.sortAsc {
		sortInfo += cellDim.Render(" ↑")
	} else {
		sortInfo += cellDim.Render(" ↓")
	}
	b.WriteString("  " + sortInfo + "\n")

	// Table header
	b.WriteString("  " + renderTableHeader(m.width) + "\n")

	// Table rows
	visible := m.visibleRows()
	end := m.scrollOffset + visible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := m.scrollOffset; i < end; i++ {
		prefix := "  "
		if i == m.cursor {
			prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true).Render("▸ ")
		}
		b.WriteString(prefix + renderTableRow(m.filtered[i], i == m.cursor, m.width) + "\n")
	}

	// Pad remaining space
	rendered := end - m.scrollOffset
	for i := rendered; i < visible; i++ {
		b.WriteString("\n")
	}

	// Filter bar or footer
	if m.filterActive {
		b.WriteString("\n  " + filterPromptStyle.Render("Filter: ") + filterInputStyle.Render(m.filterText+"█"))
	} else {
		footer := footerHelp()
		b.WriteString("\n" + footerStyle.Render("  "+footer))
	}

	return b.String()
}

func (m Model) viewDetail() string {
	if m.cursor >= len(m.filtered) {
		return m.viewTable()
	}
	return renderDetail(m.filtered[m.cursor], m.width, m.height)
}

func (m Model) viewHelp() string {
	return renderHelp(m.width)
}

func footerHelp() string {
	items := []struct {
		key  string
		desc string
	}{
		{"↑↓", "navigate"},
		{"enter", "detail"},
		{"/", "filter"},
		{"s", "sort"},
		{"?", "help"},
		{"q", "quit"},
	}

	var parts []string
	for _, item := range items {
		k := helpKeyStyle.Render(item.key)
		d := helpDescStyle.Render(item.desc)
		parts = append(parts, k+" "+d)
	}

	return strings.Join(parts, "  ")
}

// Run starts the TUI application.
func Run(projectRoot string) error {
	m := New(projectRoot)
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	// Launch analysis in a goroutine so we can pass *tea.Program for progress updates
	go func() {
		reports, err := analyzer.Run(projectRoot, func(step string) {
			p.Send(analysisProgressMsg{step: step})
		})
		if err != nil {
			p.Send(analysisErrorMsg{err: err})
			return
		}
		p.Send(analysisDoneMsg{reports: reports})
	}()

	_, err := p.Run()
	return err
}
