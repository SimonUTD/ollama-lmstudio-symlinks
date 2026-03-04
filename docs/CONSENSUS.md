# 共识与需求（LM Studio ↔ Ollama 双向模型同步 + GUI）

**日期**：2026-03-04  
**项目**：ollama-lmstudio-symlinks（原名：ollama-to-lmstudio-symlinks）

## 背景

本工具通过 **符号链接（symlink）** 在 **Ollama** 与 **LM Studio** 之间复用模型文件，避免重复下载与重复占用磁盘。

## 需求（冻结）

1. **双向同步**
   - 保留：Ollama → LM Studio
   - 新增：LM Studio → Ollama（仅 `.gguf`）
2. **同步时识别 symlink**
   - 区分：不存在 / 普通文件 / 普通目录 / symlink（指向正确 / 指向错误 / 断链）
3. **GUI 替代 CLI**
   - 本地 Web GUI：扫描 → 勾选 → 执行
   - 可选择同步哪些模型
4. **Ollama 可用性检测**
   - LM → O 同步前必须检测：`ollama` 可执行文件是否可用、Ollama 服务是否可连接
5. **环境变量继承问题**
   - 编译后二进制可能从 Finder 启动，`PATH` 不可靠；需提供可配置/可探测的 `ollama` 路径策略
6. **README 全中文**

## 关键设计决策

### 1) LM Studio → Ollama 的实现方式（准确优先）

- 通过 `ollama create <name> -f <Modelfile>` 走官方链路创建模型（不手写 manifests）。
- 创建完成后解析 manifests，定位新模型的 model layer digest，对应 blob 文件路径：
  - `~/.ollama/models/blobs/sha256-...`
- 将该 blob **替换为指向 LM Studio `.gguf` 的 symlink**，实现复用文件（节省空间）。

### 2) symlink 策略（默认保守）

- 默认不覆盖普通文件/目录。
- 修复“指向错误的 symlink / 断链 symlink”必须显式开启开关才会执行。
- LM → O 中“允许替换已存在 blob”默认关闭（digest 共享存在影响其他模型的风险）。

### 3) Ollama 检测策略（兼容 Finder 启动）

`ollama` 路径探测优先级：
1. GUI 配置（持久化）
2. 环境变量 `OLLAMA_BIN`
3. `PATH`
4. 固定路径（macOS：`/Applications/Ollama.app/Contents/Resources/ollama` 等）

服务连通性：
- 使用 `OLLAMA_HOST`/配置（默认 `127.0.0.1:11434`）做 TCP dial 检测
- **不自动启动** `ollama serve`

## 安全与边界

- GUI 前端渲染使用 DOM + `textContent`（不使用 `innerHTML`）避免 XSS。
- 后端对 LM → O 的导入做强校验：仅允许 `.gguf` 且路径必须在 LM Studio models 目录内（解析 symlink 后比对）。
- HTTP API 的 JSON 写请求要求 `Content-Type: application/json`，避免浏览器 simple-request 触发本地 CSRF。
- 写 Modelfile 前拒绝包含 `\\r/\\n` 的 `ggufPath`，避免 Modelfile 语义注入。

## 非目标

- 不做“LM Studio 任意格式 → Ollama”的万能转换（仅支持 `.gguf`）。
- 不做远程/多机同步。
- 不实现 dry-run（可作为后续增强，不属于本次冻结需求）。

## 验收口径（Done-when）

1. GUI 可启动并可扫描两端模型、勾选并执行同步。
2. LM → O：
   - 非 `.gguf`：在扫描中置灰并说明；若绕过前端调用 API 也会被后端拒绝。
   - Ollama 不可用/服务不可连接：执行时返回明确错误并在 GUI 展示。
3. O → L：保持现有能力，并准确识别已有 symlink（正确/错误/断链/冲突）。
4. README 为中文，包含使用方式与风险提示。

