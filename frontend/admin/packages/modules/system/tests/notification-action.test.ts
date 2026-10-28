import assert from "node:assert/strict";
import { test } from "node:test";
import { isNotificationActionRouteAvailable, resolveNotificationAction } from "../src/utils/notification-action.js";

test("审批消息动作解析为带任务参数的管理端审批路由", () => {
  assert.deepEqual(
    resolveNotificationAction(
      "MESSAGE_INBOX",
      JSON.stringify({ admin: { route: "WorkflowApproval", source: "project", task_id: "123", list_type: "PENDING" } })
    ),
    { name: "WorkflowApproval", query: { source: "project", task_id: "123", list_type: "PENDING" } }
  );
});

test("审批结果消息动作解析为带实例参数的管理端审批路由", () => {
  assert.deepEqual(
    resolveNotificationAction(
      "MESSAGE_INBOX",
      JSON.stringify({ admin: { route: "WorkflowApproval", source: "tenant", instance_id: "456", list_type: "SUBMITTED" } })
    ),
    { name: "WorkflowApproval", query: { source: "tenant", instance_id: "456", list_type: "SUBMITTED" } }
  );
});

test("普通消息仍按原有 viewKey 路由，非法审批参数回退收件箱", () => {
  assert.deepEqual(resolveNotificationAction("PROFILE", "{}"), { name: "Profile" });
  assert.deepEqual(resolveNotificationAction("MESSAGE_INBOX", JSON.stringify({ admin: { route: "Other" } })), {
    name: "NotificationInbox"
  });
  assert.deepEqual(
    resolveNotificationAction("MESSAGE_INBOX", JSON.stringify({ admin: { route: "WorkflowApproval", task_id: "not-a-number" } })),
    { name: "NotificationInbox" }
  );
});

test("审批消息在菜单路由未注册时不允许导航", () => {
  const action = { name: "WorkflowApproval", query: { task_id: "123" } };
  assert.equal(isNotificationActionRouteAvailable({ hasRoute: () => false }, action), false);
  assert.equal(isNotificationActionRouteAvailable({ hasRoute: name => name === "WorkflowApproval" }, action), true);
});
