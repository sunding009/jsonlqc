# jsonlqc 项目 Makefile

GO     ?= go
BINARY := jsonlqc
PKG    := .

.PHONY: all build test lint vet fmt clean docker-build help

all: build

## 编译二进制（-trimpath 去路径；-s -w 剥离符号与调试信息缩小体积）
build:
	$(GO) build -trimpath -ldflags="-s -w" -o $(BINARY) $(PKG)

## 运行单元测试（-race 开启数据竞争检测）
test:
	$(GO) test -race -v ./...

## go vet 静态检查（常见代码缺陷、可疑构造）
vet:
	$(GO) vet ./...

## 代码质量检查：gofmt 格式 + go vet（等价于 CI 中的质量门禁）
lint: vet
	@echo "==> gofmt 检查"
	@test -z "$$(gofmt -l .)" || { echo "以下文件未通过 gofmt，请运行 make fmt:"; gofmt -l .; exit 1; }
	@echo "==> 全部检查通过"

## 格式化所有 Go 源文件
fmt:
	gofmt -w .

## 清理本地构建产物
clean:
	rm -f $(BINARY)

## 构建 Docker 镜像（多阶段构建 + scratch 基础镜像）
docker-build:
	docker build -t $(BINARY) .

## 显示本帮助
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //'
