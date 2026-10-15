# @liujitcn/kratos-admin-rag

RAG 知识库管理模块，属于当前管理端默认宿主的一部分。模块只通过管理端 core 的公开入口复用表格、表单、请求和权限能力。

## 功能

- 知识库分页、新增、编辑、删除和启停。
- 绑定 embedding 模型并保存向量维度配置。
- 文档分页、文本或文件上传、删除和重新处理。
- 文档处理状态、错误信息、文件哈希和切片数量展示。
- 按知识库检索文档切片，供 AI 会话选择知识库并注入检索上下文。

## 目录

```text
src
├── api/rag/admin/v1          # 知识库、文档和检索请求
├── rpc/rag/admin/v1          # 由 Proto 生成的类型
├── views/ai-knowledge        # 知识库管理页面
├── locales                   # zh-CN、zh-TW、en-US、ja-JP
├── module.ts                 # ragAdminModule 模块定义
└── index.ts                  # 公共入口
```

后端契约位于 `backend/api/proto/rag/admin/v1`，RPC 只能通过仓库命令生成：

```bash
make -C frontend ts-admin
```

知识库数据由后端按 `data.databases.rag` 配置保存；未配置独立数据源时使用默认数据源。模型凭据不由本模块保存或回传，统一由后端敏感字段接口处理。

## 宿主装配

默认宿主在 `apps/admin/src/module-manifest.ts` 中按 `system`、`rag` 顺序加载本模块。页面路由和语言包由模块定义提供，菜单和 API 权限由后端初始化资源控制。

## 验证

```bash
pnpm --filter @liujitcn/kratos-admin-rag type:check
pnpm --filter @liujitcn/kratos-admin-rag test
```
