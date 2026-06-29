# Mini-Agent

运行在开发者本机终端上的 Coding Agent，在本地代码库中读取、写入并执行操作。

## 术语

**Coding Agent**：
以帮开发者在本地代码库中修改和理解代码为主要职责的自主助手。
_避免使用_：chatbot、general assistant、conversational AI（通用对话 AI）

**Workspace（工作区）**：
Agent 在一次 Session 内可检视和操作的本地目录树，固定为进程启动时的工作目录。
_避免使用_：project root、cwd、working folder

**Approval Gate（审批门）**：
在 Tool 动作执行前，开发者必须明确允许的暂停点，以模态浮层形式覆盖在 Transcript 之上。
_避免使用_：confirmation prompt、permission dialog、inline y/n

**Shell Execution（Shell 执行）**：
在工作区内或针对工作区运行操作系统命令；始终须经过 Approval Gate。
_避免使用_：bash、terminal command、exec

**File Mutation（文件变更）**：
在工作区内创建、修改或删除文件；无需 Approval Gate 即可执行。
_避免使用_：write、edit、patch

**Session（会话）**：
一次应用启动过程中，开发者与 Agent 之间的内存对话；进程退出即丢弃。
_避免使用_：chat history、conversation log、thread

**Inference Backend（推理后端）**：
Agent 在一次 Session 内将推理委托给的语言模型服务。
_避免使用_：LLM provider、API、Ollama

**Transcript（对话记录）**：
Session 内所有事件按时间顺序排列、可滚动的单列记录——包括开发者消息、Agent 回复、Tool 调用与审批提示。
_避免使用_：chat log、message list、history pane

**Turn（轮次）**：
从开发者发出一条消息开始，到 Agent 完成回复（含其中所有 Tool 循环）结束的一个周期。
_避免使用_：round、exchange、iteration

**Tool（工具）**：
Agent 可调用的、用于检视或操作 Workspace 的具名能力。
_避免使用_：function、plugin、skill

**Built-in Tool（内置工具）**：
随应用提供、无需额外配置即可使用的 Tool。
_避免使用_：native tool、default tool、core tool

**Workspace Search（工作区搜索）**：
按模式在 Workspace 内查找文件或文本。
_避免使用_：grep、ripgrep、find

**Slash Command（斜杠命令）**：
在 Turn 循环之外、由开发者在输入框键入 `/` 加命令名触发的操作。
_避免使用_：CLI command、meta command、shortcut

**System Prompt（系统提示词）**：
定义 Agent 在整个 Session 内行为的常驻指令；提供 Coding Agent 默认值，并可在启动时覆盖。
_避免使用_：system message、persona、instructions

**Reasoning（推理过程）**：
模型在回复之前或同时输出的思维链文本；当 Inference Backend 提供时，内联显示在 Transcript 中。
_避免使用_：thinking、internal monologue、scratchpad
