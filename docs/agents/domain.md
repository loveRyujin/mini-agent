# 领域文档

工程技能在探索代码库时，应如何消费本仓库的领域文档。

## 文档语言

本仓库面向人类的文档统一使用**简体中文**。Issue、PRD、ADR、`CONTEXT.md` 等正文均用中文撰写；代码中的标识符、GitHub 标签名、环境变量名保持英文。

## 探索前先读这些

- 仓库根目录的 **`CONTEXT.md`**，或
- 若存在 **`CONTEXT-MAP.md`**——它指向各上下文的 `CONTEXT.md`，请阅读与当前主题相关的每一份。
- **`docs/adr/`**——阅读与你即将改动区域相关的 ADR。多上下文仓库中，还需查看 `src/<context>/docs/adr/` 下的上下文级决策。

若上述文件不存在，**静默继续**。不要主动指出缺失，也不要建议预先创建。`/domain-modeling` 技能（经 `/grill-with-docs` 与 `/improve-codebase-architecture` 触发）会在术语或决策真正落定后惰性创建它们。

## 文件结构

单上下文仓库（大多数情况）：

```
/
├── CONTEXT.md
├── docs/adr/
│   ├── 0001-event-sourced-orders.md
│   └── 0002-postgres-for-write-model.md
└── src/
```

多上下文仓库（根目录存在 `CONTEXT-MAP.md`）：

```
/
├── CONTEXT-MAP.md
├── docs/adr/                          ← 系统级决策
└── src/
    ├── ordering/
    │   ├── CONTEXT.md
    │   └── docs/adr/                  ← 上下文级决策
    └── billing/
        ├── CONTEXT.md
        └── docs/adr/
```

## 使用术语表词汇

当你的输出涉及领域概念（Issue 标题、重构提案、假设、测试名等）时，使用 `CONTEXT.md` 中定义的术语。不要改用术语表 `_避免使用_` 中列出的同义词。

若所需概念尚不在术语表中——要么你在发明项目未采用的语言（请重新斟酌），要么确实存在缺口（记给 `/domain-modeling` 处理）。

## 标注 ADR 冲突

若你的输出与现有 ADR 矛盾，应明确指出，而非静默覆盖：

> _与 ADR-0007（事件溯源订单）矛盾——但值得重新讨论，因为……_
