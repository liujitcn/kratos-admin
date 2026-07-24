FROM alpine:3.22

LABEL maintainer="Liu Jun<liujun@newcapec.net>"
ENV VERSION=1.0
ENV LANG=C.UTF-8

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN echo "Asia/Shanghai" > /etc/timezone
RUN mkdir -p /app/data /app/configs /app/certs

VOLUME ["/app/data", "/app/configs", "/app/certs"]

COPY bin/server /app/server
COPY configs /app/configs

EXPOSE 6001 7001

ENTRYPOINT ["/app/server", "--conf", "./configs"]
