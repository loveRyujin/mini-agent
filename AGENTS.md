## Agent 技能

### Issue 跟踪

使用 GitHub Issues（`gh` CLI）；外部 PR 不作为分流入口。详见 `docs/agents/issue-tracker.md`。

### 分流标签

默认五标签映射（`needs-triage` / `needs-info` / `ready-for-agent` / `ready-for-human` / `wontfix`）。详见 `docs/agents/triage-labels.md`。

### 领域文档

单上下文仓库：根目录 `CONTEXT.md` + `docs/adr/`。详见 `docs/agents/domain.md`。

### 文档语言

本仓库**所有面向人类的文档**（`CONTEXT.md`、ADR、README、Issue/PRD 正文等）使用**简体中文**编写。代码标识符、环境变量名、GitHub 标签名等保持英文。
