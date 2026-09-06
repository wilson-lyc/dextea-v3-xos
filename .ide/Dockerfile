# .ide/Dockerfile
# 云原生开发（Cloud Native Build）环境镜像：CNB 默认开发环境 + Go 工具链
#
# 基础镜像 cnbcool/default-dev-env 基于 debian:trixie，已内置 code-server
# （因此本镜像走单容器模式）、git、curl、Node.js 等，代码目录为 /workspace。
#
# 用法见仓库根目录 .cnb.yml 的 docker.build：镜像按内容哈希缓存并推送到制品库，
# 本文件未变更时直接复用缓存镜像，不会每次启动都重新构建。

FROM cnbcool/default-dev-env:latest

# Go 版本，需 >= go.mod 中 go 指令声明的版本（当前为 go 1.27.0）
ARG GO_VERSION=1.27.1

# Go 环境变量
# GOMODCACHE / GOCACHE 显式声明，便于在「个人设置 > 云原生开发」中把这两个目录
# 配成自定义漫游目录，环境重建后可复用缓存，避免重复下载依赖
ENV GOPATH=/go \
    GOBIN=/go/bin \
    GOMODCACHE=/go/pkg/mod \
    GOCACHE=/root/.cache/go-build \
    GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.google.cn \
    PATH=/usr/local/go/bin:/go/bin:$PATH

# 安装 Go 工具链：官方预编译包，按容器架构自动选择
RUN set -eux; \
    arch="$(dpkg --print-architecture 2>/dev/null || uname -m)"; \
    case "$arch" in \
        amd64|x86_64) go_arch=amd64 ;; \
        arm64|aarch64) go_arch=arm64 ;; \
        *) echo "unsupported arch: $arch" >&2; exit 1 ;; \
    esac; \
    curl -fsSLO "https://go.dev/dl/go${GO_VERSION}.linux-${go_arch}.tar.gz"; \
    tar -C /usr/local -xzf "go${GO_VERSION}.linux-${go_arch}.tar.gz"; \
    rm -f "go${GO_VERSION}.linux-${go_arch}.tar.gz"; \
    mkdir -p /go/bin /go/pkg/mod /root/.cache/go-build; \
    go version

# 编译工具链：cgo、go test -race 以及部分依赖的构建需要 gcc / make
# 上游镜像约定开发环境保留 apt 缓存，故此处不执行 apt-get clean，也不删除 lists
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        gcc \
        libc6-dev \
        make \
    && gcc --version

# VSCode Go 扩展（语法高亮、跳转、诊断、调试面板）
RUN code-server --install-extension golang.go

# Go 开发工具：语言服务 gopls、import 整理 goimports、调试器 dlv
# 如需 golangci-lint / staticcheck 等，在此追加 go install 命令即可
RUN go install golang.org/x/tools/gopls@latest \
    && go install golang.org/x/tools/cmd/goimports@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest

# 常用验证命令：
#   go build ./...            编译校验
#   go vet ./...              静态检查
#   go test ./...             单元测试
#   go test -race ./...       竞态检测（已装 gcc，可用）
#   go run ./cmd/server -config configs/config.local.yaml
