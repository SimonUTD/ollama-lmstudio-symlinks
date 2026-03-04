# Ollama ↔ LM Studio 模型同步（Symlink 省空间）

本项目是一个 Go 工具，用 **符号链接（symlink）** 在 **Ollama** 与 **LM Studio** 之间同步模型，避免重复下载/重复占用磁盘空间。

目前支持两种方向：

- **Ollama → LM Studio**：在 LM Studio 中复用 Ollama 的 blob 文件
- **LM Studio → Ollama（仅 .gguf）**：让 LM Studio 的 `.gguf` 模型在 Ollama 中可用，并尽可能通过 symlink 复用文件

并提供 **GUI（本地 Web 页面）** 来替代原先的纯 CLI 操作，支持勾选选择要同步的模型。

---

## 功能特性

- 自动扫描两端模型（Ollama manifests / LM Studio `.gguf`）
- 同步时会识别目标是否已是 **正确的 symlink**（已同步/冲突/指向错误/断链）
- GUI 勾选同步：扫描 → 勾选 → 执行
- LM → O 会检测：
  - `ollama` 是否安装/可执行（支持配置路径、环境变量、固定路径探测）
  - Ollama 服务是否可连接（默认 `127.0.0.1:11434`）

---

## 快速开始

### 运行（启动 GUI）

```bash
./ollama-symlinks
```

默认在本机启动 GUI（地址会打印在终端）：

```text
GUI 已启动: http://127.0.0.1:48289
配置文件: <你的 config 路径>
```

然后在浏览器中打开该地址即可。

### 常用参数

```bash
# 指定 GUI 监听地址
./ollama-symlinks --listen 127.0.0.1:48289

# 指定配置文件路径
./ollama-symlinks --config /path/to/config.json

# 查看版本
./ollama-symlinks --version
```

---

## 默认目录（可在 GUI 配置里修改）

- Ollama models：`~/.ollama/models`
- LM Studio models：`~/.cache/lm-studio/models`

### 同步后落盘位置

- **Ollama → LM Studio**：会在 LM Studio models 下创建 provider 目录：
  - `~/.cache/lm-studio/models/ollama/<model>/<model>.gguf`（以及可选的 projector 文件）

- **LM Studio → Ollama**：会在 Ollama models 下生成 manifests，并将相关 blob 变为指向 `.gguf` 的 symlink：
  - `~/.ollama/models/manifests/...`
  - `~/.ollama/models/blobs/sha256-...` → symlink 到 LM Studio `.gguf`

---

## LM Studio → Ollama 重要说明

1) **仅支持 `.gguf`**  
LM Studio 的 MLX 等非 `.gguf` 模型不会出现在“LM Studio → Ollama”扫描列表里。

2) **需要 Ollama 服务在运行**  
GUI 会检查 `OLLAMA_HOST`（默认 `127.0.0.1:11434`）。如果不可连接，请：
- 打开 Ollama App（macOS）
- 或在终端运行：`ollama serve`

3) **“已存在 blob”替换选项有风险**  
Ollama 的 blob 是按 digest 共享的：多个模型可能引用同一个 blob。  
GUI 中的“允许替换已存在的 Ollama blob”为 symlink（LM→O）默认关闭，开启前请确认风险。

4) **Ollama 名称默认沿用 LM Studio 目录**  
默认使用 `provider/modelDir` 作为 Ollama 名称（不指定 tag 则为 `latest`）。若 GUI 提示名称不合法，请在列表中编辑为更短/更符合规则的名称后再同步。

5) **并非所有 `.gguf` 都能被当前 Ollama 版本加载**  
如果 `ollama run <model>` 报 `500 Internal Server Error: unable to load model ...`，请查看 `~/.ollama/logs/server.log`。常见原因是 `unknown model architecture`（需要升级 Ollama 或更换兼容的 GGUF）。

---

## Ollama 可执行文件发现（解决 Finder 启动 PATH 不继承）

在 macOS 上，如果你双击运行二进制，`PATH` 往往不会加载 `.zshrc/.zprofile`，因此找不到 `ollama`。

本工具按以下优先级寻找 `ollama`：

1. GUI 中配置的 `ollama` 路径
2. 环境变量 `OLLAMA_BIN`
3. `PATH` 中的 `ollama`
4. 固定路径（macOS）：`/Applications/Ollama.app/Contents/Resources/ollama` 等

---

## 从源码构建

### Make（推荐）

```bash
make build
make test
```

### build.sh

```bash
./build.sh
```

---

## 常见问题

### 安全提示（仅本地使用）

本工具会修改本机模型目录并可能调用 `ollama` 执行创建操作。建议仅监听 `127.0.0.1`，不要将 GUI 暴露到公网或不可信网络。

### Windows 上创建 symlink 提示权限不足

Windows 创建 symlink 可能需要管理员权限或开启开发者模式，请用管理员终端运行或调整系统设置。

### LM Studio 里看不到同步后的模型

1. 重启 LM Studio
2. 检查 symlink 是否存在且指向正确：`ls -la ~/.cache/lm-studio/models/ollama/*/`

### 同步后删除模型导致不可用

本工具创建的是 symlink，不是复制文件：删除/移动源文件会导致断链，模型不可用。

## 贡献

欢迎提交 Issue 或 Pull Request 来改进这个工具。

## 许可证

本项目使用 MIT 许可证，详见 `LICENSE`。
