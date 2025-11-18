# 多阶段构建 Dockerfile for Go Gin application
# Stage 1: Build stage
FROM golang:1.23.4-alpine AS builder

# 安装必要的包和工具
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

# 设置工作目录
WORKDIR /app

# 复制go.mod和go.sum文件
COPY go.mod go.sum ./

# 下载依赖包（利用Docker层缓存）
RUN go mod download

# 复制源代码
COPY . .

# 构建应用程序
# 禁用CGO以获得静态二进制文件
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o achobeta.server.forge ./cmd

# Stage 2: Production stage
FROM alpine:latest

# 安装必要的包
RUN apk --no-cache add ca-certificates tzdata

# 创建非root用户
RUN adduser -D -g '' appuser

# 设置工作目录
WORKDIR /app

# 复制构建的二进制文件
COPY --from=builder /app/achobeta.server.forge .

# 复制配置文件和模板文件（通过数据卷挂载方式）
# 在部署时将配置文件挂载到 /app/conf 目录


# 确保二进制文件可执行
RUN chmod +x /app/achobeta.server.forge

# 更改文件所有者
RUN chown -R appuser:appuser /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 运行应用程序
CMD ["./achobeta.server.forge"]