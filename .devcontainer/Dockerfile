# 使用golang:1.21-alpine3.18作为基础镜像（与生产环境保持一致）
FROM golang:1.21-alpine3.18

# 安装Node.js和开发工具
RUN sed -i "s@dl-cdn.alpinelinux.org@mirrors.aliyun.com@g" /etc/apk/repositories \
    && apk add --no-cache bash curl gcc git musl-dev nodejs npm tzdata su-exec ca-certificates

# 配置npm使用国内源
RUN npm config set registry https://registry.npmmirror.com/ \
    && npm install -g pnpm \
    && pnpm config set registry https://registry.npmmirror.com/

# 配置Go使用多个代理源，提高可靠性
RUN go env -w GOPROXY=https://goproxy.cn,https://goproxy.io,https://proxy.golang.org,direct \
    && go env -w GO111MODULE=on \
    && go install -a -v github.com/go-bindata/go-bindata/...@latest \
    && go install -a -v github.com/elazarl/go-bindata-assetfs/...@latest

# 设置工作目录
WORKDIR /workspace

# 预装一些常用的Go开发工具（使用与Go 1.21兼容的版本）
# 分开执行安装命令，即使某个安装失败也不会终止整个构建
RUN go install -v golang.org/x/tools/gopls@v0.14.2 || echo "Failed to install gopls, continuing..."
RUN go install -v github.com/go-delve/delve/cmd/dlv@v1.21.0 || echo "Failed to install dlv, continuing..."
RUN go install -v honnef.co/go/tools/cmd/staticcheck@v0.4.6 || echo "Failed to install staticcheck, continuing..."
