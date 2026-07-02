# Approval Gate 作为 TUI 侧 seam

Shell 等 Gated Tool 在执行前需要用户批准。审批的同步机制（阻塞等待 Y/N）属于 TUI 关注点，不应通过 `Event.ApprovalReplyCh` 泄漏到 Agent 事件协议。

## 决策

- **`ApprovalGate` 位于 `internal/tui/approval_gate.go`**，实现 `agent.ApprovalGate` 接口。
- **同步**：gate 内 `replyCh`；`RequestApproval` 阻塞；TUI model 收到 Y/N 后调用 `gate.Respond()`。
- **组装**：`main` 创建 gate，注入 `a.ApprovalGate`，并传入 `tui.Run(a, gate)`。
- **Event 协议**：保留 `EventApprovalRequired` + `Command`（Transcript 仍显示 `EntryApproval` 行）；删除 `ApprovalReplyCh`。
- **`ApprovalGate == nil`**：Agent 明确报错，无 channel fallback。
- **Turn 生命周期**：每 Turn 使用 `context.WithCancel`；`/quit` / Ctrl+C 先 `cancelTurn()` 再退出（fire-and-forget，不等 goroutine 结束）。

## 后果

- Agent 核心与 TUI 审批 UI 通过窄接口解耦，单元测试可用 `NewStaticApprovalGate`。
- 生产路径仅走 `ApprovalGate`；channel 不再出现在 Event 或 Agent fallback 中。
- 退出时取消 in-flight Turn，审批等待中的 Shell 不会继续执行。
