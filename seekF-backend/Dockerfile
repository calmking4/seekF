# 第一阶段：使用 Go 编译后端和 MCP。
FROM golang:1.25-alpine AS builder

WORKDIR /src
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN go build -o /out/seekf-server ./cmd/seekf-server
RUN go build -o /out/mcp-server ./cmd/mcp-server

# 第二阶段：只放入编译好的程序，不包含 Go 编译器和源码。
FROM alpine:3.22

WORKDIR /app
COPY --from=builder /out/seekf-server /out/mcp-server /app/bin/

# 后端通过此路径启动 MCP 子进程。
ENV MCP_SERVER_COMMAND=/app/bin/mcp-server

# 配置文件运行时挂载到 /app/config/config.toml，密钥通过环境变量注入。
EXPOSE 8080
ENTRYPOINT ["/app/bin/seekf-server"]
