# image-processor

Go 图片处理服务，目前支持上传 JPEG/PNG 图片、按指定宽度等比缩放并返回图片结果。

重新熟悉代码时，先阅读[源码导读](docs/源码导读.md)：包含目录职责、启动与请求调用链、逐文件说明、Go 语法速查、测试覆盖和当前实现边界。

项目正按[优化与深化计划](docs/优化与深化计划.md)推进可投递版本：明确资源限制，接入 MySQL 持久化异步任务和 Redis 结果缓存，使用 Docker Compose 启动完整环境，并保留性能与故障验证证据。

亲手扩展项目时，按[学习与构建路线](docs/学习与构建路线.md)逐步推进：每个功能先回顾现有流程，再设计并实现第一版，随后审查、验证和提交；关键阶段安排流量增长、卡顿排查与故障恢复的面试追问。

详细讲义按一章一章编写。当前从[第 1 章：源图像素预检与解码内存](docs/课程/01_源图像素预检与解码内存.md)开始，用现有请求、内存估算、Reader 实验和分步接入讲清检查原理；示例在独立副本中核验，正式功能由用户亲手实现。

[代码实现与理论核验](docs/代码实现与理论核验.md)提供全部 14 个学习单元的完整参考工程，包含源码、SQL、Docker/Compose、Redis、测试、压测和 CI，可通过[提取脚本](scripts/提取参考实现.ps1)写入独立目录运行。参考代码已通过编译和独立测试，真实 MySQL/Redis、容器运行和 CI 仍待验收；正式业务进度继续以现行代码与阶段记录为准。

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

默认监听 `:8080`。配置位于 `configs/config.yaml`：当前请求体上限 11 MiB、单文件上限 10 MiB、计算并发上限 4、处理 context 超时 15 秒。

上传大小配置如下，单位为字节：

```yaml
process:
  max_file_size: 10485760
  max_request_size: 11534336
```

`max_request_size` 限制 HTTP 请求体读取，包含文件、表单字段与 multipart 边界；它必须为正数并大于 `max_file_size`，给表单开销留空间。使用自己的配置文件时也需要添加这个必填项。请求体超限返回 `413 {"error":"request body too large"}`，单文件超限返回 `413 {"error":"file too large"}`，其他表单解析错误返回 400。

服务在解析 multipart 之前检查已知请求长度，并通过 `http.MaxBytesReader` 限制实际读取，因此长度未知的 chunked 上传同样受限。`ParseMultipartForm(32 << 20)` 的内存阈值与请求体限制是不同参数。

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

R1-1 验证覆盖请求体低于上限、恰好达到上限和超过上限，已知/未知长度、表单字段导致超限、400/413 的区分及统计。另有配置加载校验和通过真实 HTTP 连接执行的 Content-Length/chunked 超限场景。

## 已知边界

- 请求体已在表单解析前限读；单文件上限仍在 Service 读取时检查，输入与输出像素限制将在后续 R1 完善。
- 当前并发上限约束图片处理，上传解析和等待队列尚无完整的容量约束。
- context 在等待计算名额时被检查；解码、缩放、编码调用不会自动因 context 超时立即停止。
- 当前池化读取按最大文件大小申请缓冲区，分级池尚未接入；性能收益尚未测量。
- 当前进程退出后没有持久化任务或结果恢复能力。

## 版本管理与交付

每个功能阶段验证后提交并推送开发分支，记录行为变化、检查结果和限制。可投递版本必须完成 Docker/MySQL/Redis 的实际接入、关键故障验证和可复现的性能对照；尚未完成的功能不写入简历成果。
