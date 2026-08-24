FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY bin/server ./server
COPY configs ./configs
COPY data/admin /opt/kratos-admin/static/admin
COPY data/uni-app /opt/kratos-admin/static/uni-app
COPY data/taro-app /opt/kratos-admin/static/taro-app
COPY scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN chmod +x /usr/local/bin/docker-entrypoint.sh \
    && mkdir -p /app/data

VOLUME ["/app/data"]

EXPOSE 6001 7001

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["./server", "-c", "./configs", "-e", "dev"]
