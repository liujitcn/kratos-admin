# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS backend-builder

ARG TARGETOS
ARG TARGETARCH
ARG BUILD_FLAGS
ARG GOPROXY=https://proxy.golang.org,direct

WORKDIR /src

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build $BUILD_FLAGS -o /out/server ./internal/cmd/server

FROM alpine:3.22

ENV TZ=Asia/Shanghai

RUN apk add --no-cache ca-certificates gzip mysql-client openssl tzdata \
    && ln -sf /usr/share/zoneinfo/$TZ /etc/localtime \
    && echo $TZ > /etc/timezone \
    && addgroup -S kratos \
    && adduser -S -G kratos -h /app kratos \
    && mkdir -p /app/data /app/logs /app/backups /opt/kratos \
    && chown -R kratos:kratos /app /opt/kratos

WORKDIR /app

COPY --from=backend-builder --chown=kratos:kratos /out/server ./server
COPY --chown=kratos:kratos configs ./configs
COPY --chown=kratos:kratos configs /opt/kratos/configs
COPY --chown=kratos:kratos web/admin /opt/kratos/web/admin
COPY --chown=kratos:kratos web/uni-app /opt/kratos/web/uni-app
COPY --chown=kratos:kratos web/taro-app /opt/kratos/web/taro-app
COPY --chown=kratos:kratos scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN chmod +x /usr/local/bin/docker-entrypoint.sh

VOLUME ["/app/data", "/app/logs", "/app/backups"]

EXPOSE 6001 7001

USER kratos:kratos

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["./server", "-c", "./configs", "-e", "prod"]
