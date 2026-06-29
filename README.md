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

```sh
go run .
```

可选环境变量：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `LLM_API_URL` | OpenAI 兼容 API 地址 | `http://localhost:11434/v1/chat/completions` |
| `LLM_API_KEY` | API 密钥（本地 Ollama 通常可留空） | — |
| `LLM_MODEL` | 模型名称 | `gpt-oss:latest` |

## 文档

- 领域术语：[`CONTEXT.md`](CONTEXT.md)
- 架构决策：[`docs/adr/`](docs/adr/)
- Agent 工作流：[`docs/agents/`](docs/agents/)
