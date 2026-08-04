# 新增业务流程

本文是仓库内新增一项后端和前端业务能力的最短闭环。跨模块详细示例见 [../../docs/服务接入指南.md](../../docs/服务接入指南.md)。

## 开发顺序

1. 读取目标目录的 `AGENTS.md`，确认能力属于 `base`、`system.admin`、`system.app` 或新的外部模块。
2. 有新表或字段时，先按 `configs/data.yaml` 连接开发库并实际调整表结构，再执行 `make gorm-gen`。
3. 在 `api/proto` 定义接口、HTTP 路径、OpenAPI 描述和 `buf.validate`。
4. 执行 `make api openapi`；按消费端执行 `make ts`、`make ts-uni-app` 或 `make ts-taro-app`。
5. 实现 biz、service 和传输注册；依赖集合变化后执行 `make wire`。
6. 实现管理端或应用端请求与页面，类型只使用生成 RPC。
7. 同步版本化迁移中的默认数据、菜单、按钮和服务方法权限。
8. 完成后统一执行生成、后端测试和对应前端检查。

## 迁移结构

迁移目录按“版本 → 数据库类型 → 数据源”组织：

```text
backend/migration/assets/vX.Y.Z/mysql/<feature>.up.sql
backend/migration/assets/vX.Y.Z/mysql/<feature>.description.md
backend/migration/assets/vX.Y.Z/mysql/<data-source>/<feature>.up.sql
```

MySQL 默认数据源使用 `mysql` 直系文件；命名数据源放一级子目录。一个功能的 SQL 和描述使用相同文件名并与代码一起交付。脚本应可重复执行，不使用 `TRUNCATE` 或无条件删除业务数据。

## 生成命令

```bash
cd backend
make api openapi ts ts-uni-app ts-taro-app gorm-gen wire fmt
# 或执行全部：make gen
```

只执行改动涉及的前端 RPC 生成，避免把无关端的生成结果带入变更。生成产物只能由这些命令更新。

## 验证

```bash
cd backend && go test ./...
cd frontend/admin && pnpm lint:oxlint && pnpm type:check
cd frontend/uni-app && pnpm lint && pnpm tsc
cd frontend/taro-app && pnpm lint && pnpm tsc
```

只修改单个前端时执行对应前端检查；接口、生成器、模块契约或公共运行时变化时需要扩大验证范围。
