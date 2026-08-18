# 项目文档

本文档目录只保留当前实现、操作流程和稳定约束。历史调研、候选方案和已经废弃的实施草稿不作为项目依据。

## 使用入口

| 主题 | 文档 | 适用场景 |
| --- | --- | --- |
| 系统总体设计 | [系统总体设计](系统总体设计.md) | 了解模块边界、运行关系和生成链路 |
| 服务接入 | [服务接入指南](服务接入指南.md) | 新增后端、管理端或应用端业务 |
| 数据库迁移 | [数据库与初始化数据设计](数据库与初始化数据设计.md) | 新表、初始化数据、菜单和权限迁移 |
| 接口校验 | [接口参数校验设计](接口参数校验设计.md) | 修改 Proto 字段、校验和前端表单 |
| 登录与密码 | [登录与密码加密流程](登录与密码加密流程.md) | 排查登录、OAuth、密码和 Token |
| AI 助手 | [AI助手设计](AI助手设计.md) | 接入或排查 AI 会话和流式消息 |
| 前端组件 | [前端组件清单](前端组件清单.md) | 复用管理端、uni-app、Taro 组件 |
| 国际化设计 | [国际化最终方案](国际化最终方案.md) | 了解固定文案、动态翻译和运行时回退 |
| 新增语言 | [国际化语言扩展指南](国际化语言扩展指南.md) | 增加语言包、生成产物和数据库迁移 |

## 后端细则

后端目录下的文档面向后端开发，规则更具体：

- [新增业务流程](../backend/docs/new-feature.md)：后端能力的最短闭环。
- [Proto 与 HTTP 契约](../backend/docs/api.md)：协议、路径、生成和权限同步。
- [错误处理](../backend/docs/errors.md)：错误分类、metadata 和分层职责。

## 文档维护

- 文档只描述仓库当前已经存在的能力；规划中的能力必须明确标记为“未实现”，不能写成现状。
- 命令、路径、包名和接口以代码、Makefile、Proto 和 package `exports` 为准。
- 修改 README、`docs` 或后端细则后，执行 `make i18n-docs` 更新内嵌项目文档。
- `I18N_LOCALES` 使用逗号分隔的 BCP 47 语言代码列表，同时控制 OpenAPI 和项目 Markdown 文档的目标语言；`make i18n-docs` 先收集 Markdown，再由 `../kratos-kit/cmd/i18n/project_docs.py` 补充缺失语言；`make i18n-openapi` 生成 OpenAPI 多语言 YAML。无网络环境可设置 `I18N_OFFLINE=1`。
- 生成的 `backend/internal/docs/assets/docs.json` 和 `backend/internal/docs/docs.go` 只能通过 `make i18n-docs` 更新。
