package tui

import (
	"alirun/pkg/highlighter"
	"alirun/pkg/initsys"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ViewState int

const (
	ViewStateDashboard ViewState = iota
	ViewStateWizard
)

type servicesLoadedMsg struct {
	services []initsys.ServiceInfo
	err      error
}

type serviceDetailMsg struct {
	detail *initsys.ServiceInfo
	err    error
}

type serviceActionMsg struct {
	action string
	name   string
	err    error
}

type logLineMsg string

// Model is the main Bubble Tea model for Autolirun
type Model struct {
	mgr       initsys.Manager
	sType     initsys.ServiceType
	viewState ViewState

	width  int
	height int

	// Dashboard components
	table       table.Model
	services    []initsys.ServiceInfo
	rawServices []initsys.ServiceInfo

	selectedDetail *initsys.ServiceInfo
	logsViewport   viewport.Model
	logsLines      []string

	focusPane int // 0 = table, 1 = logs

	searchMode  bool
	searchInput textinput.Model

	statusMessage string
	statusIsError bool

	confirmDelete bool
	deletingName  string

	logCancel context.CancelFunc

	// Wizard components
	wName       textinput.Model
	wDesc       textinput.Model
	wExec       textinput.Model
	wWorkDir    textinput.Model
	wPreset     int // 0 = Daemon, 1 = Web Server, 2 = One-shot Task
	wScope      initsys.ServiceType
	wFocusField int // 0: Name, 1: Desc, 2: Exec, 3: WorkDir, 4: Preset, 5: Scope, 6: [Save & Start], 7: [Cancel]
	wError      string
}

// NewModel creates an initialized TUI Model
func NewModel(mgr initsys.Manager, sType initsys.ServiceType) *Model {
	// Initialize table
	columns := []table.Column{
		{Title: "ST", Width: 4},
		{Title: "SERVICE", Width: 26},
		{Title: "STATE", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(ColorPrimary).
		Bold(true)
	t.SetStyles(s)

	// Initialize search input
	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.CharLimit = 50
	ti.Width = 30

	vp := viewport.New(40, 10)
	vp.SetContent("Waiting for service logs...")

	m := &Model{
		mgr:          mgr,
		sType:        sType,
		viewState:    ViewStateDashboard,
		table:        t,
		logsViewport: vp,
		searchInput:  ti,
		focusPane:    0,
	}

	m.initWizardInputs()
	return m
}

func (m *Model) initWizardInputs() {
	m.wName = textinput.New()
	m.wName.Placeholder = "e.g. my-daemon"
	m.wName.CharLimit = 50

	m.wDesc = textinput.New()
	m.wDesc.Placeholder = "e.g. My Background Service"
	m.wDesc.CharLimit = 100

	m.wExec = textinput.New()
	m.wExec.Placeholder = "e.g. /home/user/app.sh or python3 -m http.server 8080"
	m.wExec.CharLimit = 250

	cwd, _ := os.Getwd()
	m.wWorkDir = textinput.New()
	m.wWorkDir.Placeholder = "Working directory path"
	m.wWorkDir.SetValue(cwd)
	m.wWorkDir.CharLimit = 150

	m.wPreset = 0 // Daemon
	m.wScope = m.sType
	m.wFocusField = 0
	m.wName.Focus()
	m.wError = ""
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadServicesCmd(),
		tea.EnterAltScreen,
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcLayout()
		return m, nil

	case servicesLoadedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading services: %v", msg.err)
			m.statusIsError = true
		} else {
			m.rawServices = msg.services
			m.applyFilter()
			if len(m.services) > 0 {
				return m, m.loadSelectedDetailCmd()
			}
		}
		return m, nil

	case serviceDetailMsg:
		if msg.err == nil && msg.detail != nil {
			m.selectedDetail = msg.detail
		}
		return m, nil

	case serviceActionMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Action failed: %v", msg.err)
			m.statusIsError = true
		} else {
			m.statusMessage = fmt.Sprintf("✔ Successfully performed %s on %s", msg.action, msg.name)
			m.statusIsError = false
			return m, m.loadServicesCmd()
		}
		return m, nil

	case logLineMsg:
		m.logsLines = append(m.logsLines, string(msg))
		if len(m.logsLines) > 500 {
			m.logsLines = m.logsLines[len(m.logsLines)-500:]
		}
		m.logsViewport.SetContent(strings.Join(m.logsLines, "\n"))
		m.logsViewport.GotoBottom()
		return m, nil
	}

	// Dispatch by view state
	if m.viewState == ViewStateWizard {
		return m.updateWizard(msg)
	}

	return m.updateDashboard(msg)
}

func (m *Model) updateDashboard(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If in search input mode
		if m.searchMode {
			switch msg.String() {
			case "tab", "shift+tab", "enter":
				// Exit search input mode and focus table
				m.searchMode = false
				m.searchInput.Blur()
				m.focusPane = 0
				m.table.Focus()
				return m, nil

			case "esc":
				// Clear search and return focus to table
				m.searchMode = false
				m.searchInput.Blur()
				if m.searchInput.Value() != "" {
					m.searchInput.SetValue("")
					m.applyFilter()
				}
				m.focusPane = 0
				m.table.Focus()
				return m, nil

			case "down", "up":
				// Pressing Down/Up arrow exits search input and navigates filtered list directly
				m.searchMode = false
				m.searchInput.Blur()
				m.focusPane = 0
				m.table.Focus()
				var cmd tea.Cmd
				m.table, cmd = m.table.Update(msg)
				return m, tea.Batch(cmd, m.loadSelectedDetailCmd(), m.startLogsStreamCmd())

			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.applyFilter()
				return m, cmd
			}
		}

		// If in confirm delete mode
		if m.confirmDelete {
			switch msg.String() {
			case "y", "Y":
				m.confirmDelete = false
				name := m.deletingName
				return m, m.deleteServiceCmd(name)
			default:
				m.confirmDelete = false
				m.statusMessage = "Delete cancelled."
				return m, nil
			}
		}

		// Global navigation keys
		switch msg.String() {
		case "ctrl+c", "q":
			if m.logCancel != nil {
				m.logCancel()
			}
			return m, tea.Quit

		case "esc":
			// If not in search mode, pressing Esc clears active filter if any
			if m.searchInput.Value() != "" {
				m.searchInput.SetValue("")
				m.applyFilter()
				m.statusMessage = "Search filter cleared."
				m.statusIsError = false
				return m, nil
			}

		case "n", "N", "c", "C":
			// Open New Service / Daemon Creation Wizard!
			m.initWizardInputs()
			m.viewState = ViewStateWizard
			return m, nil

		case "tab":
			m.focusPane = (m.focusPane + 1) % 2
			if m.focusPane == 0 {
				m.table.Focus()
			} else {
				m.table.Blur()
			}
			return m, nil

		case "shift+tab":
			m.focusPane = (m.focusPane + 1) % 2
			if m.focusPane == 0 {
				m.table.Focus()
			} else {
				m.table.Blur()
			}
			return m, nil

		case "/":
			m.searchMode = true
			m.table.Blur()
			m.searchInput.Focus()
			return m, nil

		case "u", "U":
			// Toggle User / System mode
			if m.sType == initsys.TypeUser {
				m.sType = initsys.TypeSystem
			} else {
				m.sType = initsys.TypeUser
			}
			m.statusMessage = fmt.Sprintf("Switched to %s mode", m.sType)
			m.statusIsError = false
			cmds = append(cmds, m.loadServicesCmd())

		case "s", "S":
			if cur := m.currentSelected(); cur != nil {
				return m, m.actionCmd("start", cur.Name)
			}

		case "x", "X":
			if cur := m.currentSelected(); cur != nil {
				return m, m.actionCmd("stop", cur.Name)
			}

		case "r", "R":
			if cur := m.currentSelected(); cur != nil {
				return m, m.actionCmd("restart", cur.Name)
			}

		case "e", "E":
			if cur := m.currentSelected(); cur != nil {
				return m, m.actionCmd("enable", cur.Name)
			}

		case "d", "D":
			if cur := m.currentSelected(); cur != nil {
				return m, m.actionCmd("disable", cur.Name)
			}

		case "delete", "backspace":
			if cur := m.currentSelected(); cur != nil {
				m.confirmDelete = true
				m.deletingName = cur.Name
				m.statusMessage = fmt.Sprintf("Delete service %q? Press 'y' to confirm, any other key to cancel.", cur.Name)
				m.statusIsError = true
				return m, nil
			}
		}

		// Delegate to focused component
		if m.focusPane == 0 {
			prevIndex := m.table.Cursor()
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			cmds = append(cmds, cmd)

			if m.table.Cursor() != prevIndex {
				cmds = append(cmds, m.loadSelectedDetailCmd(), m.startLogsStreamCmd())
			}
		} else {
			var cmd tea.Cmd
			m.logsViewport, cmd = m.logsViewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateWizard(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		// Return to dashboard
		m.viewState = ViewStateDashboard
		return m, nil

	case "tab", "down":
		m.wFocusField = (m.wFocusField + 1) % 8
		m.syncWizardFocus()
		return m, nil

	case "shift+tab", "up":
		m.wFocusField = (m.wFocusField + 7) % 8
		m.syncWizardFocus()
		return m, nil

	case " ":
		// Space toggles preset or scope if focused on them
		if m.wFocusField == 4 {
			m.wPreset = (m.wPreset + 1) % 3
			return m, nil
		}
		if m.wFocusField == 5 {
			if m.wScope == initsys.TypeUser {
				m.wScope = initsys.TypeSystem
			} else {
				m.wScope = initsys.TypeUser
			}
			return m, nil
		}

	case "enter":
		if m.wFocusField == 6 || (m.wFocusField < 4 && m.wName.Value() != "" && m.wExec.Value() != "") {
			// Submit form and create daemon!
			return m, m.submitWizard(true)
		}
		if m.wFocusField == 7 {
			m.viewState = ViewStateDashboard
			return m, nil
		}
		// Move to next field on Enter in inputs
		m.wFocusField = (m.wFocusField + 1) % 8
		m.syncWizardFocus()
		return m, nil
	}

	// Update active text inputs
	var cmd tea.Cmd
	switch m.wFocusField {
	case 0:
		m.wName, cmd = m.wName.Update(msg)
	case 1:
		m.wDesc, cmd = m.wDesc.Update(msg)
	case 2:
		m.wExec, cmd = m.wExec.Update(msg)
	case 3:
		m.wWorkDir, cmd = m.wWorkDir.Update(msg)
	}

	return m, cmd
}

func (m *Model) syncWizardFocus() {
	m.wName.Blur()
	m.wDesc.Blur()
	m.wExec.Blur()
	m.wWorkDir.Blur()

	switch m.wFocusField {
	case 0:
		m.wName.Focus()
	case 1:
		m.wDesc.Focus()
	case 2:
		m.wExec.Focus()
	case 3:
		m.wWorkDir.Focus()
	}
}

func (m *Model) generateWizardConfig() (initsys.ServiceConfig, string, error) {
	name := strings.TrimSpace(m.wName.Value())
	if name == "" {
		name = "custom-daemon"
	}
	desc := strings.TrimSpace(m.wDesc.Value())
	if desc == "" {
		desc = fmt.Sprintf("%s service", name)
	}
	execCmd := strings.TrimSpace(m.wExec.Value())
	if execCmd == "" {
		execCmd = "/bin/echo 'running custom service'"
	}
	workDir := strings.TrimSpace(m.wWorkDir.Value())

	cfg := initsys.ServiceConfig{
		Name:             name,
		Description:      desc,
		ExecStart:        execCmd,
		WorkingDirectory: workDir,
		Type:             m.wScope,
		RestartSec:       5,
	}

	switch m.wPreset {
	case 1:
		cfg.Preset = initsys.PresetWeb
		cfg.WantsNetwork = true
	case 2:
		cfg.Preset = initsys.PresetOneshot
	default:
		cfg.Preset = initsys.PresetDaemon
	}

	content, err := m.mgr.GenerateConfig(cfg)
	return cfg, content, err
}

func (m *Model) submitWizard(startNow bool) tea.Cmd {
	name := strings.TrimSpace(m.wName.Value())
	execCmd := strings.TrimSpace(m.wExec.Value())

	if name == "" {
		m.wError = "Service name cannot be empty"
		return nil
	}
	if execCmd == "" {
		m.wError = "Command (ExecStart) cannot be empty"
		return nil
	}

	cfg, content, err := m.generateWizardConfig()
	if err != nil {
		m.wError = fmt.Sprintf("Generation error: %v", err)
		return nil
	}

	mgr := m.mgr
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		savedPath, err := mgr.InstallService(ctx, cfg, content, startNow)
		if err != nil {
			return serviceActionMsg{action: "create", name: name, err: err}
		}

		return serviceActionMsg{
			action: "create & start",
			name:   fmt.Sprintf("%s (%s)", name, savedPath),
			err:    nil,
		}
	}
}

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing Autolirun..."
	}

	if m.viewState == ViewStateWizard {
		return m.renderWizardView()
	}

	return m.renderDashboardView()
}

func (m *Model) renderDashboardView() string {
	// 1. Header
	modeBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#E91E63")).
		Padding(0, 1).
		Render(strings.ToUpper(string(m.sType)))

	headerText := fmt.Sprintf(" Alirun (%s) | Scope: %s ", m.mgr.Name(), modeBadge)
	if m.searchMode {
		headerText += fmt.Sprintf(" | 🔍 %s  [Tab/Enter: Select, Esc: Clear]", m.searchInput.View())
	} else if m.searchInput.Value() != "" {
		headerText += fmt.Sprintf(" | 🔍 Filter: \"%s\"  [Esc: Clear, /: Edit]", m.searchInput.Value())
	}
	header := HeaderStyle.Width(m.width).Render(headerText)

	// 2. Main content panes
	leftWidth := m.width * 45 / 100
	if leftWidth < 30 {
		leftWidth = 30
	}
	rightWidth := m.width - leftWidth - 4

	bodyHeight := m.height - 4
	if bodyHeight < 10 {
		bodyHeight = 10
	}

	// Left pane: Table
	tableStyle := PaneBaseStyle
	if m.focusPane == 0 {
		tableStyle = PaneFocusedStyle
	}
	leftPane := tableStyle.
		Width(leftWidth).
		Height(bodyHeight).
		Render(m.table.View())

	// Right Top: Details
	detailsHeight := bodyHeight * 38 / 100
	detailsPane := PaneBaseStyle.
		Width(rightWidth).
		Height(detailsHeight).
		Render(m.renderDetails())

	// Right Bottom: Logs
	logsHeight := bodyHeight - detailsHeight - 3
	m.logsViewport.Width = rightWidth - 2
	m.logsViewport.Height = logsHeight

	logsStyle := PaneBaseStyle
	if m.focusPane == 1 {
		logsStyle = PaneFocusedStyle
	}
	logsPane := logsStyle.
		Width(rightWidth).
		Height(logsHeight).
		Render(fmt.Sprintf("%s\n%s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("Journalctl Logs"),
			m.logsViewport.View()))

	rightColumn := lipgloss.JoinVertical(lipgloss.Left, detailsPane, logsPane)
	mainBody := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightColumn)

	// 3. Footer / Status bar
	statusContent := m.renderFooter()
	footer := FooterStyle.Width(m.width).Render(statusContent)

	return lipgloss.JoinVertical(lipgloss.Left, header, mainBody, footer)
}

func (m *Model) renderWizardView() string {
	header := HeaderStyle.Width(m.width).Render(" Alirun Wizard: Create New Service / Daemon ")

	bodyHeight := m.height - 4
	if bodyHeight < 15 {
		bodyHeight = 15
	}

	leftWidth := m.width * 50 / 100
	if leftWidth < 40 {
		leftWidth = 40
	}
	rightWidth := m.width - leftWidth - 4

	// Left: Interactive form
	var formLines []string
	formLines = append(formLines, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("⚙ Service Parameters"))
	formLines = append(formLines, "")

	renderField := func(index int, title, desc string, widget string) string {
		tStyle := lipgloss.NewStyle().Bold(true)
		if m.wFocusField == index {
			tStyle = tStyle.Foreground(ColorActive)
		} else {
			tStyle = tStyle.Foreground(ColorText)
		}
		return fmt.Sprintf("%s %s\n%s\n%s",
			tStyle.Render("▶ "+title),
			lipgloss.NewStyle().Faint(true).Render("("+desc+")"),
			widget,
			"")
	}

	formLines = append(formLines, renderField(0, "Service Name", "unique identifier without .service", m.wName.View()))
	formLines = append(formLines, renderField(1, "Description", "human-readable description", m.wDesc.View()))
	formLines = append(formLines, renderField(2, "ExecStart Command", "command or path to binary/script", m.wExec.View()))
	formLines = append(formLines, renderField(3, "Working Directory", "execution directory", m.wWorkDir.View()))

	// Preset selector
	presetNames := []string{"Daemon (Restart=always)", "Web Server / API (network-online)", "One-shot Task (runs once)"}
	presetBadge := fmt.Sprintf("[%s]  (press Space to cycle)", presetNames[m.wPreset])
	presetStyle := lipgloss.NewStyle()
	if m.wFocusField == 4 {
		presetStyle = presetStyle.Bold(true).Foreground(ColorActive)
	}
	formLines = append(formLines, presetStyle.Render("▶ Execution Preset: ")+presetBadge+"\n")

	// Scope selector
	scopeBadge := fmt.Sprintf("[%s]  (press Space to toggle)", strings.ToUpper(string(m.wScope)))
	scopeStyle := lipgloss.NewStyle()
	if m.wFocusField == 5 {
		scopeStyle = scopeStyle.Bold(true).Foreground(ColorActive)
	}
	formLines = append(formLines, scopeStyle.Render("▶ Target Scope: ")+scopeBadge+"\n")

	// Buttons
	btnStart := "[ Save & Start Daemon Now ]"
	if m.wFocusField == 6 {
		btnStart = lipgloss.NewStyle().Bold(true).Background(ColorActive).Foreground(lipgloss.Color("#000000")).Render(btnStart)
	} else {
		btnStart = lipgloss.NewStyle().Bold(true).Foreground(ColorActive).Render(btnStart)
	}

	btnCancel := "[ Cancel ]"
	if m.wFocusField == 7 {
		btnCancel = lipgloss.NewStyle().Bold(true).Background(ColorFailed).Foreground(lipgloss.Color("#FFFFFF")).Render(btnCancel)
	} else {
		btnCancel = lipgloss.NewStyle().Faint(true).Render(btnCancel)
	}

	formLines = append(formLines, fmt.Sprintf("%s    %s", btnStart, btnCancel))

	if m.wError != "" {
		formLines = append(formLines, "\n"+lipgloss.NewStyle().Bold(true).Foreground(ColorFailed).Render("⚠ "+m.wError))
	}

	leftBox := PaneFocusedStyle.
		Width(leftWidth).
		Height(bodyHeight).
		Render(strings.Join(formLines, "\n"))

	// Right: Live Transparency Preview
	_, unitContent, _ := m.generateWizardConfig()
	highlightedUnit := highlighter.HighlightUnit(unitContent)

	destPath := m.mgr.GetConfigPath(strings.TrimSpace(m.wName.Value()), m.wScope)
	rightContent := fmt.Sprintf("%s\n%s: %s\n\n%s",
		lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("Live Unit Preview (.service)"),
		lipgloss.NewStyle().Faint(true).Render("Destination"),
		lipgloss.NewStyle().Bold(true).Render(destPath),
		highlightedUnit,
	)

	rightBox := PaneBaseStyle.
		Width(rightWidth).
		Height(bodyHeight).
		Render(rightContent)

	mainBody := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)

	// Footer hints for wizard
	wizardFooter := FooterStyle.Width(m.width).Render(
		KeyHintStyle.Render("[Tab/Down]") + " Next Field  " +
			KeyHintStyle.Render("[Shift+Tab/Up]") + " Prev Field  " +
			KeyHintStyle.Render("[Space]") + " Cycle Option  " +
			KeyHintStyle.Render("[Enter]") + " Save & Start  " +
			KeyHintStyle.Render("[Esc]") + " Cancel / Back to Dashboard",
	)

	return lipgloss.JoinVertical(lipgloss.Left, header, mainBody, wizardFooter)
}

func (m *Model) renderDetails() string {
	if m.selectedDetail == nil {
		return lipgloss.NewStyle().Faint(true).Render("No service selected.")
	}

	info := m.selectedDetail
	var statusBadge string
	switch info.Status {
	case initsys.StatusActive:
		statusBadge = BadgeActive.Render("● ACTIVE (" + info.SubState + ")")
	case initsys.StatusFailed:
		statusBadge = BadgeFailed.Render("✖ FAILED (" + info.SubState + ")")
	default:
		statusBadge = BadgeInactive.Render("○ INACTIVE (" + info.SubState + ")")
	}

	var sb strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(info.Name)
	sb.WriteString(fmt.Sprintf("%s  [%s]  Enabled: %t\n", title, statusBadge, info.Enabled))

	if info.Description != "" {
		sb.WriteString(fmt.Sprintf("Desc:    %s\n", info.Description))
	}
	if info.PID > 0 {
		sb.WriteString(fmt.Sprintf("PID:     %d\n", info.PID))
	}
	if info.MemoryBytes > 0 {
		mb := float64(info.MemoryBytes) / (1024 * 1024)
		sb.WriteString(fmt.Sprintf("Memory:  %.1f MB\n", mb))
	}
	if info.ConfigPath != "" {
		sb.WriteString(fmt.Sprintf("Unit:    %s\n", info.ConfigPath))
	}
	if info.ExecPath != "" {
		sb.WriteString(fmt.Sprintf("Exec:    %s\n", info.ExecPath))
	}

	return sb.String()
}

func (m *Model) renderFooter() string {
	if m.statusMessage != "" {
		color := ColorActive
		if m.statusIsError {
			color = ColorFailed
		}
		return lipgloss.NewStyle().Foreground(color).Bold(true).Render(m.statusMessage)
	}

	hints := []string{
		lipgloss.NewStyle().Bold(true).Background(ColorActive).Foreground(lipgloss.Color("#000000")).Render(" [N]ew Daemon ") + " ",
		KeyHintStyle.Render("[Tab]") + " Focus",
		KeyHintStyle.Render("[S]") + "tart",
		KeyHintStyle.Render("[X]") + "top",
		KeyHintStyle.Render("[R]") + "estart",
		KeyHintStyle.Render("[E]") + "nable",
		KeyHintStyle.Render("[D]") + "isable",
		KeyHintStyle.Render("[Del]") + "ete",
		KeyHintStyle.Render("[U]") + "ser/Sys",
		KeyHintStyle.Render("[/]") + "Search",
		KeyHintStyle.Render("[Q]") + "uit",
	}

	return strings.Join(hints, "  ")
}

func (m *Model) recalcLayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	leftWidth := m.width * 45 / 100
	if leftWidth < 30 {
		leftWidth = 30
	}
	m.table.SetWidth(leftWidth - 4)
	m.table.SetHeight(m.height - 7)

	rightWidth := m.width - leftWidth - 6
	bodyHeight := m.height - 4
	detailsHeight := bodyHeight * 38 / 100
	logsHeight := bodyHeight - detailsHeight - 3

	m.logsViewport.Width = rightWidth
	m.logsViewport.Height = logsHeight
}

func (m *Model) currentSelected() *initsys.ServiceInfo {
	idx := m.table.Cursor()
	if idx >= 0 && idx < len(m.services) {
		return &m.services[idx]
	}
	return nil
}

func (m *Model) applyFilter() {
	query := strings.ToLower(m.searchInput.Value())
	var filtered []initsys.ServiceInfo
	var rows []table.Row

	for _, s := range m.rawServices {
		if query != "" {
			nameMatch := strings.Contains(strings.ToLower(s.Name), query)
			descMatch := strings.Contains(strings.ToLower(s.Description), query)
			if !nameMatch && !descMatch {
				continue
			}
		}

		var stIcon string
		switch s.Status {
		case initsys.StatusActive:
			stIcon = "●"
		case initsys.StatusFailed:
			stIcon = "✖"
		default:
			stIcon = "○"
		}

		rows = append(rows, table.Row{
			stIcon,
			s.Name,
			s.SubState,
		})
		filtered = append(filtered, s)
	}

	m.services = filtered
	m.table.SetRows(rows)
	if m.table.Cursor() >= len(rows) && len(rows) > 0 {
		m.table.SetCursor(len(rows) - 1)
	}
}

// Commands
func (m *Model) loadServicesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		services, err := m.mgr.ListServices(ctx, m.sType)
		return servicesLoadedMsg{services: services, err: err}
	}
}

func (m *Model) loadSelectedDetailCmd() tea.Cmd {
	cur := m.currentSelected()
	if cur == nil {
		return nil
	}
	name := cur.Name
	sType := m.sType
	mgr := m.mgr

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		detail, err := mgr.GetStatus(ctx, name, sType)
		return serviceDetailMsg{detail: detail, err: err}
	}
}

func (m *Model) startLogsStreamCmd() tea.Cmd {
	if m.logCancel != nil {
		m.logCancel()
		m.logCancel = nil
	}

	cur := m.currentSelected()
	if cur == nil {
		return nil
	}

	m.logsLines = []string{fmt.Sprintf("Loading logs for %s...", cur.Name)}
	m.logsViewport.SetContent(strings.Join(m.logsLines, "\n"))

	name := cur.Name
	sType := m.sType
	mgr := m.mgr

	ctx, cancel := context.WithCancel(context.Background())
	m.logCancel = cancel

	return func() tea.Msg {
		// Read initial lines
		ch, err := mgr.StreamLogs(ctx, name, sType, 40, false)
		if err != nil {
			return logLineMsg(fmt.Sprintf("Logs error: %v", err))
		}
		var lines []string
		for line := range ch {
			lines = append(lines, line)
		}
		if len(lines) == 0 {
			lines = append(lines, "(no recent logs found)")
		}
		return logLineMsg(strings.Join(lines, "\n"))
	}
}

func (m *Model) actionCmd(action, name string) tea.Cmd {
	sType := m.sType
	mgr := m.mgr

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var err error
		switch action {
		case "start":
			err = mgr.Start(ctx, name, sType)
		case "stop":
			err = mgr.Stop(ctx, name, sType)
		case "restart":
			err = mgr.Restart(ctx, name, sType)
		case "enable":
			err = mgr.Enable(ctx, name, sType)
		case "disable":
			err = mgr.Disable(ctx, name, sType)
		}

		return serviceActionMsg{action: action, name: name, err: err}
	}
}

func (m *Model) deleteServiceCmd(name string) tea.Cmd {
	sType := m.sType
	mgr := m.mgr

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := mgr.DeleteService(ctx, name, sType)
		return serviceActionMsg{action: "delete", name: name, err: err}
	}
}
