# TUI 层采用 Bubble Tea

Mini-agent 正从 ANSI 打印的 CLI 输出演进为全屏终端 UI：可滚动的 Transcript、流式更新、模态 Approval Gate，以及常驻输入区。我们选用 Charm Bracelet（`bubbletea`、`lipgloss`、`bubbles`）作为 TUI 技术栈。

**曾考虑的方案：** 原生 ANSI 控制序列；`tview`；Charm Bracelet。

Charm Bracelet 胜出，因为其 Elm 风格的 model/update/view 循环能自然映射流式 Agent 事件（Reasoning 增量、Tool 调用、审批模态），无需与终端机制对抗。它是 Go 生态的事实标准，示例和社区资源丰富。原生 ANSI 需要自行实现布局、滚动和模态焦点管理。`tview` 更偏向表单/控件类 UI，与 Turn 过程中持续更新的聊天 Transcript 不够契合。

**后果：** Agent 核心逻辑必须与 Bubble Tea model 解耦，以便推理循环和 Tool 在无终端环境下仍可测试。日后更换 TUI 框架成本较高——可接受，因为继续用 ANSI 也会在无形中积累同等程度的锁定。
