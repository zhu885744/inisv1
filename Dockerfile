# 运行镜像：单个已内嵌 Mellow 前端的二进制（GoReleaser 构建 dockers_v2 时使用）
#
# 构建上下文由 GoReleaser 准备：它会先把当前平台的二进制放进临时上下文，
# 文件名与本项目 builds.binary 一致（inis），所以这里直接 COPY inis。
# 手动构建示例：
#   go build -tags embed -trimpath -ldflags "-s -w" -o inis main.go
#   docker build -t inis:local .
FROM alpine:3.21

# ca-certificates：HTTPS 调用（短信 / 对象存储 / OAuth / 邮件）
# tzdata：时区（容器默认 TZ=Asia/Shanghai，可用 -e TZ= 覆盖）
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S inis \
    && adduser -S -G inis -h /app inis

WORKDIR /app

# 二进制已在构建机完成 strip + UPX，这里只补权限与数据目录
COPY inis /usr/local/bin/inis
RUN chmod +x /usr/local/bin/inis \
    && mkdir -p /app/config /app/runtime /app/storage \
    && chown -R inis:inis /app

# 非 root 运行：程序只写 /app 下的 config / runtime / storage
USER inis

ENV TZ=Asia/Shanghai
EXPOSE 8642

# 首次启动会在 /app 下生成 config：数据库、短信、存储密钥都在这里，务必挂载持久化
VOLUME ["/app/config", "/app/runtime", "/app/storage"]

ENTRYPOINT ["/usr/local/bin/inis"]
