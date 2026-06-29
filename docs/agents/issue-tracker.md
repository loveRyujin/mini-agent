# Issue 跟踪：GitHub

本仓库的 Issue 与 PRD 以 GitHub Issue 形式存在。所有操作使用 `gh` CLI。

## 约定

- **创建 Issue**：`gh issue create --title "..." --body "..."`。多行正文请用 heredoc。
- **查看 Issue**：`gh issue view <number> --comments`，可用 `jq` 过滤评论并获取标签。
- **列出 Issue**：`gh issue list --state open --json number,title,body,labels,comments --jq '[.[] | {number, title, body, labels: [.labels[].name], comments: [.comments[].body]}]'`，配合 `--label`、`--state` 过滤。
- **评论 Issue**：`gh issue comment <number> --body "..."`
- **添加 / 移除标签**：`gh issue edit <number> --add-label "..."` / `--remove-label "..."`
- **关闭**：`gh issue close <number> --comment "..."`

在 clone 目录内运行 `gh` 时，仓库可从 `git remote -v` 自动推断。

## Pull Request 作为分流入口

**PR 作为需求入口：否。** _（若本仓库将外部 PR 视为功能请求，改为 `yes`；`/triage` 会读取此标志。）_

设为 `yes` 时，PR 与 Issue 使用相同标签与状态，对应 `gh pr` 命令：

- **查看 PR**：`gh pr view <number> --comments`，`gh pr diff <number>` 查看 diff。
- **列出待分流的外部 PR**：`gh pr list --state open --json number,title,body,labels,author,authorAssociation,comments`，仅保留 `authorAssociation` 为 `CONTRIBUTOR`、`FIRST_TIME_CONTRIBUTOR` 或 `NONE` 的项（排除 `OWNER`/`MEMBER`/`COLLABORATOR`）。
- **评论 / 打标签 / 关闭**：`gh pr comment`、`gh pr edit --add-label`/`--remove-label`、`gh pr close`。

GitHub 的 Issue 与 PR 共用编号空间，单独的 `#42` 可能是其一——先用 `gh pr view 42`，失败再用 `gh issue view 42`。

## 当技能要求「发布到 issue tracker」

创建 GitHub Issue。**正文使用简体中文。**

## 当技能要求「获取相关 ticket」

运行 `gh issue view <number> --comments`。
