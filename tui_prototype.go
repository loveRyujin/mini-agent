package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const protoBanner = "PROTOTYPE — Crush 风格对话区（默认截断 10 行，非折叠隐藏）"

type protoModel struct {
	viewport       viewport.Model
	textarea       textarea.Model
	entries        []protoEntry
	expanded       map[int]bool
	focusIdx       int
	transcriptFocus bool
	variantIdx     int
	themeIdx       int
	transcriptMode int
	width          int
	height         int
	ready          bool
	turnActive     bool
}

func newProtoModel() protoModel {
	ta := textarea.New()
	ta.Placeholder = ""
	ta.Focus()
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.SetWidth(80)
	ta.SetHeight(1)

	focused, blurred := textarea.DefaultStyles()
	focused.Base = lipgloss.NewStyle()
	focused.CursorLine = lipgloss.NewStyle()
	focused.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	focused.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	blurred.Base = lipgloss.NewStyle()
	blurred.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	blurred.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	ta.FocusedStyle = focused
	ta.BlurredStyle = blurred

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().Padding(0, 1)

	entries := prototypeTranscript()
	m := protoModel{
		textarea:       ta,
		viewport:       vp,
		entries:        entries,
		expanded:       initProtoExpanded(entries),
		focusIdx:        -1,
		transcriptFocus: false,
		variantIdx:     1,
		themeIdx:       0,
		transcriptMode: 0,
	}
	m.applyInputChrome()
	return m
}

func (m protoModel) renderCtx() protoRenderCtx {
	return protoRenderCtx{
		width: m.width, height: m.height,
		viewport: m.viewport, textarea: m.textarea,
		entries: m.entries, turnActive: m.turnActive,
		theme: protoThemes[m.themeIdx], transcriptMode: m.transcriptMode,
		expanded: m.expanded, focusIdx: m.focusIdx, transcriptFocus: m.transcriptFocus,
	}
}

func (m protoModel) Init() tea.Cmd { return nil }

func (m protoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.layout()
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
			return m, tea.Quit
		case tea.KeyCtrlO:
			m.toggleExpandKind(protoReasoning)
			m.syncViewport()
			return m, nil
		case tea.KeyCtrlG:
			m.toggleExpandKind(protoToolResult)
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
		case tea.KeyF7:
			m.transcriptMode = (m.transcriptMode - 1 + len(transcriptModes)) % len(transcriptModes)
			m.syncViewport()
			return m, nil
		case tea.KeyF8:
			m.transcriptMode = (m.transcriptMode + 1) % len(transcriptModes)
			m.syncViewport()
			return m, nil
		case tea.KeyF9:
			m.variantIdx = (m.variantIdx - 1 + len(protoVariants)) % len(protoVariants)
			m.layout()
			return m, nil
		case tea.KeyF10:
			m.variantIdx = (m.variantIdx + 1) % len(protoVariants)
			m.layout()
			return m, nil
		case tea.KeyF11:
			m.themeIdx = (m.themeIdx - 1 + len(protoThemes)) % len(protoThemes)
			m.syncViewport()
			return m, nil
		case tea.KeyF12:
			m.themeIdx = (m.themeIdx + 1) % len(protoThemes)
			m.syncViewport()
			return m, nil
		case tea.KeyEnter:
			text := strings.TrimSpace(m.textarea.Value())
			if text != "" {
				m.entries = append(m.entries, protoEntry{kind: protoUser, text: text})
				m.textarea.SetValue("")
				m.entries = append(m.entries, protoEntry{kind: protoAnswer, text: "（原型占位回复）收到：" + text})
				m.syncViewport()
			}
			return m, nil
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

func (m protoModel) View() string {
	if !m.ready {
		return "初始化原型…"
	}
	banner := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("11")).Width(m.width).Render(protoBanner)
	return lipgloss.JoinVertical(lipgloss.Left, banner, renderProtoView(m.variantIdx, m.renderCtx()), renderProtoSwitcher(m))
}

func (m *protoModel) applyInputChrome() {
	switch m.variantIdx {
	case 2:
		m.textarea.Prompt = ""
	default:
		if strings.TrimSpace(m.textarea.Value()) == "" {
			m.textarea.Prompt = ""
		} else {
			m.textarea.Prompt = "> "
		}
	}
	m.textarea.Placeholder = ""
}

func (m *protoModel) inputHeight() int {
	if strings.Contains(m.textarea.Value(), "\n") {
		return inputMinHeight
	}
	return 1
}

func (m *protoModel) layout() {
	m.applyInputChrome()
	inputH := m.inputHeight()
	reserved := protoReservedLines(m.variantIdx, inputH)
	vpH := m.height - reserved
	if vpH < 3 {
		vpH = 3
	}

	m.textarea.SetHeight(inputH)
	inputW := m.width
	switch m.variantIdx {
	case 1:
		inputW = m.width - 6
		m.viewport.Width = m.width - 4
	case 2:
		inputW = m.width - 4
		m.viewport.Width = m.width
	default:
		m.viewport.Width = m.width
	}
	m.textarea.SetWidth(inputW)
	m.viewport.Height = vpH
	m.syncViewport()
}

func (m *protoModel) syncViewport() {
	m.viewport.SetContent(renderProtoTranscript(m.variantIdx, m.entries, protoThemes[m.themeIdx], m.transcriptMode, m.expanded, m.focusIdx, m.transcriptFocus))
	m.viewport.GotoBottom()
}

func renderProtoSwitcher(m protoModel) string {
	v := protoVariants[m.variantIdx]
	t := protoThemes[m.themeIdx]
	tm := transcriptModes[m.transcriptMode]
	mode := "输入"
	if m.transcriptFocus {
		mode = "对话区 j/k/e"
	}
	label := fmt.Sprintf(" %s │ F9/10:%s F7/8:%s F11/12:%s │ Ctrl+T 切换 ", mode, v.name, tm.name, t.name)
	return lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("230")).Bold(true).Width(m.width).
		Render(label)
}

func (m *protoModel) toggleTranscriptFocus() {
	m.transcriptFocus = !m.transcriptFocus
	if m.transcriptFocus {
		m.textarea.Blur()
		if m.focusIdx < 0 {
			m.focusIdx = firstFocusable(m.entries)
		}
	} else {
		m.textarea.Focus()
	}
}

func (m *protoModel) handleTranscriptKeys(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlT:
		m.transcriptFocus = false
		m.textarea.Focus()
		return true
	case tea.KeyEnter, tea.KeySpace:
		m.toggleFocusedExpand()
		return true
	case tea.KeyUp:
		m.moveFocus(-1)
		return true
	case tea.KeyDown:
		m.moveFocus(1)
		return true
	case tea.KeyPgUp, tea.KeyPgDown:
		m.viewport, _ = m.viewport.Update(msg)
		return true
	}
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		switch msg.Runes[0] {
		case 'j':
			m.moveFocus(1)
			return true
		case 'k':
			m.moveFocus(-1)
			return true
		case 'e', 'E':
			m.toggleFocusedExpand()
			return true
		}
	}
	return false
}

func runTUIPrototype() error {
	p := tea.NewProgram(newProtoModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
