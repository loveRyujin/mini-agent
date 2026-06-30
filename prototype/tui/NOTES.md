# TUI 样式原型 — 结论

> **运行：** `go run . -tui-prototype`（主包内 `tui_prototype*.go`）
> **参考：** [charmbracelet/crush](https://github.com/charmbracelet/crush) `internal/ui/chat/`
>
> 本目录 `prototype/tui/` 为早期独立原型，**当前以主包原型为准**；fold 完成后可删 `tui_prototype*.go` 与本目录冗余文件。

## 已定

| 维度 | 决策 |
|------|------|
| 外壳 | **B「冒险岛面板」** — 金标题栏、棕双线框、无填充底色 |
| 对话默认 | **Crush 风格** — 非默认折叠隐藏，长内容截断 10 行 |
| 区分方式 | 左竖线 `▌`（用户/Agent）+ 工具头 `✓ Read path` + 行号代码块 |
| 配色默认 | **Maple**（F11/F12 可切 Mono / Ocean / Vivid） |
| 布局变体 | 仅保留 B 进正式版；A/C 作对比，fold 时不必带入 |

## 对话展示（F7/F8）

| 模式 | 说明 |
|------|------|
| **Crush**（默认） | 对齐 crush：截断 + 展开提示，工具 call/result **视觉合并为一块** |
| 混合 | `› 你` `✓` 行号块，合并逻辑同 Crush |
| 标签 | 中文标签 + 截断，合并逻辑同 Crush |

## 截断 / 展开

WSL/部分终端下 **F2、Ctrl+J/K 不可用**，故采用「对话区聚焦」模式：

1. **Ctrl+T** 切换到「对话区」模式（底栏显示 `对话区 j/k/e`）
2. **j / k** 或 **↑ / ↓** 在**可选块**间移动
3. **e** 或 **Space** 展开/截断当前块
4. **Esc** 或再按 **Ctrl+T** 回到输入框

### 选块规则（已验证）

可选块仅两类：

| 类型 | 焦点落点 | 展开行为 |
|------|----------|----------|
| 推理 | `Thought` 头行 | 默认显示**尾部** 10 行 + earlier-lines 提示 |
| 工具 | **`✓ Read path` 工具头行** | 展开/截断下方代码体（行号块） |

**不可选：** `protoToolResult` 单独条目 — 与 `protoToolCall` 合并渲染，展开状态记在 **call 的 index** 上。`j/k` 不会跳进代码体内部。

| 键 | 作用 |
|----|------|
| Ctrl+O | 切换全部推理块 |
| Ctrl+G | 切换全部工具块（代码体） |
| F5 / F6 | 全部展开 / 全部截断 |

常量：`protoTruncateLines = 10`（fold 时改名为 `responseContextHeight` 或同类命名）

## 输入框

- `bubbles/textarea` 对中文 Placeholder 有 UTF-8 bug → **禁用 `Placeholder`**，改在 `renderProtoInput()` 外侧渲染中文提示
- 单行时高度 1，含换行时 `inputMinHeight = 3`

## 结论

### 对话展示

- **Crush 模式**作为正式默认：截断非隐藏、工具头 + 行号代码体、推理尾部截断
- 工具 call/result 在数据层可仍分列，渲染与交互层**合并为一块**；聚焦、展开、全选工具均操作 call index
- 混合/标签模式可保留为次要样式或 fold 后删除，优先保证 Crush 路径

### 配色主题

| 主题 | 用途 |
|------|------|
| **Maple**（默认） | 冒险岛金 + 棕框，绿用户 / 青 Agent / 橙工具 |
| Mono | 灰阶，终端友好 |
| Ocean | 蓝绿系 |
| Vivid | 高饱和对比 |

fold 时抽出 `Theme` 结构体，支持配置或环境变量切换，默认 Maple。

### 外壳

- B 面板：标题栏状态（就绪/生成中）、副标题（模型 / 工作区 / 对话模式 / 主题）、双线框对话区与输入区、底栏快捷键提示
- 顶栏 switcher（F9–F12）为原型调试用途，正式版可收敛为设置或去掉

## fold 进 `tui.go` 时

- [ ] `Theme` 可配置配色（默认 Maple）
- [ ] Transcript item 化（user / assistant / tool call+result / reasoning / usage）
- [ ] 截断常量 `responseContextHeight = 10`
- [ ] 对话区聚焦：`Ctrl+T` + `j/k/e`，`isFocusable` 仅 reasoning + tool call
- [ ] 工具块合并渲染 + 展开状态挂 call index
- [ ] 中文输入提示外侧渲染（不用 textarea Placeholder）
- [ ] 正式环境 transcript 聚焦时 Space 展开，输入框失焦

## 实现文件（主包）

| 文件 | 职责 |
|------|------|
| `tui_prototype.go` | model、快捷键、对话区聚焦 |
| `tui_prototype_crush.go` | Crush 渲染、选块/展开逻辑 |
| `tui_prototype_theme.go` | 主题、混合/标签模式 |
| `tui_prototype_variants.go` | B 外壳 + A/C 变体 |
| `tui_prototype_mock.go` | Mock 对话数据 |
| `tui_prototype_focus_test.go` | 选块跳过 tool result 等测试 |
