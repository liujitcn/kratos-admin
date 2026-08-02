<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      :title="t('system.menu.title.list')"
      row-key="id"
      :indent="20"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestMenuTable"
      :pagination="false"
      :default-expand-all="false"
      :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.editing ? 'system.menu.action.edit' : 'system.menu.action.create')"
      width="1180px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="120px"
      :col-span="12"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    >
      <template #menuIcon>
        <SelectIcon v-model:icon-value="menuIconValue" :placeholder="t('system.menu.placeholder.icon')" />
      </template>

      <template #translations>
        <DynamicTranslationEditor
          v-model="translationValues"
          :resource-id="formData.id"
          :resource-type="TranslationResourceType.TRANSLATION_RESOURCE_TYPE_MENU"
          :draft-enabled="configStore.translationDraftEnabled"
          :maxlength="100"
        />
      </template>

      <template #apiTransferItem="slotScope">
        <el-popover effect="light" trigger="hover" placement="top" width="auto">
          <template #default>
            <div>{{ t("system.api.field.operation") }}：{{ slotScope.option.operation }}</div>
            <div>{{ t("system.api.field.method") }}：{{ slotScope.option.method }}</div>
            <div>{{ t("system.api.field.path") }}：{{ slotScope.option.path }}</div>
          </template>
          <template #reference>{{ slotScope.option.label }}</template>
        </el-popover>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref, resolveComponent, resolveDynamicComponent, watch } from "vue";
import { ElIcon, ElTag, type FormRules } from "element-plus";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type {
  ColumnProps,
  HeaderActionProps,
  ProTableInstance,
  RenderScope
} from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import SelectIcon from "@liujitcn/kratos-admin-core/components/SelectIcon/index.vue";
import { defBaseMenuService } from "@liujitcn/kratos-admin-system/api/system/base_menu";
import { defBaseApiService } from "@liujitcn/kratos-admin-system/api/system/base_api";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import type { BaseApi } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api";
import type {
  BaseMenu,
  BaseMenuAppMeta,
  BaseMenuForm,
  BaseMenuMeta
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_menu";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import { BaseMenuType } from "@liujitcn/kratos-admin-system/rpc/system/common/v1/enum";
import { normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { useConfigStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import DynamicTranslationEditor from "@liujitcn/kratos-admin-system/components/DynamicTranslationEditor.vue";
import DynamicTranslationCell from "@liujitcn/kratos-admin-system/components/DynamicTranslationCell.vue";
import {
  normalizeDynamicTranslations,
  serializeDynamicTranslations,
  type DynamicTranslationRecord,
  type DynamicTranslationValue
} from "@liujitcn/kratos-admin-system/components/dynamicTranslation";
import { TranslationResourceType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_translation";
import type { BaseMenuTranslation } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_menu";

defineOptions({
  name: "BaseMenu",
  inheritAttrs: false
});

const APP_MENU_ROOT_ID = 999;

/**
 * 菜单表单状态，统一补齐 meta 字段，便于 ProForm 直接双向绑定。
 */
type MenuFormState = Omit<BaseMenuForm, "meta"> & {
  /** 菜单元信息。 */
  meta: BaseMenuMeta;
};

const { BUTTONS } = useAuthButtons();
const configStore = useConfigStore();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const menuOptions = ref<ProFormOption[]>([]);
const parentMenuTypeMap = ref(new Map<number, BaseMenuType>());
const appMenuIds = ref(new Set<number>([APP_MENU_ROOT_ID]));
const apiList = ref<BaseApi[]>([]);
const translationValues = ref<DynamicTranslationValue[]>(normalizeDynamicTranslations(undefined, "title"));

const dialog = reactive({
  editing: false,
  visible: false,
  parentType: BaseMenuType.UNKNOWN_MT,
  parentLocked: true
});

/** 创建默认移动端页面配置。 */
function createDefaultAppMenuMeta(): BaseMenuAppMeta {
  return {
    view_key: "",
    access: "AUTHENTICATED",
    in_tab_bar: false,
    selected_icon: undefined
  };
}

/** 创建默认菜单元信息。 */
function createDefaultMenuMeta(): BaseMenuMeta {
  return {
    title: "",
    icon: "",
    always_show: false,
    hidden: false,
    keep_alive: false,
    full: false,
    affix: false,
    params: [],
    app: undefined
  };
}

/** 创建默认菜单表单。 */
function createDefaultMenuForm(): MenuFormState {
  return {
    id: 0,
    parent_id: undefined,
    type: BaseMenuType.FOLDER,
    path: "",
    name: "",
    component: "",
    redirect: "",
    meta: createDefaultMenuMeta(),
    api: [],
    translations: [],
    sort: 1,
    status: Status.ENABLE
  };
}

const formData = reactive<MenuFormState>(createDefaultMenuForm());

/** 统一接管菜单图标字段，规避可选字段类型带来的模板告警。 */
const menuIconValue = computed({
  get: () => formData.meta.icon ?? "",
  set: value => {
    formData.meta.icon = value;
  }
});

const menuTypeOptions = computed<ProFormOption[]>(() => [
  { label: t("system.menu.type.folder"), value: BaseMenuType.FOLDER },
  { label: t("system.menu.type.menu"), value: BaseMenuType.MENU },
  { label: t("system.menu.type.button"), value: BaseMenuType.BUTTON },
  { label: t("system.menu.type.external"), value: BaseMenuType.EXT_LINK }
]);

/** 判断父节点是否属于固定移动端菜单树。 */
function isAppMenu(parentId?: number) {
  return parentId !== undefined && appMenuIds.value.has(parentId);
}

/** 判断页面是否为固定移动端根目录的直属 tab。 */
function isAppTab(parentId?: number) {
  return parentId === APP_MENU_ROOT_ID;
}

/** 根据三、五、七、九位编号识别菜单层级。 */
function getMenuLevel(menuId: number) {
  if (menuId >= 100 && menuId <= 999) return 1;
  if (menuId >= 10000 && menuId <= 99999) return 2;
  if (menuId >= 1000000 && menuId <= 9999999) return 3;
  if (menuId >= 100000000 && menuId <= 999999999) return 4;
  return 0;
}

/** 判断当前菜单是否还能继续新增下级节点。 */
function canCreateChild(menu: BaseMenu) {
  const level = getMenuLevel(menu.id);
  if (menu.id === APP_MENU_ROOT_ID || isAppMenu(menu.parent_id)) return level >= 1 && level < 4;
  if (menu.type === BaseMenuType.FOLDER) return level === 1 || level === 2;
  if (menu.type === BaseMenuType.MENU) return level === 2 || level === 3;
  return false;
}

/** 根据父级层级限制可创建的菜单类型。 */
const availableMenuTypeOptions = computed(() => {
  const parentLevel = getMenuLevel(formData.parent_id ?? 0);

  if (formData.id === APP_MENU_ROOT_ID) return menuTypeOptions.value.filter(item => item.value === BaseMenuType.FOLDER);
  if (isAppMenu(formData.parent_id)) return menuTypeOptions.value.filter(item => item.value === BaseMenuType.MENU);
  if (formData.id > 0 && parentLevel === 0) return menuTypeOptions.value.filter(item => item.value === BaseMenuType.FOLDER);
  if (dialog.parentType === BaseMenuType.FOLDER && parentLevel === 1)
    return menuTypeOptions.value.filter(
      item => item.value === BaseMenuType.FOLDER || item.value === BaseMenuType.MENU || item.value === BaseMenuType.EXT_LINK
    );
  if (dialog.parentType === BaseMenuType.FOLDER && parentLevel === 2)
    return menuTypeOptions.value.filter(item => item.value === BaseMenuType.MENU || item.value === BaseMenuType.EXT_LINK);
  if (dialog.parentType === BaseMenuType.MENU && (parentLevel === 2 || parentLevel === 3))
    return menuTypeOptions.value.filter(item => item.value === BaseMenuType.BUTTON);
  return [];
});

watch(
  () => formData.parent_id,
  () => {
    if (isAppMenu(formData.parent_id)) {
      formData.meta.app ??= createDefaultAppMenuMeta();
      formData.meta.app.in_tab_bar = isAppTab(formData.parent_id);
    } else if (formData.id === 0) {
      formData.meta.app = undefined;
    }
    dialog.parentType = parentMenuTypeMap.value.get(formData.parent_id ?? 0) ?? BaseMenuType.UNKNOWN_MT;
    if (formData.id > 0 || availableMenuTypeOptions.value.some(item => item.value === formData.type)) return;
    formData.type = (availableMenuTypeOptions.value[0]?.value as BaseMenuType) ?? BaseMenuType.UNKNOWN_MT;
  }
);

const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.ENABLE },
  { label: t("common.status.disabled"), value: Status.DISABLE }
]);

const appAccessOptions = computed<ProFormOption[]>(() => [
  { label: t("system.menu.access.public"), value: "PUBLIC" },
  { label: t("system.menu.access.guest"), value: "GUEST_ONLY" },
  { label: t("system.menu.access.authenticated"), value: "AUTHENTICATED" }
]);

/**
 * 渲染菜单图标单元格，统一兼容 Element Plus 图标和本地 svg 图标。
 */
function renderMenuIconCell(scope: RenderScope<BaseMenu>) {
  const icon = scope.row.meta?.icon;
  if (isAppMenu(scope.row.parent_id)) return icon || "--";
  const iconName = resolveElementIcon(icon);
  if (iconName) {
    return h(
      ElIcon,
      { size: 18 },
      {
        default: () => [h(resolveDynamicComponent(iconName) as any)]
      }
    );
  }
  if (icon) return h(resolveComponent("svg-icon"), { iconClass: icon });
  return "--";
}

/**
 * 渲染菜单显示状态标签，减少页面模板中的重复判断。
 */
function renderHiddenCell(scope: RenderScope<BaseMenu>) {
  const isHidden = Boolean(scope.row.meta?.hidden);
  return h(ElTag, { type: isHidden ? "info" : "success" }, () =>
    t(isHidden ? "system.menu.status.hidden" : "system.menu.status.visible")
  );
}

/** 渲染菜单标题翻译预览，并复用当前页面的编辑弹窗。 */
function renderMenuTitleCell(scope: RenderScope<BaseMenu>) {
  const row = scope.row;
  return h(DynamicTranslationCell, {
    source: row.meta?.title ?? "",
    translations: row.translations,
    textField: "title",
    editable: BUTTONS.value["base:menu:update"],
    onEdit: () => handleOpenDialog(undefined, row.id)
  });
}

/** 菜单表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  {
    type: "selection",
    width: 55,
    selectable: row => {
      const menu = row as BaseMenu;
      return menu.parent_id !== 0 && menu.id !== APP_MENU_ROOT_ID;
    }
  },
  {
    prop: "meta.title",
    label: t("system.menu.field.name"),
    minWidth: 220,
    align: "left",
    search: { el: "input", key: "title" },
    showOverflowTooltip: false,
    render: scope => renderMenuTitleCell(scope as unknown as RenderScope<BaseMenu>)
  },
  { prop: "type", label: t("system.menu.field.type"), minWidth: 120, dictCode: "base_menu_type", search: { el: "select" } },
  {
    prop: "meta.icon",
    label: t("system.menu.field.icon"),
    width: 90,
    render: scope => renderMenuIconCell(scope as unknown as RenderScope<BaseMenu>)
  },
  { prop: "path", label: t("system.menu.field.pathOrPermission"), minWidth: 260, search: { el: "input" } },
  { prop: "name", label: t("system.menu.field.routeName"), minWidth: 180, search: { el: "input" } },
  { prop: "component", label: t("system.menu.field.component"), minWidth: 260 },
  { prop: "redirect", label: t("system.menu.field.redirect"), minWidth: 220 },
  { prop: "sort", label: t("system.common.field.sort"), minWidth: 80, align: "right" },
  {
    prop: "status",
    label: t("system.common.field.status"),
    width: 100,
    dictCode: "status",
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !BUTTONS.value["base:menu:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseMenu)
    }
  },
  {
    prop: "meta.hidden",
    label: t("system.menu.field.displayStatus"),
    width: 100,
    render: scope => renderHiddenCell(scope as unknown as RenderScope<BaseMenu>)
  },
  { prop: "created_at", label: t("system.common.field.createdAt"), minWidth: 180 },
  { prop: "updated_at", label: t("system.common.field.updatedAt"), minWidth: 180 },
  {
    prop: "operation",
    label: t("system.common.field.action"),
    width: 220,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.create"),
        type: "primary",
        link: true,
        icon: CirclePlus,
        hidden: scope => !BUTTONS.value["base:menu:create"] || !canCreateChild(scope.row as BaseMenu),
        onClick: scope => handleOpenDialog(scope.row as BaseMenu)
      },
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:menu:update"],
        onClick: scope => handleOpenDialog(undefined, (scope.row as BaseMenu).id)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: scope => {
          const menu = scope.row as BaseMenu;
          return !BUTTONS.value["base:menu:delete"] || menu.parent_id === 0 || menu.id === APP_MENU_ROOT_ID;
        },
        onClick: scope => handleDeleteMenu(scope.row as BaseMenu)
      }
    ]
  }
]);

/** 菜单表格顶部按钮配置。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:menu:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:menu:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDeleteMenu(scope.selectedList as BaseMenu[])
  }
]);

/** API 穿梭框可选数据。 */
const transferData = computed<ProFormOption[]>(() => {
  return apiList.value.map(item => ({
    ...item,
    value: item.operation,
    label: `${item.service_desc}/${item.desc}`
  }));
});

/** 菜单表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "parent_id",
    label: t("system.menu.field.parent"),
    component: "tree-select",
    options: menuOptions.value,
    props: () => ({
      nodeKey: "value",
      props: { label: "label", children: "children" },
      checkStrictly: true,
      clearable: false,
      filterable: true,
      placeholder: t("system.menu.placeholder.parent"),
      disabled: dialog.parentLocked,
      style: { width: "100%" }
    })
  },
  {
    prop: "type",
    label: t("system.menu.field.type"),
    component: "radio-group",
    options: availableMenuTypeOptions.value,
    props: model => ({
      disabled: model.id > 0 && model.parent_id === 0
    })
  },
  {
    prop: "meta.title",
    label: t(formData.type === BaseMenuType.BUTTON ? "system.menu.field.buttonName" : "system.menu.field.title"),
    component: "input",
    itemProps: {
      required: true
    },
    props: () => ({
      placeholder: t(
        formData.type === BaseMenuType.BUTTON ? "system.menu.placeholder.buttonName" : "system.menu.placeholder.title"
      )
    })
  },
  {
    prop: "translations",
    label: t("system.translation.field.translations"),
    component: "slot",
    slotName: "translations",
    colSpan: 24
  },
  {
    prop: "path",
    label: getPathFieldLabel(),
    component: "input",
    labelTooltip: getPathFieldTooltip(),
    itemProps: model => ({
      required: model.type !== BaseMenuType.FOLDER
    }),
    props: () => ({
      placeholder: getPathFieldPlaceholder()
    }),
    visible: model => model.type !== BaseMenuType.FOLDER
  },
  {
    prop: "redirect",
    label: getRedirectFieldLabel(),
    component: "input",
    props: () => ({ placeholder: getRedirectFieldPlaceholder() }),
    itemProps: model => ({
      required: model.type === BaseMenuType.EXT_LINK
    }),
    visible: model => model.type === BaseMenuType.FOLDER || model.type === BaseMenuType.EXT_LINK
  },
  {
    prop: "meta.icon",
    label: t("system.menu.field.icon"),
    component: "slot",
    slotName: "menuIcon",
    itemProps: model => ({
      required: model.type !== BaseMenuType.BUTTON
    }),
    visible: model => !isAppMenu(model.parent_id) && model.type !== BaseMenuType.BUTTON
  },
  {
    prop: "meta.icon",
    label: t("system.menu.field.defaultIcon"),
    component: "input",
    labelTooltip: t("system.menu.tooltip.mobileIcon"),
    itemProps: model => ({
      required: Boolean(model.meta?.app?.in_tab_bar)
    }),
    props: { placeholder: t("system.menu.placeholder.defaultIcon") },
    visible: model => isAppMenu(model.parent_id)
  },
  {
    prop: "name",
    label: t(isAppMenu(formData.parent_id) ? "system.menu.field.logicalName" : "system.menu.field.routeName"),
    component: "input",
    labelTooltip: isAppMenu(formData.parent_id) ? t("system.menu.tooltip.logicalName") : t("system.menu.tooltip.routeName"),
    itemProps: model => ({
      required: model.type === BaseMenuType.MENU
    }),
    props: () => ({
      placeholder: isAppMenu(formData.parent_id)
        ? t("system.menu.placeholder.logicalName")
        : t("system.menu.placeholder.routeName")
    }),
    visible: model => model.type === BaseMenuType.MENU
  },
  {
    prop: "meta.app.view_key",
    label: t("system.menu.field.viewKey"),
    component: "input",
    labelTooltip: t("system.menu.tooltip.viewKey"),
    itemProps: { required: true },
    props: { placeholder: t("system.menu.placeholder.viewKey") },
    visible: model => isAppMenu(model.parent_id)
  },
  {
    prop: "meta.app.access",
    label: t("system.menu.field.access"),
    component: "select",
    options: appAccessOptions.value,
    itemProps: { required: true },
    props: {
      clearable: false,
      placeholder: t("system.menu.placeholder.access")
    },
    visible: model => isAppMenu(model.parent_id)
  },
  {
    prop: "meta.app.selected_icon",
    label: t("system.menu.field.selectedIcon"),
    component: "input",
    labelTooltip: t("system.menu.tooltip.mobileIcon"),
    itemProps: { required: true },
    props: { placeholder: t("system.menu.placeholder.selectedIcon") },
    visible: model => isAppTab(model.parent_id)
  },
  {
    prop: "component",
    label: t("system.menu.field.component"),
    component: "input",
    labelTooltip: t("system.menu.tooltip.component"),
    itemProps: model => ({
      required: model.type === BaseMenuType.MENU && !isAppMenu(model.parent_id)
    }),
    props: { placeholder: "system/base/user/index" },
    visible: model => model.type === BaseMenuType.MENU && !isAppMenu(model.parent_id)
  },
  {
    prop: "meta.hidden",
    label: t("system.menu.field.hidden"),
    component: "switch",
    props: {
      inlinePrompt: true,
      activeText: t("system.common.value.yes"),
      inactiveText: t("system.common.value.no"),
      activeValue: true,
      inactiveValue: false
    },
    visible: model => !isAppMenu(model.parent_id) && model.type !== BaseMenuType.BUTTON
  },
  {
    prop: "meta.always_show",
    label: t("system.menu.field.alwaysShow"),
    component: "switch",
    labelTooltip: t("system.menu.tooltip.alwaysShow"),
    props: {
      inlinePrompt: true,
      activeText: t("system.common.value.yes"),
      inactiveText: t("system.common.value.no"),
      activeValue: true,
      inactiveValue: false
    },
    visible: model => !isAppMenu(model.parent_id) && (model.type === BaseMenuType.FOLDER || model.type === BaseMenuType.MENU)
  },
  {
    prop: "meta.keep_alive",
    label: t("system.menu.field.keepAlive"),
    component: "switch",
    props: {
      inlinePrompt: true,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      activeValue: true,
      inactiveValue: false
    },
    visible: model => !isAppMenu(model.parent_id) && model.type === BaseMenuType.MENU
  },
  {
    prop: "meta.full",
    label: t("system.menu.field.fullscreen"),
    component: "switch",
    props: {
      inlinePrompt: true,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      activeValue: true,
      inactiveValue: false
    },
    visible: model => !isAppMenu(model.parent_id) && model.type === BaseMenuType.MENU
  },
  {
    prop: "meta.affix",
    label: t("system.menu.field.affix"),
    component: "switch",
    props: {
      inlinePrompt: true,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      activeValue: true,
      inactiveValue: false
    },
    visible: model => !isAppMenu(model.parent_id) && model.type === BaseMenuType.MENU
  },
  {
    prop: "meta.params",
    label: t("system.menu.field.routeParams"),
    component: "kv-list",
    labelTooltip: t("system.menu.tooltip.routeParams"),
    props: {
      addText: t("system.menu.action.addRouteParam"),
      keyInputProps: { placeholder: t("common.field.parameterName") },
      valueInputProps: { placeholder: t("common.field.parameterValue") }
    },
    itemProps: {
      class: "menu-form__params"
    },
    visible: model => !isAppMenu(model.parent_id) && model.type === BaseMenuType.MENU
  },
  {
    prop: "api",
    label: t("system.menu.field.apiList"),
    component: "transfer",
    slotName: "apiTransferItem",
    options: transferData.value,
    props: {
      filterable: true,
      titles: [t("system.menu.value.availableApi"), t("system.menu.value.selectedApi")]
    },
    visible: model => model.id === APP_MENU_ROOT_ID || model.type === BaseMenuType.MENU || model.type === BaseMenuType.BUTTON,
    colSpan: 24
  },
  {
    prop: "sort",
    label: t("system.common.field.sort"),
    component: "input-number",
    itemProps: {
      required: true
    },
    props: {
      min: 1,
      controlsPosition: "right",
      precision: 0,
      step: 1,
      style: { width: "100%" }
    }
  },
  {
    prop: "status",
    label: t("system.common.field.status"),
    component: "radio-group",
    itemProps: {
      required: true
    },
    options: statusOptions.value
  }
]);

const rules = computed<FormRules>(() => ({
  parent_id: formData.id
    ? []
    : [{ required: true, type: "number", min: 1, message: t("system.menu.placeholder.parent"), trigger: "change" }],
  type: [
    {
      required: true,
      message: t("system.common.validation.requiredSelect", { field: t("system.menu.field.type") }),
      trigger: "change"
    }
  ],
  "meta.title": [
    {
      validator: (_rule, value, callback) => {
        if (value) return callback();
        callback(
          new Error(
            t(formData.type === BaseMenuType.BUTTON ? "system.menu.placeholder.buttonName" : "system.menu.placeholder.title")
          )
        );
      },
      trigger: "blur"
    }
  ],
  "meta.icon": [
    {
      validator: (_rule, value, callback) => {
        if (formData.type === BaseMenuType.BUTTON) return callback();
        if (isAppMenu(formData.parent_id) && !isAppTab(formData.parent_id)) return callback();
        if (value) return callback();
        callback(
          new Error(t(isAppMenu(formData.parent_id) ? "system.menu.validation.defaultIcon" : "system.menu.placeholder.icon"))
        );
      },
      trigger: "change"
    }
  ],
  "meta.app.view_key": [
    {
      validator: (_rule, value, callback) => {
        if (!isAppMenu(formData.parent_id) || value) return callback();
        callback(new Error(t("system.menu.validation.viewKey")));
      },
      trigger: "blur"
    }
  ],
  "meta.app.access": [
    {
      validator: (_rule, value, callback) => {
        if (!isAppMenu(formData.parent_id) || value) return callback();
        callback(new Error(t("system.menu.placeholder.access")));
      },
      trigger: "change"
    }
  ],
  "meta.app.selected_icon": [
    {
      validator: (_rule, value, callback) => {
        if (!isAppTab(formData.parent_id) || value) return callback();
        callback(new Error(t("system.menu.validation.selectedIcon")));
      },
      trigger: "blur"
    }
  ],
  path: [
    {
      max: 1024,
      message: t("system.common.validation.maxLength", { field: t("system.menu.field.path"), max: 1024 }),
      trigger: "blur"
    },
    {
      validator: (_rule, value, callback) => {
        if (formData.type === BaseMenuType.FOLDER) return callback();
        if (!value) {
          if (formData.type === BaseMenuType.BUTTON) return callback(new Error(t("system.menu.validation.permission")));
          if (formData.type === BaseMenuType.EXT_LINK) return callback(new Error(t("system.menu.validation.externalUrl")));
          return callback(
            new Error(
              t(isAppMenu(formData.parent_id) ? "system.menu.validation.logicalPath" : "system.menu.validation.routePath")
            )
          );
        }
        if (isAppMenu(formData.parent_id) && !String(value).startsWith("app/")) {
          return callback(new Error(t("system.menu.validation.appPathPrefix")));
        }
        callback();
      },
      trigger: "blur"
    }
  ],
  redirect: [
    {
      max: 1024,
      message: t("system.common.validation.maxLength", { field: t("system.menu.field.redirect"), max: 1024 }),
      trigger: "blur"
    },
    {
      validator: (_rule, value, callback) => {
        if (formData.type !== BaseMenuType.EXT_LINK) return callback();
        if (/^https?:\/\/\S+$/.test(value)) return callback();
        callback(new Error(t("system.menu.validation.httpUrl")));
      },
      trigger: "blur"
    }
  ],
  name: [
    {
      max: 255,
      message: t("system.common.validation.maxLength", { field: t("system.menu.field.routeName"), max: 255 }),
      trigger: "blur"
    },
    {
      validator: (_rule, value, callback) => {
        if (formData.type !== BaseMenuType.MENU) return callback();
        if (!value)
          return callback(
            new Error(
              t(isAppMenu(formData.parent_id) ? "system.menu.validation.logicalName" : "system.menu.placeholder.routeName")
            )
          );
        if (isAppMenu(formData.parent_id) && !String(value).startsWith("App")) {
          return callback(new Error(t("system.menu.validation.appNamePrefix")));
        }
        callback();
      },
      trigger: "blur"
    }
  ],
  component: [
    {
      max: 255,
      message: t("system.common.validation.maxLength", { field: t("system.menu.field.component"), max: 255 }),
      trigger: "blur"
    },
    {
      validator: (_rule, value, callback) => {
        if (formData.type !== BaseMenuType.MENU || isAppMenu(formData.parent_id)) return callback();
        if (value) return callback();
        callback(new Error(t("system.menu.validation.component")));
      },
      trigger: "blur"
    }
  ],
  sort: [{ required: true, type: "number", min: 1, message: t("system.common.validation.sortPositive"), trigger: "blur" }],
  status: [
    {
      required: true,
      message: t("system.common.validation.requiredSelect", { field: t("system.common.field.status") }),
      trigger: "change"
    }
  ]
}));

/** 计算当前路径字段文案。 */
function getPathFieldLabel() {
  if (isAppMenu(formData.parent_id)) return t("system.menu.field.logicalPath");
  if (formData.type === BaseMenuType.BUTTON) return t("system.menu.field.permission");
  if (formData.type === BaseMenuType.EXT_LINK) return t("system.menu.field.internalPath");
  return t("system.menu.field.path");
}

/** 计算当前路径字段占位文案。 */
function getPathFieldPlaceholder() {
  if (isAppMenu(formData.parent_id)) return t("system.menu.placeholder.logicalPath");
  if (formData.type === BaseMenuType.BUTTON) return t("system.menu.placeholder.permission");
  if (formData.type === BaseMenuType.EXT_LINK) return "external/baidu";
  if (formData.type === BaseMenuType.FOLDER) return "base";
  return "user";
}

/** 计算当前路径字段提示文案。 */
function getPathFieldTooltip() {
  if (isAppMenu(formData.parent_id)) return t("system.menu.tooltip.logicalPath");
  if (formData.type === BaseMenuType.BUTTON) return t("system.menu.tooltip.permission");
  if (formData.type === BaseMenuType.EXT_LINK) return t("system.menu.tooltip.internalPath");
  if (formData.type === BaseMenuType.FOLDER) return t("system.menu.tooltip.folderPath");
  return t("system.menu.tooltip.routePath");
}

/** 计算重定向字段文案。 */
function getRedirectFieldLabel() {
  return t(formData.type === BaseMenuType.EXT_LINK ? "system.menu.field.externalUrl" : "system.menu.field.redirectRoute");
}

/** 计算重定向字段占位文案。 */
function getRedirectFieldPlaceholder() {
  return formData.type === BaseMenuType.EXT_LINK ? "https://www.example.com/" : t("system.menu.placeholder.redirectRoute");
}

/** 判断当前图标是否为 Element Plus 图标。 */
function resolveElementIcon(icon?: string) {
  if (!icon) return "";
  if (icon.startsWith("el-icon-")) return icon.replace("el-icon-", "");
  if (/^[A-Z]/.test(icon)) return icon;
  return "";
}

/**
 * 将菜单接口返回的 API 字段统一转换为穿梭框可识别的 operation 列表。
 */
function normalizeMenuApiSelection(api?: unknown[]) {
  if (!Array.isArray(api)) return [];

  const apiOperationSet = new Set(apiList.value.map(item => item.operation));
  const apiIdMap = new Map(apiList.value.map(item => [String(item.id), item.operation]));

  return api
    .map(item => {
      if (typeof item === "string") {
        if (apiOperationSet.has(item)) return item;
        return apiIdMap.get(item) ?? "";
      }

      if (typeof item === "number") {
        return apiIdMap.get(String(item)) ?? "";
      }

      if (item && typeof item === "object") {
        const currentItem = item as Record<string, unknown>;
        if (typeof currentItem.operation === "string") return currentItem.operation;
        if (currentItem.id !== undefined) return apiIdMap.get(String(currentItem.id)) ?? "";
      }

      return "";
    })
    .filter((item, index, currentList) => item && currentList.indexOf(item) === index);
}

/** 将服务端菜单表单补齐为前端可编辑结构。 */
function normalizeMenuForm(data?: Partial<BaseMenuForm>): MenuFormState {
  const defaultForm = createDefaultMenuForm();
  const parentId = data?.parent_id === 0 ? undefined : data?.parent_id;
  const normalizedMeta = {
    ...createDefaultMenuMeta(),
    ...(data?.meta ?? {}),
    params: data?.meta?.params ?? [],
    app: data?.meta?.app ?? (isAppMenu(parentId) ? createDefaultAppMenuMeta() : undefined)
  };

  return {
    ...defaultForm,
    ...data,
    parent_id: parentId,
    type: data?.type ?? BaseMenuType.FOLDER,
    status: data?.status ?? Status.ENABLE,
    api: normalizeMenuApiSelection(data?.api),
    sort: data?.sort ?? 1,
    meta: normalizedMeta
  };
}

/** 构建可选父级菜单树，并记录父节点类型用于约束下级菜单类型。 */
function buildMenuOptions(menuList: BaseMenu[] = []) {
  const typeMap = new Map<number, BaseMenuType>();
  const mobileIds = new Set<number>([APP_MENU_ROOT_ID]);
  const convert = (list: BaseMenu[]): ProFormOption[] =>
    list
      .filter(item => item.type === BaseMenuType.FOLDER || item.type === BaseMenuType.MENU)
      .map(item => {
        typeMap.set(item.id, item.type);
        if (mobileIds.has(item.parent_id ?? 0)) mobileIds.add(item.id);
        return {
          label: item.meta?.title || item.name || item.path,
          value: item.id,
          children: convert(item.children ?? [])
        };
      });
  const options = convert(menuList);
  parentMenuTypeMap.value = typeMap;
  appMenuIds.value = mobileIds;
  return options;
}

/** 根据菜单类型清理无效字段，避免提交脏数据。 */
function buildSubmitPayload(): BaseMenuForm {
  const payload = normalizeMenuForm(formData);
  payload.translations = serializeDynamicTranslations(translationValues.value, "title") as BaseMenuTranslation[];
  // 一级菜单在表单中保持空白，提交时仍按接口约定传回根节点标识。
  if (payload.id > 0 && payload.parent_id === undefined) payload.parent_id = 0;
  payload.meta.params = (payload.meta.params ?? []).filter(item => item.key || item.value);

  if (isAppMenu(payload.parent_id)) {
    payload.component = "";
    payload.redirect = "";
    payload.meta.app ??= createDefaultAppMenuMeta();
    payload.meta.always_show = false;
    payload.meta.hidden = false;
    payload.meta.keep_alive = false;
    payload.meta.full = false;
    payload.meta.affix = false;
    payload.meta.params = [];
    payload.meta.app.in_tab_bar = isAppTab(payload.parent_id);
    if (!payload.meta.app.in_tab_bar) payload.meta.app.selected_icon = undefined;
  }

  if (payload.type === BaseMenuType.BUTTON) {
    payload.name = "";
    payload.component = "";
    payload.redirect = "";
    payload.meta.icon = "";
    payload.meta.always_show = false;
    payload.meta.hidden = true;
    payload.meta.keep_alive = false;
    payload.meta.full = false;
    payload.meta.affix = false;
    payload.meta.params = [];
  }

  if (payload.type === BaseMenuType.FOLDER) {
    payload.path = "";
    payload.name = "";
    payload.component = "Layout";
    if (payload.id !== APP_MENU_ROOT_ID) payload.api = [];
    payload.meta.keep_alive = false;
    payload.meta.full = false;
    payload.meta.affix = false;
    payload.meta.params = [];
  }

  if (payload.type === BaseMenuType.EXT_LINK) {
    payload.name = "";
    payload.component = "";
    payload.api = [];
    payload.meta.keep_alive = false;
    payload.meta.full = false;
    payload.meta.affix = false;
    payload.meta.params = [];
  }

  return payload;
}

/** 加载菜单树选项和 API 列表，确保弹窗打开时相关数据已可用。 */
async function loadDialogResources() {
  const [menuData, apiData] = await Promise.all([defBaseMenuService.TreeBaseMenu({}), defBaseApiService.OptionBaseApi({})]);
  menuOptions.value = buildMenuOptions(menuData.base_menus ?? []);
  apiList.value = apiData.base_apis ?? [];
}

/** 根据关键字递归过滤菜单树，保留匹配节点及其父级。 */
function filterMenuTree(menuList: BaseMenu[], keywordMap: Record<string, string>) {
  const titleKeyword = (keywordMap.title ?? "").trim().toLowerCase();
  const nameKeyword = (keywordMap.name ?? "").trim().toLowerCase();
  const pathKeyword = (keywordMap.path ?? "").trim().toLowerCase();

  return menuList.reduce<BaseMenu[]>((result, item) => {
    const children = filterMenuTree(item.children ?? [], keywordMap);
    const title = item.meta?.title?.toLowerCase() ?? "";
    const name = item.name?.toLowerCase() ?? "";
    const path = item.path?.toLowerCase() ?? "";
    const matched =
      (!titleKeyword || title.includes(titleKeyword)) &&
      (!nameKeyword || name.includes(nameKeyword)) &&
      (!pathKeyword || path.includes(pathKeyword));

    if (!matched && !children.length) return result;

    result.push({
      ...item,
      children
    });
    return result;
  }, []);
}

/** 请求菜单表格数据，并按搜索条件过滤树形结构。 */
async function requestMenuTable(params: Record<string, string>) {
  const data = await defBaseMenuService.TreeBaseMenu({});
  const keywordMap = {
    title: params.title ?? "",
    name: params.name ?? "",
    path: params.path ?? ""
  };

  return {
    data: filterMenuTree(data.base_menus ?? [], keywordMap)
  };
}

/** 刷新菜单表格。 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 打开菜单弹窗。
 * parentMenu 为新增时的固定父节点，menuId 为编辑时的菜单 ID。
 */
async function handleOpenDialog(parentMenu?: BaseMenu, menuId?: number) {
  await loadDialogResources();
  dialog.parentLocked = Boolean(parentMenu || menuId);
  dialog.editing = Boolean(menuId);
  dialog.parentType = parentMenu?.type ?? BaseMenuType.UNKNOWN_MT;
  resetForm(menuId ? undefined : { parent_id: parentMenu?.id });
  dialog.visible = true;

  if (menuId) {
    const data = await defBaseMenuService.GetBaseMenu({ id: menuId });
    resetForm(data);
    return;
  }
}

/** 关闭菜单弹窗并显式重置表单与校验状态。 */
function handleCloseDialog() {
  dialog.visible = false;
  dialog.parentType = BaseMenuType.UNKNOWN_MT;
  dialog.parentLocked = true;
  resetForm();
}

/** 重置当前表单数据和校验状态，避免新增与编辑切换时残留旧值。 */
function resetForm(data?: Partial<BaseMenuForm>) {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  Object.assign(formData, normalizeMenuForm(data));
  translationValues.value = normalizeDynamicTranslations(data?.translations as DynamicTranslationRecord[] | undefined, "title");
}

/** 提交菜单表单，并在成功后关闭弹窗、刷新表格。 */
async function handleSubmit() {
  const valid = await formDialogRef.value?.validate();
  if (!valid) return;

  const payload = buildSubmitPayload();
  if (payload.id > 0) {
    await defBaseMenuService.UpdateBaseMenu({ base_menu: payload });
    ElMessage.success(t("system.menu.message.updateSuccess"));
  } else {
    await defBaseMenuService.CreateBaseMenu({ base_menu: payload });
    ElMessage.success(t("system.menu.message.createSuccess"));
  }

  handleCloseDialog();
  refreshTable();
}

/**
 * 在菜单状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseMenu) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const action = t(nextStatus === Status.ENABLE ? "common.status.enabled" : "common.status.disabled");
  const menuName = row.meta?.title || row.name || row.path || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(t("system.menu.message.confirmStatus", { action, name: menuName }), t("common.title.notice"), {
      confirmButtonText: t("common.action.confirm"),
      cancelButtonText: t("common.action.cancel"),
      type: "warning"
    });
    await defBaseMenuService.SetBaseMenuStatus({ id: row.id, status: nextStatus });
    ElMessage.success(t("system.common.message.statusSuccess", { action }));
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除菜单，兼容单条删除与批量删除。
 */
function handleDeleteMenu(selected?: number | string | Array<number | string> | BaseMenu | BaseMenu[]) {
  const menuList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseMenu[])
    : selected && typeof selected === "object"
      ? [selected as BaseMenu]
      : [];
  if (menuList.some(item => item.parent_id === 0 || item.id === APP_MENU_ROOT_ID)) {
    ElMessage.warning(t("system.menu.message.protectedDelete"));
    return;
  }
  const menuIds = (
    menuList.length ? menuList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!menuIds) {
    ElMessage.warning(t("system.common.message.selectDeleteItem"));
    return;
  }

  const singleMenuName = menuList[0]?.meta?.title || menuList[0]?.name || menuList[0]?.path || `ID:${menuList[0]?.id ?? ""}`;
  const confirmMessage = menuList.length
    ? menuList.length === 1
      ? t("system.menu.message.confirmDeleteSingle", { name: singleMenuName })
      : t("system.menu.message.confirmDeleteBatch", { count: menuList.length })
    : t("system.menu.message.confirmDeleteSelected");

  ElMessageBox.confirm(confirmMessage, t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () => {
      defBaseMenuService.DeleteBaseMenu({ id: menuIds }).then(() => {
        ElMessage.success(t("system.menu.message.deleteSuccess"));
        refreshTable();
      });
    },
    () => {
      ElMessage.info(t("system.menu.message.deleteCanceled"));
    }
  );
}
</script>

<style scoped lang="scss">
.table-box {
  height: 100%;
}
:deep(.menu-form__params .el-form-item__content) {
  align-items: flex-start;
}
</style>
