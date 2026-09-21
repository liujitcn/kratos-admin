import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { test } from "node:test";

function readSource(path: string) {
  return readFile(join(process.cwd(), path), "utf8");
}

test("菜单搜索使用 ProDialog 并保留主题样式", async () => {
  const source = await readSource("../../core/src/layouts/components/Header/components/SearchMenu.vue");

  assert.match(source, /<ProDialog[\s\S]*class="search-dialog"/);
  assert.match(source, /import ProDialog from "@\/components\/Dialog\/ProDialog\.vue"/);
  assert.match(source, /:global\(\.search-dialog\)/);
  assert.match(source, /:global\(\.search-dialog \.el-dialog__header\)/);
  assert.doesNotMatch(source, /:global\(\.search-dialog\)[\s\S]*\.el-dialog__header\s*\{/);
});

test("登录策略提交前执行表单校验", async () => {
  const source = await readSource("src/views/base/login-policy/index.vue");
  const submitBlock = source.match(/async function handleSubmit\(\) \{[\s\S]*?\n\}/)?.[0];

  assert.ok(submitBlock, "缺少登录策略提交方法");
  assert.match(submitBlock, /const valid = await formDialogRef\.value\?\.validate\(\);/);
  assert.match(submitBlock, /if \(!valid\) return;/);
  assert.ok(submitBlock.indexOf("validate()") < submitBlock.indexOf("const baseLoginPolicy"));
});

test("登录策略弹窗自适应标签并展示初始化密码强度", async () => {
  const source = await readSource("src/views/base/login-policy/index.vue");

  assert.match(source, /<FormDialog[\s\S]*label-width="auto"/);
  assert.match(source, /import PasswordStrength from "@liujitcn\/kratos-admin-core\/components\/PasswordStrength\/index\.vue"/);
  assert.match(source, /<template #initialPasswordStrength>[\s\S]*<PasswordStrength :password="formData\.initial_password" \/>/);
  assert.match(source, /prop: "initialPasswordStrength"[\s\S]*component: "slot"[\s\S]*slotName: "initialPasswordStrength"/);
});

test("登录策略租户和用户未选择时不显示零值", async () => {
  const source = await readSource("src/views/base/login-policy/index.vue");

  assert.match(source, /tenant_id: undefined,[\s\S]*?user_id: undefined/);
  assert.match(source, /formData\.tenant_id = undefined;[\s\S]*?formData\.user_id = undefined;/);
});

test("新增租户成功后三行文本展示一次性随机管理员凭据", async () => {
  const [pageSource, apiSource] = await Promise.all([
    readSource("src/views/base/tenant/index.vue"),
    readSource("src/api/system/admin/v1/base_tenant.ts")
  ]);

  assert.match(apiSource, /CreateBaseTenant\(request: CreateBaseTenantRequest\): Promise<CreateBaseTenantResponse>/);
  assert.match(pageSource, /const response = await defBaseTenantService\.CreateBaseTenant/);
  assert.match(pageSource, /if \(response\.initial_password\)/);
  assert.match(pageSource, /response\.tenant_code/);
  assert.match(pageSource, /tenant-credentials-row[\s\S]*credentialsDialog\.tenant_code[\s\S]*credentialsDialog\.admin_user_name[\s\S]*credentialsDialog\.initial_password/);
  assert.doesNotMatch(pageSource, /<el-form[\s\S]*credentialsDialog\.tenant_code/);
  assert.doesNotMatch(pageSource, /tenant-credentials-toolbar/);
  assert.match(pageSource, /credentialsDialog\.tenant_code[\s\S]*tooltip\.copy_credentials[\s\S]*CopyDocument/);
  assert.match(pageSource, /credentialsDialog\.passwordVisible/);
  assert.match(pageSource, /<View v-if="!credentialsDialog\.passwordVisible" \/>[\s\S]*<Hide v-else \/>/);
  assert.match(pageSource, /CopyDocument/);
  assert.match(pageSource, /copyText\(content\)/);
  assert.match(pageSource, /system\.base\.tenant\.message\.initial_credentials_warning/);
});

test("管理端共享同一套 Vue 运行时并延后初始化页签拖拽", async () => {
  const [workspaceSource, tabsSource, dashboardSource] = await Promise.all([
    readSource("../../../package.json"),
    readSource("../../core/src/layouts/components/Tabs/index.vue"),
    readSource("src/views/base/dashboard/index.vue")
  ]);
  const workspace = JSON.parse(workspaceSource) as {
    pnpm?: { overrides?: Record<string, string> };
  };

  assert.equal(workspace.pnpm?.overrides?.vue, "3.5.42");
  assert.equal(workspace.pnpm?.overrides?.["element-plus"], "2.14.5");
  assert.match(tabsSource, /void nextTick\(tabsDrop\)/);
  assert.match(tabsSource, /const tabsNav = document\.querySelector<HTMLElement>\("\.el-tabs__nav"\);\s*if \(!tabsNav\) return;/);
  assert.match(tabsSource, /onBeforeUnmount\(\(\) => \{\s*tabsSortable\?\.destroy\(\)/);
  assert.match(dashboardSource, /const loading = ref\(true\);/);
});

test("归档和备份恢复拒绝零值记录ID", async () => {
  const [archiveRestore, backupRestore] = await Promise.all([
    readSource("src/views/base/backup-management/archive-restore/index.vue"),
    readSource("src/views/base/backup-management/backup-restore/index.vue")
  ]);

  assert.match(
    archiveRestore,
    /archive_record_id[\s\S]*?props:\s*\{\s*min:\s*0[\s\S]*?archive_record_id:\s*\[[\s\S]*?type:\s*"number"[\s\S]*?min:\s*1[\s\S]*?archive_record_id_positive/
  );
  assert.match(archiveRestore, /Number\.isInteger\(formData\.archive_record_id\)/);
  assert.match(
    backupRestore,
    /backup_record_id[\s\S]*?props:\s*\{\s*min:\s*0[\s\S]*?backup_record_id:\s*\[[\s\S]*?type:\s*"number"[\s\S]*?min:\s*1[\s\S]*?backup_record_id_positive/
  );
  assert.match(backupRestore, /Number\.isInteger\(formData\.backup_record_id\)/);
});

test("状态切换先确认再调用接口", async () => {
  const sources = await Promise.all([
    readSource("src/views/base/message-category/index.vue"),
    readSource("src/views/base/backup-management/archive-config/index.vue"),
    readSource("src/views/base/backup-management/backup-config/index.vue")
  ]);

  for (const source of sources) {
    const statusBlock = source.match(/(?:async function handleSetStatus|async function setStatus)\([\s\S]*?\n\}/)?.[0];
    assert.ok(statusBlock, "缺少状态切换方法");
    assert.match(statusBlock, /ElMessageBox\.confirm/);
    assert.match(statusBlock, /await defBase.*Status\(/);
    assert.ok(statusBlock.indexOf("ElMessageBox.confirm") < statusBlock.indexOf("await defBase"));
  }
});

test("通知组件显式接管并透传顶部工具属性", async () => {
  const source = await readSource("src/components/notification/Notification.vue");

  assert.match(source, /defineOptions\(\{ name: "Notification", inheritAttrs: false \}\)/);
  assert.match(source, /<el-popover[\s\S]*v-bind="\$attrs"/);
});

test("文件资产详情使用可关闭的内容预览弹窗", async () => {
  const source = await readSource("src/views/base/file/index.vue");

  assert.match(source, /<ProDialog[\s\S]*:show-footer="false"/);
  assert.match(source, /GetFileBlob/);
  assert.match(source, /URL\.revokeObjectURL/);
  assert.doesNotMatch(source, /ElMessageBox\.alert/);
});

test("出库脱敏响应字段缺失 ref 时使用空引用", async () => {
  const source = await readSource("src/views/base/redact-output-policy/index.vue");

  assert.match(source, /function normalizeRef\(ref\?: string\)/);
  assert.match(source, /\(ref \?\? ""\)\.split\("\/"\)/);
});

test("脱敏列表和下拉使用中文名称并保持简洁选择器", async () => {
  const [storageSource, outputSource] = await Promise.all([
    readSource("src/views/base/redact-storage-policy/index.vue"),
    readSource("src/views/base/redact-output-policy/index.vue")
  ]);
  const requestApisBlock = outputSource.match(/async function requestApis\(\) \{[\s\S]*?\n\}/)?.[0];

  assert.match(storageSource, /defBaseRedactStoragePolicyService\.ListBaseRedactStorageTable/);
  assert.match(storageSource, /item\.comment/);
  assert.match(storageSource, /tableCommentMap/);
  assert.match(outputSource, /item\.service_desc/);
  assert.match(outputSource, /api\.desc/);
  assert.match(outputSource, /apiLabel/);
  assert.match(outputSource, /isGetApi/);
  assert.match(outputSource, /api\.method\.toUpperCase\(\) === "GET"/);
  assert.match(outputSource, /OptionBaseApi\(\{ include_public: true, tenant_response: true \}\)/);
  assert.ok(requestApisBlock, "缺少 API 选项请求方法");
  assert.doesNotMatch(requestApisBlock, /requestResponseFields\(api\.id\)/);
  assert.match(outputSource, /mode: BaseRedactOutputPolicyMode\.BASE_REDACT_OUTPUT_POLICY_MODE_FULL/);
  assert.doesNotMatch(outputSource, /apiOptionTooltip|<el-tooltip/);
});

test("消息标题单独打开正文，发送详情只展示投递信息", async () => {
  const source = await readSource("src/views/base/message/index.vue");
  const sendDetailDialog = source.match(/<ProDialog(?=[^>]*v-model="detail\.visible")[\s\S]*?<\/ProDialog>/)?.[0];

  assert.match(source, /prop: "title"[\s\S]*?openContent\(row\.id\)/);
  assert.match(source, /system\.base\.message\.content\.title/);
  assert.match(source, /system\.base\.message\.send_detail\.title/);
  assert.match(source, /function openContent\(id: number\)/);
  assert.match(source, /prop: "operation"[\s\S]*?message-operation-column/);
  assert.doesNotMatch(source, /prop: "operation"[\s\S]*?width: 380/);
  assert.match(source, /whiteSpace: "nowrap"[\s\S]*?\n\s*\},\n\s*row\.title/);
  assert.ok(sendDetailDialog, "缺少发送详情弹窗");
  assert.doesNotMatch(sendDetailDialog, /message-detail-content|detail\.data\.form\?\.content/);
});

test("消息页面单删和提交都使用正确的表单数据", async () => {
  const source = await readSource("src/views/base/message/index.vue");
  const submitBlock = source.match(/async function handleSubmit\(\) \{[\s\S]*?\n\}/)?.[0];
  const deleteBlock = source.match(/async function handleDelete\([\s\S]*?\n\}/)?.[0];
  assert.ok(submitBlock, "缺少消息提交方法");
  assert.ok(deleteBlock, "缺少消息删除方法");
  assert.match(submitBlock, /const valid = await formDialogRef\.value\?\.validate\(\);/);
  assert.match(deleteBlock, /typeof item === "object" \? item\.id : item/);
});

test("缓存和 API 日志搜索使用后端 keyword 字段", async () => {
  const [cacheSource, apiLogSource] = await Promise.all([
    readSource("src/views/tool/cache/index.vue"),
    readSource("src/views/base/api-log/index.vue")
  ]);
  assert.match(cacheSource, /prop: "key"[\s\S]*search: \{ el: "input", key: "keyword"/);
  assert.match(apiLogSource, /prop: "operation"[\s\S]*search: \{ el: "input", key: "keyword"/);
});

test("入库脱敏策略清空已有规则时提交删除请求", async () => {
  const source = await readSource("src/views/base/redact-storage-policy/index.vue");
  assert.match(source, /const removedIds = form\.column_rows\.filter\(row => row\.id > 0 && !row\.rule_id\)/);
  assert.match(source, /DeleteBaseRedactStoragePolicy\(\{ id: removedIds\.join\(","\) \}\)/);
});

test("SSE 和资源地址只使用有效的协议与路由格式", async () => {
  const [sseSource, utilsSource] = await Promise.all([
    readSource("src/api/base/v1/sse.ts"),
    readSource("../../core/src/utils/utils.ts")
  ]);
  assert.match(sseSource, /new URL\(`\$\{SSE_URL\}\/\$\{encodeURIComponent\(request\.stream\)\}`/);
  assert.match(sseSource, /url\.searchParams\.set\("channel_id", request\.channel_id\)/);
  assert.ok(utilsSource.includes('if (/^[a-z][a-z0-9+.-]*:/i.test(value)) return "";'));
});

test("用户默认性别为保密且凭据轮换需要确认", async () => {
  const [userSource, oauthSource] = await Promise.all([
    readSource("src/views/base/user/index.vue"),
    readSource("src/views/base/oauth-client/index.vue")
  ]);
  assert.match(userSource, /gender: 1/);
  assert.match(oauthSource, /await ElMessageBox\.confirm\(/);
  assert.match(oauthSource, /system\.base\.oauth_client\.confirm\.rotate_credentials/);
});

test("新增回归校验使用的国际化键在四种语言中均存在", async () => {
  const locales = await Promise.all(["zh-CN", "en-US", "ja-JP", "zh-TW"].map(locale => readSource(`src/locales/${locale}.json`)));
  const requiredKeys = [
    "system.base.message_category.resource",
    "system.backup.validation.archive_record_id_positive",
    "system.backup.validation.backup_record_id_positive"
  ];

  for (const source of locales) {
    const messages = JSON.parse(source) as Record<string, string>;
    for (const key of requiredKeys) assert.ok(messages[key], `缺少国际化键: ${key}`);
  }
});

test("项目授权新增弹窗默认使用有效授权类型", async () => {
  const source = await readSource("src/views/base/tenant-project-grant/index.vue");

  assert.match(
    source,
    /const DEFAULT_SUBJECT_TYPE = BaseTenantProjectGrantSubjectType\.BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_POST;/
  );
  assert.match(source, /subject_type: DEFAULT_SUBJECT_TYPE/);
  assert.match(source, /formData\.subject_type = DEFAULT_SUBJECT_TYPE;/);
  assert.match(source, /await Promise\.all\(\[loadProjectOptions\(\), loadSubjectOptions\(\)\]\)/);
});

test("角色分配权限默认关闭父子联动", async () => {
  const source = await readSource("src/views/base/role/index.vue");

  assert.match(source, /const parentChildLinked = ref\(false\);/);
  assert.match(source, /:check-strictly="!parentChildLinked"/);
});

test("定时任务操作列使用统一操作按钮配置", async () => {
  const source = await readSource("src/views/base/job/index.vue");

  assert.match(source, /prop: "operation"[\s\S]*?cellType: "actions"[\s\S]*?actions:/);
  assert.doesNotMatch(source, /renderOperationCell|job-operation|job-action/);
});

test("项目授权范围提供开关含义说明", async () => {
  const source = await readSource("src/views/base/tenant-project-grant/index.vue");
  assert.match(source, /labelTooltip: t\("system\.base\.tenant_project_grant\.tooltip\.scope"\)/);

  for (const locale of ["zh-CN", "en-US", "ja-JP", "zh-TW"]) {
    const messages = JSON.parse(await readSource(`src/locales/${locale}.json`)) as Record<string, string>;
    assert.ok(messages["system.base.tenant_project_grant.tooltip.scope"], `缺少 ${locale} 授权范围说明`);
  }
});

test("项目授权列表沿用统一租户列展示规则", async () => {
  const source = await readSource("src/views/base/tenant-project-grant/index.vue");
  const tenantColumn = source.match(/\.\.\.tenantColumns\(\{[^}]+\}\)/)?.[0];

  assert.ok(tenantColumn, "项目授权列表缺少租户列配置");
  assert.doesNotMatch(tenantColumn, /isShow:\s*false/);
  assert.doesNotMatch(tenantColumn, /isSetting:\s*false/);
});

test("文件管理列表使用统一租户列展示规则", async () => {
  const source = await readSource("src/views/base/file/index.vue");

  assert.match(source, /tenantColumns\(\{ label: t\("common\.field\.tenant"\), minWidth: 100 \}\)/);
  assert.doesNotMatch(source, /system\.base\.file\.field\.tenant/);
});

test("携带租户查询的列表统一将租户作为第一业务列和第一个查询字段", async () => {
  const pages = [
    "api-log/index.vue",
    "data-access-log/index.vue",
    "dept/index.vue",
    "file/index.vue",
    "login-log/index.vue",
    "message/index.vue",
    "oauth-client/index.vue",
    "online-session/index.vue",
    "operation-log/index.vue",
    "permission-log/index.vue",
    "policy-evaluation-log/index.vue",
    "post/index.vue",
    "role/index.vue",
    "tenant-project-grant/index.vue",
    "user/index.vue"
  ];

  for (const page of pages) {
    const source = await readSource(`src/views/base/${page}`);
    const columns = source.match(/const columns = computed<ColumnProps\[\]>\(\(\) => \[([\s\S]*?)\n\]\);/)?.[1];
    assert.ok(columns, `${page} 缺少标准列配置`);
    assert.ok(columns.indexOf("tenantColumns(") >= 0, `${page} 缺少统一租户列`);
    assert.ok(columns.indexOf("tenantColumns(") < columns.indexOf('{ prop: "'), `${page} 的租户列不是第一业务列`);
  }

  const tenantSource = await readSource("../../core/src/tenant.ts");
  assert.match(tenantSource, /order: options\.order \?\? 1/);
});

test("项目授权列表不展示主体编码列且不保留无用翻译", async () => {
  const source = await readSource("src/views/base/tenant-project-grant/index.vue");
  assert.doesNotMatch(source, /prop: "subject_code"[\s\S]*?field\.subject_code/);

  for (const locale of ["zh-CN", "en-US", "ja-JP", "zh-TW"]) {
    const messages = JSON.parse(await readSource(`src/locales/${locale}.json`)) as Record<string, string>;
    assert.equal(messages["system.base.tenant_project_grant.field.subject_code"], undefined, `仍保留 ${locale} 主体编码翻译`);
  }
});

test("国际化自定义翻译使用响应式语言选项并锁定编辑键和区域", async () => {
  const source = await readSource("src/views/base/i18n-custom/index.vue");

  assert.match(source, /search: \{ el: "select", enum: localeOptions \}/);
  assert.match(source, /async function requestBaseI18nCustomTable[\s\S]*?await loadLanguages\(\);/);
  assert.match(source, /async function handleOpenDialog\(id\?: number\) \{[\s\S]*?await formDialogRef\.value\?\.open\(/);
  assert.match(source, /field: "key"|prop: "key"[\s\S]*?disabled: dialog\.editing/);
  assert.match(source, /prop: "locale"[\s\S]*?disabled: dialog\.editing/);
});

test("基础管理编辑弹窗统一使用通用异步打开控制器", async () => {
  const pages = [
    "area/index.vue",
    "config/index.vue",
    "dept/index.vue",
    "dict/index.vue",
    "dict/item.vue",
    "i18n-custom/index.vue",
    "job/index.vue",
    "language/index.vue",
    "menu/index.vue",
    "oauth-client/index.vue",
    "post/index.vue",
    "role/index.vue",
    "tenant/index.vue",
    "user/index.vue"
  ];
  const sources = await Promise.all(pages.map(page => readSource(`src/views/base/${page}`)));
  for (const source of sources) {
    assert.doesNotMatch(source, /dialogRequestSerial|detailRequestSerial/);
    assert.match(source, /\.value\?\.open\(/);
    assert.match(source, /\.value\?\.close\(/);
  }
});

test("租户项目列表按默认租户展示租户字段并支持扩展列定位", async () => {
  const source = await readSource("src/components/tenant-project/TenantProjectManager.vue");

  assert.match(source, /tenantColumns\(\{ label: t\("common\.field\.tenant"\), order: 1 \}\)/);
  assert.match(source, /tenantFormField\(\{ label: t\("common\.field\.tenant"\), disabledOnEdit: true \}\)/);
  assert.match(source, /tenant_id: toRequestTenantId\(params\.tenant_id\)/);
  assert.match(source, /arrangeTenantProjectColumns\(baseColumns, props\.extraColumns/);
  assert.match(source, /prop: "name", label: t\("system\.base\.tenant_project\.field\.name"\),[\s\S]*?search: \{ el: "input" \}/);
  assert.match(source, /prop: "code", label: t\("system\.base\.tenant_project\.field\.code"\),[\s\S]*?search: \{ el: "input" \}/);
  assert.doesNotMatch(source, /TenantProjectText|common\.field\.tenant\} \/ \$\{t\("common\.field\.project"\)\}/);
});

test("项目目录仅允许默认租户执行维护操作", async () => {
  const source = await readSource("src/components/tenant-project/TenantProjectManager.vue");

  assert.match(source, /disabled: \(\) => !isDefaultTenant\.value \|\|[^\n]*base:tenant:project:status/);
  assert.match(source, /hidden: \(\) => !isDefaultTenant\.value \|\|[^\n]*base:tenant:project:update/);
  assert.match(source, /hidden: \(\) => !isDefaultTenant\.value \|\|[^\n]*base:tenant:project:delete/);
  assert.match(source, /hidden: \(\) => !isDefaultTenant\.value \|\|[^\n]*base:tenant:project:create/);
});

test("普通租户管理员在用户列表中不可删除但仍可重置密码", async () => {
  const source = await readSource("src/views/base/user/index.vue");

  assert.match(source, /selectable: row => !isDeleteProtectedManagementUser\(row as BaseUser\)/);
  assert.match(
    source,
    /hidden: scope => isDeleteProtectedManagementUser\(scope\.row as BaseUser\) \|\| !BUTTONS\.value\["base:user:delete"\]/
  );
  assert.match(
    source,
    /hidden: scope => isProtectedManagementUser\(scope\.row as BaseUser\) \|\| !BUTTONS\.value\["base:user:pwd"\]/
  );
  assert.match(source, /updateUndeletableRoleIds\(options\)/);
  assert.match(source, /option\.disabled/);
});
