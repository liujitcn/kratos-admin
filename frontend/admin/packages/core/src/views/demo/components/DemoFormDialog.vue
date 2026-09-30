<!--
  Demo 共享表单弹窗：一次性覆盖 ProForm 支持的全部字段组件类型。

  设计说明：
  - 三个 Demo 表格页的“新增 / 编辑”按钮都打开本弹窗，作为全组件表单的单一演示源。
  - 每个字段都标注了组件类型与关键 props 的作用，真实项目中按需摘取，不要整段照抄。
  - 全部文案走 t() 语言包 key，页面不写中文 UI 字符串。
  - component: "slot" 的分组标题字段演示了如何在字段流中插入自定义内容。
-->
<template>
  <FormDialog
    v-model="visible"
    ref="formDialogRef"
    :title="t('core.demo.form.title.demo_form')"
    width="960px"
    label-width="120px"
    :col-span="12"
    :model="formData"
    :fields="formFields"
    :rules="rules"
    @confirm="handleSubmit"
    @close="handleClose"
  >
    <!-- 分组标题插槽：配合 component: "slot" 字段渲染分区标题 -->
    <template #sectionBasic>
      <el-divider content-position="left">{{ t("core.demo.form.group.basic") }}</el-divider>
    </template>
    <template #sectionChoice>
      <el-divider content-position="left">{{ t("core.demo.form.group.choice") }}</el-divider>
    </template>
    <template #sectionAdvanced>
      <el-divider content-position="left">{{ t("core.demo.form.group.advanced") }}</el-divider>
    </template>
    <!-- 后缀插槽：渲染在 name 输入框的 append 位置 -->
    <template #nameSuffix>
      <el-button @click="handleRandomCode">{{ t("core.demo.form.action.random_code") }}</el-button>
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import type { UploadUserFile } from "element-plus";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import { t } from "@/locales";
import { demoDeptTree, demoLevelOptions } from "../demo-data";

defineOptions({
  name: "DemoFormDialog"
});

const emit = defineEmits<{
  /** 校验通过后的提交事件，payload 为表单数据深拷贝。 */
  submit: [data: Record<string, any>];
}>();

/** 表单数据模型：字段名与 formFields 的 prop 一一对应。 */
const formData = reactive({
  /** 项目名称，input 演示。 */
  name: "",
  /** 访问口令，password 演示。 */
  password: "",
  /** 项目简介，textarea 演示。 */
  intro: "",
  /** 任务数量，input-number 演示。 */
  count: 10,
  /** 主题色，color-picker 演示。 */
  color: "",
  /** 是否启用，switch 演示，同时控制 invite_code 字段显隐。 */
  enabled: true,
  /** 已阅读说明，单checkbox 演示。 */
  agreed: false,
  /** 邀请码，visible 联动演示字段：仅 enabled 打开时显示。 */
  invite_code: "",
  /** 展示模式，segmented 演示。 */
  mode: "card",
  /** 项目等级，select 演示（可搜索、可清空）。 */
  level: "",
  /** 租户，tenant-select 演示（静态 options 形态）。 */
  tenant_id: undefined as number | undefined,
  /** 性别，dict 字典组件演示。 */
  gender: 1,
  /** 可见范围，radio-group 演示。 */
  visibility: "public",
  /** 兴趣标签，checkbox-group 演示。 */
  hobbies: [] as string[],
  /** 所属部门，tree-select 演示。 */
  dept_id: undefined as number | undefined,
  /** 成立日期，date-picker 单日演示。 */
  birthday: "",
  /** 活动区间，date-picker 区间选择演示。 */
  range: [] as string[],
  /** 调度表达式，cron-expression 演示。 */
  cron: "",
  /** 协作成员，transfer 穿梭框演示。 */
  members: [] as number[],
  /** 封面图，image-upload 单图上传演示。 */
  avatar: "",
  /** 相册，images-upload 多图上传演示。 */
  gallery: [],
  /** 附件，file-upload 单文件上传演示（组件 fileInfo 为对象类型，空值用 undefined）。 */
  attachment: undefined as UploadUserFile | undefined,
  /** 附件组，files-upload 多文件上传演示。 */
  attachments: [],
  /** 详细内容，rich-text 富文本演示。 */
  content: "",
  /** 部署配置，yaml-editor 演示。 */
  config: "",
  /** 别名列表，dynamic-list 演示。 */
  aliases: [] as string[],
  /** 环境变量，kv-list 键值对列表演示。 */
  properties: [] as { key: string; value: string }[]
});

/** 部门树静态选项：把 DemoDeptNode 映射成 el-tree-select 需要的 value/label/children 结构。 */
const deptTreeOptions: ProFormOption[] = demoDeptTree.map(dept => ({
  label: dept.name,
  value: Number(dept.id),
  children: dept.children?.map(child => ({ label: child.name, value: Number(child.id) }))
}));

/** 租户静态选项：tenant-select 传入 options 后不再请求租户接口，纯静态可渲染。 */
const tenantOptions: ProFormOption[] = [
  { label: t("core.demo.form.tenant.main"), value: 1 },
  { label: t("core.demo.form.tenant.sub"), value: 2 }
];

/** 协作成员穿梭框数据源：value 即选中后的 key。 */
const memberOptions: ProFormOption[] = demoDeptTree[0].children!.map((dept, index) => ({
  label: t("core.demo.form.leader_title", { name: dept.name }),
  value: index + 1
}));

/**
 * 全量字段配置：每个字段注释了组件类型与关键 props，属性透传规则见
 * core/src/components/ProForm/components/ProFormItem.vue。
 */
const formFields = computed<ProFormField[]>(() => [
  // ── 分组标题：component: "slot" + colSpan 24 渲染通栏插槽 ──
  { prop: "section_basic", label: "", component: "slot", slotName: "sectionBasic", colSpan: 24 },
  {
    // input：suffixSlotName 渲染输入框 append 插槽
    prop: "name",
    label: t("core.demo.form.field.name"),
    component: "input",
    suffixSlotName: "nameSuffix",
    props: { placeholder: t("common.placeholder.input"), clearable: true, maxlength: 30 }
  },
  {
    // password：ProFormItem 内置 show-password，仅需透传占位等 props
    prop: "password",
    label: t("core.demo.form.field.password"),
    component: "password",
    props: { placeholder: t("common.placeholder.input") }
  },
  {
    // textarea：maxlength + show-word-limit 演示字数限制
    prop: "intro",
    label: t("core.demo.form.field.intro"),
    component: "textarea",
    colSpan: 24,
    props: { rows: 3, maxlength: 200, showWordLimit: true, placeholder: t("common.placeholder.input") }
  },
  {
    // input-number：步进、精度与控制按钮位置
    prop: "count",
    label: t("core.demo.form.field.count"),
    component: "input-number",
    labelTooltip: t("core.demo.form.tooltip.count"),
    props: { min: 0, max: 999, step: 5, precision: 0, controlsPosition: "right", style: { width: "100%" } }
  },
  {
    // color-picker：预置色板与透明度
    prop: "color",
    label: t("core.demo.form.field.color"),
    component: "color-picker",
    props: { showAlpha: true, predefine: ["#409eff", "#67c23a", "#e6a23c", "#f56c6c"] }
  },
  {
    // switch：inlinePrompt 文案内嵌；该值同时联动 invite_code 的显隐
    prop: "enabled",
    label: t("core.demo.form.field.enabled"),
    component: "switch",
    props: { inlinePrompt: true, activeText: t("common.status.enabled"), inactiveText: t("common.status.disabled") }
  },
  {
    // checkbox：单个复选框，文案来自 checkboxLabel
    prop: "agreed",
    label: "",
    component: "checkbox",
    checkboxLabel: t("core.demo.form.field.agreed")
  },
  {
    // visible 联动演示：仅 enabled 打开时渲染，随 switch 实时显隐
    prop: "invite_code",
    label: t("core.demo.form.field.invite_code"),
    component: "input",
    visible: model => Boolean(model.enabled),
    props: { placeholder: t("common.placeholder.input"), maxlength: 12 }
  },
  // ── 选择与日期分组 ──
  { prop: "section_choice", label: "", component: "slot", slotName: "sectionChoice", colSpan: 24, rowBreakBefore: true },
  {
    // segmented：分段选择器，block 撑满整行
    prop: "mode",
    label: t("core.demo.form.field.mode"),
    component: "segmented",
    options: [
      { label: t("core.demo.form.mode.card"), value: "card" },
      { label: t("core.demo.form.mode.list"), value: "list" },
      { label: t("core.demo.form.mode.table"), value: "table" }
    ],
    props: { block: true }
  },
  {
    // select：filterable 可搜索、clearable 可清空；options 里的 tagType 仅供标签列使用
    prop: "level",
    label: t("core.demo.form.field.level"),
    component: "select",
    options: demoLevelOptions,
    props: { clearable: true, filterable: true, placeholder: t("common.placeholder.select") }
  },
  {
    // tenant-select：真实项目无需 options（组件自动请求租户接口），
    // Demo 为纯静态页传入固定 options 跳过接口请求。
    prop: "tenant_id",
    label: t("common.field.tenant"),
    component: "tenant-select",
    props: { options: tenantOptions, placeholder: t("common.placeholder.select") }
  },
  {
    // dict：字典组件按 code 拉取字典项（依赖字典接口权限，管理端默认已授权），
    // codeType: "number" 表示字典值为数字。
    prop: "gender",
    label: t("core.demo.form.field.gender"),
    component: "dict",
    props: { code: "base_user_gender", codeType: "number", style: { width: "100%" } }
  },
  {
    // radio-group：单选组
    prop: "visibility",
    label: t("core.demo.form.field.visibility"),
    component: "radio-group",
    options: [
      { label: t("core.demo.form.visibility.public"), value: "public" },
      { label: t("core.demo.form.visibility.private"), value: "private" }
    ]
  },
  {
    // checkbox-group：多选组，绑定数组
    prop: "hobbies",
    label: t("core.demo.form.field.hobbies"),
    component: "checkbox-group",
    options: [
      { label: t("core.demo.form.hobby.review"), value: "review" },
      { label: t("core.demo.form.hobby.deploy"), value: "deploy" },
      { label: t("core.demo.form.hobby.metric"), value: "metric" }
    ]
  },
  {
    // tree-select：树形下拉，checkStrictly 允许选中任意层级
    prop: "dept_id",
    label: t("common.field.department"),
    component: "tree-select",
    options: deptTreeOptions,
    props: {
      clearable: true,
      filterable: true,
      checkStrictly: true,
      renderAfterExpand: false,
      placeholder: t("common.placeholder.select"),
      style: { width: "100%" }
    }
  },
  {
    // date-picker：单日选择，value-format 控制绑定值格式
    prop: "birthday",
    label: t("core.demo.form.field.birthday"),
    component: "date-picker",
    props: { type: "date", valueFormat: "YYYY-MM-DD", placeholder: t("common.placeholder.select"), style: { width: "100%" } }
  },
  {
    // date-picker：区间选择，绑定数组值，通栏展示
    prop: "range",
    label: t("core.demo.form.field.range"),
    component: "date-picker",
    colSpan: 24,
    props: {
      type: "datetimerange",
      valueFormat: "YYYY-MM-DD HH:mm:ss",
      startPlaceholder: t("common.placeholder.start_date"),
      endPlaceholder: t("common.placeholder.end_date")
    }
  },
  // ── 高级组件分组 ──
  { prop: "section_advanced", label: "", component: "slot", slotName: "sectionAdvanced", colSpan: 24, rowBreakBefore: true },
  {
    // cron-expression：Cron 表达式编辑器（弹窗面板生成表达式）
    prop: "cron",
    label: t("core.demo.form.field.cron"),
    component: "cron-expression",
    colSpan: 24,
    rowBreakBefore: true,
    props: { placeholder: t("core.demo.form.placeholder.cron") }
  },
  {
    // transfer：穿梭框，选项 value 即选中 key，通栏展示
    prop: "members",
    label: t("core.demo.form.field.members"),
    component: "transfer",
    colSpan: 24,
    options: memberOptions,
    props: {
      filterable: true,
      targetOrder: "unshift",
      titles: [t("core.demo.form.transfer.candidates"), t("core.demo.form.transfer.selected")]
    }
  },
  {
    // image-upload：单图上传（默认走文件服务上传接口，选择文件后才发起请求）
    prop: "avatar",
    label: t("core.demo.form.field.avatar"),
    component: "image-upload",
    props: { width: "120px", height: "120px" }
  },
  {
    // images-upload：多图上传，limit 限制数量
    prop: "gallery",
    label: t("core.demo.form.field.gallery"),
    component: "images-upload",
    props: { limit: 4, drag: false }
  },
  {
    // file-upload：单文件上传
    prop: "attachment",
    label: t("core.demo.form.field.attachment"),
    component: "file-upload"
  },
  {
    // files-upload：多文件上传，fileSize 单位 MB
    prop: "attachments",
    label: t("core.demo.form.field.attachments"),
    component: "files-upload",
    props: { limit: 3, fileSize: 10 }
  },
  {
    // rich-text：富文本编辑器，通栏展示，height 控制编辑区高度（wangEditor 要求 ≥300px，否则 hoverbar 定位告警）
    prop: "content",
    label: t("core.demo.form.field.content"),
    component: "rich-text",
    colSpan: 24,
    props: { height: "320px" }
  },
  {
    // yaml-editor：YAML 编辑器（CodeMirror + 工具栏），min/maxHeight 控制伸缩范围
    prop: "config",
    label: t("core.demo.form.field.config"),
    component: "yaml-editor",
    colSpan: 24,
    props: { minHeight: "160px", maxHeight: "320px" }
  },
  {
    // dynamic-list：动态行文本列表
    prop: "aliases",
    label: t("core.demo.form.field.aliases"),
    component: "dynamic-list",
    colSpan: 24,
    props: { inputProps: { placeholder: t("common.placeholder.input_content") } }
  },
  {
    // kv-list：键值对列表，addText 自定义新增按钮文案
    prop: "properties",
    label: t("core.demo.form.field.properties"),
    component: "kv-list",
    colSpan: 24,
    props: { addText: t("core.demo.form.action.add_property") }
  }
]);

/** 校验规则演示：必填、长度与数字范围。 */
const rules = computed(() => ({
  name: [
    { required: true, message: t("common.validation.required_input", { field: t("core.demo.form.field.name") }), trigger: "blur" }
  ],
  count: [{ required: true, type: "number", min: 0, max: 999, message: t("common.validation.sort_positive"), trigger: "blur" }],
  invite_code: [
    {
      max: 12,
      message: t("common.validation.max_length", { field: t("core.demo.form.field.invite_code"), max: 12 }),
      trigger: "blur"
    }
  ]
}));

const visible = ref(false);
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

/** 恢复表单初始值：打开弹窗前调用，避免新增/编辑之间互相污染。 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.name = "";
  formData.password = "";
  formData.intro = "";
  formData.count = 10;
  formData.color = "";
  formData.enabled = true;
  formData.agreed = false;
  formData.invite_code = "";
  formData.mode = "card";
  formData.level = "";
  formData.tenant_id = undefined;
  formData.gender = 1;
  formData.visibility = "public";
  formData.hobbies = [];
  formData.dept_id = undefined;
  formData.birthday = "";
  formData.range = [];
  formData.cron = "";
  formData.members = [];
  formData.avatar = "";
  formData.gallery = [];
  formData.attachment = undefined;
  formData.attachments = [];
  formData.content = "";
  formData.config = "";
  formData.aliases = [];
  formData.properties = [];
}

/**
 * 打开弹窗：支持传入初始值（编辑回填演示）。
 * 真实项目中可在 open 的 load 回调里请求详情接口后再回填。
 */
function open(initial?: Partial<typeof formData>) {
  resetForm();
  if (initial) Object.assign(formData, initial);
  visible.value = true;
}

/** 随机生成一个编码填入名称，演示 suffixSlotName 插槽交互。 */
function handleRandomCode() {
  formData.name = `DEMO-${Math.random().toString(36).slice(2, 8).toUpperCase()}`;
}

/** 提交表单：校验通过后向父组件抛出数据并关闭（纯静态演示，不调用保存接口）。 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;
    const submitData = JSON.parse(JSON.stringify(formData)) as Record<string, any>;
    // 真实项目在此调用 CreateXxx / UpdateXxx RPC，成功后刷新表格；
    // Demo 中父页面监听 submit 事件演示本地落库（树形表格页的新增子级）。
    emit("submit", submitData);
    // eslint-disable-next-line no-console
    console.log("[DemoFormDialog] 提交数据：", submitData);
    ElMessage.success(t("core.demo.form.message.submit_success"));
    handleClose();
  });
}

/** 关闭弹窗并清理表单状态。 */
function handleClose() {
  formDialogRef.value?.close();
  resetForm();
}

defineExpose({ open });
</script>
