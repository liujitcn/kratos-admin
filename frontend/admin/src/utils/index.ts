import { isArray } from "@/utils/is";
import type { RouteItem, RouteMeta } from "@/rpc/system/admin/v1/auth";
import { BaseMenuType } from "@/rpc/system/common/v1/enum";

const mode = import.meta.env.VITE_ROUTER_MODE;

/** 树形或枚举数据字段映射配置。 */
type FieldNamesProps = {
  label?: string;
  value?: string;
  children?: string;
};

/**
 * 获取菜单图标。
 */
export function getRouteMetaIcon(meta?: RouteMeta) {
  return meta?.icon ?? "";
}

/**
 * 获取菜单标题。
 */
export function getRouteMetaTitle(meta?: RouteMeta) {
  return meta?.title ?? "未命名菜单";
}

/**
 * 判断菜单是否隐藏。
 */
export function getRouteMetaHidden(meta?: RouteMeta) {
  return meta?.hidden ?? false;
}

/**
 * 判断目录是否始终显示。
 */
export function getRouteMetaAlwaysShow(meta?: RouteMeta) {
  return meta?.always_show ?? false;
}

/**
 * 判断菜单是否全屏。
 */
export function getRouteMetaFull(meta?: RouteMeta) {
  return meta?.full ?? false;
}

/**
 * 判断菜单是否固定标签。
 */
export function getRouteMetaAffix(meta?: RouteMeta) {
  return meta?.affix ?? false;
}

/**
 * 判断菜单是否缓存。
 */
export function getRouteMetaKeepAlive(meta?: RouteMeta) {
  return meta?.keep_alive ?? false;
}

/**
 * @description 获取localStorage
 * @param {String} key Storage名称
 * @returns {String}
 */
export function localGet(key: string) {
  const value = window.localStorage.getItem(key);
  try {
    return JSON.parse(window.localStorage.getItem(key) as string);
  } catch {
    return value;
  }
}

/**
 * @description 存储localStorage
 * @param {String} key Storage名称
 * @param {*} value Storage值
 * @returns {void}
 */
export function localSet(key: string, value: any) {
  window.localStorage.setItem(key, JSON.stringify(value));
}

/**
 * @description 清除localStorage
 * @param {String} key Storage名称
 * @returns {void}
 */
export function localRemove(key: string) {
  window.localStorage.removeItem(key);
}

/**
 * @description 清除所有localStorage
 * @returns {void}
 */
export function localClear() {
  window.localStorage.clear();
}

/**
 * @description 判断数据类型
 * @param {*} val 需要判断类型的数据
 * @returns {String}
 */
export function isType(val: any) {
  if (val === null) return "null";
  if (typeof val !== "object") return typeof val;
  else return Object.prototype.toString.call(val).slice(8, -1).toLocaleLowerCase();
}

/**
 * @description 生成唯一 uuid
 * @returns {String}
 */
export function generateUUID() {
  let uuid = "";
  for (let i = 0; i < 32; i++) {
    let random = (Math.random() * 16) | 0;
    if (i === 8 || i === 12 || i === 16 || i === 20) uuid += "-";
    uuid += (i === 12 ? 4 : i === 16 ? (random & 3) | 8 : random).toString(16);
  }
  return uuid;
}

/**
 * 判断两个对象是否相同
 * @param {Object} a 要比较的对象一
 * @param {Object} b 要比较的对象二
 * @returns {Boolean} 相同返回 true，反之 false
 */
export function isObjectValueEqual(a: { [key: string]: any }, b: { [key: string]: any }) {
  if (!a || !b) return false;
  let aProps = Object.getOwnPropertyNames(a);
  let bProps = Object.getOwnPropertyNames(b);
  if (aProps.length != bProps.length) return false;
  for (let i = 0; i < aProps.length; i++) {
    let propName = aProps[i];
    let propA = a[propName];
    let propB = b[propName];
    if (!b.hasOwnProperty(propName)) return false;
    if (propA instanceof Object) {
      if (!isObjectValueEqual(propA, propB)) return false;
    } else if (propA !== propB) {
      return false;
    }
  }
  return true;
}

/**
 * @description 生成随机数
 * @param {Number} min 最小值
 * @param {Number} max 最大值
 * @returns {Number}
 */
export function randomNum(min: number, max: number): number {
  let num = Math.floor(Math.random() * (min - max) + max);
  return num;
}

/**
 * @description 获取当前时间对应的提示语
 * @returns {String}
 */
export function getTimeState() {
  let timeNow = new Date();
  let hours = timeNow.getHours();
  if (hours >= 6 && hours <= 10) return `早上好 ⛅`;
  if (hours >= 10 && hours <= 14) return `中午好 🌞`;
  if (hours >= 14 && hours <= 18) return `下午好 🌞`;
  if (hours >= 18 && hours <= 24) return `晚上好 🌛`;
  if (hours >= 0 && hours <= 6) return `凌晨好 🌛`;
}

/**
 * @description 获取不同路由模式所对应的 url + params
 * @returns {String}
 */
export function getUrlWithParams() {
  const url = {
    hash: location.hash.substring(1),
    history: location.pathname + location.search
  };
  return url[mode];
}

/**
 * 判断路径是否为外部链接。
 */
export function isExternalPath(path?: string) {
  if (!path) return false;
  return /^(https?:|mailto:|tel:)/.test(path);
}

/** 获取菜单项实际跳转目标，外链优先使用 redirect。 */
export function getRouteTarget(item: RouteItem): string {
  if (item.type === BaseMenuType.EXT_LINK) return item.redirect || item.path || "";
  return item.path ?? "";
}

/** 获取菜单节点的首个可跳转子路径。 */
function getFirstRoutePath(item: RouteItem): string {
  if (item.path) return item.path;
  for (const child of item.children ?? []) {
    const path = getFirstRoutePath(child);
    if (path) return path;
  }
  return "";
}

/** 获取菜单节点在 Element Plus 菜单树中的稳定索引。 */
export function getRouteMenuKey(item: RouteItem): string {
  if (item.path) return item.path;
  return `folder:${getFirstRoutePath(item) || getRouteMetaTitle(item.meta)}`;
}

/** 判断路由路径是否匹配菜单路径，兼容动态参数。 */
function matchRoutePath(menuPath: string, currentPath: string): boolean {
  const menuSegments = menuPath.split("/").filter(Boolean);
  const currentSegments = currentPath.split("/").filter(Boolean);
  if (menuSegments.length !== currentSegments.length) return false;
  return menuSegments.every((segment, index) => segment.startsWith(":") || segment === currentSegments[index]);
}

/** 判断菜单节点或其子节点是否对应当前路由。 */
export function isRouteMenuActive(item: RouteItem, currentPath: string): boolean {
  if (item.path && matchRoutePath(item.path, currentPath)) return true;
  return (item.children ?? []).some(child => isRouteMenuActive(child, currentPath));
}

/**
 * @description 使用递归扁平化菜单，方便添加动态路由
 * @param {Array} menuList 菜单列表
 * @returns {Array}
 */
export function getFlatMenuList(menuList: RouteItem[]): RouteItem[] {
  let newMenuList: RouteItem[] = JSON.parse(JSON.stringify(menuList));
  return newMenuList.flatMap(item => [item, ...(item.children ? getFlatMenuList(item.children) : [])]);
}

/**
 * @description 使用递归过滤出需要渲染在左侧菜单的列表 (需剔除 hide == true 的菜单)
 * @param {Array} menuList 菜单列表
 * @returns {Array}
 * */
export function getShowMenuList(menuList: RouteItem[]) {
  let newMenuList: RouteItem[] = JSON.parse(JSON.stringify(menuList));
  return newMenuList.filter(item => {
    item.children?.length && (item.children = getShowMenuList(item.children));
    return !getRouteMetaHidden(item.meta);
  });
}

/**
 * @description 使用递归找出所有面包屑存储到 pinia/vuex 中
 * @param {Array} menuList 菜单列表
 * @param {Array} parent 父级菜单
 * @param {Object} result 处理后的结果
 * @returns {Object}
 */
export const getAllBreadcrumbList = (
  menuList: RouteItem[],
  parent: RouteItem[] = [],
  result: Record<string, RouteItem[]> = {}
) => {
  for (const item of menuList) {
    const routePath = item.path ?? "";
    const breadcrumb = [...parent, item];
    if (routePath) result[routePath] = breadcrumb;
    if (item.children) getAllBreadcrumbList(item.children, breadcrumb, result);
  }
  return result;
};

/**
 * @description 使用递归处理路由菜单 path，生成一维数组 (第一版本地路由鉴权会用到，该函数暂未使用)
 * @param {Array} menuList 所有菜单列表
 * @param {Array} menuPathArr 菜单地址的一维数组 ['**','**']
 * @returns {Array}
 */
export function getMenuListPath(menuList: RouteItem[], menuPathArr: string[] = []): string[] {
  for (const item of menuList) {
    if (typeof item === "object" && item.path) menuPathArr.push(item.path);
    if (item.children?.length) getMenuListPath(item.children, menuPathArr);
  }
  return menuPathArr;
}

/**
 * @description 递归查询当前 path 所对应的菜单对象 (该函数暂未使用)
 * @param {Array} menuList 菜单列表
 * @param {String} path 当前访问地址
 * @returns {Object | null}
 */
export function findMenuByPath(menuList: RouteItem[], path: string): RouteItem | null {
  for (const item of menuList) {
    if (item.path === path) return item;
    if (item.children) {
      const res = findMenuByPath(item.children, path);
      if (res) return res;
    }
  }
  return null;
}

/**
 * @description 使用递归过滤需要缓存的菜单 name (该函数暂未使用)
 * @param {Array} menuList 所有菜单列表
 * @param {Array} keepAliveNameArr 缓存的菜单 name ['**','**']
 * @returns {Array}
 * */
export function getKeepAliveRouterName(menuList: RouteItem[], keepAliveNameArr: string[] = []) {
  menuList.forEach(item => {
    getRouteMetaKeepAlive(item.meta) && item.name && keepAliveNameArr.push(item.name);
    item.children?.length && getKeepAliveRouterName(item.children, keepAliveNameArr);
  });
  return keepAliveNameArr;
}

/**
 * @description 格式化表格单元格默认值 (el-table-column)
 * @param {Number} row 行
 * @param {Number} col 列
 * @param {*} callValue 当前单元格值
 * @returns {String}
 * */
export function formatTableColumn(row: number, col: number, callValue: any) {
  // 如果当前值为数组，使用 / 拼接（根据需求自定义）
  if (isArray(callValue)) return callValue.length ? callValue.join(" / ") : "--";
  return callValue ?? "--";
}

/**
 * @description 处理 ProTable 值为数组 || 无数据
 * @param {*} callValue 需要处理的值
 * @returns {String}
 * */
export function formatValue(callValue: any) {
  // 如果当前值为数组，使用 / 拼接（根据需求自定义）
  if (isArray(callValue)) return callValue.length ? callValue.join(" / ") : "--";
  return callValue ?? "--";
}

/**
 * @description 处理 prop 为多级嵌套的情况，返回的数据 (列如: prop: user.name)
 * @param {Object} row 当前行数据
 * @param {String} prop 当前 prop
 * @returns {*}
 * */
export function handleRowAccordingToProp(row: { [key: string]: any }, prop: string) {
  if (!prop.includes(".")) return row[prop] ?? "--";
  prop.split(".").forEach(item => (row = row[item] ?? "--"));
  return row;
}

/**
 * @description 处理 prop，当 prop 为多级嵌套时 ==> 返回最后一级 prop
 * @param {String} prop 当前 prop
 * @returns {String}
 * */
export function handleProp(prop: string) {
  const propArr = prop.split(".");
  if (propArr.length == 1) return prop;
  return propArr[propArr.length - 1];
}

/**
 * @description 根据枚举列表查询当需要的数据（如果指定了 label 和 value 的 key值，会自动识别格式化）
 * @param {String} callValue 当前单元格值
 * @param {Array} enumData 字典列表
 * @param {Array} fieldNames label && value && children 的 key 值
 * @param {String} type 过滤类型（目前只有 tag）
 * @returns {String}
 * */
export function filterEnum(callValue: any, enumData?: any, fieldNames?: FieldNamesProps, type?: "tag") {
  const value = fieldNames?.value ?? "value";
  const label = fieldNames?.label ?? "label";
  const children = fieldNames?.children ?? "children";
  let filterData: { [key: string]: any } = {};
  // 判断 enumData 是否为数组
  if (Array.isArray(enumData)) filterData = findItemNested(enumData, callValue, value, children);
  // 判断是否输出的结果为 tag 类型
  if (type == "tag") {
    return filterData?.tag_type ?? "";
  } else {
    return filterData ? filterData[label] : "--";
  }
}

/**
 * @description 递归查找 callValue 对应的 enum 值
 * */
export function findItemNested(enumData: any, callValue: any, value: string, children: string) {
  return enumData.reduce((accumulator: any, current: any) => {
    if (accumulator) return accumulator;
    if (current[value] === callValue) return current;
    if (current[children]) return findItemNested(current[children], callValue, value, children);
  }, null);
}
