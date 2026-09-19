# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS backend-builder

ARG TARGETOS
ARG TARGETARCH
ARG BUILD_FLAGS

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

RUN apk add --no-cache ca-certificates gzip mysql-client openssl tzdata \
    && addgroup -S kratos \
    && adduser -S -G kratos -h /app kratos \
    && mkdir -p /app/data /app/web /app/logs /app/backups /opt/kratos-admin \
    && chown -R kratos:kratos /app /opt/kratos-admin

WORKDIR /app

COPY --from=backend-builder --chown=kratos:kratos /out/server ./server
COPY --chown=kratos:kratos configs ./configs
COPY --chown=kratos:kratos configs /opt/kratos-admin/configs
COPY --chown=kratos:kratos migration/assets ./migration/assets
COPY --chown=kratos:kratos web /opt/kratos-admin/static
COPY --chown=kratos:kratos scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN chmod +x /usr/local/bin/docker-entrypoint.sh

VOLUME ["/app/data", "/app/logs", "/app/backups"]

EXPOSE 6001 7001

USER kratos:kratos

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["./server", "-c", "./configs", "-e", "prod"]
