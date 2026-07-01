package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loveRyujin/mini-agent/internal/agent"
	"github.com/loveRyujin/mini-agent/internal/slash"
	"github.com/loveRyujin/mini-agent/internal/tools"
	"github.com/loveRyujin/mini-agent/internal/transcript"
)

const inputMinHeight = 3

const tuiInputHint = "输入消息，Enter 发送…"

type eventMsg struct {
	event agent.Event
}

type turnStartedMsg struct {
	ch <-chan agent.Event
}

type turnDoneMsg struct{}

type model struct {
	agent      *agent.Agent
	transcript *transcript.Transcript
	viewport   viewport.Model
	textarea   textarea.Model
	eventCh    <-chan agent.Event

	expanded        map[int]bool
	focusIdx        int
	transcriptFocus bool
	theme           transcript.Theme
	workspace       string

	approvalCommand string
	approvalReplyCh chan<- bool

	width, height  int
	turnInProgress bool
	followTail     bool
	ready          bool

	transcriptOriginY int
	transcriptOriginX int
	transcriptHitW    int
	transcriptHitH    int
	plainLines        []string
	selecting         bool
	selStart          textPos
	selEnd            textPos
	copyNotice        string
}

func newModel(a *agent.Agent) *model {
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

	return &model{
		agent:      a,
		transcript: transcript.New(),
		textarea:   ta,
		viewport:   vp,
		expanded:   make(map[int]bool),
		focusIdx:   -1,
		theme:      transcript.DefaultTheme,
		workspace:  tools.WorkspaceDisplay(),
		followTail: true,
		selStart:   textPos{line: -1, col: -1},
		selEnd:     textPos{line: -1, col: -1},
	}
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.clearSelection()
		m.copyNotice = ""
		if msg.event.Kind == agent.EventApprovalRequired {
			m.approvalCommand = msg.event.Command
			m.approvalReplyCh = msg.event.ApprovalReplyCh
			m.textarea.Blur()
		}
		m.syncViewport()
		return m, m.waitEvent()

	case turnDoneMsg:
		m.turnInProgress = false
		m.eventCh = nil
		m.syncViewport()
		return m, nil

	case tea.KeyMsg:
		if m.approvalReplyCh != nil {
			m.handleApprovalKeys(msg)
			return m, nil
		}
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
			if m.hasSelection() {
				m.copySelection()
				m.clearSelection()
				m.syncViewport()
				return m, nil
			}
			if m.transcriptFocus {
				m.transcriptFocus = false
				m.textarea.Focus()
				m.syncViewport()
				return m, nil
			}
			return m, tea.Quit
		case tea.KeyCtrlO:
			m.toggleExpandKind(transcript.EntryReasoning)
			m.syncViewport()
			return m, nil
		case tea.KeyCtrlG:
			m.toggleExpandKind(transcript.EntryToolCall)
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
			switch result, arg := slash.Parse(text); result {
			case slash.Quit:
				return m, tea.Quit
			case slash.Clear:
				m.agent.ClearSession()
				m.transcript.Reset()
				m.expanded = make(map[int]bool)
				m.focusIdx = -1
				m.transcriptFocus = false
				m.textarea.Focus()
				m.syncViewport()
				return m, nil
			case slash.Help:
				m.transcript.AddSystemMessage(slash.HelpText())
				m.syncViewport()
				return m, nil
			case slash.Unknown:
				msgText := "未知命令。"
				if arg != "" {
					msgText = fmt.Sprintf("未知命令 /%s。", arg)
				}
				m.transcript.AddSystemMessage(msgText + " 输入 /help 查看可用命令。")
				m.syncViewport()
				return m, nil
			default:
				m.transcript.AddUserMessage(text)
				m.syncViewport()
				return m, startTurn(m.agent, text)
			}
		case tea.KeyPgUp, tea.KeyPgDown:
			m.scrollViewport(msg)
			return m, nil
		}

		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		m.layout()
		return m, cmd

	case tea.MouseMsg:
		return m.handleMouse(msg)
	}

	return m, nil
}

func (m *model) View() string {
	if !m.ready {
		return lipgloss.NewStyle().Render("初始化...")
	}
	panel := renderPanel(m)
	if m.approvalReplyCh != nil {
		return overlayModal(renderApprovalModal(m), m.width, m.height)
	}
	return panel
}

func renderPanel(m *model) string {
	t := m.theme
	titleStyle := lipgloss.NewStyle().Background(t.Gold).Foreground(lipgloss.Color("16")).Bold(true).Padding(0, 1)
	panelStyle := lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(t.Border).Padding(0, 1)
	footerStyle := lipgloss.NewStyle().Foreground(t.Dim).Padding(0, 1)

	status := lipgloss.NewStyle().Foreground(t.User).Bold(true).Render("● 就绪")
	if m.approvalReplyCh != nil {
		status = lipgloss.NewStyle().Foreground(t.Gold).Bold(true).Render("● 等待批准")
	} else if m.turnInProgress {
		status = lipgloss.NewStyle().Foreground(t.Tool).Bold(true).Render("● 生成中")
	}
	title := titleStyle.Width(m.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Top,
			" mini-agent ",
			lipgloss.PlaceHorizontal(m.width-lipgloss.Width(" mini-agent ")-lipgloss.Width(status), lipgloss.Right, status),
		),
	)
	subtitle := lipgloss.NewStyle().Foreground(t.Dim).Width(m.width).Padding(0, 1).
		Render(m.agent.Model + "  ·  " + m.workspace + "  ·  " + t.Name)
	transcriptPanel := panelStyle.Width(m.width - 2).Render(m.viewport.View())
	inputPanel := panelStyle.Width(m.width - 2).Render(renderInput(m))
	footer := footerStyle.Width(m.width).Render(m.footerText())
	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, transcriptPanel, inputPanel, footer)
}

func (m *model) footerText() string {
	if m.approvalReplyCh != nil {
		return "Y 允许执行  ·  N 拒绝"
	}
	if m.copyNotice != "" {
		return m.copyNotice + "  ·  鼠标拖拽选中复制  ·  PgUp/PgDn 滚动  ·  Enter 发送"
	}
	return "鼠标拖拽选中复制  ·  PgUp/PgDn 滚动  ·  Ctrl+T 对话区  ·  Enter 发送"
}

func renderInput(m *model) string {
	t := m.theme
	if strings.TrimSpace(m.textarea.Value()) == "" {
		hint := lipgloss.NewStyle().Foreground(t.Dim).Italic(true).Render(tuiInputHint)
		return lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Foreground(t.Tool).Render("> "), hint)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Foreground(t.Tool).Render("> "), m.textarea.View())
}

func (m *model) inputHeight() int {
	if strings.Contains(m.textarea.Value(), "\n") {
		return inputMinHeight
	}
	return 1
}

func (m *model) layout() {
	inputH := m.inputHeight()
	m.textarea.SetHeight(inputH)
	m.textarea.SetWidth(m.width - 6)
	m.viewport.Width = m.width - 4

	const chrome = 10
	vpH := m.height - chrome - inputH
	if vpH < 3 {
		vpH = 3
	}
	m.viewport.Height = vpH
	m.updateTranscriptHitbox()
	m.syncViewport()
}

func (m *model) syncViewport() {
	prevYOffset := m.viewport.YOffset
	wasAtBottom := m.viewport.AtBottom()

	content := m.transcript.Render(transcript.RenderOpts{
		Theme:           m.theme,
		Expanded:        m.expanded,
		FocusIdx:        m.focusIdx,
		TranscriptFocus: m.transcriptFocus,
	})
	m.plainLines = splitPlainLines(content)
	if m.selActive() {
		content = highlightSelection(content, m.plainLines, m.selStart, m.selEnd)
	}

	m.viewport.SetContent(content)

	if m.followTail || wasAtBottom {
		m.viewport.GotoBottom()
		m.followTail = true
		return
	}

	maxY := max(0, m.viewport.TotalLineCount()-m.viewport.Height)
	if prevYOffset > maxY {
		prevYOffset = maxY
	}
	m.viewport.SetYOffset(prevYOffset)
}

func (m *model) scrollViewport(msg tea.KeyMsg) {
	m.viewport, _ = m.viewport.Update(msg)
	m.followTail = m.viewport.AtBottom()
}

func (m *model) waitEvent() tea.Cmd {
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

func startTurn(a *agent.Agent, userText string) tea.Cmd {
	ch := make(chan agent.Event)
	go func() {
		emit := func(e agent.Event) { ch <- e }
		_ = a.RunTurn(context.Background(), userText, emit)
		close(ch)
	}()
	return func() tea.Msg {
		return turnStartedMsg{ch: ch}
	}
}

func Run(a *agent.Agent) error {
	p := tea.NewProgram(newModel(a), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
