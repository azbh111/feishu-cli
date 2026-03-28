# GitHub PR 提交指南

本文档规范从本地 `dev` 分支向官方仓库 `main` 提交 PR 的完整流程。

## 仓库信息

| 项目 | 值 |
|------|-----|
| 官方仓库 | `https://github.com/riba2534/feishu-cli` |
| Fork 仓库 | `git@github.com:azbh111/feishu-cli.git`（remote 名：`fork`） |
| 官方仓库 remote | `origin` |

## 官方仓库规范

| 项目 | 规范 |
|------|------|
| Base 分支 | `main` |
| 分支命名 | `feat/xxx`、`fix/xxx`、`docs/xxx`（kebab-case） |
| 提交信息 | Conventional Commits：`feat(scope): 描述` / `fix: 描述` |
| PR 标题 | 同提交信息格式，中英文均可 |
| CI | 无自动化，需自行确保 `go test ./...` 和 `go vet ./...` 通过 |

## 分支策略

### 分支总览

| 分支 | 所在仓库 | 作用 | 上游 |
|------|---------|------|------|
| `main` | origin（官方） | 官方主分支，只读 | — |
| `master` | fork（个人） | 个人主分支，官方改动的中转站 | 跟踪 `origin/main` |
| `fdoc` | fork（个人） | fdoc 工具分支，包含所有改动 | 基于 `master` |
| `dev` | 本地 | 日常开发，包含未拆解的混合改动 | — |
| `fix/xxx`、`feat/xxx` | fork（个人） | feature 分支，每个变更一个 | 基于 `origin/main` |

### 分支关系图

```
origin/main（官方，只读）
 │
 ├─── master（个人主分支，fork 上）
 │     │
 │     └─── fdoc（fdoc 工具分支，fork 上）
 │
 └─── fix/xxx, feat/xxx（feature 分支）
       │
       └─→ 先合入 master 验证 ─→ 再人工 PR 到 origin/main

dev（本地日常开发）
 │
 └─→ 拆解为独立 feature 分支
```

### 改动流转规则

| 改动分类 | 流转路径 | 说明 |
|---------|---------|------|
| base / feishu-cli | `dev → feature 分支 → master → fdoc`，验证通过后人工 PR 到 `origin/main` | 官方需要的改动 |
| fdoc | `dev → fdoc`（直接合入） | 官方不需要的改动，跳过 master |

**官方改动流转详细步骤**：

1. 从 `dev` 拆解改动，创建 feature 分支（基于 `origin/main`）
2. feature 分支推送到 fork，合入 `master`
3. 在 `master` 上回归测试（`go test ./...` + `go vet ./...`）
4. 测试通过后，将 `master` 合入 `fdoc`，确保 fdoc 也包含最新改动
5. 人工从 fork 的 feature 分支向 `origin/main` 创建 PR

**fdoc 改动流转**：

1. 从 `dev` 提取 fdoc 相关改动
2. 直接合入 `fdoc` 分支（不经过 master）
3. 不向官方提交

### 分支同步规则

| 场景 | 操作 |
|------|------|
| 官方 main 有更新 | `git fetch origin` → master rebase origin/main → fdoc rebase master |
| feature 合入 master 后 | fdoc merge/rebase master |
| fdoc 独有改动 | 直接在 fdoc 上提交，不影响 master |

### 禁止提交给官方的内容

- `docs/` 目录下的本地文档（如设计文档、本指南、`docs/features/`）
- `CLAUDE.md` 中仅用于本地开发的章节（如「开发流程」TDD 5 步、`fdoc push/pull` 命令）
- `fdoc/` 目录（fdoc 二进制）
- `internal/diff/` 包（仅 fdoc 使用）
- `internal/frontmatter/` 包（仅 fdoc 使用）
- `test/input/` 目录下的 fdoc 测试数据
- 分类为 `fdoc` 的所有变更（见「变更拆解规范」）

## 变更拆解规范

### 拆解原则

dev 分支的改动在提交前，必须拆解为**独立可提交的变更单元**，记录在 `docs/features/` 目录下。每个变更一个文档。

### 文件命名

```
<seq>.<category>-<description>.md
```

| 字段 | 说明 |
|------|------|
| `seq` | 两位数序号（01, 02, ...），表示理论改动顺序。按 seq 从小到大依次应用即可还原到最新功能 |
| `category` | 变更分类，见下表 |
| `description` | kebab-case 简要描述 |

### 分类规则

| 分类 | 文件名前缀 | 提交官方 | 说明 |
|------|-----------|---------|------|
| `base` | `<seq>.base-xxx.md` | **是** | 基础库改动（`internal/converter`、`internal/client`、`internal/config` 等）。不涉及 CLI 命令层 |
| `feishu-cli` | `<seq>.feishu-cli-xxx.md` | **是** | CLI 命令层改动（`cmd/` 目录）。直接影响 `feishu-cli` 命令行为 |
| `fdoc` | `<seq>.fdoc-xxx.md` | **否** | fdoc 专用改动。不提交则不影响 feishu-cli 的使用和修复 |

### 判定标准：fdoc vs feishu-cli

一个改动属于哪个分类，取决于**谁依赖它**：

| 判定条件 | 分类 |
|---------|------|
| 仅被 `fdoc/` 目录下的代码引用 | fdoc |
| 被 `cmd/` 目录下的代码引用（feishu-cli 命令） | base 或 feishu-cli |
| 在 `internal/` 中但无人引用（仅测试覆盖） | 看设计意图，通常归 fdoc |

**关键原则**：fdoc 相关改动如果不提交，feishu-cli 的所有命令不受影响、所有测试照常通过。

### 混合 commit 处理

dev 分支的一个 commit 可能同时包含 base、feishu-cli、fdoc 的改动。拆解时需要：

1. 按文件级别区分：同一文件的不同 hunk 可能属于不同分类
2. 在文档中标注 cherry-pick 时需要的 hunk 筛选方式
3. 标注与其他 seq 的依赖关系

### 排序规则

1. `base` 排在最前（底层依赖）
2. `feishu-cli` 居中（依赖 base）
3. `fdoc` 排在最后（依赖 base + feishu-cli）
4. 同分类内按逻辑依赖排序（被依赖方在前）

### 文档模板

每个变更文档应包含：

```markdown
# <seq>. <标题>

- **分类**: base / feishu-cli / fdoc
- **提交官方**: 是 / 否
- **类型**: feat / fix / perf / refactor
- **PR 分支**: `fix/xxx` 或 `feat/xxx`（仅提交官方的需要）
- **PR 标题**: Conventional Commits 格式（仅提交官方的需要）

## 变更文件
| 文件 | 改动说明 |
|------|---------|

## 问题描述 / 说明
## 改动内容
## 依赖
## Cherry-pick 方式（仅提交官方的需要）
## 验证（仅提交官方的需要）
```

### 当前拆解总览与提交待办

> **使用说明**：按表格从上到下的顺序提交。每个 PR 合入后勾选 `[x]`。
> 有前置依赖的条目必须等依赖项合入后才能提交。

#### 提交给官方（base + feishu-cli）

| 状态 | seq | PR 分支 | 描述 | 前置依赖 | 备注 |
|------|-----|---------|------|---------|------|
| [ ] | 01 | `fix/config-test-isolation` | 修复 config 测试隔离 | 无 | 独立，可随时提交 |
| [ ] | 02 | `fix/quote-block-import-export` | 修复引用块导入导出 | 无 | 独立，可随时提交 |
| [ ] | 03+04 | `fix/list-nesting-and-numbering` | 修复 Todo 嵌套 + 有序列表编号和缩进 | 无 | **必须合并为一个 PR**（见下方说明） |
| [ ] | 05 | `fix/quotecontainer-empty-block` | QuoteContainer 空块清理 | 建议 02 先合入 | 软依赖：逻辑关联但代码独立 |
| [ ] | 06 | `feat/configurable-doc-url` | DocURL 可配置化 | 无 | 独立，可随时提交 |
| [ ] | 07 | `perf/table-batch-fill` | 表格批量填充优化 | 无 | 独立，可随时提交 |
| [ ] | 08 | `feat/mermaid-text-drawing` | Mermaid 改用 TextDrawing | 无 | 独立，可随时提交 |

#### 不提交（fdoc 自用）

| seq | 描述 | 前置依赖 |
|-----|------|---------|
| 09 | ConvertPerBlock 方法 | 02, 03+04 |
| 10 | UpdateBlockText 方法 | 无 |
| 11 | frontmatter 包 + FolderID | 无 |
| 12 | import/export front matter 支持 | 11 |
| 13 | 语义树 diff 引擎 | 09 |
| 14 | fdoc 二进制和命令 | 09-13 全部 |

#### 依赖关系图

```
独立可提交（无依赖，可并行）:
  01  config 测试隔离
  02  引用块修复
  06  DocURL 可配置
  07  表格批量填充
  08  Mermaid TextDrawing

必须合并提交:
  03+04  Todo 嵌套 + 有序列表（代码交叉，无法拆分）

软依赖:
  02 ──(建议先)──→ 05  QuoteContainer 空块清理

fdoc 依赖链（不提交）:
  02, 03+04 ──→ 09 ──→ 13 ──→ 14
                               ↑
  10, 11 ──→ 12 ──────────────┘
```

#### 为什么 seq 03 和 04 必须合并？

`convertTodoWithDepth`（seq 03 新增）直接引用了 `c.orderedSeq` 字段（seq 04 新增）。
两者在 `convertBullet`/`convertOrdered` 的子块递归逻辑中共享相同的代码 hunk，无法按 hunk 拆分。
分开提交任一方都无法编译通过。

**合并后的 PR 建议**：
- 分支名：`fix/list-nesting-and-numbering`
- 标题：`fix(converter): 修复列表嵌套、编号和缩进问题`
- 包含：Todo 嵌套导入导出 + 有序列表真实编号 + 自定义起始编号 + 缩进 2→4 空格 + 嵌套计数隔离

#### 提交进度

| 批次 | seq | 状态 | 说明 |
|------|-----|------|------|
| 第一批 | 01 config 测试隔离 | ✅ 已合入 master，待官方合并 PR | `fix/config-test-isolation` |
| 第一批 | 02 引用块修复 | ✅ 已合入 master，待官方合并 PR | `fix/quote-block-import-export` |
| 第二批 | 03+04 列表嵌套+编号 | ✅ 已合入 master，待官方合并 PR | `fix/list-nesting-and-numbering` |
| 第三批 | 05 QuoteContainer 空块清理 | 待提交 | 02 合入后可提 |
| 第三批 | 06 DocURL 可配置 | 待提交 | |
| 第三批 | 07 表格批量填充 | 待提交 | |
| 第三批 | 08 Mermaid TextDrawing | 待提交 | |

详见 `docs/features/` 目录下各文档。

---

## 完整提交流程（官方改动）

### 1. 创建 feature 分支并提取改动

```bash
# 基于最新 origin/main 创建 feature 分支
git fetch origin
git checkout -b fix/xxx origin/main

# 从 dev 提取改动（按 docs/features/ 中的 cherry-pick 方式）
git diff main..dev -- <file> | git apply
# 或 cherry-pick 整个 commit
git cherry-pick <commit-hash>

# 提交
git add <files>
git commit -m 'fix(scope): 描述'
```

### 2. 推送 feature 分支到 fork

```bash
git push fork fix/xxx
```

### 3. 合入 master 并回归测试

```bash
git checkout master
git merge fix/xxx

# 全量回归
go clean -testcache && go test ./...
go vet ./...
```

**必须零失败**（已知的 main 上游 bug 除外）。

### 4. 同步到 fdoc

```bash
git checkout fdoc
git merge master
```

### 5. 同步更新官方 CLAUDE.md（如需要）

如果本次变更涉及**新增功能、新命令、新 API、块类型变化**等，在 feature 分支上更新 CLAUDE.md：

| 变更类型 | 需更新章节 |
|---------|-----------|
| 新增命令 | 「常用命令」、「功能测试验证」 |
| 新增块类型 | 「块类型映射」 |
| API 行为发现 | 「SDK 注意事项」、「API 限制与处理」 |
| 新增权限需求 | 「权限要求」 |
| 新增技能 | 「Claude Code 技能」 |
| 项目结构变化 | 「项目结构」 |
| 新增依赖 | 「技术栈」 |
| Bug 修复 | 「已知问题」（移除已修复项）、「功能测试验证」 |

**注意**：不要把 dev 分支的「开发流程」「fdoc 命令」等本地内容带上去。

### 6. 人工创建 PR 到官方

从 fork 的 feature 分支向 `riba2534/feishu-cli:main` 创建 PR。

### 7. PR 合并后同步

```bash
# 更新 main
git checkout main && git pull origin main

# 同步 master
git checkout master && git rebase origin/main

# 同步 fdoc
git checkout fdoc && git rebase master

# 删除已合入的 feature 分支
git branch -D fix/xxx
git push fork --delete fix/xxx
```

## 提交 fdoc 改动流程

```bash
# 从 dev 提取 fdoc 专用改动，直接合入 fdoc 分支
git checkout fdoc
git diff dev~N..dev -- <fdoc-files> | git apply
git add <files>
git commit -m 'feat: fdoc xxx'
git push fork fdoc
```

## PR 描述模板

PR 描述**必须包含**以下章节（按顺序）：

| 章节 | 必填 | 说明 |
|------|------|------|
| Summary | 是 | 1-4 条核心改动 |
| Background | 是 | 问题背景、原因分析、动机 |
| Usage（使用示例） | 按需 | 新增功能或行为变更时必填，展示使用方式或前后对比 |
| Changes（改动文件） | 是 | 列出改动文件及每个文件的改动说明 |
| Test Plan | 是 | 测试方法、新增用例、回归结果 |

```markdown
## Summary
- 核心改动 1
- 核心改动 2

## Background
问题背景和原因分析

## Usage（如有新功能或行为变更）
改动前后的使用示例或对比

## Changes
| 文件 | 改动说明 |
|------|---------|
| `path/to/file.go` | 说明 |

## Test Plan
### 常规检查
- [ ] `go build ./...` 编译通过
- [ ] `go vet ./...` 静态检查通过
- [ ] `go clean -testcache && go test ./...` 全部测试通过

### 新增/变更用例
- 说明新增或修改的测试用例及覆盖的场景
```

## Checklist（提交前对照）

- [ ] `go test ./...` 全部通过
- [ ] `go vet ./...` 无警告
- [ ] feature 分支不包含 `docs/`、`fdoc/`、`internal/diff/`、`internal/frontmatter/`、`test/input/` 等本地/fdoc 内容
- [ ] feature 分支的 CLAUDE.md 不包含本地定制内容（TDD 流程、fdoc 命令等）
- [ ] 本次提交的变更对应 `docs/features/` 中分类为 base 或 feishu-cli 的文档
- [ ] 新增功能已同步更新 CLAUDE.md 对应章节
- [ ] 提交信息符合 Conventional Commits 格式
- [ ] PR 标题简洁，描述包含 Summary 和 Test Plan
- [ ] feature 分支已合入 master 且测试通过
