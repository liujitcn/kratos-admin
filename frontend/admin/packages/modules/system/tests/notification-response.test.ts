import assert from "node:assert/strict";
import { test } from "node:test";
import { normalizeNotificationSummary, normalizePageNotificationResponse } from "../src/utils/notification-response.js";

test("空分页响应提供可读取的消息列表和分页默认值", () => {
  const response = normalizePageNotificationResponse({});

  assert.deepEqual(response, {
    notifications: [],
    total: 0,
    next_cursor_id: 0,
    has_more: false
  });
  assert.equal(response.notifications.length, 0);
});

test("空汇总响应提供可用于角标计算的默认值", () => {
  const response = normalizeNotificationSummary({});

  assert.deepEqual(response, {
    unread_total: 0,
    latest_delivery_id: 0,
    category_unread: []
  });
  assert.equal(Number.isFinite(Math.min(response.unread_total, 99)), true);
});
