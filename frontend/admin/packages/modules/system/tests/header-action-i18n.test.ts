import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { test } from "node:test";

const pagePaths = ["src/views/base/message-category/index.vue", "src/views/base/message/index.vue"];
const localeNames = ["zh-CN", "en-US", "ja-JP", "zh-TW"];

function readSource(path: string) {
  return readFile(join(process.cwd(), path), "utf8");
}

function headerActionKeys(source: string) {
  const block = source.match(/const headerActions[\s\S]*?^\]\);/m)?.[0];
  assert.ok(block, "页面缺少 headerActions 配置");
  return [...block.matchAll(/label:\s*t\("([^"]+)"\)/g)].map(match => match[1]);
}

test("消息页面顶部操作按钮使用四种语言均存在的国际化键", async () => {
  const pages = await Promise.all(pagePaths.map(async path => ({ path, source: await readSource(path) })));

  for (const locale of localeNames) {
    const [coreSource, systemSource] = await Promise.all([
      readSource(`../../core/src/locales/${locale}.json`),
      readSource(`src/locales/${locale}.json`)
    ]);
    const messages = { ...JSON.parse(coreSource), ...JSON.parse(systemSource) };
    for (const page of pages) {
      for (const key of headerActionKeys(page.source)) {
        assert.ok(Object.hasOwn(messages, key), `${page.path} 的 ${key} 缺少 ${locale} 翻译`);
      }
    }
  }
});
