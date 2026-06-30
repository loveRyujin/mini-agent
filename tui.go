package main

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const inputMinHeight = 3

const tuiInputHint = "输入消息，Enter 发送…"

type eventMsg struct {
	event Event
}

type turnStartedMsg struct {
	ch <-chan Event
}

type turnDoneMsg struct{}

type tuiModel struct {
	agent      *Agent
	transcript *Transcript
	viewport   viewport.Model
	textarea   textarea.Model
	eventCh    <-chan Event

	expanded        map[int]bool
	focusIdx        int
	transcriptFocus bool
	theme           tuiTheme
	workspace       string

	width, height  int
	turnInProgress bool
	ready          bool
}

func newTUIModel(agent *Agent) *tuiModel {
	ta := textarea.New()
	ta.Prompt = ""
	ta.Placeholder = ""
	ta.Focus()
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.SetWidth(80)
	ta.SetHeight(1)

	focused, blurred := textarea.DefaultStyles()
	focused.Base = lipgloss.NewStyle()
	focused.CursorLine = lipgloss.NewStyle()
	focused.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	blurred.Base = lipgloss.NewStyle()
	blurred.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	ta.FocusedStyle = focused
	ta.BlurredStyle = blurred

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().Padding(0, 1)

	return &tuiModel{
		agent:      agent,
		transcript: NewTranscript(),
		textarea:   ta,
		viewport:   vp,
		expanded:   make(map[int]bool),
		focusIdx:   -1,
		theme:      defaultTUITheme,
		workspace:  WorkspaceDisplay(),
	}
}

func (m *tuiModel) Init() tea.Cmd {
	return nil
}

func (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.layout()
		return m, nil

	case turnStartedMsg:
		m.turnInProgress = true
		m.eventCh = msg.ch
		m.syncViewport()
		return m, m.waitEvent()

	case eventMsg:
		m.transcript.Apply(msg.event)
		m.syncViewport()
		return m, m.waitEvent()

	case turnDoneMsg:
		m.turnInProgress = false
		m.eventCh = nil
		m.syncViewport()
		return m, nil

	case tea.KeyMsg:
		if m.transcriptFocus {
			if m.handleTranscriptKeys(msg) {
				m.syncViewport()
				return m, nil
			}
		}
		switch msg.Type {
		case tea.KeyCtrlT:
			m.toggleTranscriptFocus()
			m.syncViewport()
			return m, nil
		case tea.KeyCtrlC, tea.KeyEsc:
			if m.transcriptFocus {
				m.transcriptFocus = false
				m.textarea.Focus()
				m.syncViewport()
				return m, nil
			}
			return m, tea.Quit
		case tea.KeyCtrlO:
			m.toggleExpandKind(entryReasoning)
			m.syncViewport()
			return m, nil
		case tea.KeyCtrlG:
			m.toggleExpandKind(entryToolCall)
			m.syncViewport()
			return m, nil
		case tea.KeyF5:
			m.setAllExpanded(true)
			m.syncViewport()
			return m, nil
		case tea.KeyF6:
			m.setAllExpanded(false)
			m.syncViewport()
			return m, nil
		case tea.KeyEnter:
			if m.turnInProgress || m.transcriptFocus {
				return m, nil
			}
			text := strings.TrimSpace(m.textarea.Value())
			if text == "" {
				return m, nil
			}
			m.textarea.SetValue("")
			m.transcript.AddUserMessage(text)
			m.syncViewport()
			return m, startTurn(m.agent, text)
		case tea.KeyPgUp, tea.KeyPgDown:
			m.viewport, _ = m.viewport.Update(msg)
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	m.layout()
	return m, cmd
}

func (m *tuiModel) View() string {
	if !m.ready {
		return lipgloss.NewStyle().Render("初始化...")
	}
	return renderTUIPanel(m)
}

func renderTUIPanel(m *tuiModel) string {
	t := m.theme
	titleStyle := lipgloss.NewStyle().Background(t.gold).Foreground(lipgloss.Color("16")).Bold(true).Padding(0, 1)
	panelStyle := lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(t.border).Padding(0, 1)
	footerStyle := lipgloss.NewStyle().Foreground(t.dim).Padding(0, 1)

	status := lipgloss.NewStyle().Foreground(t.user).Bold(true).Render("● 就绪")
	if m.turnInProgress {
		status = lipgloss.NewStyle().Foreground(t.tool).Bold(true).Render("● 生成中")
	}
	title := titleStyle.Width(m.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Top,
			" mini-agent ",
			lipgloss.PlaceHorizontal(m.width-lipgloss.Width(" mini-agent ")-lipgloss.Width(status), lipgloss.Right, status),
		),
	)
	subtitle := lipgloss.NewStyle().Foreground(t.dim).Width(m.width).Padding(0, 1).
		Render(m.agent.Model + "  ·  " + m.workspace + "  ·  " + t.name)
	transcriptPanel := panelStyle.Width(m.width - 2).Render(m.viewport.View())
	inputPanel := panelStyle.Width(m.width - 2).Render(renderTUIInput(m))
	footer := footerStyle.Width(m.width).Render("Ctrl+T 对话区  ·  j/k 选块  ·  e/Space 展开  ·  Esc 回输入  ·  Enter 发送")
	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, transcriptPanel, inputPanel, footer)
}

func renderTUIInput(m *tuiModel) string {
	t := m.theme
	if strings.TrimSpace(m.textarea.Value()) == "" {
		hint := lipgloss.NewStyle().Foreground(t.dim).Italic(true).Render(tuiInputHint)
		return lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Foreground(t.tool).Render("> "), hint)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Foreground(t.tool).Render("> "), m.textarea.View())
}

func (m *tuiModel) inputHeight() int {
	if strings.Contains(m.textarea.Value(), "\n") {
		return inputMinHeight
	}
	return 1
}

func (m *tuiModel) layout() {
	inputH := m.inputHeight()
	m.textarea.SetHeight(inputH)
	inputW := m.width - 6
	m.textarea.SetWidth(inputW)
	m.viewport.Width = m.width - 4

	const chrome = 10 // 标题 + 副标题 + 双线框 + 底栏
	vpH := m.height - chrome - inputH
	if vpH < 3 {
		vpH = 3
	}
	m.viewport.Height = vpH
	m.syncViewport()
}

func (m *tuiModel) syncViewport() {
	m.viewport.SetContent(m.transcript.Render(TranscriptRenderOpts{
		Theme:           m.theme,
		Expanded:        m.expanded,
		FocusIdx:        m.focusIdx,
		TranscriptFocus: m.transcriptFocus,
	}))
	m.viewport.GotoBottom()
}

func (m *tuiModel) waitEvent() tea.Cmd {
	if m.eventCh == nil {
		return nil
	}
	ch := m.eventCh
	return func() tea.Msg {
		e, ok := <-ch
		if !ok {
			return turnDoneMsg{}
		}
		return eventMsg{event: e}
	}
}

func startTurn(agent *Agent, userText string) tea.Cmd {
	ch := make(chan Event)
	go func() {
		emit := func(e Event) { ch <- e }
		_ = agent.RunTurn(context.Background(), userText, emit)
		close(ch)
	}()
	return func() tea.Msg {
		return turnStartedMsg{ch: ch}
	}
}

func runTUI(agent *Agent) error {
	p := tea.NewProgram(newTUIModel(agent), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
