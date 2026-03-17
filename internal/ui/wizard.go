// Package ui provides the main Bubble Tea wizard model.
package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"

	"clashctl/internal/core"
	"clashctl/internal/mihomo"
)

// WizardModel is the main TUI state.
type WizardModel struct {
	screen    Screen
	appCfg    *core.AppConfig
	width     int
	height    int
	quitting  bool

	// Subscription URL input
	urlInput textinput.Model

	// Mode selection
	modeIndex int // 0 = TUN, 1 = mixed-port

	// Advanced settings
	advancedIndex   int
	advancedFields  []string
	advancedInputs  []textinput.Model

	// Result
	execSteps   []ExecStep
	execError   string

	// Nodes screen
	nodes               []NodeItem
	nodeIndex           int
	nodesLoading        bool
	nodesError          string
	testingAll          bool
	controllerAvailable bool
	selectedNodeName    string
	nodeAction          string // "fetch", "test-all", "select"
}

// ExecStep represents a single execution step result.
type ExecStep struct {
	Label   string
	Success bool
	Detail  string
}

// NodeItem represents a proxy node in the TUI list.
type NodeItem struct {
	Name     string
	Type     string
	Delay    int  // ms, 0=unknown, <0=timeout/error
	Selected bool // currently active in PROXY group
	Testing  bool // currently being tested
}

// --- Bubble Tea messages for async operations ---

type nodesFetchedMsg struct {
	nodes []NodeItem
	err   string
}

type nodeTestResultMsg struct {
	index int
	delay int
}

type allNodesTestedMsg struct {
	results []struct {
		index int
		delay int
	}
}

type nodeSwitchedMsg struct {
	nodeName string
	err      string
}

// NewWizard creates a new WizardModel with defaults.
func NewWizard() WizardModel {
	// URL input
	urlInput := textinput.New()
	urlInput.Placeholder = "https://example.com/subscription"
	urlInput.Focus()
	urlInput.Width = 60
	urlInput.Prompt = "› "
	urlInput.PromptStyle = InputStyle
	urlInput.TextStyle = InputStyle

	// Advanced fields
	fields := []string{
		"配置目录",
		"控制器地址",
		"mixed-port",
		"Provider 路径",
		"健康检查",
		"systemd 服务",
		"自动启动",
	}
	advInputs := make([]textinput.Model, len(fields))
	for i, label := range fields {
		ti := textinput.New()
		ti.Width = 40
		ti.Prompt = "› "
		ti.PromptStyle = InputStyle
		ti.TextStyle = InputStyle
		switch label {
		case "配置目录":
			ti.SetValue("/etc/mihomo")
		case "控制器地址":
			ti.SetValue("127.0.0.1:9090")
		case "mixed-port":
			ti.SetValue("7890")
		case "Provider 路径":
			ti.SetValue("./providers/airport.yaml")
		case "健康检查":
			ti.SetValue("是")
		case "systemd 服务":
			ti.SetValue("是")
		case "自动启动":
			ti.SetValue("是")
		}
		advInputs[i] = ti
	}

	return WizardModel{
		screen:         ScreenWelcome,
		appCfg:         core.DefaultAppConfig(),
		urlInput:       urlInput,
		advancedFields: fields,
		advancedInputs: advInputs,
	}
}

func (m WizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case nodesFetchedMsg:
		return m.handleNodesFetched(msg)
	case allNodesTestedMsg:
		return m.handleAllNodesTested(msg)
	case nodeSwitchedMsg:
		return m.handleNodeSwitched(msg)
	}
	return m, nil
}

func (m WizardModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit
	if msg.String() == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}

	switch m.screen {
	case ScreenWelcome:
		return m.updateWelcome(msg)
	case ScreenSubscription:
		return m.updateSubscription(msg)
	case ScreenMode:
		return m.updateMode(msg)
	case ScreenAdvanced:
		return m.updateAdvanced(msg)
	case ScreenPreview:
		return m.updatePreview(msg)
	case ScreenResult:
		return m.updateResult(msg)
	case ScreenNodes:
		return m.updateNodes(msg)
	}
	return m, nil
}

func (m WizardModel) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.screen = ScreenSubscription
		return m, nil
	case "q", "esc":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m WizardModel) updateSubscription(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		url := strings.TrimSpace(m.urlInput.Value())
		if url == "" {
			return m, nil // don't proceed without URL
		}
		m.appCfg.SubscriptionURL = url
		m.screen = ScreenMode
		return m, nil
	case "esc":
		m.screen = ScreenWelcome
		return m, nil
	default:
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	}
}

func (m WizardModel) updateMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.modeIndex > 0 {
			m.modeIndex--
		}
	case "down", "j":
		if m.modeIndex < 1 {
			m.modeIndex++
		}
	case "enter":
		if m.modeIndex == 0 {
			m.appCfg.Mode = "tun"
		} else {
			m.appCfg.Mode = "mixed"
		}
		m.screen = ScreenAdvanced
		return m, nil
	case "esc":
		m.screen = ScreenSubscription
		return m, nil
	}
	return m, nil
}

func (m WizardModel) updateAdvanced(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.advancedIndex > 0 {
			m.advancedIndex--
		}
	case "down", "j":
		if m.advancedIndex < len(m.advancedFields)-1 {
			m.advancedIndex++
		}
	case "enter":
		// Collect advanced values
		m.collectAdvancedValues()
		m.screen = ScreenPreview
		return m, nil
	case "esc":
		m.screen = ScreenMode
		return m, nil
	default:
		// Update the focused input
		if m.advancedIndex < len(m.advancedInputs) {
			var cmd tea.Cmd
			m.advancedInputs[m.advancedIndex], cmd = m.advancedInputs[m.advancedIndex].Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *WizardModel) collectAdvancedValues() {
	for i, field := range m.advancedFields {
		val := m.advancedInputs[i].Value()
		switch field {
		case "配置目录":
			m.appCfg.ConfigDir = val
		case "控制器地址":
			m.appCfg.ControllerAddr = val
		case "Provider 路径":
			m.appCfg.ProviderPath = val
		case "健康检查":
			m.appCfg.EnableHealthCheck = (val == "是" || val == "yes" || val == "true" || val == "1")
		case "systemd 服务":
			m.appCfg.EnableSystemd = (val == "是" || val == "yes" || val == "true" || val == "1")
		case "自动启动":
			m.appCfg.AutoStart = (val == "是" || val == "yes" || val == "true" || val == "1")
		}
	}
}

func (m WizardModel) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Execute full pipeline!
		m.execSteps = m.executeFull()
		m.screen = ScreenResult
		return m, nil
	case "esc":
		m.screen = ScreenAdvanced
		return m, nil
	}
	return m, nil
}

func (m WizardModel) updateResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Check if controller API is available → go to nodes screen
		if m.controllerAvailable {
			m.screen = ScreenNodes
			m.nodesLoading = true
			m.nodesError = ""
			m.nodeIndex = 0
			return m, m.fetchNodesCmd()
		}
		// Otherwise quit
		m.quitting = true
		return m, tea.Quit
	case "q":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

// --- Nodes screen handlers ---

func (m WizardModel) updateNodes(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.nodesLoading || m.testingAll {
		// Only allow quit during loading/testing
		switch msg.String() {
		case "q", "esc":
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.nodeIndex > 0 {
			m.nodeIndex--
		}
	case "down", "j":
		if m.nodeIndex < len(m.nodes)-1 {
			m.nodeIndex++
		}
	case "t":
		// Test all nodes
		if len(m.nodes) > 0 {
			m.testingAll = true
			for i := range m.nodes {
				m.nodes[i].Testing = true
				m.nodes[i].Delay = 0
			}
			return m, m.testAllNodesCmd()
		}
	case "enter":
		// Select the highlighted node
		if len(m.nodes) > 0 && m.nodeIndex < len(m.nodes) {
			nodeName := m.nodes[m.nodeIndex].Name
			m.nodeAction = "select"
			return m, m.switchNodeCmd(nodeName)
		}
	case "r":
		// Refresh nodes list
		m.nodesLoading = true
		m.nodesError = ""
		return m, m.fetchNodesCmd()
	case "q":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		// Go back to result screen
		m.screen = ScreenResult
		return m, nil
	}
	return m, nil
}

// --- Async commands ---

func (m WizardModel) fetchNodesCmd() tea.Cmd {
	return func() tea.Msg {
		client := mihomo.NewClient("http://" + m.appCfg.ControllerAddr)

		// Retry up to 5 times with 1s delay (wait for mihomo to start)
		var detail *mihomo.ProxyGroupDetail
		var err error
		for i := 0; i < 5; i++ {
			detail, err = client.GetProxyGroupDetail("PROXY")
			if err == nil {
				break
			}
			if i < 4 {
				time.Sleep(1 * time.Second)
			}
		}
		_ = detail

		if err != nil {
			return nodesFetchedMsg{err: "无法获取节点列表: " + err.Error()}
		}

		nodes := make([]NodeItem, 0, len(detail.Nodes))
		for _, n := range detail.Nodes {
			nodes = append(nodes, NodeItem{
				Name:     n.Name,
				Type:     n.Type,
				Delay:    n.Delay,
				Selected: n.Selected,
			})
		}

		return nodesFetchedMsg{nodes: nodes}
	}
}

func (m WizardModel) testAllNodesCmd() tea.Cmd {
	return func() tea.Msg {
		client := mihomo.NewClient("http://" + m.appCfg.ControllerAddr)

		results := make([]struct {
			index int
			delay int
		}, len(m.nodes))

		for i := range m.nodes {
			delay := client.TestNode("PROXY", m.nodes[i].Name)
			results[i] = struct {
				index int
				delay int
			}{index: i, delay: delay}
		}

		return allNodesTestedMsg{results: results}
	}
}

func (m WizardModel) switchNodeCmd(nodeName string) tea.Cmd {
	return func() tea.Msg {
		client := mihomo.NewClient("http://" + m.appCfg.ControllerAddr)
		err := client.SwitchProxy("PROXY", nodeName)
		if err != nil {
			return nodeSwitchedMsg{err: "切换节点失败: " + err.Error()}
		}
		return nodeSwitchedMsg{nodeName: nodeName}
	}
}

// --- Message handlers ---

func (m WizardModel) handleNodesFetched(msg nodesFetchedMsg) (tea.Model, tea.Cmd) {
	m.nodesLoading = false
	if msg.err != "" {
		m.nodesError = msg.err
		return m, nil
	}
	m.nodes = msg.nodes
	m.nodeIndex = 0
	m.sortNodesByDelay()
	return m, nil
}

func (m WizardModel) handleAllNodesTested(msg allNodesTestedMsg) (tea.Model, tea.Cmd) {
	m.testingAll = false
	for _, r := range msg.results {
		if r.index >= 0 && r.index < len(m.nodes) {
			m.nodes[r.index].Delay = r.delay
			m.nodes[r.index].Testing = false
		}
	}
	m.sortNodesByDelay()
	return m, nil
}

func (m WizardModel) handleNodeSwitched(msg nodeSwitchedMsg) (tea.Model, tea.Cmd) {
	if msg.err != "" {
		m.nodesError = msg.err
		return m, nil
	}
	// Update selected state
	for i := range m.nodes {
		m.nodes[i].Selected = (m.nodes[i].Name == msg.nodeName)
	}
	m.selectedNodeName = msg.nodeName
	m.nodesError = ""
	return m, nil
}

// sortNodesByDelay sorts nodes by latency (fastest first).
func (m *WizardModel) sortNodesByDelay() {
	if len(m.nodes) == 0 {
		return
	}
	// Simple bubble sort - nodes are usually <100
	for i := 0; i < len(m.nodes); i++ {
		for j := i + 1; j < len(m.nodes); j++ {
			ai, aj := m.nodes[i].Delay, m.nodes[j].Delay
			if shouldSwap(ai, aj) {
				m.nodes[i], m.nodes[j] = m.nodes[j], m.nodes[i]
			}
		}
	}
}

func shouldSwap(a, b int) bool {
	// Both unknown or timeout: keep order
	if a <= 0 && b <= 0 {
		return false
	}
	// Unknown/timeout goes after known delays
	if a <= 0 {
		return false
	}
	if b <= 0 {
		return true
	}
	// Both known: lower delay first
	return a > b
}

// View renders the current screen.
func (m WizardModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(TitleStyle.Render("🧙 clashctl 配置向导"))
	b.WriteString("\n")

	// Step indicator (except welcome)
	if m.screen != ScreenWelcome {
		b.WriteString(StepStyle.Render(m.screen.StepLabel()))
		b.WriteString("\n\n")
	}

	switch m.screen {
	case ScreenWelcome:
		b.WriteString(m.viewWelcome())
	case ScreenSubscription:
		b.WriteString(m.viewSubscription())
	case ScreenMode:
		b.WriteString(m.viewMode())
	case ScreenAdvanced:
		b.WriteString(m.viewAdvanced())
	case ScreenPreview:
		b.WriteString(m.viewPreview())
	case ScreenResult:
		b.WriteString(m.viewResult())
	case ScreenNodes:
		b.WriteString(m.viewNodes())
	}

	return b.String()
}

// --- Node delay formatting ---

func formatNodeDelay(delay int) string {
	switch {
	case delay == 0:
		return "未测试"
	case delay < 0:
		return "超时"
	case delay < 100:
		return fmt.Sprintf("%dms ✨", delay)
	case delay < 300:
		return fmt.Sprintf("%dms", delay)
	case delay < 1000:
		return fmt.Sprintf("%dms ⚠️", delay)
	default:
		return fmt.Sprintf("%.1fs 🔴", float64(delay)/1000)
	}
}

func nodeCursor(selected bool) string {
	if selected {
		return "▸"
	}
	return " "
}
