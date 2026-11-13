import type { AdminComponentApi, AdminComponentDoc } from "@/demo";
import componentApi from "./component-api.json";

const apiByFile: Record<string, AdminComponentApi> = componentApi;
const tableExample = () => import("./TableExample.vue");
const formExample = () => import("./FormExample.vue");
const filterExample = () => import("./FilterExample.vue");
const dialogExample = () => import("./DialogExample.vue");
const uploadExample = () => import("./UploadExample.vue");
const importExample = () => import("./ImportExample.vue");
const editorExample = () => import("./EditorExample.vue");
const displayExample = () => import("./DisplayExample.vue");
const tenantExample = () => import("./TenantExample.vue");
const commonExample = () => import("./CommonExample.vue");
const securityExample = () => import("./SecurityExample.vue");

// 包内示例转义源码别名，避免发布构建改写展示字符串。
const coreSourceAlias = "\u0040/";

/** 组合可直接复制到页面的完整 Vue 示例。 */
function vueCode(imports: string, state: string, template: string) {
  return `<template>\n${template}\n</template>\n\n<script setup lang="ts">\n${imports}\n${state}\n</script>`;
}

const tableCode = vueCode(
  'import { ref } from "vue";\nimport ProTable from "@liujitcn/kratos-admin-core/components/ProTable";\nimport type { ColumnProps } from "@liujitcn/kratos-admin-core/components/ProTable/interface";',
  `
const projects = ref([
  { id: 1, name: "智慧校园平台", owner: "张明", enabled: true },
  { id: 2, name: "校园支付平台", owner: "李华", enabled: false }
]);
const columns: ColumnProps[] = [
  { prop: "name", label: "项目名称", search: { el: "input" } },
  { prop: "owner", label: "负责人" },
  { prop: "enabled", label: "状态", cellType: "status",
    statusProps: { activeValue: true, inactiveValue: false, activeText: "启用", inactiveText: "停用" } }
];

async function requestProjectTable(params: Record<string, any>) {
  const list = projects.value.filter(row => !params.name || row.name.includes(params.name));
  const start = (params.page_num - 1) * params.page_size;
  return { data: { list: list.slice(start, start + params.page_size), total: list.length } };
}`,
  `  <ProTable row-key="id" :columns="columns" :request-api="requestProjectTable" :search-col="2" />`
);

const searchCode = vueCode(
  'import { reactive, ref } from "vue";\nimport SearchForm from "@liujitcn/kratos-admin-core/components/SearchForm/index.vue";\nimport type { ColumnProps } from "@liujitcn/kratos-admin-core/components/ProTable/interface";',
  `
const searchParam = reactive({ name: "", owner: "" });
const result = ref("");
const columns: ColumnProps[] = [
  { prop: "name", label: "项目名称", search: { el: "input" } },
  { prop: "owner", label: "负责人", search: { el: "select" },
    enum: [{ label: "张明", value: "张明" }, { label: "李华", value: "李华" }] }
];
function search() { result.value = JSON.stringify(searchParam); }
function reset() { Object.assign(searchParam, { name: "", owner: "" }); result.value = ""; }`,
  `  <SearchForm :columns="columns" :search-param="searchParam" :search-col="2" :search="search" :reset="reset" />
  <p>{{ result }}</p>`
);

const formCode = vueCode(
  'import { reactive, ref } from "vue";\nimport ProForm from "@liujitcn/kratos-admin-core/components/ProForm/index.vue";\nimport type { ProFormField, ProFormInstance } from "@liujitcn/kratos-admin-core/components/ProForm/interface";',
  `
const formRef = ref<ProFormInstance>();
const model = reactive({ name: "智慧校园平台", notify: true, email: "admin@example.com" });
const result = ref("");
const fields: ProFormField[] = [
  { prop: "name", label: "项目名称", component: "input",
    rules: [{ required: true, message: "请输入项目名称", trigger: "blur" }] },
  { prop: "notify", label: "发送通知", component: "switch" },
  { prop: "email", label: "通知邮箱", component: "input", visible: value => value.notify }
];
async function validate() { result.value = await formRef.value?.validate() ? "校验通过" : "请补齐必填字段"; }`,
  `  <ProForm ref="formRef" :model="model" :fields="fields" :col-span="12" />
  <el-button type="primary" @click="validate">校验表单</el-button>
  <p>{{ result }}</p>`
);

const formDialogCode = vueCode(
  'import { reactive, ref } from "vue";\nimport FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";\nimport type { ProFormField, ProFormInstance } from "@liujitcn/kratos-admin-core/components/ProForm/interface";',
  `
const visible = ref(false);
const formRef = ref<ProFormInstance>();
const model = reactive({ name: "", remark: "" });
const saved = ref("");
const fields: ProFormField[] = [
  { prop: "name", label: "项目名称", component: "input",
    rules: [{ required: true, message: "请输入项目名称", trigger: "blur" }] },
  { prop: "remark", label: "项目说明", component: "textarea" }
];
function save() { saved.value = model.name; visible.value = false; }
function reset() { Object.assign(model, { name: "", remark: "" }); formRef.value?.clearValidate(); }`,
  `  <el-button @click="visible = true">新增项目</el-button>
  <FormDialog ref="formRef" v-model="visible" title="新增项目" :model="model" :fields="fields" @confirm="save" @closed="reset" />
  <p v-if="saved">已保存：{{ saved }}</p>`
);

const proDialogCode = vueCode(
  'import { ref } from "vue";\nimport ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";',
  "\nconst visible = ref(false);\nfunction confirm() { visible.value = false; }",
  `  <el-button @click="visible = true">查看项目</el-button>
  <ProDialog v-model="visible" title="项目预览" @confirm="confirm">
    <p>智慧校园平台</p>
  </ProDialog>`
);

const dynamicCode = vueCode(
  'import { ref } from "vue";\nimport DynamicList from "@liujitcn/kratos-admin-core/components/ProForm/components/DynamicList.vue";',
  '\nconst contacts = ref(["admin@example.com"]);',
  '  <DynamicList v-model="contacts" :input-props="{ placeholder: \'输入通知收件人\' }" />\n  <pre>{{ contacts }}</pre>'
);
const kvCode = vueCode(
  'import { ref } from "vue";\nimport KvList from "@liujitcn/kratos-admin-core/components/ProForm/components/KvList.vue";',
  '\nconst variables = ref([{ key: "ENV", value: "demo" }]);',
  '  <KvList v-model="variables" add-text="新增参数" />\n  <pre>{{ variables }}</pre>'
);
const cronCode = vueCode(
  'import { ref } from "vue";\nimport CronExpression from "\u0040/components/CronExpression/index.vue";',
  '\nconst cron = ref("0 */5 * * * *");',
  '  <CronExpression v-model="cron" />\n  <p>当前表达式：{{ cron }}</p>'
);
const treeCode = vueCode(
  'import { ref } from "vue";\nimport TreeFilter from "@liujitcn/kratos-admin-core/components/TreeFilter/index.vue";',
  `
const selected = ref("");
const categories = [
  { id: "campus", label: "校园服务", children: [
    { id: "payment", label: "校园支付" }, { id: "visitor", label: "访客管理" }
  ] }
];`,
  '  <TreeFilter title="项目分类" :data="categories" @change="selected = $event" />\n  <p>当前分类：{{ selected }}</p>'
);
const filterCode = vueCode(
  'import { ref } from "vue";\nimport SelectFilter from "\u0040/components/SelectFilter/index.vue";',
  `
const selected = ref({});
const filters = [{ title: "项目状态", key: "status", options: [
  { label: "全部", value: "" }, { label: "启用", value: "enabled" }, { label: "停用", value: "disabled" }
] }];`,
  '  <SelectFilter :data="filters" :default-values="{ status: \'\' }" @change="selected = $event" />\n  <pre>{{ selected }}</pre>'
);

const imageUploadCode = vueCode(
  'import { onBeforeUnmount, ref } from "vue";\nimport UploadImg from "@liujitcn/kratos-admin-core/components/Upload/Img.vue";\nimport type { FileInfo } from "@liujitcn/kratos-admin-core/rpc/base/v1/file";',
  `
const imageUrl = ref("");
const urls: string[] = [];
async function previewImage(file: File): Promise<FileInfo> {
  const url = URL.createObjectURL(file);
  urls.push(url);
  return { name: file.name, url, extname: file.name.split(".").at(-1) ?? "" };
}
onBeforeUnmount(() => urls.forEach(url => URL.revokeObjectURL(url)));`,
  '  <UploadImg v-model:image-url="imageUrl" :api="previewImage" width="180px" height="150px" />'
);
const imagesUploadCode = imageUploadCode
  .replace(
    'import UploadImg from "@liujitcn/kratos-admin-core/components/Upload/Img.vue";',
    'import UploadImgs from "\u0040/components/Upload/Imgs.vue";\nimport type { UploadUserFile } from "element-plus";'
  )
  .replace('const imageUrl = ref("");', "const images = ref<UploadUserFile[]>([]);")
  .replace(
    '<UploadImg v-model:image-url="imageUrl" :api="previewImage" width="180px" height="150px" />',
    '<UploadImgs v-model:file-list="images" :api="previewImage" :limit="3" />'
  );
const fileUploadCode = vueCode(
  'import { ref } from "vue";\nimport UploadFile from "@liujitcn/kratos-admin-core/components/Upload/File.vue";\nimport type { UploadUserFile } from "element-plus";',
  "\nconst file = ref<UploadUserFile>();",
  '  <UploadFile v-model:file-info="file" upload-type="document" :file-size="20" />'
);
const filesUploadCode = vueCode(
  'import { ref } from "vue";\nimport UploadFiles from "\u0040/components/Upload/Files.vue";\nimport type { UploadUserFile } from "element-plus";',
  "\nconst files = ref<UploadUserFile[]>([]);",
  '  <UploadFiles v-model:file-list="files" upload-type="document" :limit="5" :file-size="20" />'
);
const importCode = vueCode(
  'import { ref } from "vue";\nimport ImportExcel from "\u0040/components/ImportExcel/index.vue";',
  `
const importRef = ref<InstanceType<typeof ImportExcel>>();
const result = ref("");
function openImport() {
  importRef.value?.acceptParams({
    title: "项目",
    // 业务页面在这里传入模板下载接口。
    importApi: async (form: FormData) => {
      const file = form.get("file") as File;
      result.value = "已选择 " + file.name + "，覆盖选项：" + form.get("isCover");
    }
  });
}`,
  '  <el-button @click="openImport">打开导入弹窗</el-button>\n  <ImportExcel ref="importRef" />\n  <p>{{ result }}</p>'
);

const codeEditorCode = vueCode(
  'import { ref } from "vue";\nimport CodeEditor from "@liujitcn/kratos-admin-core/components/CodeEditor/index.vue";',
  "\nconst code = ref('const enabled = true;');",
  '  <CodeEditor v-model="code" language="typescript" height="280px" />'
);
const yamlCode = vueCode(
  'import { ref } from "vue";\nimport YamlEditor from "@liujitcn/kratos-admin-core/components/YamlEditor/index.vue";',
  '\nconst yaml = ref("app:\\n  name: 智慧校园平台\\n  enabled: true");',
  '  <YamlEditor v-model="yaml" min-height="240px" max-height="360px" />'
);
const markdownEditorCode = vueCode(
  'import { ref } from "vue";\nimport MarkdownEditor from "@liujitcn/kratos-admin-core/components/MarkdownEditor/index.vue";\nimport { useGlobalStore } from "\u0040/stores/modules/global";',
  '\nconst globalStore = useGlobalStore();\nconst markdown = ref("# 项目说明\\n\\n欢迎使用 **智慧校园平台**。");',
  '  <MarkdownEditor v-model="markdown" :is-dark="globalStore.isDark" height="340px" />'
);
const wangCode = vueCode(
  'import { ref } from "vue";\nimport WangEditor from "@liujitcn/kratos-admin-core/components/WangEditor/index.vue";',
  `
const html = ref("<h2>项目公告</h2><p>欢迎使用智慧校园平台。</p>");
const toolbar = { toolbarKeys: ["headerSelect", "bold", "italic", "underline", "bulletedList", "undo", "redo"] };
const editorConfig = { pasteIgnoreImg: true, customPaste: (_editor: unknown, event: ClipboardEvent) => {
  if (!event.clipboardData?.files.length) return true;
  event.preventDefault();
  return false;
} };`,
  '  <div @drop.capture.prevent>\n    <WangEditor v-model:value="html" height="320px" :toolbar-config="toolbar" :editor-config="editorConfig" />\n  </div>'
);
const markdownPreviewCode = vueCode(
  'import { ref } from "vue";\nimport MarkdownPreview from "@liujitcn/kratos-admin-core/components/MarkdownPreview/index.vue";\nimport { useGlobalStore } from "\u0040/stores/modules/global";',
  '\nconst globalStore = useGlobalStore();\nconst markdown = ref("# 项目说明\\n\\n支持 **统一入口** 与消息通知。");',
  '  <MarkdownPreview :model-value="markdown" :is-dark="globalStore.isDark" />'
);
const richPreviewCode = vueCode(
  'import RichTextPreview from "@liujitcn/kratos-admin-core/components/RichTextPreview/index.vue";',
  '\nconst html = "<h2>项目公告</h2><p>欢迎使用 <strong>智慧校园平台</strong>。</p>";',
  '  <RichTextPreview :model-value="html" />'
);
const codeBlockCode = vueCode(
  'import CodeBlock from "@liujitcn/kratos-admin-core/components/CodeBlock/index.vue";',
  '\nconst code = "const enabled = true;";',
  '  <CodeBlock :code="code" language="typescript" max-height="280px" />'
);

const dictCode = vueCode(
  'import { ref } from "vue";\nimport Dict from "\u0040/components/Dict/index.vue";',
  "\n// 使用业务已配置的字典编码，选项由现有字典缓存和服务提供。\nconst status = ref<number>();",
  '  <Dict v-model="status" code="status" code-type="number" type="select" />'
);
const dictLabelCode = vueCode(
  'import DictLabel from "@liujitcn/kratos-admin-core/components/Dict/DictLabel.vue";',
  "\nconst status = 1;",
  '  <DictLabel code="status" :model-value="status" />'
);
const tenantCode = vueCode(
  'import { ref } from "vue";\nimport TenantSelect from "@liujitcn/kratos-admin-core/components/TenantSelect/index.vue";\nimport TenantText from "@liujitcn/kratos-admin-core/components/TenantText/index.vue";',
  '\nconst tenantId = ref<string | number>();\nconst options = [{ label: "北校区", value: 1, disabled: false }, { label: "南校区", value: 2, disabled: false }];',
  '  <TenantSelect v-model="tenantId" :options="options" placeholder="选择租户" clearable />\n  <TenantText :tenant-id="tenantId" :options="options" />'
);
const tenantTextCode = vueCode(
  'import TenantText from "@liujitcn/kratos-admin-core/components/TenantText/index.vue";',
  '\nconst options = [{ label: "北校区", value: 1, disabled: false }];',
  '  <TenantText :tenant-id="1" :options="options" />\n  <TenantText tenant-name="智慧校园示例租户" />'
);

const chartCode = vueCode(
  'import ECharts from "@liujitcn/kratos-admin-core/components/ECharts/index.vue";\nimport type { ECOption } from "\u0040/components/ECharts/config";',
  `
const option: ECOption = {
  tooltip: { trigger: "axis" },
  xAxis: { type: "category", data: ["第一周", "第二周", "第三周", "第四周"] },
  yAxis: { type: "value" },
  series: [{ type: "bar", data: [1860, 2450, 3200, 5350] }]
};`,
  '  <div style="height: 280px"><ECharts :option="option" /></div>'
);
const cardCode = vueCode(
  'import { ref } from "vue";\nimport DataPanelCard from "\u0040/components/Card/DataPanelCard.vue";',
  "\nconst visits = ref(12860);",
  '  <DataPanelCard title="本月访问" description="最近四周的本地示例数据" primary>\n    <template #extra><el-button @click="visits += 120">模拟新增</el-button></template>\n    <p>{{ visits.toLocaleString() }} 次访问</p>\n  </DataPanelCard>'
);
const gridCode = vueCode(
  'import Grid from "\u0040/components/Grid/index.vue";\nimport GridItem from "\u0040/components/Grid/components/GridItem.vue";',
  "",
  '  <Grid :cols="{ xs: 1, sm: 2, md: 2, lg: 3, xl: 3 }" :gap="16">\n    <GridItem>项目概览</GridItem>\n    <GridItem>待办事项</GridItem>\n    <GridItem :span="2" :xs="{ span: 1 }">最近活动</GridItem>\n  </Grid>'
);
const selectIconCode = vueCode(
  'import { ref } from "vue";\nimport SelectIcon from "@liujitcn/kratos-admin-core/components/SelectIcon/index.vue";',
  '\nconst icon = ref("Setting");',
  '  <SelectIcon v-model:icon-value="icon" title="选择图标" />\n  <p>{{ icon }}</p>'
);
const svgCode = vueCode(
  'import SvgIcon from "\u0040/components/SvgIcon/index.vue";',
  "",
  '  <svg width="0" height="0" aria-hidden="true">\n    <symbol id="example-circle" viewBox="0 0 24 24">\n      <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="2" />\n    </symbol>\n  </svg>\n  <SvgIcon icon-class="circle" prefix="example" :size="32" />'
);
const darkCode = vueCode('import SwitchDark from "\u0040/components/SwitchDark/index.vue";', "", "  <SwitchDark />");
const localeCode = vueCode('import LocaleSwitch from "\u0040/components/LocaleSwitch/index.vue";', "", "  <LocaleSwitch />");
const loadingCode = vueCode(
  'import { ref } from "vue";\nimport Loading from "\u0040/components/Loading/index.vue";',
  "\nconst loading = ref(true);",
  '  <el-switch v-model="loading" active-text="显示加载状态" />\n  <div style="height: 240px"><Loading v-if="loading" /></div>'
);
const errorCode = vueCode(
  'import ErrorMessage from "\u0040/components/ErrorMessage/index.vue";\nimport image from "@liujitcn/kratos-admin-core/assets/images/404.png";',
  "",
  '  <ErrorMessage code="404" :image="image" message="未找到你要访问的页面" />'
);
const secretCode = vueCode(
  'import { ref } from "vue";\nimport SecretInput from "\u0040/components/ProForm/components/SecretInput.vue";',
  '\nconst secret = ref("Demo-2026!");',
  '  <SecretInput v-model="secret" placeholder="输入示例密钥" />'
);
const strengthCode = vueCode(
  'import { ref } from "vue";\nimport PasswordStrength from "@liujitcn/kratos-admin-core/components/PasswordStrength/index.vue";',
  '\nconst password = ref("Demo-2026!");',
  '  <el-input v-model="password" type="password" show-password />\n  <PasswordStrength :password="password" tip="建议使用至少 8 位字符，组合大小写字母、数字和符号。" />'
);
const passwordVerifyCode = vueCode(
  'import { ref } from "vue";\nimport PasswordVerifyDialog from "@liujitcn/kratos-admin-core/components/Dialog/PasswordVerifyDialog.vue";\nimport type { SecretCrypto } from "@liujitcn/kratos-admin-core/rpc/common/v1/types";',
  `
const visible = ref(false);
const result = ref("");
function confirm(password: SecretCrypto) {
  // 业务页面将此加密结果传给需要二次认证的接口。
  result.value = password ? "已取得加密密码" : "";
  visible.value = false;
}`,
  '  <el-button @click="visible = true">验证密码</el-button>\n  <PasswordVerifyDialog v-model="visible" description="确认敏感操作前验证当前密码" @confirm="confirm" />\n  <p>{{ result }}</p>'
);
const mfaCode = vueCode(
  'import MfaSetupPanel from "@liujitcn/kratos-admin-core/components/Mfa/MfaSetupPanel.vue";',
  '\n// 公开演示密钥；业务页面使用 MFA 绑定接口返回的 URI。\nconst uri = "otpauth://totp/Demo:component-docs?secret=JBSWY3DPEHPK3PXP&issuer=Demo";',
  '  <MfaSetupPanel :uri="uri" />'
);
const recoveryCode = vueCode(
  'import { ref } from "vue";\nimport MfaRecoveryCodesDialog from "@liujitcn/kratos-admin-core/components/Mfa/MfaRecoveryCodesDialog.vue";',
  '\nconst visible = ref(false);\nconst codes = ["DEMO-0001-ABCD", "DEMO-0002-EFGH", "DEMO-0003-IJKL", "DEMO-0004-MNOP"];',
  '  <el-button @click="visible = true">查看恢复码</el-button>\n  <MfaRecoveryCodesDialog v-model="visible" :codes="codes" />'
);

/** 为 Core 组件附加所属包和源码提取的 API。 */
function componentDoc(
  id: string,
  label: string,
  group: string,
  file: string,
  description: string,
  usage: string,
  code: string,
  example?: AdminComponentDoc["example"],
  related: string[] = []
): AdminComponentDoc {
  return {
    id,
    label,
    group,
    owner: "Core",
    file,
    description,
    usage: code.includes(coreSourceAlias) ? `${usage}\n此示例用于 Core 源码，${coreSourceAlias} 指向 Core 的 src 目录。` : usage,
    code,
    api: apiByFile[file],
    example,
    related
  };
}

/** Core 全部自定义组件的中文说明、准确用法和本地演示。 */
export const coreComponentDocs: AdminComponentDoc[] = [
  componentDoc(
    "ProTable",
    "高级表格",
    "表格与筛选",
    "ProTable/index.vue",
    "配置驱动的业务表格，统一查询、分页、列设置、预置单元格和行内操作。",
    "用 columns 声明搜索项与单元格类型，用 headerActions 配置表头按钮。页面负责请求与业务操作；分页接口返回 { data: { list, total } }，非分页接口返回 { data: list }。示例可以搜索、新增、编辑、删除、切换状态与体验树形表格。",
    tableCode,
    tableExample,
    ["SearchForm", "TableColumn", "Pagination", "ColSetting", "FormDialog"]
  ),
  componentDoc(
    "SearchForm",
    "搜索表单",
    "表格与筛选",
    "SearchForm/index.vue",
    "根据列配置生成搜索条件，支持响应式栅格、展开收起和字段插槽。",
    "通过 columns、searchParam 与 searchCol 声明表单，search 和 reset 由页面实现。search.el 选择控件，search.key 可独立指定请求字段；通常由 ProTable 统一装配。本地表格示例展示了真实搜索与重置。",
    searchCode,
    tableExample,
    ["SearchFormItem", "Grid", "GridItem", "ProTable"]
  ),
  componentDoc(
    "TreeFilter",
    "树形筛选",
    "表格与筛选",
    "TreeFilter/index.vue",
    "用树节点筛选数据，支持名称搜索、单选、多选与异步懒加载。",
    "静态场景传入 data，监听 change 获取节点 ID；multiple 模式返回 ID 数组。requestApi 的响应使用 { data: nodes }，懒加载按 parentKey 指定的字段请求子节点。示例只使用本地树数据。",
    treeCode,
    filterExample,
    ["ProTable", "SelectFilter"]
  ),
  componentDoc(
    "SelectFilter",
    "条件筛选",
    "表格与筛选",
    "SelectFilter/index.vue",
    "将枚举条件展示为可选择的筛选项，支持多组条件和多选。",
    "data 按组配置 title、key 和 options，defaultValues 提供初值，change 返回全部当前条件。多选组的首个选项表示全部，选择其他选项时会清除全部状态。",
    filterCode,
    filterExample,
    ["TreeFilter", "SearchForm"]
  ),
  componentDoc(
    "Pagination",
    "分页器",
    "表格与筛选",
    "ProTable/components/Pagination.vue",
    "ProTable 内部分页器，同步当前页、每页条数与总数。",
    "由 ProTable 传入 pageable 和两个分页回调。pageable 使用 page_num、page_size、total；业务页通常只需提供表格请求，不直接组装分页器。示例底部可切换页码和每页条数。",
    tableCode,
    tableExample,
    ["ProTable"]
  ),
  componentDoc(
    "ColSetting",
    "列设置",
    "表格与筛选",
    "ProTable/components/ColSetting.vue",
    "ProTable 内部列设置抽屉，调整各列的显隐与排序能力。",
    "由表格工具栏按钮打开，colSetting 传入可配置列；isSetting: false 可隐藏配置项。当前组件支持列显隐与是否允许排序，不提供拖动列顺序的能力。示例使用实际抽屉，可通过右上角关闭。",
    tableCode,
    tableExample,
    ["ProTable", "TableColumn"]
  ),
  componentDoc(
    "TableColumn",
    "表格列渲染",
    "表格与筛选",
    "ProTable/components/TableColumn.vue",
    "ProTable 内部列渲染器，处理嵌套列、字典、预置单元格与操作按钮。",
    "通过 ProTable 的 columns 使用。cellType 支持 image、status、actions、money；金额默认将分转换为元，状态列的回调用于同步业务结果。render 与具名插槽只用于配置无法表达的内容。",
    tableCode,
    tableExample,
    ["ProTable", "DictLabel"]
  ),
  componentDoc(
    "SearchFormItem",
    "搜索字段",
    "表格与筛选",
    "SearchForm/components/SearchFormItem.vue",
    "SearchForm 内部字段渲染器，将搜索配置转换为对应输入控件。",
    "随 SearchForm 或 ProTable 使用，通过 column.search 指定控件、透传属性、枚举与字典。search.enum 优先于列 enum，选择值写入 searchParam。示例中的项目名称和负责人即由该组件渲染。",
    searchCode,
    tableExample,
    ["SearchForm", "ProTable"]
  ),
  componentDoc(
    "ProForm",
    "配置式表单",
    "表单",
    "ProForm/index.vue",
    "用字段配置描述表单，统一校验、栅格布局、动态显隐与插槽扩展。",
    "model 持有页面数据，fields 声明字段、控件、规则、选项和显隐函数。使用 colSpan 与字段 colSpan 调整布局；validate 返回校验结果。示例可切换通知开关与渠道，体验字段联动、动态列表和键值参数。",
    formCode,
    formExample,
    ["ProFormItem", "DynamicList", "KvList", "CronExpression", "FormDialog"]
  ),
  componentDoc(
    "ProFormItem",
    "表单字段",
    "表单",
    "ProForm/components/ProFormItem.vue",
    "ProForm 内部字段渲染器，按 component 配置选择输入、选择、上传或编辑器。",
    "通过 ProForm 的 fields 使用，field.prop 对应 model 字段，field.props 可按当前模型动态生成。组件只负责控件绑定，字段标签、校验和布局由 ProForm 管理。",
    formCode,
    formExample,
    ["ProForm", "SecretInput", "DynamicList", "KvList"]
  ),
  componentDoc(
    "DynamicList",
    "动态列表",
    "表单",
    "ProForm/components/DynamicList.vue",
    "编辑字符串列表，支持新增、删除条目与输入属性配置。",
    "v-model 绑定字符串数组，inputProps 透传占位、长度限制等输入属性。也可以在 ProForm 中将 component 设为 dynamic-list。新增和删除会回传新的数组。",
    dynamicCode,
    formExample,
    ["ProForm", "KvList"]
  ),
  componentDoc(
    "KvList",
    "键值列表",
    "表单",
    "ProForm/components/KvList.vue",
    "编辑键值对数组，适合环境变量、扩展属性和请求参数。",
    "v-model 绑定 { key, value } 数组；keyInputProps、valueInputProps 分别控制两个输入框，addText 配置新增按钮文案。组件不自动去重或转换为对象，业务校验由页面处理。",
    kvCode,
    formExample,
    ["ProForm", "DynamicList"]
  ),
  componentDoc(
    "CronExpression",
    "定时表达式",
    "表单",
    "CronExpression/index.vue",
    "通过可视化分段编辑器和常用预设生成定时表达式。",
    "v-model 双向绑定表达式。当前实现使用秒、分、时、日、月、年六个字段；点击编辑入口调整分段并确认后回填。提交到业务服务前，页面应采用服务端约定的 Cron 格式。",
    cronCode,
    formExample,
    ["ProForm"]
  ),
  componentDoc(
    "FormDialog",
    "表单弹窗",
    "弹窗",
    "Dialog/FormDialog.vue",
    "组合通用弹窗与配置式表单，统一表单布局、校验与打开流程。",
    "v-model 控制显示，model 与 fields 声明表单。点击确定会先校验，通过后触发 confirm，由页面执行提交并关闭；closed 中重置表单与校验。open 可异步加载并只应用最新一次打开结果。",
    formDialogCode,
    dialogExample,
    ["ProDialog", "ProForm"]
  ),
  componentDoc(
    "ProDialog",
    "通用弹窗",
    "弹窗",
    "Dialog/ProDialog.vue",
    "通用弹窗容器，支持确认、取消、加载状态和自定义底部操作。",
    "默认插槽放置内容，v-model 控制显示，confirm 由页面处理；取消会关闭并通知调用方。showFooter 可隐藏默认底部，header 和 footer 插槽用于特殊布局。规则表单优先使用 FormDialog。",
    proDialogCode,
    dialogExample,
    ["FormDialog"]
  ),
  componentDoc(
    "PasswordVerifyDialog",
    "密码验证弹窗",
    "弹窗",
    "Dialog/PasswordVerifyDialog.vue",
    "在敏感操作前采集密码，校验后向业务页面返回加密结果。",
    "confirm 返回 SecretCrypto，页面将其传入敏感操作接口，并用 confirmLoading 表达提交状态。组件确认时固定调用密码加密流程获取服务端公钥，没有可替换的加密 API，因此此页提供业务接入代码，不运行密码确认预览。关闭后会清理密码与校验。",
    passwordVerifyCode,
    undefined,
    ["ProDialog", "SecretInput", "PasswordStrength"]
  ),
  componentDoc(
    "UploadImg",
    "单图上传",
    "上传与导入",
    "Upload/Img.vue",
    "选择单张图片，提供预览、替换、删除和文件校验。",
    "v-model:imageUrl 绑定地址，api 可替换上传过程并返回 FileInfo。默认限制 JPG、PNG、GIF 和 5 MB；uploadType 与 accessMode 控制正式文件服务的业务分类和访问方式。本例 api 仅创建浏览器对象地址。",
    imageUploadCode,
    uploadExample,
    ["UploadImgs", "ProForm"]
  ),
  componentDoc(
    "UploadImgs",
    "多图上传",
    "上传与导入",
    "Upload/Imgs.vue",
    "管理多张图片，支持数量限制、预览、删除与上传状态反馈。",
    "v-model:fileList 绑定 Element Plus 文件数组，api 返回 FileInfo，limit 设置最大数量。上传和删除会同步文件列表。本例最多选择 3 张，图片只保留在浏览器中，离开示例后释放临时地址。",
    imagesUploadCode,
    uploadExample,
    ["UploadImg", "ProForm"]
  ),
  componentDoc(
    "UploadFile",
    "单文件上传",
    "上传与导入",
    "Upload/File.vue",
    "管理单个文件，支持上传校验、文件卡片、下载和删除。",
    "v-model:fileInfo 绑定文件信息，api 可替换上传，fileType 传入允许的 MIME 类型。下载固定经过现有文件服务，不能随本地 api 替换；为保证文档页不访问业务文件接口，此页提供正式接入代码。",
    fileUploadCode,
    undefined,
    ["UploadFiles", "ProForm"]
  ),
  componentDoc(
    "UploadFiles",
    "多文件上传",
    "上传与导入",
    "Upload/Files.vue",
    "管理文件列表，支持数量、大小和格式限制，以及下载与移除。",
    "v-model:fileList 绑定文件数组，limit 设置数量限制，api 可替换上传。文件卡片的下载固定调用文件服务，缺少独立的本地下载回调；此页提供业务接入代码，文件访问权限由服务端校验。",
    filesUploadCode,
    undefined,
    ["UploadFile", "ProForm"]
  ),
  componentDoc(
    "ImportExcel",
    "表格导入",
    "上传与导入",
    "ImportExcel/index.vue",
    "通过弹窗下载模板、选择 Excel 文件并设置数据覆盖方式。",
    "通过实例 acceptParams 传入 title、tempApi、importApi 与可选刷新回调。importApi 收到包含 file 和 isCover 的 FormData。本例下载本地 Excel 模板，并仅显示所选文件及覆盖选项，不解析或提交文件。",
    importCode,
    importExample,
    ["ProDialog", "UploadFile"]
  ),
  componentDoc(
    "CodeEditor",
    "代码编辑器",
    "编辑与预览",
    "CodeEditor/index.vue",
    "基于 Monaco 编辑源码，支持语言、高度与只读模式。",
    "v-model 绑定代码文本，language 使用 Monaco 语言标识，height 设置容器高度。readonly 在编辑器初始化时应用；需要切换只读状态时重新挂载组件。示例按需加载编辑器及本地 worker。",
    codeEditorCode,
    editorExample,
    ["YamlEditor", "CodeBlock"]
  ),
  componentDoc(
    "YamlEditor",
    "配置编辑器",
    "编辑与预览",
    "YamlEditor/index.vue",
    "编辑 YAML 配置，提供语法高亮、行号、折叠和光标位置状态。",
    "v-model 绑定配置文本，minHeight 与 maxHeight 控制编辑区域；工具栏提供复制和换行切换，invalid 可标记外部校验失败。组件负责编辑呈现，YAML 解析与业务校验由调用方处理。",
    yamlCode,
    editorExample,
    ["CodeEditor", "CodeBlock"]
  ),
  componentDoc(
    "MarkdownEditor",
    "文档编辑器",
    "编辑与预览",
    "MarkdownEditor/index.vue",
    "编辑 Markdown 文本，支持主题、高度、占位提示和长度限制。",
    "v-model 绑定 Markdown 原文，isDark 与全局主题同步，height 控制编辑高度，maxLength 设置字符上限。组件不负责业务保存；预览展示使用 MarkdownPreview。",
    markdownEditorCode,
    editorExample,
    ["MarkdownPreview", "CodeBlock"]
  ),
  componentDoc(
    "WangEditor",
    "富文本编辑器",
    "编辑与预览",
    "WangEditor/index.vue",
    "富文本编辑封装，支持工具栏、编辑器配置和文件服务上传。",
    "v-model:value 绑定 HTML。toolbarConfig 和 editorConfig 透传编辑器配置，uploadType 指定图片视频分类。组件内置媒体上传连接文件服务；本例只提供文本工具并阻止粘贴、拖入图片，正式媒体上传需在业务环境使用。",
    wangCode,
    editorExample,
    ["RichTextPreview", "UploadImg"]
  ),
  componentDoc(
    "MarkdownPreview",
    "文档预览",
    "编辑与预览",
    "MarkdownPreview/index.vue",
    "展示 Markdown 内容、代码高亮与代码块复制操作。",
    "modelValue 接收原文，isDark 同步主题，maxCodeHeight 控制长代码块高度。组件清洗原生 HTML，并为中文及重复标题生成锚点；多个实例可传入独立 id。",
    markdownPreviewCode,
    editorExample,
    ["MarkdownEditor", "CodeBlock"]
  ),
  componentDoc(
    "RichTextPreview",
    "富文本预览",
    "编辑与预览",
    "RichTextPreview/index.vue",
    "清洗并展示 HTML 正文，统一内容排版和媒体地址。",
    "modelValue 传入 HTML，组件使用现有清洗库过滤不安全内容，并规范化图片和视频地址。适用于已保存的富文本详情；业务仍需在服务端执行内容校验。",
    richPreviewCode,
    editorExample,
    ["WangEditor", "MarkdownPreview"]
  ),
  componentDoc(
    "CodeBlock",
    "代码展示",
    "编辑与预览",
    "CodeBlock/index.vue",
    "高亮展示代码原文，提供语言识别、复制和最大高度限制。",
    "code 传入原文，language 指定高亮语言，未指定时自动检测；maxHeight 限制代码区高度，超出后内部滚动。适合文档、配置预览与调试结果展示。",
    codeBlockCode,
    editorExample,
    ["CodeEditor", "MarkdownPreview"]
  ),
  componentDoc(
    "Dict",
    "字典选择",
    "数据选择",
    "Dict/index.vue",
    "按字典编码加载选项，支持下拉、单选、多选和不同值类型。",
    "code 对应业务字典编码，codeType 决定字符串或数值，type 选择 select、radio、checkbox。组件从共享字典缓存取值，缓存未命中时自动请求字典服务，且没有本地 options 参数；此页提供用法代码，不写入共享缓存。示例编码需替换为业务已配置的字典。",
    dictCode,
    undefined,
    ["DictLabel", "ProForm", "SearchForm"]
  ),
  componentDoc(
    "DictLabel",
    "字典标签",
    "数据选择",
    "Dict/DictLabel.vue",
    "把字典值转换为可读标签，并按字典配置展示标签颜色。",
    "code 指定字典，modelValue 指定值；配置有效 tag_type 时使用标签，否则输出文本。组件复用共享字典缓存，未命中会请求服务，没有本地选项入口，因此此页仅展示接入代码。",
    dictLabelCode,
    undefined,
    ["Dict", "TableColumn"]
  ),
  componentDoc(
    "TenantSelect",
    "租户选择",
    "数据选择",
    "TenantSelect/index.vue",
    "选择租户上下文，支持显式传入选项和加载状态。",
    "只向默认租户账号展示，普通租户账号按现有权限规则隐藏。传入 options 时直接使用页面选项，不发起租户加载请求；v-model 和 change 返回所选租户。示例选项为本地校园数据。",
    tenantCode,
    tenantExample,
    ["TenantText", "ProForm", "SearchForm"]
  ),
  componentDoc(
    "TenantText",
    "租户文本",
    "数据选择",
    "TenantText/index.vue",
    "展示租户名称，支持直接传入名称或按编号匹配选项。",
    "tenantName 优先于编号匹配；tenantId 配合 options 查找标签，无法匹配时显示编号。此组件只负责展示，不发起租户接口请求。",
    tenantTextCode,
    tenantExample,
    ["TenantSelect"]
  ),
  componentDoc(
    "ECharts",
    "图表容器",
    "数据展示",
    "ECharts/index.vue",
    "封装图表生命周期、尺寸变化和图表事件。",
    "页面准备 ECOption 并传入 option；父容器需有明确高度。onClick 与 onMouseup 接收图表事件，实例暴露 draw、resize 与 getInstance。当前默认配置注册了 Canvas 渲染器，直接使用默认 renderer。",
    chartCode,
    displayExample,
    ["DataPanelCard"]
  ),
  componentDoc(
    "DataPanelCard",
    "数据面板卡片",
    "数据展示",
    "Card/DataPanelCard.vue",
    "统一数据面板的标题、辅助说明、扩展操作和内容区域。",
    "title 为标题，description 为辅助说明，primary 选择主要样式；extra 插槽放置操作，默认插槽放置指标和图表。内容区使用组件的固定面板高度，适合概览与仪表盘。",
    cardCode,
    displayExample,
    ["ECharts", "Grid"]
  ),
  componentDoc(
    "Grid",
    "响应式栅格",
    "数据展示",
    "Grid/index.vue",
    "按屏幕断点排列内容，支持列数、间距和折叠行数。",
    "cols 可传固定列数或 xs/sm/md/lg/xl 断点配置，gap 可传数值或横纵间距数组。内部使用 GridItem；折叠列表应通过 v-for 渲染并传入 index，便于识别折叠位置。",
    gridCode,
    displayExample,
    ["GridItem", "SearchForm"]
  ),
  componentDoc(
    "GridItem",
    "栅格项",
    "数据展示",
    "Grid/components/GridItem.vue",
    "响应式栅格内部项，配置跨列、偏移与各断点布局。",
    "放在 Grid 内部使用，span 设置占位，offset 设置偏移，断点属性可覆盖当前配置；suffix 将操作项靠后布局。父容器提供列数、间距和折叠上下文，不建议脱离 Grid 单独使用。",
    gridCode,
    displayExample,
    ["Grid", "SearchForm"]
  ),
  componentDoc(
    "SelectIcon",
    "图标选择",
    "通用",
    "SelectIcon/index.vue",
    "从平台内置图标中搜索和选择，支持清空与当前值回显。",
    "v-model:iconValue 绑定图标名称，title 与 placeholder 自定义文案。点击输入框打开选择弹窗，可按英文图标名搜索；该组件返回名称，图标渲染由使用方处理。",
    selectIconCode,
    commonExample,
    ["SvgIcon"]
  ),
  componentDoc(
    "SvgIcon",
    "矢量图标",
    "通用",
    "SvgIcon/index.vue",
    "按 SVG symbol 名称渲染图标，统一前缀、尺寸和自定义样式。",
    "iconClass 与 prefix 组合为 symbol ID，图标需由项目 SVG 构建插件或页面 symbol 提供。size 支持数值像素或 CSS 尺寸，iconStyle 可补充样式；本例使用本地 symbol 展示不同尺寸。",
    svgCode,
    commonExample,
    ["SelectIcon"]
  ),
  componentDoc(
    "SwitchDark",
    "明暗切换",
    "通用",
    "SwitchDark/index.vue",
    "切换管理端明暗主题，复用全局主题状态与样式。",
    "直接放置组件即可。交互会修改当前管理端全局主题，而不是独立的示例主题；灰色和色弱模式沿用管理端全局滤镜。",
    darkCode,
    commonExample,
    ["LocaleSwitch"]
  ),
  componentDoc(
    "LocaleSwitch",
    "语言切换",
    "通用",
    "LocaleSwitch/index.vue",
    "切换管理端语言，同步更新页面和组件文案。",
    "组件使用管理端已配置的语言选项，多于一种语言时显示入口。选择后更新全局语言并刷新动态本地化数据，业务页面无需单独维护语言状态。",
    localeCode,
    commonExample,
    ["SwitchDark"]
  ),
  componentDoc(
    "Loading",
    "加载状态",
    "通用",
    "Loading/index.vue",
    "统一展示内容区域的加载动画。",
    "使用 v-if 或条件分支控制显示，父容器决定动画区域大小。全屏加载由既有 Loading 工具管理；局部内容加载直接使用此组件。",
    loadingCode,
    commonExample,
    ["ErrorMessage"]
  ),
  componentDoc(
    "ErrorMessage",
    "错误状态页",
    "通用",
    "ErrorMessage/index.vue",
    "展示错误编号、插图与说明，用于权限、资源和服务错误页面。",
    "code、image、message 均由页面提供。返回按钮调用路由历史返回，适合 403、404、500 等状态页面；本例使用项目自带的 404 插图。",
    errorCode,
    commonExample,
    ["Loading"]
  ),
  componentDoc(
    "SecretInput",
    "敏感字段输入",
    "账号安全",
    "ProForm/components/SecretInput.vue",
    "敏感字段输入与按需查看，支持本地掩码和已配置密钥的查看流程。",
    "v-model 绑定输入文本，点击眼睛可查看本地已有值。业务查看需要 secret 配置及 resource、fieldName、recordId、configured，只有满足这些条件时才请求明文；本例不传查看配置，仅演示本地输入与显隐。",
    secretCode,
    securityExample,
    ["ProFormItem", "PasswordVerifyDialog"]
  ),
  componentDoc(
    "PasswordStrength",
    "密码强度",
    "账号安全",
    "PasswordStrength/index.vue",
    "实时展示密码强度，并显示页面提示或当前密码策略。",
    "password 接收输入值，tip 优先展示自定义提示，也可传入 policy。只有显式启用 loadCurrentPolicy 且没有 policy 时才请求账号密码策略；本例使用本地提示，不调用账号接口。",
    strengthCode,
    securityExample,
    ["SecretInput", "PasswordVerifyDialog"]
  ),
  componentDoc(
    "MfaSetupPanel",
    "多重认证绑定",
    "账号安全",
    "Mfa/MfaSetupPanel.vue",
    "展示 TOTP 绑定二维码和可复制的绑定地址。",
    "uri 使用绑定流程返回的 otpauth 地址，二维码在浏览器中生成，组件不发起绑定请求。业务页面负责验证码确认与后续流程；示例使用公开演示密钥，不代表当前账号的绑定信息。",
    mfaCode,
    securityExample,
    ["MfaRecoveryCodesDialog"]
  ),
  componentDoc(
    "MfaRecoveryCodesDialog",
    "认证恢复码",
    "账号安全",
    "Mfa/MfaRecoveryCodesDialog.vue",
    "在弹窗中展示和复制一次性恢复码，供绑定流程确认保存。",
    "v-model 控制显示，codes 接收本次返回的恢复码数组，confirm 表示用户已完成保存确认。页面负责决定后续流程；本例全部为演示文本，不读取或生成账号凭据。",
    recoveryCode,
    securityExample,
    ["MfaSetupPanel", "ProDialog"]
  )
];
