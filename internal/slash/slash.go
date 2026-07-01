package slash

import "strings"

type Result int

const (
	None Result = iota
	Quit
	Clear
	Help
	Unknown
)

func Parse(text string) (Result, string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return None, ""
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return Unknown, ""
	}
	cmd := strings.ToLower(strings.TrimPrefix(fields[0], "/"))
	switch cmd {
	case "quit":
		return Quit, ""
	case "clear":
		return Clear, ""
	case "help":
		return Help, ""
	default:
		return Unknown, cmd
	}
}

func HelpText() string {
	return strings.TrimSpace(`
Slash Command：
  /quit   退出 TUI
  /clear  清空 Session 与 Transcript
  /help   显示此帮助

Transcript 快捷键：
  鼠标拖拽     选中文本，松开后自动复制
  PgUp/PgDn    滚动对话记录
  Ctrl+T       进入对话区（j/k 选块，Ctrl+Y 复制块）
  G            跳至最新
  Y  允许执行 Shell 命令
  N  拒绝执行

配置（启动前设置环境变量）：
  Inference Backend
    LLM_API_URL   OpenAI 兼容 API 地址
    LLM_API_KEY   API 密钥
    LLM_MODEL     模型名称

  System Prompt
    MINI_AGENT_SYSTEM_PROMPT       直接覆盖系统提示词
    MINI_AGENT_SYSTEM_PROMPT_FILE  从文件读取系统提示词（优先于上者）
`)
}
