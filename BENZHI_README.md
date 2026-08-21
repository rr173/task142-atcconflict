# ATC Conflict Engine

ATC Conflict Engine 是一个面向区域管制席位的空域冲突探测服务。它保存飞行计划、航空器性能、航迹报告和移交记录；对 ACTIVE 飞行的最新航迹做短期 CPA 预测，输出水平与 RVSM 垂直间隔同时失效的冲突，并在服务重启后从 SQLite 中恢复可重算状态。

## 本地运行

```bash
go build ./...
go test ./...
go vet ./...
go run . --smoke-test
go run . --addr=:8080
```

`--smoke-test` 使用固定输入验证扇区容量告警和对头冲突探测，不依赖外部服务。默认运行模式在当前目录创建 `atc.db`，提供 `/health`、`/metrics`、飞行计划、航迹、冲突和移交接口；静态页面由嵌入式前端在 `/` 提供。

## Docker

```bash
./build_benzhi_docker.sh atc-conflict linux/amd64
docker run --rm atc-conflict --smoke-test
./build_benzhi_docker.sh atc-conflict-arm64 linux/arm64
docker run --rm atc-conflict-arm64 --smoke-test
```

`build_benzhi_docker.sh` 的第一个参数是镜像名，第二个参数是目标平台。镜像构建时会下载 Go module 并执行 `go build ./...`；容器默认启动服务，也可显式传入 `--smoke-test` 运行后退出。
