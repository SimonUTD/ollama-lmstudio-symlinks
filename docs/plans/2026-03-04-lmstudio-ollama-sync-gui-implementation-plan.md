# LM Studio ↔ Ollama 双向模型同步（含 GUI）实现计划

> 说明：本仓库将按 TDD（先红后绿）推进；除非用户明确要求，本执行环境不自动 `git commit`。

**Goal**：实现 LM Studio ↔ Ollama 双向模型同步（symlink 去重 + GUI 勾选），并将 README 更新为中文。  
**Architecture**：Go 单二进制 + localhost Web GUI；核心分为 Discovery/Plan/Apply；LM→O 通过 `ollama create` 创建模型后将目标 blob 替换为 symlink。  
**Tech Stack**：Go（标准库 `net/http`, `embed`, `os`, `filepath`），最小依赖。  

---

## Task 1：引入 Go Module 与目录结构

**Files:**
- Create: `go.mod`
- Modify: `go.sum`（去掉无效注释，允许为空或由 Go 自动生成）
- Create: `VERSION`（让 `build.sh` 与 GitHub Actions 可用）
- Modify: `build.sh`、`Makefile`、`.github/workflows/build.yml`（按新 main 包路径构建）

**Steps:**
1. 创建 `go.mod`（module `github.com/SimonUTD/ollama-lmstudio-symlinks`）
2. 修正 `go.sum` 为合法格式（可为空，后续 `go test` 会生成）
3. 创建 `VERSION`（先用 `v0.0.0`，后续可手动调整）
4. 更新构建入口（例如 `go build ./cmd/ollama-symlinks`）

**Validation:**
- Run: `go test ./...`（此时可能还没有测试，但至少不应出现 “go.mod/go.sum 解析错误”）
- Run: `go build ./cmd/ollama-symlinks`

---

## Task 2：实现并测试 symlink 识别（文件系统层）

**Files:**
- Create: `internal/symlink/status.go`
- Test: `internal/symlink/status_test.go`

**Behaviors:**
- 不存在 / 普通文件 / 普通目录 / symlink（指向正确/错误/断链）
- 目标为相对 symlink 时，能解析为绝对路径进行比较（基于 link 所在目录拼接）

**Validation:**
- Run: `go test ./internal/symlink -v`

---

## Task 3：实现 Ollama 模型发现（manifests 解析）并补测试

**Files:**
- Create: `internal/ollama/manifest.go`
- Create: `internal/ollama/discover.go`
- Test: `internal/ollama/discover_test.go`

**Behaviors:**
- 从 `~/.ollama/models/manifests` 扫描出模型 `repo:tag`
- 解析 layers：定位 `application/vnd.ollama.image.model` digest 与 projector digest（如存在）
- 提供 digest → blob filename 转换（`sha256:<hex>` → `sha256-<hex>`）

**Validation:**
- Run: `go test ./internal/ollama -v`

---

## Task 4：实现 LM Studio 模型发现（`.gguf` 扫描）并补测试

**Files:**
- Create: `internal/lmstudio/discover.go`
- Test: `internal/lmstudio/discover_test.go`

**Behaviors:**
- 扫描 `~/.cache/lm-studio/models` 下的 `.gguf`
- 记录 provider/modelDir/file/size/fullPath
- 多 gguf 文件同目录时：标记 “auxiliary” 与 “primary”（按文件大小择大）

**Validation:**
- Run: `go test ./internal/lmstudio -v`

---

## Task 5：实现 Ollama 安装/服务检测与命令执行抽象（可测试）

**Files:**
- Create: `internal/ollamaexec/detect.go`
- Create: `internal/ollamaexec/runner.go`
- Test: `internal/ollamaexec/detect_test.go`

**Behaviors:**
- 发现 `ollama` 可执行文件（优先级：config > env `OLLAMA_BIN` > PATH > 常见固定路径）
- `ollama --version` 校验并返回可读状态
- 检测 `OLLAMA_HOST` 连通性（默认 `127.0.0.1:11434`），不可连接时给出明确原因

**Validation:**
- Run: `go test ./internal/ollamaexec -v`

---

## Task 6：实现双向同步 Plan/Apply（先单测 plan，再实现 apply）

**Files:**
- Create: `internal/sync/plan.go`
- Create: `internal/sync/apply.go`
- Test: `internal/sync/plan_test.go`
- Test: `internal/sync/apply_test.go`

**Behaviors:**
- O→L：按 digest 生成目标路径，使用 symlink 状态决定 create/skip/conflict/fix（fix 需显式开关）
- L→O：
  - 若目标 Ollama 模型已存在且 blob symlink 指向正确：skip
  - 若已存在但不匹配：conflict（默认不覆盖）
  - 若不存在：执行 `ollama create`，解析 manifest 找到 model layer digest，对应 blob 进行替换为 symlink
  - 若 blob digest 在同步开始前已存在：默认不替换（除非显式开关），避免影响其他模型；结果需明确提示

**Validation:**
- Run: `go test ./internal/sync -v`

---

## Task 7：实现 GUI（Web）与配置持久化

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/gui/server.go`
- Create: `internal/gui/handlers.go`
- Create: `internal/gui/assets/*`（HTML/CSS/JS）
- Create: `cmd/ollama-symlinks/main.go`

**Behaviors:**
- GUI 启动后可扫描两端模型并勾选同步
- 展示 Ollama 检测结果（已安装/路径/版本/服务连通）
- 支持执行并展示逐项结果（本次不实现 dry-run）
- 配置保存与加载（目录、OLLAMA_HOST、ollama bin、覆盖开关等）

**Validation:**
- Run: `go test ./...`
- Run: `go build ./cmd/ollama-symlinks`

---

## Task 8：README 全中文更新

**Files:**
- Modify: `README.md`

**Behaviors:**
- 说明两种同步方向、GUI 用法、目录默认值、Ollama 检测、风险提示（删除源文件会断链）
- 明确限制：LM→O 仅支持 `.gguf`

**Validation:**
- 人工检查：内容为中文、命令示例可复制运行

---

## Task 9：全量验证

**Validation:**
- Run: `go test ./...`
- Run: `go build ./cmd/ollama-symlinks`
- Run: `make test`（如果 Makefile 已对齐）
- Run: `make build`
