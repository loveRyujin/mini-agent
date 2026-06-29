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

	width, height int
	turnInProgress bool
	ready          bool
}

func newTUIModel(agent *Agent) *tuiModel {
	ta := textarea.New()
	ta.Placeholder = "输入消息…"
	ta.Focus()
	ta.CharLimit = 0
	ta.SetWidth(80)
	ta.SetHeight(inputMinHeight)

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().Padding(0, 1)

	return &tuiModel{
		agent:      agent,
		transcript: NewTranscript(),
		textarea:   ta,
		viewport:   vp,
	}
}

func (m *tuiModel) Init() tea.Cmd {
	return tea.EnterAltScreen
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
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.turnInProgress {
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
	return m, cmd
}

func (m *tuiModel) View() string {
	if !m.ready {
		return "初始化…"
	}

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(strings.Repeat("─", m.width))

	status := ""
	if m.turnInProgress {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("  生成中…")
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		divider,
		m.textarea.View()+status,
	)
}

func (m *tuiModel) layout() {
	inputHeight := inputMinHeight
	viewportHeight := m.height - inputHeight - 2
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	m.textarea.SetWidth(m.width)
	m.textarea.SetHeight(inputHeight)
	m.viewport.Width = m.width
	m.viewport.Height = viewportHeight
	m.syncViewport()
}

func (m *tuiModel) syncViewport() {
	m.viewport.SetContent(m.transcript.Render())
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
