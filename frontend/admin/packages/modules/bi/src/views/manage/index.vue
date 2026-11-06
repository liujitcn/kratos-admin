<template>
  <div class="table-box">
    <ProTable
      ref="table"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestDashboardTable"
    /><FormDialog
      ref="dialog"
      v-model="creating"
      :title="editing ? t('common.action.edit') : t('bi.dashboard.create')"
      :model="form"
      :fields="fields"
      :rules="rules"
      @confirm="submit"
      @closed="reset"
    />
    <BiDashboardMenuDialog v-model="menuOpen" :dashboard="menuDashboard" />
  </div>
</template>
<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { t } from "@liujitcn/kratos-admin-core";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { useAuthStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import type { BiDashboard, PageBiDashboardRequest } from "../../rpc/bi/admin/v1/bi_dashboard";
import { defBiCardService } from "../../api/bi/admin/v1/bi_card";
import { defBiDashboardService } from "../../api/bi/admin/v1/bi_dashboard";
import BiDashboardMenuDialog from "../../components/BiDashboardMenuDialog.vue";
import { parseDashboardLayout } from "../../bi-dashboard-schema";
defineOptions({ name: "BiDashboardManage", inheritAttrs: false });
const router = useRouter(),
  authStore = useAuthStore(),
  table = ref<ProTableInstance>(),
  dialog = ref<InstanceType<typeof FormDialog>>(),
  creating = ref(false),
  form = reactive({ name: "", code: "", description: "" });
const editing = ref<BiDashboard>();
const menuOpen = ref(false),
  menuDashboard = ref<BiDashboard>();
const savingFavorite = ref(false);
const menuPath = computed(() => authStore.flatMenuListGet.find(item => item.component === "system/base/menu/index")?.path);
const fields = computed<ProFormField[]>(() => [
  { prop: "name", label: t("bi.query.save_name"), component: "input", props: { maxlength: 128 } },
  { prop: "code", label: t("bi.dashboard.code"), component: "input", props: { maxlength: 128, disabled: Boolean(editing.value) } },
  { prop: "description", label: t("bi.dashboard.description"), component: "input", props: { type: "textarea", maxlength: 500 } }
]);
const rules = {
  name: [{ required: true, message: t("bi.card.validation.name"), trigger: "blur" }],
  code: [{ required: true, message: t("bi.dashboard.code_required"), trigger: "blur" }]
};
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("bi.dashboard.create"),
    type: "primary",
    icon: CirclePlus,
    onClick: () => {
      reset();
      form.code = `bi_${Date.now().toString(36)}`;
      creating.value = true;
    }
  }
]);
const columns = computed<ColumnProps[]>(() => [
  { prop: "name", label: t("bi.query.save_name"), minWidth: 180, search: { el: "input" } },
  { prop: "code", label: t("bi.dashboard.code"), minWidth: 150 },
  {
    prop: "favorite",
    label: t("bi.dashboard.favorite"),
    width: 100,
    cellType: "status",
    statusProps: {
      activeValue: true,
      inactiveValue: false,
      activeText: t("bi.dashboard.favorited"),
      inactiveText: t("bi.dashboard.not_favorited"),
      disabled: scope => savingFavorite.value || !(scope.row as BiDashboard).published_version,
      beforeChange: scope => setFavorite(scope.row as BiDashboard, !scope.row.favorite)
    }
  },
  {
    prop: "status",
    label: t("common.field.status"),
    width: 100,
    cellType: "status",
    statusProps: {
      activeValue: 2,
      inactiveValue: 1,
      activeText: t("bi.dashboard.status_published"),
      inactiveText: t("bi.dashboard.status_draft"),
      disabled: true
    }
  },
  { prop: "draft_version", label: t("bi.dashboard.version"), width: 90 },
  { prop: "published_version", label: t("bi.dashboard.published_version"), width: 110 },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 175 },
  {
    prop: "operation",
    label: t("common.field.operation"),
    cellType: "actions",
    fixed: "right",
    actions: [
      {
        label: t("bi.dashboard.preview"),
        icon: View,
        link: true,
        type: "primary",
        disabled: scope => !(scope.row as BiDashboard).published_version,
        onClick: scope => {
          void router.push(`/bi/dashboard/${(scope.row as BiDashboard).id}`);
        }
      },
      {
        label: t("bi.dashboard.edit"),
        icon: Grid,
        link: true,
        hidden: scope => !(scope.row as BiDashboard).can_edit,
        onClick: scope => {
          void router.push({ path: "/bi/designer", query: { id: String((scope.row as BiDashboard).id) } });
        }
      },
      {
        label: t("common.action.edit"),
        icon: EditPen,
        link: true,
        hidden: scope => !(scope.row as BiDashboard).can_edit,
        onClick: scope => edit(scope.row as BiDashboard)
      },
      {
        label: t("bi.dashboard.publish"),
        icon: Promotion,
        link: true,
        hidden: scope => !(scope.row as BiDashboard).can_edit,
        disabled: scope => (scope.row as BiDashboard).draft_version === (scope.row as BiDashboard).published_version,
        onClick: scope => publish(scope.row as BiDashboard)
      },
      {
        label: t("bi.dashboard.configure_menu"),
        icon: Setting,
        link: true,
        hidden: scope => !menuPath.value || !(scope.row as BiDashboard).can_edit,
        disabled: scope => !(scope.row as BiDashboard).published_version,
        onClick: scope => configureMenu(scope.row as BiDashboard)
      },
      {
        label: t("common.action.delete"),
        icon: Delete,
        link: true,
        type: "danger",
        hidden: scope => !(scope.row as BiDashboard).can_edit,
        onClick: scope => remove(scope.row as BiDashboard)
      }
    ]
  }
]);
/** 分页请求统一转换表格条件，并同步工作台的仪表盘收藏状态。 */
async function requestDashboardTable(params: Record<string, unknown>) {
  const [data, config] = await Promise.all([
    defBiDashboardService.PageBiDashboard(buildPageRequest<PageBiDashboardRequest>(params as unknown as PageBiDashboardRequest)),
    defBiCardService.GetBiWorkbench({})
  ]);
  return {
    data: {
      list: (data.dashboards || []).map(item => ({ ...item, favorite: config.dashboard_ids.includes(item.id) })),
      total: data.total
    }
  };
}
/** 保存收藏开关，成功后再切换，保留最新排序和独立卡片收藏。 */
async function setFavorite(item: BiDashboard, favorite: boolean) {
  if (savingFavorite.value || !item.published_version) return false;
  savingFavorite.value = true;
  try {
    const latest = await defBiCardService.GetBiWorkbench({});
    if (latest.dashboard_ids.includes(item.id) === favorite) return true;
    const ids = favorite ? [...latest.dashboard_ids, item.id] : latest.dashboard_ids.filter(id => id !== item.id);
    await defBiCardService.SaveBiWorkbench({ dashboard_ids: ids, favorite_ids: latest.favorite_ids });
    ElMessage.success(t("bi.workbench.saved"));
    return true;
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
    return false;
  } finally {
    savingFavorite.value = false;
  }
}
/** 在管理列表编辑名称与说明，设计器只处理卡片布局。 */
function edit(item: BiDashboard) {
  reset();
  editing.value = item;
  Object.assign(form, { name: item.name, code: item.code, description: item.description });
  creating.value = true;
}
/** 保存仪表盘信息，新建时再进入卡片组装器。 */
async function submit() {
  if (!(await dialog.value?.validate())) return;
  try {
    if (editing.value) {
      await defBiDashboardService.SaveBiDashboard({
        dashboard: {
          ...editing.value,
          name: form.name,
          description: form.description,
          schema_json: JSON.stringify({ ...parseDashboardLayout(editing.value.schema_json), name: form.name })
        }
      });
      creating.value = false;
      table.value?.getTableList();
      return;
    }
    const item = await defBiDashboardService.SaveBiDashboard({
      dashboard: {
        id: 0,
        tenant_id: 0,
        project_id: 0,
        owner_id: 0,
        code: form.code,
        name: form.name,
        description: form.description,
        status: 1,
        draft_version: 0,
        published_version: 0,
        schema_json: JSON.stringify({ schemaVersion: 3, name: form.name, cards: [] }),
        updated_at: "",
        can_edit: true
      }
    });
    creating.value = false;
    table.value?.getTableList();
    void router.push({ path: "/bi/designer", query: { id: String(item.id) } });
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  }
}
/** 发布现有草稿，服务端负责快照和版本冲突校验。 */
async function publish(item: BiDashboard) {
  try {
    await defBiDashboardService.PublishBiDashboard({ id: item.id, version: item.draft_version });
    ElMessage.success(t("bi.dashboard.publish_success"));
    table.value?.getTableList();
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  }
}
/** 在当前页面配置仪表盘菜单，不跳转或改动通用菜单管理表单。 */
function configureMenu(item: BiDashboard) {
  menuDashboard.value = item;
  menuOpen.value = true;
}
/** 删除前按名称确认，失败时保留表格内容。 */
async function remove(item: BiDashboard) {
  try {
    await ElMessageBox.confirm(`${t("bi.query.save_name")}：${item.name}`, t("common.action.delete"), { type: "warning" });
  } catch {
    return;
  }
  try {
    await defBiDashboardService.DeleteBiDashboard({ id: item.id });
    table.value?.getTableList();
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  }
}
/** 关闭时清空表单与校验。 */
function reset() {
  editing.value = undefined;
  Object.assign(form, { name: "", code: "", description: "" });
  dialog.value?.clearValidate();
}
</script>
