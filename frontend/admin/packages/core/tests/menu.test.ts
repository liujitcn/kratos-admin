import assert from "node:assert/strict";
import test from "node:test";
import type { RouteItem } from "../src/rpc/system/admin/v1/auth.js";
import { getRouteMenuKey } from "../src/utils/menuKey.js";

test("无路径目录使用自身标题生成唯一菜单索引", () => {
  const supplierManagement: RouteItem = {
    path: "",
    meta: { title: "供应商管理", params: [] },
    type: 1,
    children: [{ path: "/supplier/info", type: 2, children: [] }]
  };
  const productCenter: RouteItem = {
    path: "",
    meta: { title: "商品中心", params: [] },
    type: 1,
    children: [supplierManagement]
  };

  assert.notEqual(getRouteMenuKey(productCenter), getRouteMenuKey(supplierManagement));
});
