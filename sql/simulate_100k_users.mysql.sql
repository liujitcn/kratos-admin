-- ============================================================================
-- kratos-admin 压测模拟数据：10 万用户 + 部门树 + 角色 + 岗位
-- 目标库：MySQL 8.x（192.168.60.83:3306/kratos_admin，默认租户 0000，tenant_id = 1）
--
-- 结构：
--   部门 209 个：模拟集团(根) → 8 个一级部门 → 各 5 个二级(40) → 各 4 个三级(160)
--   角色 8 个：sim_manager / sim_lead / sim_staff / sim_finance / sim_hr
--              / sim_sales / sim_support / sim_viewer（data_scope 覆盖 1/2/3/4）
--   岗位 10 个：sim_ceo / sim_dept_mgr / sim_lead / sim_sr_eng / sim_eng
--               / sim_specialist / sim_assistant / sim_finance / sim_hr / sim_cs
--   用户 100000 个：sim_user_000001 ~ sim_user_100000
--     - 分散映射（三维度独立）：部门 = MOD(n-1,160)，角色 = MOD(FLOOR((n-1)/160),8)，
--       岗位 = MOD(FLOOR((n-1)/1280),10)；每个部门内约 625 人含 8 种角色 × 10 种岗位
--     - 创建时间：2026-01-01 ~ 2026-12-31 全年均匀分布（每天约 274 人，
--       时分秒按 n*7919 伪随机散开），password_changed_at / updated_at 同 created_at
--     - 密码与 super 相同：New@20266（复用其 bcrypt 哈希，不强制改密）
--     - 手机号 18000000001 ~ 18000100000，邮箱 sim_user_xxxxxx@sim.test
--
-- 约定：所有模拟行 remark 统一为 'sim-data'，清理脚本按此识别。
-- 前提：压测库独占执行（脚本按“多行插入 id 连续”做父子映射，无并发才成立）。
-- 预计耗时：< 1 分钟。
-- ============================================================================

-- CTE 递归深度上限放大（每批 1 万序号）
SET SESSION cte_max_recursion_depth = 200000;

-- ============================================================================
-- 0. 清理段（重跑前手动启用：去掉注释按序执行）
-- ============================================================================
-- DELETE FROM base_user WHERE remark = 'sim-data';
-- DELETE FROM base_dept WHERE remark = 'sim-data';
-- DELETE FROM base_role WHERE remark = 'sim-data';
-- DELETE FROM base_post WHERE remark = 'sim-data';

-- ============================================================================
-- 1. 新增部门（209 个，分四层插入，逐层回填 path）
-- ============================================================================

-- 1.1 模拟集团根部门（挂在现有根部门 id=1 下）
INSERT INTO base_dept (tenant_id, parent_id, path, name, sort, status, remark, created_by, updated_by, created_at, updated_at, deleted_at)
VALUES (1, 1, '', '模拟集团', 900, 1, 'sim-data', 1, 1, NOW(), NOW(), 0);
SET @sim_root := LAST_INSERT_ID();
SET @sim_root_path := CONCAT('/0/', @sim_root);
UPDATE base_dept SET path = @sim_root_path WHERE id = @sim_root;

-- 1.2 一级部门 8 个（显式 VALUES，保证 id 连续）
INSERT INTO base_dept (tenant_id, parent_id, path, name, sort, status, remark, created_by, updated_by, created_at, updated_at, deleted_at) VALUES
(1, @sim_root, '', '01-研发中心',   901, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, @sim_root, '', '02-市场部',     902, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, @sim_root, '', '03-销售部',     903, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, @sim_root, '', '04-财务部',     904, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, @sim_root, '', '05-人力资源部', 905, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, @sim_root, '', '06-运营部',     906, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, @sim_root, '', '07-客服部',     907, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, @sim_root, '', '08-行政部',     908, 1, 'sim-data', 1, 1, NOW(), NOW(), 0);
SET @d1 := LAST_INSERT_ID();
UPDATE base_dept SET path = CONCAT(@sim_root_path, '/', id)
WHERE id BETWEEN @d1 AND @d1 + 7;

-- 1.3 二级部门 40 个：每个一级部门下 5 个组
INSERT INTO base_dept (tenant_id, parent_id, path, name, sort, status, remark, created_by, updated_by, created_at, updated_at, deleted_at)
SELECT 1, d.id, '', CONCAT(d.name, '-', g.gn, '组'), 900 + g.gn, 1, 'sim-data', 1, 1, NOW(), NOW(), 0
FROM base_dept d
CROSS JOIN (SELECT 1 gn UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5) g
WHERE d.tenant_id = 1 AND d.id BETWEEN @d1 AND @d1 + 7
ORDER BY d.id, g.gn;
SET @d2 := LAST_INSERT_ID();
UPDATE base_dept c JOIN base_dept p ON c.parent_id = p.id
SET c.path = CONCAT(p.path, '/', c.id)
WHERE c.id BETWEEN @d2 AND @d2 + 39;

-- 1.4 三级部门 160 个：每个二级部门下 4 个课（10 万用户全部分布在这一层）
INSERT INTO base_dept (tenant_id, parent_id, path, name, sort, status, remark, created_by, updated_by, created_at, updated_at, deleted_at)
SELECT 1, p.id, '', CONCAT(p.name, '-', g.gn, '课'), 900 + g.gn, 1, 'sim-data', 1, 1, NOW(), NOW(), 0
FROM base_dept p
CROSS JOIN (SELECT 1 gn UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4) g
WHERE p.tenant_id = 1 AND p.id BETWEEN @d2 AND @d2 + 39
ORDER BY p.id, g.gn;
SET @d3 := LAST_INSERT_ID();
UPDATE base_dept c JOIN base_dept p ON c.parent_id = p.id
SET c.path = CONCAT(p.path, '/', c.id)
WHERE c.id BETWEEN @d3 AND @d3 + 159;

-- ============================================================================
-- 2. 新增角色 8 个（menus 复用“普通用户 user”角色的移动端菜单，最小权限）
--    data_scope：1 全部 / 2 部门及子部门 / 3 本部门 / 4 仅本人
-- ============================================================================
INSERT INTO base_role (tenant_id, name, code, data_scope, menus, status, remark, created_by, updated_by, created_at, updated_at, deleted_at) VALUES
(1, '模拟-部门经理',   'sim_manager',  2, '[99000000,99010000,99010100,99010101,99010500,99050000,99050100,99100000,99100100,99100200,99100500,99101000]', 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-组长',       'sim_lead',     3, '[99000000,99010000,99010100,99010101,99010500,99050000,99050100,99100000,99100100,99100200,99100500,99101000]', 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-普通员工',   'sim_staff',    4, '[99000000,99010000,99010100,99010101,99010500,99050000,99050100,99100000,99100100,99100200,99100500,99101000]', 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-财务专员',   'sim_finance',  4, '[99000000,99010000,99010100,99010101,99010500,99050000,99050100,99100000,99100100,99100200,99100500,99101000]', 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-人事专员',   'sim_hr',       4, '[99000000,99010000,99010100,99010101,99010500,99050000,99050100,99100000,99100100,99100200,99100500,99101000]', 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-销售',       'sim_sales',    4, '[99000000,99010000,99010100,99010101,99010500,99050000,99050100,99100000,99100100,99100200,99100500,99101000]', 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-客服',       'sim_support',  4, '[99000000,99010000,99010100,99010101,99010500,99050000,99050100,99100000,99100100,99100200,99100500,99101000]', 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-只读访客',   'sim_viewer',   3, '[99000000,99010000,99010100,99010101,99010500,99050000,99050100,99100000,99100100,99100200,99100500,99101000]', 1, 'sim-data', 1, 1, NOW(), NOW(), 0);
SET @r := LAST_INSERT_ID();

-- ============================================================================
-- 3. 新增岗位 10 个
-- ============================================================================
INSERT INTO base_post (tenant_id, name, code, sort, status, remark, created_by, updated_by, created_at, updated_at, deleted_at) VALUES
(1, '模拟-总经理',   'sim_ceo',          901, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-部门经理', 'sim_dept_mgr',     902, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-组长',     'sim_lead',         903, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-高级工程师', 'sim_sr_eng',     904, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-工程师',   'sim_eng',          905, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-专员',     'sim_specialist',   906, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-助理',     'sim_assistant',    907, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-财务',     'sim_finance',      908, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-人事',     'sim_hr',           909, 1, 'sim-data', 1, 1, NOW(), NOW(), 0),
(1, '模拟-客服',     'sim_cs',           910, 1, 'sim-data', 1, 1, NOW(), NOW(), 0);
SET @p := LAST_INSERT_ID();

-- ============================================================================
-- 4. 生成 10 万用户（分 10 批，每批 1 万；密码 = super 的 New@20266）
--    创建时间 = 2026-01-01 + (n-1) 对 365 取模的天数 + n*7919 伪随机秒数
--    → 全年 365 天均匀覆盖（每天约 274 人），12 个月每月约 8333 人
-- ============================================================================
SET @pwd := '$2a$10$QBoSIqVtmSblc2kHET3dG.TYFBTsmr6FIZrZlYO7SsLU.e5mFHuu2';

-- 批 1：sim_user_000001 ~ sim_user_010000
INSERT INTO base_user (tenant_id, user_name, user_code, nick_name, role_id, dept_id, post_id, phone, email, id_type, id_code, password, password_changed_at, password_history, must_change_password, gender, avatar, status, remark, created_by, updated_by, created_at, updated_at, deleted_at)
WITH RECURSIVE seq AS (SELECT 1 n UNION ALL SELECT n + 1 FROM seq WHERE n < 10000)
SELECT 1,
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('压测用户', LPAD(n, 6, '0')),
       @r + MOD(FLOOR((n - 1) / 160), 8),
       @d3 + MOD(n - 1, 160),
       @p + MOD(FLOOR((n - 1) / 1280), 10),
       CONCAT('18', LPAD(n, 9, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0'), '@sim.test'),
       NULL, '', @pwd,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '[]', 1, 2 + MOD(n, 2), '', 1, 'sim-data', 1, 1,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND, 0
FROM seq;

-- 批 2：sim_user_010001 ~ sim_user_020000
INSERT INTO base_user (tenant_id, user_name, user_code, nick_name, role_id, dept_id, post_id, phone, email, id_type, id_code, password, password_changed_at, password_history, must_change_password, gender, avatar, status, remark, created_by, updated_by, created_at, updated_at, deleted_at)
WITH RECURSIVE seq AS (SELECT 10001 n UNION ALL SELECT n + 1 FROM seq WHERE n < 20000)
SELECT 1,
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('压测用户', LPAD(n, 6, '0')),
       @r + MOD(FLOOR((n - 1) / 160), 8),
       @d3 + MOD(n - 1, 160),
       @p + MOD(FLOOR((n - 1) / 1280), 10),
       CONCAT('18', LPAD(n, 9, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0'), '@sim.test'),
       NULL, '', @pwd,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '[]', 1, 2 + MOD(n, 2), '', 1, 'sim-data', 1, 1,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND, 0
FROM seq;

-- 批 3：sim_user_020001 ~ sim_user_030000
INSERT INTO base_user (tenant_id, user_name, user_code, nick_name, role_id, dept_id, post_id, phone, email, id_type, id_code, password, password_changed_at, password_history, must_change_password, gender, avatar, status, remark, created_by, updated_by, created_at, updated_at, deleted_at)
WITH RECURSIVE seq AS (SELECT 20001 n UNION ALL SELECT n + 1 FROM seq WHERE n < 30000)
SELECT 1,
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('压测用户', LPAD(n, 6, '0')),
       @r + MOD(FLOOR((n - 1) / 160), 8),
       @d3 + MOD(n - 1, 160),
       @p + MOD(FLOOR((n - 1) / 1280), 10),
       CONCAT('18', LPAD(n, 9, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0'), '@sim.test'),
       NULL, '', @pwd,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '[]', 1, 2 + MOD(n, 2), '', 1, 'sim-data', 1, 1,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND, 0
FROM seq;

-- 批 4：sim_user_030001 ~ sim_user_040000
INSERT INTO base_user (tenant_id, user_name, user_code, nick_name, role_id, dept_id, post_id, phone, email, id_type, id_code, password, password_changed_at, password_history, must_change_password, gender, avatar, status, remark, created_by, updated_by, created_at, updated_at, deleted_at)
WITH RECURSIVE seq AS (SELECT 30001 n UNION ALL SELECT n + 1 FROM seq WHERE n < 40000)
SELECT 1,
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('压测用户', LPAD(n, 6, '0')),
       @r + MOD(FLOOR((n - 1) / 160), 8),
       @d3 + MOD(n - 1, 160),
       @p + MOD(FLOOR((n - 1) / 1280), 10),
       CONCAT('18', LPAD(n, 9, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0'), '@sim.test'),
       NULL, '', @pwd,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '[]', 1, 2 + MOD(n, 2), '', 1, 'sim-data', 1, 1,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND, 0
FROM seq;

-- 批 5：sim_user_040001 ~ sim_user_050000
INSERT INTO base_user (tenant_id, user_name, user_code, nick_name, role_id, dept_id, post_id, phone, email, id_type, id_code, password, password_changed_at, password_history, must_change_password, gender, avatar, status, remark, created_by, updated_by, created_at, updated_at, deleted_at)
WITH RECURSIVE seq AS (SELECT 40001 n UNION ALL SELECT n + 1 FROM seq WHERE n < 50000)
SELECT 1,
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('压测用户', LPAD(n, 6, '0')),
       @r + MOD(FLOOR((n - 1) / 160), 8),
       @d3 + MOD(n - 1, 160),
       @p + MOD(FLOOR((n - 1) / 1280), 10),
       CONCAT('18', LPAD(n, 9, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0'), '@sim.test'),
       NULL, '', @pwd,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '[]', 1, 2 + MOD(n, 2), '', 1, 'sim-data', 1, 1,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND, 0
FROM seq;

-- 批 6：sim_user_050001 ~ sim_user_060000
INSERT INTO base_user (tenant_id, user_name, user_code, nick_name, role_id, dept_id, post_id, phone, email, id_type, id_code, password, password_changed_at, password_history, must_change_password, gender, avatar, status, remark, created_by, updated_by, created_at, updated_at, deleted_at)
WITH RECURSIVE seq AS (SELECT 50001 n UNION ALL SELECT n + 1 FROM seq WHERE n < 60000)
SELECT 1,
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('压测用户', LPAD(n, 6, '0')),
       @r + MOD(FLOOR((n - 1) / 160), 8),
       @d3 + MOD(n - 1, 160),
       @p + MOD(FLOOR((n - 1) / 1280), 10),
       CONCAT('18', LPAD(n, 9, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0'), '@sim.test'),
       NULL, '', @pwd,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '[]', 1, 2 + MOD(n, 2), '', 1, 'sim-data', 1, 1,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND, 0
FROM seq;

-- 批 7：sim_user_060001 ~ sim_user_070000
INSERT INTO base_user (tenant_id, user_name, user_code, nick_name, role_id, dept_id, post_id, phone, email, id_type, id_code, password, password_changed_at, password_history, must_change_password, gender, avatar, status, remark, created_by, updated_by, created_at, updated_at, deleted_at)
WITH RECURSIVE seq AS (SELECT 60001 n UNION ALL SELECT n + 1 FROM seq WHERE n < 70000)
SELECT 1,
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('压测用户', LPAD(n, 6, '0')),
       @r + MOD(FLOOR((n - 1) / 160), 8),
       @d3 + MOD(n - 1, 160),
       @p + MOD(FLOOR((n - 1) / 1280), 10),
       CONCAT('18', LPAD(n, 9, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0'), '@sim.test'),
       NULL, '', @pwd,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '[]', 1, 2 + MOD(n, 2), '', 1, 'sim-data', 1, 1,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND, 0
FROM seq;

-- 批 8：sim_user_070001 ~ sim_user_080000
INSERT INTO base_user (tenant_id, user_name, user_code, nick_name, role_id, dept_id, post_id, phone, email, id_type, id_code, password, password_changed_at, password_history, must_change_password, gender, avatar, status, remark, created_by, updated_by, created_at, updated_at, deleted_at)
WITH RECURSIVE seq AS (SELECT 70001 n UNION ALL SELECT n + 1 FROM seq WHERE n < 80000)
SELECT 1,
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('压测用户', LPAD(n, 6, '0')),
       @r + MOD(FLOOR((n - 1) / 160), 8),
       @d3 + MOD(n - 1, 160),
       @p + MOD(FLOOR((n - 1) / 1280), 10),
       CONCAT('18', LPAD(n, 9, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0'), '@sim.test'),
       NULL, '', @pwd,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '[]', 1, 2 + MOD(n, 2), '', 1, 'sim-data', 1, 1,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND, 0
FROM seq;

-- 批 9：sim_user_080001 ~ sim_user_090000
INSERT INTO base_user (tenant_id, user_name, user_code, nick_name, role_id, dept_id, post_id, phone, email, id_type, id_code, password, password_changed_at, password_history, must_change_password, gender, avatar, status, remark, created_by, updated_by, created_at, updated_at, deleted_at)
WITH RECURSIVE seq AS (SELECT 80001 n UNION ALL SELECT n + 1 FROM seq WHERE n < 90000)
SELECT 1,
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('压测用户', LPAD(n, 6, '0')),
       @r + MOD(FLOOR((n - 1) / 160), 8),
       @d3 + MOD(n - 1, 160),
       @p + MOD(FLOOR((n - 1) / 1280), 10),
       CONCAT('18', LPAD(n, 9, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0'), '@sim.test'),
       NULL, '', @pwd,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '[]', 1, 2 + MOD(n, 2), '', 1, 'sim-data', 1, 1,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND, 0
FROM seq;

-- 批 10：sim_user_090001 ~ sim_user_100000
INSERT INTO base_user (tenant_id, user_name, user_code, nick_name, role_id, dept_id, post_id, phone, email, id_type, id_code, password, password_changed_at, password_history, must_change_password, gender, avatar, status, remark, created_by, updated_by, created_at, updated_at, deleted_at)
WITH RECURSIVE seq AS (SELECT 90001 n UNION ALL SELECT n + 1 FROM seq WHERE n < 100000)
SELECT 1,
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0')),
       CONCAT('压测用户', LPAD(n, 6, '0')),
       @r + MOD(FLOOR((n - 1) / 160), 8),
       @d3 + MOD(n - 1, 160),
       @p + MOD(FLOOR((n - 1) / 1280), 10),
       CONCAT('18', LPAD(n, 9, '0')),
       CONCAT('sim_user_', LPAD(n, 6, '0'), '@sim.test'),
       NULL, '', @pwd,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '[]', 1, 2 + MOD(n, 2), '', 1, 'sim-data', 1, 1,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND,
       '2026-01-01 00:00:00' + INTERVAL MOD(n - 1, 365) DAY + INTERVAL MOD(n * 7919, 86400) SECOND, 0
FROM seq;

-- ============================================================================
-- 5. 验证
-- ============================================================================
SELECT '部门' AS item, COUNT(*) AS cnt FROM base_dept WHERE remark = 'sim-data'
UNION ALL
SELECT '角色', COUNT(*) FROM base_role WHERE remark = 'sim-data'
UNION ALL
SELECT '岗位', COUNT(*) FROM base_post WHERE remark = 'sim-data'
UNION ALL
SELECT '用户', COUNT(*) FROM base_user WHERE remark = 'sim-data';

-- 按月分布验证（12 个月每月约 8333 人）
SELECT DATE_FORMAT(created_at, '%Y-%m') AS ym, COUNT(*) AS users
FROM base_user WHERE remark = 'sim-data'
GROUP BY ym ORDER BY ym;

-- 用户分布抽查：部门 × 角色 × 岗位
SELECT d.name AS dept, r.name AS role, p.name AS post, COUNT(*) AS users
FROM base_user u
JOIN base_dept d ON d.id = u.dept_id
JOIN base_role r ON r.id = u.role_id
JOIN base_post p ON p.id = u.post_id
WHERE u.remark = 'sim-data'
GROUP BY d.name, r.name, p.name
ORDER BY users DESC
LIMIT 20;
