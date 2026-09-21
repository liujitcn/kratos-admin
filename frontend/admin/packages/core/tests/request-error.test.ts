import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import test from "node:test";

test("公共认证接口返回 401 时展示后端业务错误", async () => {
  const source = await readFile(join(process.cwd(), "src/utils/request.ts"), "utf8");
  const responseInterceptor = source.match(/service\.interceptors\.response\.use\([\s\S]*?\n\);/)?.[0];

  assert.ok(responseInterceptor, "缺少响应拦截器");
  assert.match(responseInterceptor, /const isUnauthorized = status === 401 \|\| code === 401/);
  assert.match(
    responseInterceptor,
    /if \(isUnauthorized && !shouldSkipAuthExpiredPrompt\(requestConfig\)\)[\s\S]*?handleAuthExpired\(\);[\s\S]*?else if \(data\)[\s\S]*?showRequestError\(message\)/
  );
});
