<template>
  <FormDialog
    ref="dialog"
    :model-value="modelValue"
    :title="t('bi.dashboard.configure_menu')"
    :model="form"
    :fields="fields"
    :rules="rules"
    :confirm-loading="saving"
    :form-props="{ disabled: loading || saving }"
    :close-on-click-modal="!saving"
    @update:model-value="emit('update:modelValue', $event)"
    @confirm="submit"
    @closed="reset"
  >
    <template #i18ns>
      <DynamicI18nEditor v-model="i18nValues" :source="form.title" :maxlength="100" :disabled="loading || saving" />
    </template>
    <template #icon>
      <SelectIcon v-model:icon-value="form.icon" :placeholder="t('system.base.menu.placeholder.icon')" />
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import SelectIcon from "@liujitcn/kratos-admin-core/components/SelectIcon/index.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { defBaseMenuService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_menu";
import { loadEnabledBaseLanguages } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_language";
import DynamicI18nEditor from "@liujitcn/kratos-admin-system/components/i18n/DynamicI18nEditor.vue";
import {
  normalizeDynamicI18ns,
  serializeDynamicI18ns,
  type DynamicI18nRecord,
  type DynamicI18nValue
} from "@liujitcn/kratos-admin-system/components/i18n/dynamicI18n";
import type { BaseMenu, BaseMenuForm } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_menu";
import { BaseMenuType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/common";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import type { BiDashboard } from "../rpc/bi/admin/v1/bi_dashboard";

const props = defineProps<{ modelValue: boolean; dashboard?: BiDashboard }>();
const emit = defineEmits<{ "update:modelValue": [value: boolean] }>();
const dialog = ref<InstanceType<typeof FormDialog>>();
const form = reactive({ title: "", icon: "DataAnalysis", parent_id: undefined as number | undefined });
const options = ref<ProFormOption[]>([]);
const existing = ref<BaseMenuForm>();
const i18nValues = ref<DynamicI18nValue[]>([]);
const loading = ref(false),
  saving = ref(false);
let request = 0;
const fields = computed<ProFormField[]>(() => [
  {
    prop: "title",
    label: t("system.base.menu.field.title"),
    component: "input",
    suffixSlotName: "i18ns",
    props: { maxlength: 100 }
  },
  { prop: "icon", label: t("system.base.menu.field.icon"), component: "slot", slotName: "icon" },
  {
    prop: "parent_id",
    label: t("bi.dashboard.menu_position"),
    component: "tree-select",
    options: options.value,
    props: { nodeKey: "value", props: { label: "label", children: "children" }, checkStrictly: true, filterable: true }
  }
]);
const rules = computed(() => ({
  title: [{ required: true, whitespace: true, message: t("system.base.menu.placeholder.title"), trigger: "blur" }],
  icon: [{ required: true, message: t("system.base.menu.placeholder.icon"), trigger: "change" }],
  parent_id: [{ required: true, message: t("system.base.menu.placeholder.parent"), trigger: "change" }]
}));

/** 加载可放置的管理端目录，并回填当前仪表盘已存在的菜单。 */
async function load() {
  const current = ++request;
  loading.value = true;
  existing.value = undefined;
  i18nValues.value = [];
  Object.assign(form, { title: props.dashboard?.name || "", icon: "DataAnalysis", parent_id: undefined });
  try {
    await loadEnabledBaseLanguages();
    if (current !== request) return;
    i18nValues.value = normalizeDynamicI18ns(undefined);
    const data = await defBaseMenuService.TreeBaseMenu({ lazy: false });
    if (current !== request) return;
    options.value = folderOptions(data.base_menus || []);
    const queue = [...(data.base_menus || [])];
    for (const item of queue) {
      queue.push(...(item.children || []));
      if (item.name !== `BiDashboard${props.dashboard?.id}` || item.component !== "bi/dashboard/index") continue;
      const value = await defBaseMenuService.GetBaseMenu({ id: item.id });
      if (current !== request) return;
      existing.value = value;
      i18nValues.value = normalizeDynamicI18ns(value.i18ns as DynamicI18nRecord[] | undefined);
      Object.assign(form, {
        title: value.meta?.title || props.dashboard?.name || "",
        icon: value.meta?.icon || "DataAnalysis",
        parent_id: value.parent_id
      });
      break;
    }
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    if (current === request) loading.value = false;
  }
}

/** 目录作为唯一可选父级，排除移动端分支和页面、按钮节点。 */
function folderOptions(items: BaseMenu[]): ProFormOption[] {
  return items
    .filter(item => item.type === BaseMenuType.BASE_MENU_TYPE_FOLDER && item.id !== 99000000)
    .map(item => ({ label: item.meta?.title || item.name, value: item.id, children: folderOptions(item.children || []) }));
}

/** 固定仪表盘路由与权限，保存菜单标题、译文、图标和父目录。 */
async function submit() {
  if (!props.dashboard || loading.value || saving.value || !(await dialog.value?.validate())) return;
  saving.value = true;
  try {
    const payload: BaseMenuForm = {
      id: existing.value?.id || 0,
      parent_id: form.parent_id,
      type: BaseMenuType.BASE_MENU_TYPE_MENU,
      path: `/reports/dashboard/${props.dashboard.id}`,
      name: `BiDashboard${props.dashboard.id}`,
      component: "bi/dashboard/index",
      redirect: "",
      meta: {
        title: form.title,
        icon: form.icon,
        hidden: false,
        always_show: false,
        keep_alive: false,
        full: false,
        affix: false,
        params: [],
        app: undefined
      },
      api: ["/bi.admin.v1.BiDashboardService/GetBiDashboardData"],
      i18ns: serializeDynamicI18ns(i18nValues.value, "base_menu.meta.title", existing.value?.id || 0, form.title),
      sort: existing.value?.sort || 1,
      status: Status.STATUS_ENABLE
    };
    if (payload.id) await defBaseMenuService.UpdateBaseMenu({ base_menu: payload });
    else await defBaseMenuService.CreateBaseMenu({ base_menu: payload });
    ElMessage.success(t("bi.dashboard.menu_saved"));
    emit("update:modelValue", false);
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    saving.value = false;
  }
}

/** 关闭时清空草稿与校验，并丢弃迟到的目录请求。 */
function reset() {
  request++;
  loading.value = false;
  existing.value = undefined;
  i18nValues.value = [];
  options.value = [];
  Object.assign(form, { title: "", icon: "DataAnalysis", parent_id: undefined });
  dialog.value?.clearValidate();
}
watch(
  () => [props.modelValue, props.dashboard?.id],
  () => {
    if (props.modelValue && props.dashboard) void load();
  }
);
</script>
