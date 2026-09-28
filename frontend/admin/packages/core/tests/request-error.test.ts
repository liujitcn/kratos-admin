import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import test from "node:test";
import { formatConflictMessage } from "../src/utils/conflict-message.js";
import { hasStructuredErrorCode, resolveStructuredErrorMessage } from "../src/utils/request-response.js";

test("成功业务响应的字符串 code 不会被识别为结构化错误码", () => {
  assert.equal(hasStructuredErrorCode({ code: "SYSTEM", name: "系统消息" }), false);
  assert.equal(hasStructuredErrorCode({ code: 500, reason: "INTERNAL_ERROR", message: "请求失败" }), true);
  assert.equal(hasStructuredErrorCode({ message: "请求成功" }), false);
});

test("成功响应拦截器使用结构化错误码识别器", async () => {
  const source = await readFile(join(process.cwd(), "src/utils/request.ts"), "utf8");
  const responseInterceptor = source.match(/service\.interceptors\.response\.use\([\s\S]*?\n\);/)?.[0];

  assert.ok(responseInterceptor, "缺少响应拦截器");
  assert.match(responseInterceptor, /if \(!hasStructuredErrorCode\(responseData\)\)/);
});

test("限流响应优先展示后端提示，缺少提示时使用本地化回退", () => {
  const translate = (key: string) => (key === "common.error.rate_limited" ? "请求过于频繁，请稍后重试" : key);
  assert.equal(resolveStructuredErrorMessage({ reason: "RATE_LIMITED" }, translate), "请求过于频繁，请稍后重试");
  assert.equal(
    resolveStructuredErrorMessage({ reason: "RATE_LIMITED", message: "请求过于频繁，请稍后重试" }, translate),
    "请求过于频繁，请稍后重试"
  );
  assert.equal(resolveStructuredErrorMessage({ reason: "INTERNAL_ERROR" }, translate), "");
});

test("HTTP 和结构化响应拦截器都使用限流提示回退", async () => {
  const source = await readFile(join(process.cwd(), "src/utils/request.ts"), "utf8");
  assert.match(source, /resolveStructuredErrorMessage\(responseData, key => t\(key\)\)/);
  assert.match(source, /resolveStructuredErrorMessage\(data, key => t\(key\)\)/);
});

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

test("唯一约束冲突显示资源和字段", () => {
  const messages: Record<string, string> = {
    "common.error.conflict": "资源状态冲突",
    "common.error.conflict.unique_location": "冲突位置：{resource}；字段：{fields}",
    "common.error.conflict.field_separator": "、",
    "common.error.conflict.detail_separator": " · ",
    "common.error.conflict.unknown_field": "未知字段",
    "common.error.conflict.with_details": "{message}（{details}）",
    "system.base.user.resource": "用户",
    "common.field.phone": "手机号"
  };
  const translate = (key: string, params: Record<string, string> = {}) => {
    const template = messages[key] ?? key;
    return template.replace(/\{(\w+)\}/g, (_, name: string) => params[name] ?? `{${name}}`);
  };

  assert.equal(
    formatConflictMessage(
      "手机号已被占用",
      {
        reason: "CONFLICT",
        metadata: {
          conflict_type: "unique_violation",
          resource: "base_user",
          field: "phone",
          constraint: "unique_base_user_phone"
        }
      },
      translate
    ),
    "手机号已被占用（冲突位置：用户；字段：手机号）"
  );
});

test("通用冲突标题由具体位置替代且不显示数据库索引名", () => {
  const messages: Record<string, string> = {
    "common.error.conflict": "资源状态冲突",
    "common.error.conflict.unique_location": "冲突位置：{resource}；字段：{fields}",
    "common.error.conflict.field_separator": "、",
    "common.error.conflict.detail_separator": " · ",
    "common.error.conflict.unknown_field": "未知字段",
    "common.error.conflict.with_details": "{message}（{details}）",
    "system.base.user.resource": "用户",
    "common.field.phone": "手机号"
  };
  const translate = (key: string, params: Record<string, string> = {}) => {
    const template = messages[key] ?? key;
    return template.replace(/\{(\w+)\}/g, (_, name: string) => params[name] ?? `{${name}}`);
  };

  assert.equal(
    formatConflictMessage(
      "资源状态冲突",
      {
        reason: "CONFLICT",
        metadata: {
          message_key: "common.error.conflict",
          conflict_type: "unique_violation",
          resource: "base_user",
          field: "phone",
          constraint: "unique_base_user_phone"
        }
      },
      translate
    ),
    "冲突位置：用户；字段：手机号"
  );
});

test("父子资源冲突显示关联资源且其他错误保持原文", () => {
  const messages: Record<string, string> = {
    "common.error.conflict.related_resource": "关联冲突：{parent} → {child}",
    "common.error.conflict.resource": "涉及资源：{resource}",
    "common.error.conflict.with_details": "{message}（{details}）",
    "system.base.message.resource": "消息",
    "system.base.dept.resource": "部门",
    "system.base.user.resource": "用户"
  };
  const translate = (key: string, params: Record<string, string> = {}) => {
    const template = messages[key] ?? key;
    return template.replace(/\{(\w+)\}/g, (_, name: string) => params[name] ?? `{${name}}`);
  };

  assert.equal(
    formatConflictMessage(
      "删除部门失败，仍有用户",
      { reason: "CONFLICT", metadata: { conflict_type: "has_children", resource: "base_dept", child_resource: "base_user" } },
      translate
    ),
    "删除部门失败，仍有用户（关联冲突：部门 → 用户）"
  );
  assert.equal(formatConflictMessage("请求参数错误", { reason: "INVALID_ARGUMENT", metadata: { resource: "base_user" } }, translate), "请求参数错误");
  assert.equal(
    formatConflictMessage("只能编辑草稿", { reason: "CONFLICT", metadata: { conflict_type: "protected_resource", resource: "base_message" } }, translate),
    "只能编辑草稿（涉及资源：消息）"
  );
});

test("组合唯一索引显示全部字段，缺少资源元数据时保留原消息", () => {
  const messages: Record<string, string> = {
    "common.error.conflict.unique_location": "冲突位置：{resource}；字段：{fields}",
    "common.error.conflict.field_separator": "、",
    "common.error.conflict.detail_separator": " · ",
    "common.error.conflict.unknown_field": "未知字段",
    "common.error.conflict.with_details": "{message}（{details}）",
    "system.base.config.resource": "系统配置",
    "system.base.config.field.site": "配置位置",
    "system.base.config.field.key": "配置键"
  };
  const translate = (key: string, params: Record<string, string> = {}) => {
    const template = messages[key] ?? key;
    return template.replace(/\{(\w+)\}/g, (_, name: string) => params[name] ?? `{${name}}`);
  };

  assert.equal(
    formatConflictMessage(
      "同一位置的配置键重复",
      {
        reason: "CONFLICT",
        metadata: {
          conflict_type: "unique_violation",
          resource: "base_config",
          field: "site,key",
          constraint: "unique_base_config"
        }
      },
      translate
    ),
    "同一位置的配置键重复（冲突位置：系统配置；字段：配置位置、配置键）"
  );
  assert.equal(formatConflictMessage("资源状态冲突", { reason: "CONFLICT" }, translate), "资源状态冲突");
});

test("冲突缺少资源元数据时显示接口路径", () => {
  const messages: Record<string, string> = {
    "common.error.conflict": "资源状态冲突",
    "common.error.conflict.operation": "接口路径：{operation}",
    "common.error.conflict.detail_separator": "；"
  };
  const translate = (key: string, params: Record<string, string> = {}) => {
    const template = messages[key] ?? key;
    return template.replace(/\{(\w+)\}/g, (_, name: string) => params[name] ?? `{${name}}`);
  };

  assert.equal(
    formatConflictMessage(
      "资源状态冲突",
      { reason: "CONFLICT", metadata: { message_key: "common.error.conflict", operation: "/api/v1/system/admin/users" } },
      translate
    ),
    "接口路径：/api/v1/system/admin/users"
  );
});
