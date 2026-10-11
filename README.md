# image-processor

Go 图片处理服务，目前支持上传 JPEG/PNG 图片、按指定宽度等比缩放并返回图片结果。

项目正按[优化与深化计划](docs/优化与深化计划.md)推进可投递版本：明确资源限制，接入 MySQL 持久化异步任务和 Redis 结果缓存，使用 Docker Compose 启动完整环境，并保留性能与故障验证证据。

## 当前状态

当前可运行的是同步服务。Docker Compose、MySQL、Redis 和异步任务尚未实现；各项完成状态以计划中的 R0–R5 进度表为准。

| 接口 | 行为 |
|---|---|
| `POST /resize` | multipart 表单上传图片，按宽度等比缩放，返回 JPEG/PNG |
| `GET /healthz` | 返回进程存活状态 |
| `GET /stats` | 返回请求计数；池统计目前尚未接入真实调用 |

## 本地运行

需要 Go 1.25.4 或满足模块要求的更新版本。从仓库根目录执行：

```sh
go run ./cmd/server -config configs/config.yaml
```

默认监听 `:8080`。配置位于 `configs/config.yaml`：当前文件上限 10 MiB、计算并发上限 4、处理 context 超时 15 秒。

上传示例（将图片路径换成自己的文件）：

```sh
curl -f -X POST http://localhost:8080/resize -F "file=@sample.png;type=image/png" -F "width=320" -o resized.png
```

Windows PowerShell 中使用 `curl.exe` 执行同样的参数，以避免旧版 PowerShell 的同名别名。

停止服务：在启动窗口按 `Ctrl+C`。

## 验证

```sh
go test ./...
go build -o .bin/server ./cmd/server
```

已有参数错误路径验证。新增基线验证覆盖真实 JPEG/PNG 的响应格式与等比输出尺寸，以及损坏图片；开启和关闭缓冲池分别验证。

## 已知边界

- 文件上限在表单解析之后检查，请求体总大小和图片像素限制将在 R1 完善。
- 当前并发上限约束图片处理，上传解析和等待队列尚无完整的容量约束。
- context 在等待计算名额时被检查；解码、缩放、编码调用不会自动因 context 超时立即停止。
- 当前池化读取按最大文件大小申请缓冲区，分级池尚未接入；性能收益尚未测量。
- 当前进程退出后没有持久化任务或结果恢复能力。

## 版本管理与交付

每个功能阶段验证后提交并推送开发分支，记录行为变化、检查结果和限制。可投递版本必须完成 Docker/MySQL/Redis 的实际接入、关键故障验证和可复现的性能对照；尚未完成的功能不写入简历成果。
