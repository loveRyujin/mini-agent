# Bubble Tea for the TUI layer

Mini-agent is evolving from ANSI-printed CLI output into a full-screen terminal UI with a scrollable Transcript, streaming updates, modal Approval Gates, and a persistent input area. We adopt Charm Bracelet (`bubbletea`, `lipgloss`, `bubbles`) as the TUI stack.

**Considered options:** raw ANSI control sequences; `tview`; Charm Bracelet.

Charm Bracelet wins because its Elm-style model/update/view loop maps cleanly onto streaming agent events (reasoning deltas, tool calls, approval modals) without fighting the terminal. It is the de facto standard in Go, which keeps examples and future contributors aligned. Raw ANSI would reinvent layout, scrolling, and modal focus management. `tview` skews toward form/widget UIs and fits less naturally with a chat transcript that updates continuously during a Turn.

**Consequences:** agent core logic must stay decoupled from Bubble Tea models so the inference loop and tools remain testable without a terminal. Swapping TUI frameworks later would be a substantial rewrite—acceptable because the alternative (ANSI) would accumulate the same lock-in informally.
