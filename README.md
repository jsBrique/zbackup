## zbackup

zbackup 是一个“像 rsync 一样但更傻瓜”的 SSH 增量备份小工具。它只做一件事：把一个目录完整同步到另一端（本地或远端），然后把状态写入目标端 `.zbackup` 目录，以便以后继续增量。当你在命令行执行一次 zbackup 后，目录里的 `.zbackup/` 将记录所有快照、日志和断点信息，下次运行时就能直接接着上次的结果。

> ⚠️ **风险提示**：这是一个自用开源小项目，会对目标目录进行创建/覆盖/删除操作。使用前请在非关键数据上充分测试，确保行为符合预期；若因使用本工具造成数据丢失或其它损失，作者概不负责。
>
> ⚠️ Windows 平台仅做基本验证，部分行为可能与类 Unix 环境不同，请慎重使用。

### 工作原理

1. **识别端点**：`-s/--source`、`-d/--dest` 可以是本地路径，也可以是 `user@host:/path` 格式。zbackup 始终在源端扫描文件，再将变更同步到目标端。
2. **扫描 & Diff**：源端扫描结果会记录目录/文件/大小/修改时间/校验和（按需），再和目标端 `.zbackup/snapshots/<latest>.json` 对比，得出新增、修改、删除列表。
3. **生成计划**：把 diff 结果转成 `TransferPlan`，包含 mkdir/upload/download/delete/skip 等动作。支持 `--mode full`（全量，删除目的端冗余）与 `--mode incr`（增量，默认）。
4. **传输 & 校验**：所有操作都通过 SSH 完成（可给 `-p/-i/-o`）。传输成功后按配置的算法计算源/目的校验和，确保一致后才算完成。
5. **快照 & 断点**：执行过程中实时写入 `.zbackup/pending.json`，即便断电/中断也能从该文件继续。任务结束会输出新的快照 JSON 和日志文件，更新 `.zbackup/latest`。
6. **进度条 & 日志**：终端显示单行进度条（含当前文件与 Mbps 速率）；日志输出前会清除进度行，避免挤在一行。日志和快照都在目标端 `.zbackup` 下，方便调试与追踪。

### 功能摘要

- **传输方式**：完全基于 SSH，支持所有常见的 SSH 选项；无需额外开放端口。
- **双向同步**：既可拉取远端到本地，也可把本地推送到远端。
- **增量/全量**：增量模式只同步变化的文件；全量模式会删除目的端多出来的文件，保持与源端一致。
- **断点续传**：执行时持续把进度写入 `.zbackup/pending.json`，中断后自动读取继续。
- **校验算法**：默认 `sha256`，也可选择 `md5/sha1/none`；校验既用于增量判断，也用于传输后验证。远端会优先尝试 `sha256sum` 等命令，不支持时回落为本地计算。
- **目录保持**：会同步空目录，路径中的空格、中文等特殊字符也会被正确识别。
- **日志与进度**：终端进度条显示百分比、文件数、实时 Mbps；日志默认写在 `.zbackup/logs/` 下，也可通过 `--log-file` 指向本地文件。

### 运行环境依赖

**本机（运行 zbackup 的机器）**
- Go 环境（编译或 `go run` 使用时）。
- `ssh`、`scp` 或兼容的 OpenSSH 工具（zbackup 会外部调用它们）。
- 对于路径含中文/空格的场景，终端需支持 UTF-8。

**远端主机**
- 必须启用 SSH，并允许执行常规 shell 命令（`find`、`stat`、`cat` 等）。
- 如希望远端参与校验，需要安装 `sha256sum`/`sha1sum`/`md5sum`（GNU coreutils）；未安装时 zbackup 会自动回退到“把文件拉回本地计算 hash”，只是会多一次读取流量。
- 能执行 `mkdir`、`rm` 等基本命令。

### 安装

```bash
go install ./cmd/zbackup
```

或直接在源码目录运行：

```bash
go run ./cmd/zbackup --help
```

### 基本用法

```bash
# 远端 → 本地
zbackup -p 13022 -s user@host:~/src/ -d ./dst/

# 本地 → 远端
zbackup -p 13022 -s ./src/ -d user@host:/data/
```

常用参数：

| 参数 | 说明 |
| --- | --- |
| `-s, --source` / `-d, --dest` | 指定源/目标路径（本地或 `[user@]host:/path`） |
| `-p, --port` / `-i, --identity` / `-o, --ssh-option` | SSH 端口、私钥、附加选项 |
| `-m, --mode` | `full` / `incr`，默认增量 |
| `--checksum` | `none` / `md5` / `sha1` / `sha256`，默认 `sha256` |
| `--exclude` | 支持 glob 的排除规则，可多次传入 |
| `--no-progress` | 关闭终端进度条（适合 CI） |
| `--log-file` / `--log-level` | 自定义日志文件和级别（默认目标端 `.zbackup/logs/`） |
| `--dry-run` | 仅展示计划，不实际传输 |
| `--snapshot-name` | 自定义快照名称（默认 UTC 时间戳） |

### 架构说明（更细一点）

- `cmd/zbackup`：Cobra CLI 入口，解析参数、校验配置。
- `pkg/core`：核心流程；负责调用扫描、diff、传输、日志与快照，内置 `checkpoint` 机制持续落盘未完成进度。
- `pkg/endpoint`：表达本地/远端端点，远端通过 SSH 命令执行 `find/stat/cat` 等。
- `pkg/transfer`：执行传输计划；支持并发、校验、mkdir/delete/skip 等动作，并通过回调将成功记录反馈给 `core`。
- `pkg/meta`：管理 `.zbackup` 下的快照、latest、pending 文件。
- `pkg/ui`：控制台进度条和日志输出互斥，保持单行刷新。
- `pkg/logging`：对 `log/slog` 的轻包装，便于输出到多个 Writer。

元数据结构示例（`meta.Snapshot`）：

```json
{
  "name": "20250101T120000Z",
  "created_at": "...",
  "source_root": "/src",
  "dest_root": "/dst",
  "files": {
    "foo.txt": {
      "rel_path": "foo.txt",
      "size": 1024,
      "mod_time": "...",
      "checksum": "...",
      "is_dir": false
    }
  },
  "completed": true
}
```

### 日志与快照

- 默认在目标端 `.zbackup/logs/backup-<snapshot>.log` 写入完整执行日志。
- 快照保存在 `.zbackup/snapshots/<snapshot>.json`，最新记录由 `.zbackup/latest` 指向。
- 执行过程中持续更新 `.zbackup/pending.json`，异常退出后可继续。
- 指定 `--log-file` 时，日志写到本地文件，但快照仍落在目标端。

### 进阶技巧

- `--exclude "*.tmp" --exclude "cache/*"` 可排除多种模式。
- `--dry-run` 查看计划，不传输；输出包括每个 action、路径和大小。
- 全量模式（`--mode full`）会同步删除目的端多余文件，适合“镜像备份”场景。
- 手动传输过程中可随时退出，下次运行会从 `.zbackup/pending.json` 接着同步。

### 为什么要把快照写在目标端？

- 换电脑、换位置都能继续，因为所有状态都保存在目标端。
- 断点续传自然可用：`.zbackup/pending.json` 会记住已完成的文件，下次开机后无需任何手工操作。
- 多个备份任务可共存，每个快照可按时间戳或自定义名字分辨。

### 测试与交叉编译

```bash
go test ./...
```

仓库提供 PowerShell 脚本帮助生成常见平台的二进制：

```powershell
pwsh ./build-linux.ps1      # 仅编译 Linux/AMD64
pwsh ./build-all.ps1        # Linux/macOS/Windows 主流架构
```

生成的文件都会放在 `build/` 目录。

### 开发提示

- 所有 SSH 行为目前通过外部 `ssh` 命令完成，后续可替换为内建 SSH/SFTP。
- 增量判断默认基于 size+mtime，若启用校验和，会在传输完成后保存 hash，幂等更强。
- Fast fail：出现错误时日志中会列出失败的文件，不会影响已成功的文件；再次运行会自动重试失败文件。
- 由于所有状态都在目标端 `.zbackup` 下，你可以把该目录备份或版本控制起来，方便回滚。

