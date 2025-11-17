## zbackup

基于 Go 编写的命令行增量备份工具，所有数据通过 SSH/SCP 通道传输，所有快照、日志统一落在目标端 `.zbackup/` 目录。支持本地 ↔ 远端双向备份、全量/增量切换、传输后校验、进度条与详细日志输出。

### 安装

```bash
go install ./cmd/zbackup
```

或直接在源码目录内运行：

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
| `-s, --source` / `-d, --dest` | 指定源/目的路径（本地或 `[user@]host:/path`） |
| `-p, --port` / `-i, --identity` / `-o, --ssh-option` | 透传 SSH 端口、私钥与额外参数 |
| `-m, --mode` | `full` / `incr`，默认增量 |
| `--checksum` | `none` / `md5` / `sha1` / `sha256`，默认 sha256 |
| `--exclude` | 排除 glob 模式，可多次 |
| `--no-progress` | 关闭进度条（CI/脚本场景） |
| `--log-file` / `--log-level` | 自定义日志输出位置与级别（默认写入目标端 `.zbackup/logs/`） |
| `--dry-run` | 只显示计划、不执行传输 |
| `--snapshot-name` | 自定义快照名称，不填则使用 UTC 时间戳 |

### 架构说明

- `cmd/zbackup`：CLI 入口，使用 Cobra 解析参数，构造 `core.BackupConfig`。
- `pkg/core`：任务编排（扫描 → Diff → 传输 → 校验 → 快照/日志写入）。
- `pkg/endpoint`：抽象本地与远端文件系统（远端通过 SSH 命令实现 `find`、`cat`、`stat` 等操作）。
- `pkg/meta`：在目标端 `.zbackup` 下读写快照与 `latest` 指针。
- `pkg/transfer`：根据计划串流复制文件、计算校验和、处理删除。
- `pkg/ui`：进度条封装，默认使用 `progressbar`（`--no-progress` 时降级为空实现）。
- `pkg/logging`：统一的 slog 包装，支持多 writer。

元数据结构（`meta.Snapshot`）：

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
      "checksum": "..."
    }
  },
  "completed": true
}
```

### 日志与快照落盘

- 默认在目标端 `.zbackup/logs/backup-<snapshot>.log` 写入完整执行日志。
- 快照保存在 `.zbackup/snapshots/<snapshot>.json`，并通过 `.zbackup/latest` 记录上一次成功快照。
- 自定义 `--log-file` 时，日志写到本地指定路径，快照仍落在目标端。

### 进阶

- 支持 glob 式 `--exclude`，例如 `--exclude "*.tmp" --exclude "cache/*"`。
- `--dry-run` 可先确认计划（会输出详细 action/path/size）。
- 全量模式（`--mode full`）会同步删除目的端多余文件。

### 测试

项目自带单元测试覆盖核心模块：

```bash
go test ./...
```

### 开发提示

- 所有 SSH 命令通过 `endpoint.RemoteFS` 抽象，后续可替换为内建 SSH/SFTP。
- 增量判断默认以 size+mtime 为主，开启校验和后会在传输完成后保存 hash，重复运行保证幂等。
- 日志、快照写入都使用统一的 `FileSystem`，因此远端/本地行为一致；重试时可直接利用 `.zbackup` 信息继续增量。

