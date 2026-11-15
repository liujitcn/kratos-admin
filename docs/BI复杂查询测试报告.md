# BI 复杂查询测试报告

测试日期：2026-11-14。测试对象为当前工作区的 BI 语义模型、参数化 SQL、公共指标和卡片发布实现。

## 结论

cloud-scp MySQL 的五表关联、字典两表查询及公共指标同步闭环通过。测试发现并修复了配置 SQL 结果被 Go 再次排序的问题。

cloud-scp Doris 的百万行事实表聚合、筛选和租户参数隔离通过，但真实交易表的复合键多表模型仍无法保存。现有模型安全拒绝了无法证明单列唯一性的关系，不能将此项视为 Doris 多表支持通过。

## 环境和数据范围

| 数据源 | 实际数据库 | 版本 | 本轮用途 |
| --- | --- | --- | --- |
| cloud-scp 管理库 | `cloud_scp_test` | MySQL 8.0.46 | 实际表结构发现、五表关联、字典查询、卡片与指标闭环 |
| cloud-scp SCP 业务库 | `cloud_scp` | Doris 3.0.8，协议版本返回 5.7.99 | 百万行交易聚合、隔离验证、复合键关联反例 |
| 测试元数据库 | 自动创建的临时 SQLite | 项目现有测试驱动 | 数据源配置、模型、查询记录、卡片、仪表盘和发布快照 |

真实数据库仅执行结构读取和只读查询。BI 测试配置均写入临时 SQLite，测试结束自动清理；本轮没有向管理端的永久数据源列表新增 cloud-scp 配置。

| 实际表 | 行数 |
| --- | ---: |
| MySQL `base_user` | 2 |
| MySQL `base_dept` | 4 |
| MySQL `base_role` | 5 |
| MySQL `base_post` | 2 |
| MySQL `base_tenant` | 1 |
| MySQL `base_dict_item` | 264 |
| MySQL `base_dict` | 74 |
| Doris `account_debit_rec` | 1,020,057 |
| Doris `base_user` | 563 |
| Doris `base_dept` | 8 |
| Doris `dev_info` | 336 |
| Doris `dev_group_catalog` | 385 |

## 执行结果

共执行 **36 个真实数据源叶子用例**。另执行 2 个离线排序回归用例、4 个既有 Adapter 上限回归用例，合计 42 个定向用例通过。

| 测试组 | 用例数 | 覆盖范围 | 结果 |
| --- | ---: | --- | --- |
| MySQL 五表 | 23 | 用户→部门、角色、岗位、租户；5 个指标；双维分组；LEFT/INNER；乱序链式关系；eq/in/between/contains；组合筛选；时间范围；注入字符串；空聚合/NULL；Top-1；6 种非法关系；多卡指标同步与发布闭环 | 通过 |
| MySQL 字典两表 | 5 | 明细聚合、74 个字典分组、双维 Top-10、指标降序、请求 1000 行但源上限 100 行的截断 | 通过 |
| Doris | 8 | 复合键关联阻断、交易聚合、项目分组、项目 IN＋状态区间、空集、4 种租户参数、项目授权边界、完整/缺失复合键的五表独立对照 | 通过；包含已知限制的拒绝验证 |
| 离线排序回归 | 2 | 数字形式字符串按数据库升序/降序取 Top-N | 通过 |
| 既有 Adapter 回归 | 4 | 源上限、查询上限、默认上限、超量结果裁剪 | 通过 |

结果由独立只读 SQL 对照；数值以有理数比较，避免使用浮点容差掩盖金额差异。日志不输出用户姓名、账号、联系方式或凭据。

卡片闭环经过真实仓储和 Biz：两张多表卡片保存 SQL → 发布仪表盘 → 预览修改公共指标 → 同步两张卡片 → 当前及历史旧发布返回 409 → 重新发布后正常取数。公式 `COUNT(*) * 2` 只用于测试同步机制。

定向运行时，MySQL 查询的 BI 计时约 0–7 ms，Doris 的四个 BI 查询约 9–22 ms。该样本来自本地网络和已运行数据库，不能作为并发性能或冷缓存性能结论；五表 MySQL 事实数据只有 2 行。

后端完整 `go test ./...`（带实际数据源配置）退出码 0；`make lint` 退出码 0，报告 `0 issues`；`git diff --check` 通过。

## 发现和处理

### 已修复：重复排序改变数据库返回顺序

数据库按字符串规则排好 `"10"`、`"2"` 后，`BiQueryCase.execute` 又调用通用结果排序，把两者解析为数字，改变了顺序。真实字典查询的双维 Top-10 和源上限 100 行用例均复现了该问题。

配置 SQL 现在保留数据库的排序、截断和排序规则；Adapter 继续使用现有结果排序及裁剪逻辑。离线回归用例先复现失败，再验证修复；真实字典查询随后与独立 SQL 一致。

相关实现：`backend/internal/biz/bi/admin/bi_query.go`；稳定回归：`bi_query_order_test.go`。

### 未实现：Doris 复合键多表模型

本次连接的 Doris `information_schema.table_constraints` 与 `key_column_usage` 查询未返回可用唯一约束。真实表定义使用复合 UNIQUE KEY，例如：

- 交易：`tenant_id, project_id, transaction_id, sn`。
- 用户：`tenant_id, project_id, user_id`。
- 部门：`tenant_id, project_id, dept_id`。
- 设备：`tenant_id, project_id, device_id`。

现有模型只接受一对关联字段，并要求右字段具备单列唯一约束。编译器会注入租户条件，但没有表示项目等额外关联条件的契约。因此现有实现不能表达这些真实复合键关系。不能将复合键中的每列分别标为唯一来绕过校验。

真实数据中，用户表按 `tenant_id,user_id` 有 6 组重复键，部门有 1 组，设备编码有 4 组，设备分组有 115 组；加入 `project_id` 后这些组合均无重复。

独立执行交易→用户→部门、交易→设备→设备分组的五表 LEFT JOIN，结果如下。**这些五表 SQL 是独立对照，尚未通过 BI 模型执行。**

| 关联方式，租户 1 | 笔数 | 金额，分 |
| --- | ---: | ---: |
| 事实表基准 | 416,494 | 508,073,238 |
| 完整租户＋项目＋业务编号关联 | 416,494 | 508,073,238 |
| 遗漏项目条件 | 945,876 | 1,297,021,686 |

遗漏项目条件使笔数变为约 2.27 倍、金额变为约 2.55 倍。后续正式接入该 Doris 多表模型，需要同时支持可信的复合唯一键发现和多字段关联，并补充项目重复编号回归。本轮保留安全拒绝行为。

可信租户参数的独立绑定验证分别得到：租户 1 为 416,494 条，租户 2 为 402,393 条，租户 1000 为 201,170 条，不存在租户为 0 条。此项验证编译器及参数绑定层；配置源的项目范围调用仍按当前边界返回 403。

## 重复运行

集成测试：`backend/internal/biz/bi/admin/bi_cloud_scp_integration_test.go`。默认无配置时跳过，不要求 CI 提供业务库凭据。

在本机创建权限为 `0600` 的私有 JSON，填写实际只读连接；不要加入版本库：

```json
{
  "cloud-scp/default": {
    "driver": "mysql",
    "address": "127.0.0.1:3306",
    "database": "cloud_scp_test",
    "username": "本机填写",
    "password": "本机填写"
  },
  "cloud-scp/scp": {
    "driver": "doris",
    "address": "127.0.0.1:9030",
    "database": "cloud_scp",
    "username": "本机填写",
    "password": "本机填写"
  }
}
```

在 `backend` 执行完整测试组，测试依赖上述真实表和数据分布：

```sh
BI_CLOUD_SCP_CONNECTIONS=/absolute/path/private-connections.json \
  go test ./internal/biz/bi/admin \
  -run '^(TestCloudSCP|TestConfiguredSQLPreservesDatabaseOrder|TestAdapterExecutionLimit)' \
  -count=1 -v
```

离线回归无需私有配置：

```sh
go test ./internal/biz/bi/admin -run '^TestConfiguredSQLPreservesDatabaseOrder$' -count=1
```

本轮临时明文连接文件已删除。未覆盖浏览器交互、HTTP 传输、真实 AI 推理、并发压力、8 表极限及大数据量多表性能；权限引擎在测试中使用授权桩。Doris 的复合键模型接入限制仍需后续实现。
