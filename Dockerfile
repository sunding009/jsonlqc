# syntax=docker/dockerfile:1

# jsonlqc Dockerfile —— 多阶段构建：
#   阶段一（builder）：golang 镜像中静态编译出无依赖的单一二进制；
#   阶段二（runtime）：scratch 空白镜像，只放入该二进制，体积最小、攻击面最小。
#
# 构建镜像：
#   docker build -t jsonlqc .
#
# 运行（用 -v 把本地数据目录挂载进容器后作为参数传入）：
#   docker run --rm -v "$PWD:/data" jsonlqc /data/testdata/sample.jsonl

# ==================== 构建阶段 ====================
# 与 go.mod 中声明的 go 1.26 保持一致。
FROM golang:1.26-alpine AS builder

WORKDIR /src

# 仅复制构建所需文件：依赖清单 + Go 源码（无第三方依赖，无需 go mod download）。
COPY go.mod ./
COPY *.go ./

# 静态编译：scratch 无 libc 与动态链接器，必须关闭 CGO。
# -trimpath 去除本机绝对路径；-ldflags "-s -w" 剥离符号与调试信息以缩小体积。
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/jsonlqc .

# ==================== 运行阶段 ====================
# scratch：完全空白的根文件系统。本工具仅读取本地文件、不做网络请求，
# 因此无需 CA 证书、时区数据、/etc/passwd 等运行时文件。
FROM scratch

LABEL org.opencontainers.image.title="jsonlqc" \
      org.opencontainers.image.description="JSONL 数据质检命令行工具（纯标准库实现）" \
      org.opencontainers.image.source="https://github.com/sunding009/jsonlqc" \
      org.opencontainers.image.licenses="MIT"

COPY --from=builder /out/jsonlqc /jsonlqc

# 默认入口；数据文件通过 -v 挂载后作为位置参数传入。
ENTRYPOINT ["/jsonlqc"]
