# 完整配置模板

本目录按当前 `kratos-kit/api/proto/config/v1` 的 `Bootstrap` 配置保留全部字段，并额外保留启动根密钥 `key.yaml`。模板只用于复制、裁剪和填写，不作为默认启动配置直接使用。默认启动配置在上级 `backend/configs` 目录。

## 文件对应关系

| 文件 | 配置内容 |
| --- | --- |
| `server.yaml` | HTTP、gRPC、MCP、SSE 服务端 |
| `client.yaml` | HTTP、gRPC、MCP、SSE 客户端 |
| `data.yaml` | 数据库、Redis、队列 |
| `trace.yaml` | 链路追踪 |
| `logger.yaml` | 日志组件 |
| `registry.yaml` | 服务注册与发现 |
| `config.yaml` | 远程配置中心 |
| `oss.yaml` | 对象存储 |
| `notify.yaml` | 通知与短信 |
| `auth.yaml` | JWT 与会话认证 |
| `authz.yaml` | Casbin 授权 |
| `pprof.yaml` | Pyroscope 性能分析 |
| `ai.yaml` | 云端或本地大模型 |
| `translator.yaml` | 机器翻译 |
| `mfa.yaml` | MFA、TOTP、WebAuthn |
| `key.yaml` | 根密钥 Provider |

## 模板字段

- `authz.yaml` 的 `authz.casbin.exclude` 包含一个空白结构示例，未使用时删除该元素。
- `data.yaml` 的 `data.databases.example` 是按名称配置数据库的结构示例，使用时将 `example` 改为实际名称或删除。
- `server.yaml` 的 `server.mcp.http_tools[0]` 是 HTTP Tool 的完整结构示例，使用时替换或删除。
- 空 map 和空 list 也是 Proto 字段的一部分，不要为了精简模板删除字段。
- `duration` 字段使用 ProtoJSON duration 格式，数值以 `s` 结尾，例如 `5s`、`0.2s`、`300s`。
- `bytes` 字段使用字符串填写 PEM 文本或其他字节内容；`tls.config` 下的证书字段对应 `cert_pem`、`key_pem`、`ca_pem`。

## 固定值

### AI

`ai.model.type`：

- `MODEL_TYPE_UNSPECIFIED`：未指定。
- `CLOUD_MODEL`：云端模型，使用 `cloud` 配置。
- `LOCAL_MODEL`：本地 Ollama 模型，使用 `local` 配置。

### 认证与客户端

`authn.jwt.method`、`client.*.middleware.auth.method` 使用 JWT 签名算法名称，常用值为 `HS256`、`HS384`、`HS512`、`RS256`、`RS384`、`RS512`。

`client.*.middleware.retry.retry_codes` 使用 gRPC 状态码名称，例如 `UNAVAILABLE`、`RESOURCE_EXHAUSTED`、`DEADLINE_EXCEEDED`。`client.*.mcp.transport`：

可填写的标准 gRPC 状态码包括 `OK`、`CANCELED`、`UNKNOWN`、`INVALID_ARGUMENT`、`DEADLINE_EXCEEDED`、`NOT_FOUND`、`ALREADY_EXISTS`、`PERMISSION_DENIED`、`UNAUTHENTICATED`、`RESOURCE_EXHAUSTED`、`FAILED_PRECONDITION`、`ABORTED`、`OUT_OF_RANGE`、`UNIMPLEMENTED`、`INTERNAL`、`UNAVAILABLE`、`DATA_LOSS`。

- `UNSPECIFIED`：未指定，默认使用 HTTP。
- `HTTP`：Streamable HTTP。
- `SSE`：Legacy SSE。
- `STDIO`：启动子进程，通过标准输入输出通信。

### 服务端传输

`server.sse.transport`：

- `UNSPECIFIED`：未指定，默认使用 HTTP。
- `HTTP`：独立监听端口的 SSE 服务。
- `IN_PROCESS`：进程内 SSE 服务。

`server.mcp.transport`：

- `UNSPECIFIED`：未指定。
- `HTTP`：独立监听端口的 Streamable HTTP MCP 服务。
- `SSE`：独立监听端口的 Legacy SSE MCP 服务。
- `STDIO`：通过标准输入输出运行 MCP 服务。
- `IN_PROCESS`：进程内 MCP 服务。

`server.mcp.http_tools[].parameters[].location`：

- `HTTP_PARAM_LOCATION_UNSPECIFIED`：未指定。
- `HTTP_PARAM_LOCATION_PATH`：写入 URL 路径变量。
- `HTTP_PARAM_LOCATION_QUERY`：写入 URL 查询参数。
- `HTTP_PARAM_LOCATION_HEADER`：写入 HTTP Header。
- `HTTP_PARAM_LOCATION_BODY`：写入请求 Body。

`server.mcp.http_tools[].body_mode`：

- `HTTP_BODY_MODE_UNSPECIFIED`：未指定。
- `HTTP_BODY_MODE_NONE`：不发送 Body。
- `HTTP_BODY_MODE_JSON`：JSON 编码 Body。
- `HTTP_BODY_MODE_FORM`：表单编码 Body。
- `HTTP_BODY_MODE_RAW`：使用 `body_template` 原始模板。

`server.mcp.http_tools[].parameters[].type` 使用 JSON Schema 类型名称：`string`、`integer`、`number`、`boolean`、`object`、`array`。`server.mcp.http_tools[].method` 使用标准 HTTP 方法名称，例如 `GET`、`POST`、`PUT`、`PATCH`、`DELETE`。

`server.http.network`、`server.grpc.network`、`server.mcp.network`、`server.sse.network` 通常填写 `tcp`，也可按监听需求使用 `tcp4` 或 `tcp6`。`server.sse.codec` 当前使用 `json`。

### 数据、密钥与存储

- `data.database.driver`：`mysql`、`postgres`、`sqlite`。
- `key.type`：`file`、`vault`、`aws`、`google`、`azure`、`kubernetes`。
- `oss.type`：`local`、`aliyun`、`ftp`、`minio`、`s3`。
- `oss.upload_security.command` 是扫描器可执行文件路径，例如 `clamdscan`，不是枚举值。

### 日志、注册与配置中心

- `logger.type`：`zap`、`logrus`、`fluent`、`aliyun`、`tencent`、`zerolog`。
- `logger.zap.level`、`logger.logrus.level`、`logger.zerolog.level`：`debug`、`info`、`warn`、`error`、`dpanic`、`panic`、`fatal`。
- `logger.logrus.formatter`：`text` 或 `json`。
- `logger.zerolog.writer`：`stdout`、`stderr`、`console`、`file`、`lumberjack`；使用 `file` 或 `lumberjack` 时填写 `filepath`。
- `registry.type`：`consul`、`etcd`、`zookeeper`、`nacos`、`kubernetes`、`eureka`、`polaris`、`servicecomb`。
- `config.type`：`etcd`、`consul`、`nacos`、`apollo`、`kubernetes`、`polaris`；留空表示不启用远程配置中心。

### 其他组件

- `notify.type`：`sms`；留空表示不启用通知发送器。
- `pprof.type`：`pyroscope`。
- `trace.exporter`：`otlp-grpc`、`otlp-http`、`zipkin`、`stdout`。
- `translator.type`：`google`、`baidu`、`alibaba`、`volc`。
- `translator.google.version`：`v1`、`v2`、`v3`。
- `mfa.totp.algorithm`：`SHA1`、`SHA256`、`SHA512`。

## 使用方式

完整配置目录不会与上级最小配置目录合并。确认已填写需要启用的组件后，通过 `make -C backend run-full` 使用该目录启动；日常本地启动使用 `make -C backend run-minimal`。
