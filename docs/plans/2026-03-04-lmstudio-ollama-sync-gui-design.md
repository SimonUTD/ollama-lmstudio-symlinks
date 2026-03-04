# LM Studio ↔ Ollama 双向模型同步（含 GUI）设计文档

**日期**：2026-03-04  
**项目**：ollama-to-lmstudio-symlinks  

## 背景

当前项目仅支持 **Ollama → LM Studio**：通过创建 symlink，让 LM Studio 复用 Ollama 的 blob 文件，避免重复占用磁盘。

新需求是支持 **双向**：
- 继续支持 Ollama → LM Studio
- 新增 LM Studio → Ollama：把 LM Studio 已下载的模型“接入” Ollama 使用，同样以 symlink 为核心，尽量避免复制大文件
- 提供 GUI 替代 CLI，并允许勾选选择要同步的模型
- 同步时要正确识别与处理既有 symlink（已同步、指向错误、断链等）
- README 更新为中文

## 目标（Goals）

1. **双向同步**：Ollama ↔ LM Studio 都能同步模型，且默认不复制大文件。
2. **symlink 识别准确**：同步过程中对目标路径的存在情况做“类型+指向”级别识别，而不是仅判断“存在/不存在”。
3. **GUI**：本地可视化界面，支持扫描、勾选、执行、结果展示。
4. **可移植**：编译后的二进制即用，不依赖 Node/Electron 等额外运行时；从终端启动或双击启动都能正确工作。
5. **安全与可解释**：默认不覆盖普通文件；遇到冲突给出明确原因与提示，不做静默降级。

## 非目标（Non-goals）

- 不做“把所有 LM Studio 格式都转成 Ollama 可用”的万能转换。
  - 仅支持 **`.gguf`** 模型导入到 Ollama。
  - MLX 等非 GGUF 目录在 GUI 中展示但禁用同步，并显示原因。
- 不实现远程同步/多机同步。
- 不在未经用户确认的情况下自动删除任何模型或目录。

## 关键约束与事实

### 1) 两端模型存储（本机观察）

- LM Studio 默认模型目录：`~/.cache/lm-studio/models/`
  - 常见结构：`~/.cache/lm-studio/models/<provider>/<model>/<files>`
  - 模型文件可能包含：`.gguf`、以及配套文件（例如 `config.json`、`mmproj*.gguf` 等）

- Ollama 默认模型目录：`~/.ollama/models/`
  - 项目当前已依赖其结构：`manifests/` 与 `blobs/`

### 2) 环境变量继承（编译后可执行文件）

Go 二进制只会继承“启动它的进程”的环境变量：
- 从终端运行：通常能继承 shell 的 `PATH`
- 从 Finder/Spotlight 双击：通常不会加载 `.zshrc/.zprofile`，`PATH` 可能不包含 Homebrew 或自定义路径

因此不能仅依赖 `PATH` 来发现 `ollama`。

## 总体方案（Architecture）

采用 **单个 Go 可执行文件 + 内置 Web GUI（localhost）**：
- 后端：Go `net/http`（标准库），提供扫描/预览/执行 API
- 前端：静态 HTML/CSS/JS（内嵌到二进制，或随二进制发布）
- 业务核心：抽象为“模型发现（Discovery）”与“同步计划（Plan）”与“执行（Apply）”

核心同步思路仍是：**创建 symlink，而非复制模型文件**。

## 同步方向与策略

### A) Ollama → LM Studio（保留并增强）

**目标**：在 LM Studio 目录下创建 provider 目录 `ollama/`，并为每个模型创建 `.gguf` symlink（现有实现）。

**增强点**：
- 目标路径存在时，不仅判断“存在”，还要识别：
  - 已存在且是 symlink，且指向正确：标记为 `AlreadySynced`
  - 已存在且是 symlink，但指向错误：标记为 `SymlinkMismatch`
  - 已存在但为普通文件/目录：标记为 `Conflict`
  - symlink 断链：标记为 `BrokenSymlink`
- 默认不覆盖 `Conflict`；对 `SymlinkMismatch` 是否允许修复由 GUI 明确开关控制（默认关闭）。

### B) LM Studio → Ollama（新增）

**目标**：让 LM Studio 的 `.gguf` 模型在 Ollama 中可用，同时尽可能不复制 `.gguf` 文件。

**推荐实现（准确性优先）**：
1. 检测 `ollama` 是否安装可用（见下一节）
2. 检测 Ollama server 是否可连接（`OLLAMA_HOST`，默认 `127.0.0.1:11434`）
3. 对每个选中的 `.gguf`：
   - 生成一个临时 Modelfile：`FROM <absolute_path_to_gguf>`
   - 调用 `ollama create <model_name> -f <Modelfile>` 创建模型
   - 从 `~/.ollama/models/blobs` 中定位新产生的 blob（通过前后快照差异或 manifests 解析）
   - 用 symlink 将该 blob 指向 LM Studio 的 `.gguf` 文件（实现“复用”）

**为什么不直接手写 manifests**：
- Ollama manifests/schema/元数据存在版本差异与兼容性风险；
- 使用 `ollama create` 走官方路径更稳，出错也更可诊断（错误信息更清晰）。

**命名规则（可配置）**：
- 默认从 LM Studio 目录推导：`<provider>/<model>/<filename>` → 规范化为 `provider-model-filename`（去扩展名）
- GUI 允许用户对每个条目编辑最终 Ollama 模型名（避免冲突）

## Ollama 安装与服务检测

### 可执行文件发现（有序优先级）

1. GUI 中用户显式配置的 `ollama` 路径（持久化）
2. 环境变量 `OLLAMA_BIN` 指定路径
3. `exec.LookPath("ollama")`
4. 常见固定路径探测：
   - macOS：`/Applications/Ollama.app/Contents/Resources/ollama`
   - Homebrew：`/opt/homebrew/bin/ollama`、`/usr/local/bin/ollama`

对候选路径执行 `ollama --version` 校验可用性，并在 GUI 展示检测结果与最终选择来源。

### 服务连通性

通过 `OLLAMA_HOST`（默认 `127.0.0.1:11434`）检查是否可连接：
- 不可连接：LM → O 执行会返回明确错误，并提示“请打开 Ollama App 或运行 `ollama serve`”
- 该策略不做静默自动启动，避免后台常驻进程与权限/资源争议

## symlink 识别与冲突策略（统一）

### 目标路径判定（必须区分类型）

对“目标文件路径”做如下判断：
- 不存在：可创建
- 存在且是 symlink：
  - readlink 后与期望源路径一致：已同步，跳过
  - 不一致：标记 mismatch（默认不修复，除非显式允许）
- 存在且是普通文件/目录：冲突（默认跳过，不覆盖）

### 可选覆盖策略（GUI 显式控制）

- `允许修复指向错误的 symlink`（默认关闭）
- `允许删除并重建断链 symlink`（默认关闭）

任何覆盖行为都必须在 GUI 明确开关下执行，并在执行日志中逐项记录。

## GUI 交互设计（最小可用）

### 页面结构

1. 顶部状态栏
   - LM Studio 目录（可配置/自动探测）
   - Ollama 状态（已安装/未安装、路径、版本、服务可连接性）
2. 同步方向选择
   - `Ollama → LM Studio`
   - `LM Studio → Ollama`
3. 扫描按钮
4. 模型表格（可筛选/搜索）
   - 勾选框
   - 名称、来源路径、目标路径、状态（可同步/已同步/冲突/不支持）
   - （LM→O）可编辑 “Ollama 名称”
5. 执行按钮
6. 结果区（逐项成功/跳过/失败原因）

## 配置与持久化

- 保存用户选择的目录（LM Studio dir、Ollama dir、ollama bin、OLLAMA_HOST）与 GUI 选项开关。
- 配置文件位置：优先使用标准用户配置目录（macOS：`~/Library/Application Support/<app>/config.json`），并允许通过环境变量覆盖。

## 测试计划（TDD）

至少覆盖以下单元测试：
- symlink 识别：不存在/普通文件/symlink 指向正确/指向错误/断链
- 模型发现：
  - Ollama manifests 扫描解析
  - LM Studio 目录扫描：识别 `.gguf` 与非 `.gguf`
- 计划生成（plan）：对不同状态条目生成正确的 action（create/skip/conflict）

对 `ollama create` 相关逻辑做可注入接口，单测用 fake runner 验证命令拼装与错误传递（不做假成功路径）。

## 验收标准（Done-when）

1. GUI 可启动并在浏览器使用：能扫描两端模型并勾选同步。
2. LM → O：
   - Ollama 未安装时：GUI 明确报错（执行时返回错误并展示）
   - Ollama 未运行时：GUI 明确提示需要启动服务（执行时返回错误并展示）
   - 选择 `.gguf` 后可导入，并在 `~/.ollama/models` 侧以 symlink 复用 LM Studio 文件
3. O → L：
   - 保持现有能力，并正确识别“已存在但指向不一致”的 symlink
4. README 全中文，包含安装、GUI 使用、限制与风险提示。
