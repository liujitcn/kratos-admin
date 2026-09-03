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
  const source = await readSource("src/components/Notification.vue");

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

test("消息标题单独打开正文，发送详情只展示投递信息", async () => {
  const source = await readSource("src/views/base/message/index.vue");
  const sendDetailDialog = source.match(/<ProDialog\s+v-model="detail\.visible"[\s\S]*?<\/ProDialog>/)?.[0];

  assert.match(source, /prop: "title"[\s\S]*?openContent\(row\.id\)/);
  assert.match(source, /system\.base\.message\.content\.title/);
  assert.match(source, /system\.base\.message\.send_detail\.title/);
  assert.match(source, /function openContent\(id: number\)/);
  assert.match(source, /prop: "operation"[\s\S]*?width: 380[\s\S]*?message-operation-column/);
  assert.match(source, /whiteSpace: "nowrap"[\s\S]*?\n\s*\},\n\s*row\.title/);
  assert.ok(sendDetailDialog, "缺少发送详情弹窗");
  assert.doesNotMatch(sendDetailDialog, /message-detail-content|detail\.data\.form\?\.content/);
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
