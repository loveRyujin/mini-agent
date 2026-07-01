# mini-agent

从零用 Go 构建 AI Agent。

## 环境准备

使用 Ollama 运行 LLM。

- 第一步，下载模型：

```sh
ollama pull <your_model_name>
```

- 第二步，查看已安装模型：

```sh
ollama list
```

- 第三步，运行模型：

```sh
ollama run <your_model_name>
```

更多说明：https://docs.ollama.com/cli

## 运行 Agent

在目标项目目录下启动（该目录即为 **Workspace**）：

```sh
go run ./cmd/mini-agent
```

在 TUI 中输入 `/help` 可查看 Slash Command 与配置说明。

### Inference Backend

任意 OpenAI 兼容 API（Ollama、云端等）均可通过环境变量配置：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `LLM_API_URL` | OpenAI 兼容 API 地址 | `http://localhost:11434/v1/chat/completions` |
| `LLM_API_KEY` | API 密钥（本地 Ollama 通常可留空） | — |
| `LLM_MODEL` | 模型名称 | `deepseek-r1:latest` |

### System Prompt

未配置时使用内置 Coding Agent 默认提示词（包含当前 Workspace 路径与可用 Built-in Tool 说明）。

| 变量 | 说明 |
|------|------|
| `MINI_AGENT_SYSTEM_PROMPT` | 直接覆盖系统提示词 |
| `MINI_AGENT_SYSTEM_PROMPT_FILE` | 从文件读取系统提示词（优先于 `MINI_AGENT_SYSTEM_PROMPT`） |

示例：

```sh
export MINI_AGENT_SYSTEM_PROMPT_FILE="$PWD/.mini-agent-prompt.txt"
go run ./cmd/mini-agent
```

## 文档

- 领域术语：[`CONTEXT.md`](CONTEXT.md)
- 架构决策：[`docs/adr/`](docs/adr/)
- Agent 工作流：[`docs/agents/`](docs/agents/)
