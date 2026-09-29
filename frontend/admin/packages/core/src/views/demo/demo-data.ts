/**
 * Demo 演示模块共享的静态 mock 数据与类型。
 *
 * 约定：
 * - 全部数据纯前端静态维护，不调用任何后端接口；真实项目中对应字段来自 RPC 返回结构。
 * - 记录内容本身是"业务数据"，与表格列头等 UI 文案不同，不进入语言包（与真实项目一致）。
 * - avatar 使用内联 SVG data URL，避免依赖外网图片服务，离线环境也能渲染图片列。
 */

/** Demo 通用状态值：与 base.v1 Status 枚举保持一致（1 启用 / 2 禁用），core 不反向依赖业务 RPC。 */
export const STATUS_ENABLE = 1;
/** 禁用状态值。 */
export const STATUS_DISABLE = 2;

/**
 * 用 canvas 生成 PNG base64 头像 data URL。
 * 图片列的 formatSrc 只放行 base64 位图（png/jpeg/webp 等），
 * 因此不能用 svg+xml 文本；canvas 生成可离线渲染且通过安全校验。
 */
function makeAvatar(background: string, text: string): string {
  const size = 80;
  const canvas = document.createElement("canvas");
  canvas.width = size;
  canvas.height = size;
  const ctx = canvas.getContext("2d");
  if (!ctx) return "";
  ctx.beginPath();
  if (typeof ctx.roundRect === "function") {
    ctx.roundRect(0, 0, size, size, 16);
  } else {
    ctx.rect(0, 0, size, size);
  }
  ctx.fillStyle = background;
  ctx.fill();
  ctx.fillStyle = "#ffffff";
  ctx.font = "30px sans-serif";
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  ctx.fillText(text, size / 2, size / 2 + 2);
  return canvas.toDataURL("image/png");
}

/** Demo 员工记录，左树右表页面的行数据结构。 */
export interface DemoEmployee {
  /** 员工ID。 */
  id: number;
  /** 姓名列表格首列。 */
  name: string;
  /** 所属部门ID，关联左侧部门树的 id。 */
  dept_id: number;
  /** 邮箱，演示普通文本列。 */
  email: string;
  /** 手机号。 */
  phone: string;
  /** 头像，演示 cellType: "image" 图片列。 */
  avatar: string;
  /** 薪资（单位：分），演示 cellType: "money" 金额列，金额列按分换算展示。 */
  salary: number;
  /** 状态，1 启用 / 0 停用，演示 cellType: "status" 状态开关列。 */
  status: number;
  /** 入职日期，演示日期区间搜索。 */
  entry_date: string;
}

/** Demo 项目记录，纯表格页面的行数据结构。 */
export interface DemoProject {
  /** 项目ID。 */
  id: number;
  /** 项目名称，演示 input 搜索 + tag 标签列。 */
  name: string;
  /** 项目等级，演示 tag: true + enum tagType 标签列。 */
  level: string;
  /** 负责人。 */
  owner: string;
  /** 状态，1 启用 / 0 停用。 */
  status: number;
  /** 预算金额（单位：分），演示 cellType: "money" 金额列，金额列按分换算展示。 */
  budget: number;
  /** 任务总数，放在多级表头“任务情况”下。 */
  task_total: number;
  /** 完成进度（0-100），演示 render 自定义渲染进度条列。 */
  progress: number;
  /** 所属城市（演示级联选择搜索），格式：省份/城市。 */
  region: string[];
  /** 创建时间，演示日期区间搜索。 */
  created_at: string;
}

/** Demo 目录节点，树形表格页面的行数据结构（children 内嵌子级行）。 */
export interface DemoCatalog {
  /** 节点ID。 */
  id: number;
  /** 目录名称，作为树形展开列。 */
  name: string;
  /** 编码。 */
  code: string;
  /** 负责人。 */
  owner: string;
  /** 状态，1 启用 / 0 停用。 */
  status: number;
  /** 文档数量。 */
  doc_count: number;
  /** 子级目录，树形表格通过 tree-props.children 识别。 */
  children?: DemoCatalog[];
}

/** 部门树节点，左侧 TreeFilter 静态数据结构。 */
export interface DemoDeptNode {
  /** 部门ID，字符串形式以匹配 TreeFilter 默认选中逻辑。 */
  id: string;
  /** 部门名称。 */
  name: string;
  /** 部门人数，展示在树节点右侧。 */
  count: number;
  /** 子部门。 */
  children?: DemoDeptNode[];
}

/** 详情抽屉单条字段：label 走 t() 文案，value 为记录内容。 */
export interface DemoDetailField {
  /** 字段中文名。 */
  label: string;
  /** 字段值。 */
  value: string;
  /** 是否用 el-tag 展示。 */
  tag?: boolean;
  /** el-tag 展示类型（success/info/warning/danger）。 */
  tagType?: "success" | "info" | "warning" | "danger";
}

/** 详情抽屉信息分组，一组对应一个 el-descriptions 区块。 */
export interface DemoDetailSection {
  /** 分组标题。 */
  title: string;
  /** 分组内字段。 */
  fields: DemoDetailField[];
}

/** 详情抽屉入参模型，由各页面从行数据组装。 */
export interface DemoDetailModel {
  /** 抽屉标题。 */
  title: string;
  /** 基本信息分组。 */
  sections: DemoDetailSection[];
  /** 流程步骤，演示 el-steps。 */
  steps: { title: string; description: string }[];
  /** 操作时间线，演示 el-timeline。 */
  timeline: { title: string; timestamp: string; type: "primary" | "success" | "warning" | "danger" | "info" }[];
  /** 原始数据，演示 JSON 预览。 */
  raw: Record<string, unknown>;
}

/** 左侧部门树静态数据：三层结构演示树的展开、搜索与联动。 */
export const demoDeptTree: DemoDeptNode[] = [
  {
    id: "1",
    name: "Demo 技术部",
    count: 12,
    children: [
      { id: "11", name: "前端组", count: 5 },
      { id: "12", name: "后端组", count: 7 }
    ]
  },
  {
    id: "2",
    name: "Demo 产品部",
    count: 6,
    children: [
      { id: "21", name: "产品设计组", count: 3 },
      { id: "22", name: "用户体验组", count: 3 }
    ]
  },
  { id: "3", name: "Demo 运营部", count: 4 }
];

/** 员工静态数据：覆盖图片列、金额列、状态列与部门树联动过滤演示。 */
export const demoEmployees: DemoEmployee[] = [
  { id: 1, name: "陈默", dept_id: 11, email: "chenmo@demo.com", phone: "13800000001", avatar: makeAvatar("#409eff", "陈"), salary: 1800000, status: STATUS_ENABLE, entry_date: "2023-03-15" },
  { id: 2, name: "林晚", dept_id: 11, email: "linwan@demo.com", phone: "13800000002", avatar: makeAvatar("#67c23a", "林"), salary: 1650000, status: STATUS_ENABLE, entry_date: "2023-07-01" },
  { id: 3, name: "苏辞", dept_id: 12, email: "suci@demo.com", phone: "13800000003", avatar: makeAvatar("#e6a23c", "苏"), salary: 2100000, status: STATUS_ENABLE, entry_date: "2022-11-20" },
  { id: 4, name: "顾言", dept_id: 12, email: "guyan@demo.com", phone: "13800000004", avatar: makeAvatar("#f56c6c", "顾"), salary: 1950000, status: STATUS_DISABLE, entry_date: "2021-06-08" },
  { id: 5, name: "沈舟", dept_id: 21, email: "shenzhou@demo.com", phone: "13800000005", avatar: makeAvatar("#909399", "沈"), salary: 1500000, status: STATUS_ENABLE, entry_date: "2024-01-10" },
  { id: 6, name: "叶蓁", dept_id: 21, email: "yezhen@demo.com", phone: "13800000006", avatar: makeAvatar("#409eff", "叶"), salary: 1580000, status: STATUS_ENABLE, entry_date: "2024-04-22" },
  { id: 7, name: "江离", dept_id: 22, email: "jiangli@demo.com", phone: "13800000007", avatar: makeAvatar("#67c23a", "江"), salary: 1420000, status: STATUS_DISABLE, entry_date: "2024-09-01" },
  { id: 8, name: "温叙", dept_id: 3, email: "wenxu@demo.com", phone: "13800000008", avatar: makeAvatar("#e6a23c", "温"), salary: 1280000, status: STATUS_ENABLE, entry_date: "2025-02-14" },
  { id: 9, name: "白鹭", dept_id: 3, email: "bailu@demo.com", phone: "13800000009", avatar: makeAvatar("#f56c6c", "白"), salary: 1350000, status: STATUS_ENABLE, entry_date: "2025-03-30" },
  { id: 10, name: "祝余", dept_id: 11, email: "zhuyu@demo.com", phone: "13800000010", avatar: makeAvatar("#909399", "祝"), salary: 1720000, status: STATUS_ENABLE, entry_date: "2023-12-05" }
];

/** 项目等级枚举选项：同时用于标签列展示与搜索下拉、表单单选。 */
export const demoLevelOptions = [
  { label: "S", value: "S", tagType: "danger" },
  { label: "A", value: "A", tagType: "warning" },
  { label: "B", value: "B", tagType: "primary" },
  { label: "C", value: "C", tagType: "info" }
];

/** 城市级联静态数据：演示搜索区 cascader 与 tree-select 控件。 */
export const demoRegionTree = [
  {
    value: "zhejiang",
    label: "浙江省",
    children: [
      { value: "hangzhou", label: "杭州市" },
      { value: "ningbo", label: "宁波市" }
    ]
  },
  {
    value: "guangdong",
    label: "广东省",
    children: [
      { value: "shenzhen", label: "深圳市" },
      { value: "guangzhou", label: "广州市" }
    ]
  },
  {
    value: "sichuan",
    label: "四川省",
    children: [{ value: "chengdu", label: "成都市" }]
  }
];

/** 项目静态数据：25 条覆盖分页、多级表头、进度条、标签列等展示场景。 */
export const demoProjects: DemoProject[] = [
  { id: 1, name: "星尘数据平台", level: "S", owner: "陈默", status: STATUS_ENABLE, budget: 128000000, task_total: 46, progress: 82, region: ["zhejiang", "hangzhou"], created_at: "2025-01-06 09:30:00" },
  { id: 2, name: "流萤网关", level: "A", owner: "苏辞", status: STATUS_ENABLE, budget: 66000000, task_total: 32, progress: 64, region: ["guangdong", "shenzhen"], created_at: "2025-02-11 14:20:00" },
  { id: 3, name: "青梧中台", level: "B", owner: "顾言", status: STATUS_DISABLE, budget: 42000000, task_total: 28, progress: 35, region: ["sichuan", "chengdu"], created_at: "2025-03-02 10:00:00" },
  { id: 4, name: "拾光相册", level: "C", owner: "林晚", status: STATUS_ENABLE, budget: 15000000, task_total: 12, progress: 90, region: ["zhejiang", "ningbo"], created_at: "2025-04-18 16:45:00" },
  { id: 5, name: "北辰监控", level: "A", owner: "沈舟", status: STATUS_ENABLE, budget: 73000000, task_total: 38, progress: 58, region: ["guangdong", "guangzhou"], created_at: "2025-05-09 11:15:00" },
  { id: 6, name: "白泽知识库", level: "S", owner: "叶蓁", status: STATUS_ENABLE, budget: 156000000, task_total: 52, progress: 47, region: ["zhejiang", "hangzhou"], created_at: "2025-06-01 08:40:00" },
  { id: 7, name: "惊蛰调度", level: "B", owner: "江离", status: STATUS_DISABLE, budget: 38000000, task_total: 22, progress: 20, region: ["sichuan", "chengdu"], created_at: "2025-06-25 13:30:00" },
  { id: 8, name: "云阶存储", level: "C", owner: "温叙", status: STATUS_ENABLE, budget: 21000000, task_total: 16, progress: 73, region: ["zhejiang", "hangzhou"], created_at: "2025-07-14 15:10:00" },
  { id: 9, name: "听雨日志", level: "B", owner: "白鹭", status: STATUS_ENABLE, budget: 34000000, task_total: 24, progress: 66, region: ["guangdong", "shenzhen"], created_at: "2025-08-03 09:55:00" },
  { id: 10, name: "扶摇发布", level: "A", owner: "祝余", status: STATUS_ENABLE, budget: 89000000, task_total: 41, progress: 51, region: ["zhejiang", "ningbo"], created_at: "2025-08-20 10:25:00" },
  { id: 11, name: "沧海备份", level: "C", owner: "陈默", status: STATUS_DISABLE, budget: 12000000, task_total: 9, progress: 15, region: ["sichuan", "chengdu"], created_at: "2025-09-01 14:00:00" },
  { id: 12, name: "照夜网盘", level: "B", owner: "苏辞", status: STATUS_ENABLE, budget: 45000000, task_total: 27, progress: 88, region: ["guangdong", "guangzhou"], created_at: "2025-09-12 17:35:00" }
];

/** 目录树静态数据：三层嵌套演示树形表格的展开、折叠与子级操作。 */
export const demoCatalogTree: DemoCatalog[] = [
  {
    id: 1,
    name: "产品文档",
    code: "prod-doc",
    owner: "沈舟",
    status: STATUS_ENABLE,
    doc_count: 32,
    children: [
      {
        id: 11,
        name: "需求说明",
        code: "prod-doc-req",
        owner: "叶蓁",
        status: STATUS_ENABLE,
        doc_count: 12,
        children: [
          { id: 111, name: "V3 需求合集", code: "prod-doc-req-v3", owner: "叶蓁", status: STATUS_ENABLE, doc_count: 8 },
          { id: 112, name: "历史归档", code: "prod-doc-req-arch", owner: "江离", status: STATUS_DISABLE, doc_count: 4 }
        ]
      },
      { id: 12, name: "设计稿说明", code: "prod-doc-ui", owner: "祝余", status: STATUS_ENABLE, doc_count: 20 }
    ]
  },
  {
    id: 2,
    name: "技术文档",
    code: "tech-doc",
    owner: "陈默",
    status: STATUS_ENABLE,
    doc_count: 58,
    children: [
      { id: 21, name: "架构设计", code: "tech-doc-arch", owner: "苏辞", status: STATUS_ENABLE, doc_count: 15 },
      { id: 22, name: "接口文档", code: "tech-doc-api", owner: "顾言", status: STATUS_ENABLE, doc_count: 36 },
      { id: 23, name: "运维手册", code: "tech-doc-ops", owner: "白鹭", status: STATUS_DISABLE, doc_count: 7 }
    ]
  },
  { id: 3, name: "测试文档", code: "qa-doc", owner: "林晚", status: STATUS_ENABLE, doc_count: 21 },
  { id: 4, name: "废弃目录", code: "deprecated", owner: "温叙", status: STATUS_DISABLE, doc_count: 0 }
];
