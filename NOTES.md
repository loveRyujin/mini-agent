# Week 1 学习笔记：五种 Agent 工作流 × mini-agent

W1 收口的是学习对照，不是再写一遍 agent。流式 chat、Built-in Tools、tool loop 已经在代码里：`agent.RunTurn` 调 `inference.Client.CallLLMStream`，用 `tools.Builtin` 在同一条 Session 历史里循环，直到模型不再发起 tool call。本笔记把 Anthropic《Building Effective Agents》里的五种 **workflow** 对照到这些真实符号上。

文章把两类系统分开：

- **workflow**：开发者预先排好 LLM 调用与代码门闩的路径。
- **agent**：模型在循环里自己决定下一步（用哪些工具、要不要再问一轮）。

本仓现在是后者的最小形态：一个 `agent.Agent`、一份 System Prompt、一个模型、一份内存 `History`。下面五种命名工作流都还没有独立编排。

## 五种模式

状态以当前树为准。缺的一行只写「若要做会落在哪」，本 PR 不实现。

| 模式 | 状态 | 本仓对照 |
|------|------|----------|
| Prompt Chaining | 缺 | 没有「步骤 A 的提示词 → 程序检查 → 步骤 B 的提示词」这种固定链。全 Session 只有 `prompt.Default` / `prompt.Resolve` 产出的一份系统提示，之后都在同一个 `RunTurn` 里打同一个模型。若要补：把「读需求 / 列计划 / 改代码」拆成多次独立的 `CallLLMStream`，中间用代码门闩决定是否进入下一步，而不是交给同一条 tool loop 自由发挥。 |
| Routing | 缺 | 没有分类器把输入分到不同提示词、模型或子流程。`tool_choice: "auto"`（`RunTurn` 里的请求体）只是让**同一个**模型在同一组工具里自选。`slash.Parse` 在 Turn 之外分发 `/quit`、`/clear`、`/help`，这是开发者命令，不是 LLM 路由。若要补：在 `startTurnCmd` 之前加一层分类（小模型或规则），再把消息送进不同的 system prompt 或 `Backend`。 |
| Parallelization | 缺 | 没有 sectioning，也没有 voting。一轮里模型即使返回多个 tool call，`(*Agent).toolCall` 也是顺序 `for` 循环逐个 `Call`。每轮只有一次 `CallLLMStream`。TUI 里 `startTurnCmd` 的 goroutine 只跑**当前这一条** Turn。若要补：对互不依赖的子任务并发调用 `CallLLMStream` 或 `Tool.Call`，再用代码汇总；或对同一任务多次采样后投票。 |
| Orchestrator-Workers | 缺 | 没有编排者动态拆任务、派给 worker LLM、再综合。`tools.Builtin` 是函数（`read_file`、`list_file`、`write_file`、`workspace_search`、`run_shell`），不是带自己 History 的子 Agent。若要补：编排者模型产出子任务，每个子任务一个独立 `Agent`（或至少独立 history / model），最后由编排者合成。 |
| Evaluator-Optimizer | 缺 | 没有「生成模型 + 评判模型」的反馈环。现有停止条件都是机械的：`maxToolRoundsPerTurn`（25）、`inference.ToolSignature` 发现连续两轮工具调用完全相同就停、`finish_reason == "tool_calls"` 但解析不出合法调用就报错。若要补：草稿生成之后再发一次评判用的 `CallLLMStream`，不通过则把评语写回 `History` 再生成，直到代码侧接受或到达轮次上限。 |

## 相关概念（文章里的积木，不是上面五种 workflow）

| 概念 | 状态 | 本仓对照 |
|------|------|----------|
| Tool-use / augmented LLM | 有。这是 W1 的代码 KPI | 核心循环是 `(*Agent).RunTurn`（`internal/agent/agent.go`）。用户消息追加进 `History` 后，请求带 `stream: true`、`tools`（`ToolDefinitions`）、`tool_choice: "auto"`。流式增量交给 `inference.ToolCallAccumulator`；`ValidToolCalls` 丢掉缺 id / 名字 / 参数的调用；有调用则 `AssistantToolCallsMessage` 写回历史，`toolCall` 按名字分发，工具结果再写回，然后进入下一轮。没有调用则把本轮拼接的正文记成 assistant 消息，发出 `EventTurnComplete`。上限与去重见上表。推理侧是自研 OpenAI 兼容 SSE：`inference.Backend`、`(*Client).CallLLMStream`（`internal/inference/client.go`）。工具注册表是 `agent.Tools`，启动时由 `tools.Builtin` 装入五个 Built-in Tool（`internal/tools/tool.go`）。路径约束在 `workspace.Workspace.Resolve`。Session 记忆只是进程内 `History`（`ClearSession` 清空；进程退出即丢，见 `CONTEXT.md`）。没有在模型调用前自动检索的 RAG 层；`workspace_search` 仍是模型自己决定要不要调的工具。TUI 只消费事件：`EventAnswerDelta`、`EventReasoningDelta`、`EventToolCall`、`EventToolResult`（`internal/tui/tui.go` 的 `startTurnCmd`）。 |
| HITL / Approval Gate | 有，而且超前于 W1 | Shell 执行前必须过人。`tools.GatedTool` 在 `Tool` 上多了 `ApprovalSummary`；目前只有 `RunShell`（`run_shell`）实现它。`(*Agent).toolCall` 遇到 `GatedTool` 就走 `requestApproval`：`ApprovalGate == nil` 直接报错，拒绝则 `tools.FailResp` + `tools.ErrShellDenied`，允许才 `Call`。接口在 `agent.ApprovalGate` / `ApprovalRequest`；测试用 `agent.NewStaticApprovalGate`。生产实现是 `tui.ApprovalGate`：`RequestApproval` 阻塞，`Respond` 由 `handleApprovalKeys` 的 Y/N 唤醒，模态在 `renderApprovalModal`。组装在 `cmd/mini-agent/main.go`（`tui.NewApprovalGate` 赋给 `a.ApprovalGate`）。决策记录：`docs/adr/0002-approval-gate-seam.md`。文件变更（`write_file`）不过门，与 `CONTEXT.md` 里 File Mutation 的定义一致。这是人对工具的暂停，不是 Evaluator-Optimizer。按学习计划它属于更后面的能力，W1 笔记只记录它已经在，不往下扩。 |

## 诚实缺口

对照下来，W1 缺的是编排，不是循环：

- **Routing**：模型不会被分到专门的提示词或后端；斜杠命令不要算成这个模式。
- **Parallelization**：工具与 LLM 调用都是串行的；多 tool call 同轮返回也不会并行执行。
- **Orchestrator-Workers**：只有一个 Agent。内置工具是函数调用，不是 worker 模型。
- **Evaluator-Optimizer**：没有第二套评判提示。轮次上限和 `ToolSignature` 只防死循环。
- **Prompt Chaining** 同样没有。多轮 tool loop 看起来像「一步接一步」，但每一步的提示词相同、下一步由模型临时决定，文章把这算 agent，不算 chaining。

已有、且不该在本笔记里重做的部分：流式 tool loop，以及超前的 Approval Gate。

## 下一周（W2，本 PR 不做）

把现有 Built-in Tools 用官方 [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk)（模块 `github.com/modelcontextprotocol/go-sdk`）暴露成 MCP Server，让 Cursor 能调 `read_file` / `list_file` / `write_file` / `workspace_search` / `run_shell`。`RunShell` 的 Approval Gate 语义要保留在服务器侧，而不是在适配层里悄悄放开。不在本 PR 加依赖或改 `tools` 包。

## 来源

- [Building Effective Agents](https://www.anthropic.com/research/building-effective-agents)（Erik Schluntz、Barry Zhang）
- Cookbook：[patterns/agents](https://github.com/anthropics/anthropic-cookbook/tree/main/patterns/agents)。五种 workflow 的示例在 `basic_workflows.ipynb`（Prompt Chaining、Routing、Parallelization）、`orchestrator_workers.ipynb`、`evaluator_optimizer.ipynb`。
