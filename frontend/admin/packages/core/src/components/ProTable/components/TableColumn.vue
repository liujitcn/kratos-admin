<template>
  <RenderTableColumn v-bind="column" />
</template>

<script setup lang="ts" name="TableColumn">
import { h, inject, isProxy, markRaw, onUnmounted, ref, toRaw, useSlots, withDirectives } from "vue";
import type { ObjectDirective } from "vue";
import {
  ElButton,
  ElDropdown,
  ElDropdownItem,
  ElDropdownMenu,
  ElImage,
  ElSwitch,
  ElTableColumn,
  ElTag,
  ElText
} from "element-plus";
import DictLabel from "@/components/Dict/DictLabel.vue";
import { ColumnProps, HeaderRenderScope, RenderScope, TableActionProps } from "@/components/ProTable/interface";
import type { TableAlign } from "@/utils/proTable";
import { filterEnum, formatValue, handleProp, handleRowAccordingToProp } from "@/utils";
import { formatPrice, formatSrc } from "@/utils/utils";
import { t } from "@/locales";

const props = defineProps<{
  column: ColumnProps;
  resolveAlign?: (column: ColumnProps) => TableAlign;
}>();

const slots = useSlots();

const enumMap = inject("enumMap", ref(new Map()));

const actionWidths = ref(new Map<string, number>());
const measuredCells = new Map<HTMLElement, string>();
let resizeObserver: ResizeObserver | undefined;
let measureFrame = 0;

/** 合并当前列所有可见行和表头的实际宽度，支持语言、字体和权限变化。 */
const scheduleActionMeasurement = () => {
  cancelAnimationFrame(measureFrame);
  measureFrame = requestAnimationFrame(() => {
    const widths = new Map<string, number>();
    measuredCells.forEach((key, element) => {
      if (!element.isConnected || !element.getClientRects().length) return;
      const cell = element.closest(".cell");
      if (!cell) return;
      const style = getComputedStyle(cell);
      const width = Math.ceil(
        element.getBoundingClientRect().width + parseFloat(style.paddingLeft) + parseFloat(style.paddingRight) + 2
      );
      widths.set(key, Math.max(widths.get(key) ?? 80, width));
    });
    if (widths.size !== actionWidths.value.size || [...widths].some(([key, width]) => actionWidths.value.get(key) !== width)) {
      actionWidths.value = widths;
    }
  });
};

/** 观察渲染后的操作内容，避免按字符个数估算不同语言的文本宽度。 */
const measureActionContent: ObjectDirective<HTMLElement, string> = {
  mounted(element, binding) {
    measuredCells.set(element, binding.value);
    resizeObserver ??= new ResizeObserver(scheduleActionMeasurement);
    resizeObserver.observe(element);
    scheduleActionMeasurement();
  },
  updated(element, binding) {
    measuredCells.set(element, binding.value);
    scheduleActionMeasurement();
  },
  unmounted(element) {
    measuredCells.delete(element);
    resizeObserver?.unobserve(element);
    scheduleActionMeasurement();
  }
};

onUnmounted(() => {
  resizeObserver?.disconnect();
  cancelAnimationFrame(measureFrame);
  measuredCells.clear();
});

/**
 * 透传给 Element Plus 前移除图标组件上的响应式代理，避免 Vue 对组件对象发出性能告警。
 */
const normalizeActionIcon = (icon: unknown): any => {
  if (!icon || (typeof icon !== "object" && typeof icon !== "function")) return icon;
  const rawIcon = isProxy(icon) ? toRaw(icon) : icon;
  return typeof rawIcon === "object" ? markRaw(rawIcon) : rawIcon;
};

// 渲染表格数据
const renderCellData = (item: ColumnProps, scope: RenderScope<any>) => {
  return enumMap.value.get(item.prop) && item.isFilterEnum
    ? filterEnum(handleRowAccordingToProp(scope.row, item.prop!), enumMap.value.get(item.prop)!, item.fieldNames)
    : formatValue(handleRowAccordingToProp(scope.row, item.prop!));
};

// 获取 tag 类型
const getTagType = (item: ColumnProps, scope: RenderScope<any>) => {
  return (
    filterEnum(handleRowAccordingToProp(scope.row, item.prop!), enumMap.value.get(item.prop), item.fieldNames, "tag") || "primary"
  );
};

/**
 * 渲染字典列内容，统一复用 DictLabel 组件处理标签与文本展示。
 */
const renderDictCellData = (item: ColumnProps, scope: RenderScope<any>) => {
  if (!item.dictCode || !item.prop) return null;
  return h(DictLabel, { code: item.dictCode, modelValue: handleRowAccordingToProp(scope.row, item.prop) });
};

/**
 * 解析列级自定义参数，统一兼容静态对象与函数返回值。
 */
const resolveColumnParams = (params: any, scope: RenderScope<any>) => {
  if (!params) return undefined;
  return typeof params === "function" ? params(scope) : params;
};

/**
 * 根据 prop 将开关值同步回行数据，兼容多级路径字段。
 */
const setRowValueByProp = (row: Record<string, any>, prop: string, value: any) => {
  if (!prop.includes(".")) {
    row[prop] = value;
    return;
  }
  const propList = prop.split(".");
  const lastProp = propList.pop() as string;
  let currentRow = row;
  propList.forEach(item => {
    if (typeof currentRow[item] !== "object" || currentRow[item] === null) {
      currentRow[item] = {};
    }
    currentRow = currentRow[item];
  });
  currentRow[lastProp] = value;
};

/**
 * 统一解析按钮显隐与禁用状态，避免渲染分支里重复判断。
 */
const getBooleanValue = (value: boolean | ((scope: RenderScope<any>) => boolean) | undefined, scope: RenderScope<any>) => {
  if (typeof value === "function") return value(scope);
  return Boolean(value);
};

/**
 * 渲染图片预览列，统一处理缩略图与大图预览。
 */
const renderImageCell = (item: ColumnProps, scope: RenderScope<any>) => {
  const imageProps = item.imageProps ?? {};
  const rawSrc =
    typeof imageProps.src === "function"
      ? imageProps.src(scope)
      : (imageProps.src ?? handleRowAccordingToProp(scope.row, item.prop!));
  const rawPreviewSrc = typeof imageProps.previewSrc === "function" ? imageProps.previewSrc(scope) : (imageProps.previewSrc ?? rawSrc);
  if (!rawSrc || rawSrc === "--") return "--";
  const src = formatSrc(String(rawSrc));
  const previewSrc = formatSrc(String(rawPreviewSrc));
  const thumbWidth = typeof imageProps.width === "number" ? `${imageProps.width}px` : (imageProps.width ?? "60px");
  const thumbHeight = typeof imageProps.height === "number" ? `${imageProps.height}px` : (imageProps.height ?? "60px");
  return h(
    ElImage,
    {
      src,
      previewSrcList: [previewSrc],
      previewTeleported: true,
      zoomRate: 1.2,
      maxScale: 7,
      minScale: 0.2,
      showProgress: true,
      initialIndex: 0,
      fit: "cover",
      style: {
        width: thumbWidth,
        height: thumbHeight,
        borderRadius: "var(--admin-page-radius)"
      }
    },
    {
      error: () => h(ElText, { type: "info", size: "small" }, () => t("common.message.no_data"))
    }
  );
};

/**
 * 渲染状态开关列，并在切换后回调页面业务方法。
 */
const renderStatusCell = (item: ColumnProps, scope: RenderScope<any>) => {
  if (!item.prop || !item.statusProps) return renderCellData(item, scope);
  const statusProps = item.statusProps;
  const params = resolveColumnParams(statusProps.params, scope);
  return h(ElSwitch, {
    modelValue: handleRowAccordingToProp(scope.row, item.prop),
    inlinePrompt: true,
    activeValue: statusProps.activeValue,
    inactiveValue: statusProps.inactiveValue,
    activeText: statusProps.activeText,
    inactiveText: statusProps.inactiveText,
    disabled: getBooleanValue(statusProps.disabled, scope),
    beforeChange: () => statusProps.beforeChange?.(scope, params) ?? true,
    "onUpdate:modelValue": value => {
      setRowValueByProp(scope.row, item.prop!, value);
      statusProps.onChange?.(value, scope, params);
    }
  });
};

/**
 * 渲染金额列，统一将分单位金额转换为元字符串，并支持简单前后缀。
 */
const renderMoneyCell = (item: ColumnProps, scope: RenderScope<any>) => {
  const moneyProps = item.moneyProps ?? {};
  const rawValue =
    typeof moneyProps.value === "function"
      ? moneyProps.value(scope)
      : (moneyProps.value ?? handleRowAccordingToProp(scope.row, item.prop!));
  if (rawValue === undefined || rawValue === null || rawValue === "") return "--";
  const amount = typeof rawValue === "number" ? rawValue : Number(rawValue);
  if (!Number.isFinite(amount)) return String(rawValue);
  return `${moneyProps.prefix ?? ""}${formatPrice(amount)}${moneyProps.suffix ?? ""}`;
};

/**
 * 渲染操作按钮列，优先展示编辑、删除，超过三项时将其他操作折叠。
 */
const renderActionsCell = (item: ColumnProps, scope: RenderScope<any>) => {
  if (!item.actions?.length) return "--";
  const visibleActions = item.actions.filter(action => !getBooleanValue(action.hidden, scope));
  if (!visibleActions.length) return "--";
  const primaryLabels = [t("common.action.edit"), t("common.action.delete")];
  const primaryActions = primaryLabels.flatMap(label => visibleActions.filter(action => action.label === label));
  const otherActions = visibleActions.filter(action => !primaryLabels.includes(action.label));
  const inlineActions = visibleActions.length > 3 ? primaryActions : [...primaryActions, ...otherActions];
  const buttons = inlineActions.map((action: TableActionProps) => {
    const params = resolveColumnParams(action.params, scope);
    return h(
      ElButton,
      {
        key: action.label,
        size: "small",
        type: action.type ?? "primary",
        link: action.link ?? true,
        icon: normalizeActionIcon(action.icon),
        disabled: getBooleanValue(action.disabled, scope),
        onClick: () => action.onClick(scope, params)
      },
      { default: () => action.label }
    );
  });
  if (visibleActions.length > 3 && otherActions.length)
    buttons.push(
      h(
        ElDropdown,
        { trigger: "click", placement: "bottom-end", size: "small" },
        {
          default: () =>
            h(ElButton, { type: "primary", link: true, size: "small", icon: ArrowDown }, () => t("common.action.more")),
          dropdown: () =>
            h(ElDropdownMenu, null, () =>
              otherActions.map(action => {
                const disabled = getBooleanValue(action.disabled, scope);
                const params = resolveColumnParams(action.params, scope);
                return h(
                  ElDropdownItem,
                  {
                    key: action.label,
                    icon: normalizeActionIcon(action.icon),
                    disabled,
                    style: disabled ? undefined : { color: `var(--el-color-${action.type ?? "primary"})` },
                    onClick: () => {
                      if (!disabled) action.onClick(scope, params);
                    }
                  },
                  () => action.label
                );
              })
            )
        }
      )
    );
  const content = h("span", { class: "pro-table-actions" }, buttons);
  return item.width == null && item.minWidth == null
    ? withDirectives(content, [[measureActionContent, item.prop ?? item.label ?? ""]])
    : content;
};

/** 渲染默认表头标题，避免窄列换行并保留完整标题提示。 */
const renderHeaderLabel = (item: ColumnProps) => {
  if (!item.label || item.showOverflowTooltip === false) return item.label;
  return h("span", { title: item.label }, item.label);
};

/**
 * 渲染预置列类型，统一收敛图片、状态和操作按钮等通用场景。
 */
const renderPresetCell = (item: ColumnProps, scope: RenderScope<any>) => {
  switch (item.cellType) {
    case "image":
      return renderImageCell(item, scope);
    case "status":
      return renderStatusCell(item, scope);
    case "actions":
      return renderActionsCell(item, scope);
    case "money":
      return renderMoneyCell(item, scope);
    default:
      return null;
  }
};

/** 渲染表格列，为操作列提供自适应宽度和右侧固定默认值。 */
const RenderTableColumn = (item: ColumnProps) => {
  if (!item.isShow) return null;
  const isActionColumn = item.cellType === "actions" || item.prop === "operation";
  const autoActionWidth = item.cellType === "actions" && !item.render && !(item.prop && slots[handleProp(item.prop)]);
  return h(
    ElTableColumn,
    {
      ...item,
      width:
        item.width ??
        (isActionColumn && item.minWidth == null
          ? autoActionWidth
            ? (actionWidths.value.get(item.prop ?? item.label ?? "") ?? 80)
            : 160
          : undefined),
      fixed: item.fixed ?? (isActionColumn ? "right" : undefined),
      align: item.align ?? props.resolveAlign?.(item) ?? "left",
      showOverflowTooltip: item.showOverflowTooltip ?? item.prop !== "operation"
    },
    {
      default: (scope: RenderScope<any>) => {
        if (item._children) return item._children.map(child => RenderTableColumn(child));
        if (item.render) return item.render(scope);
        if (item.prop && slots[handleProp(item.prop)]) return slots[handleProp(item.prop)]!(scope);
        if (item.cellType) return renderPresetCell(item, scope);
        if (item.dictCode) return renderDictCellData(item, scope);
        if (item.tag) return h(ElTag, { type: getTagType(item, scope) }, { default: () => renderCellData(item, scope) });
        return renderCellData(item, scope);
      },
      header: (scope: HeaderRenderScope<any>) => {
        if (item.headerRender) return item.headerRender(scope);
        if (item.prop && slots[`${handleProp(item.prop)}Header`]) return slots[`${handleProp(item.prop)}Header`]!(scope);
        if (autoActionWidth && item.width == null && item.minWidth == null) {
          return withDirectives(h("span", { class: "pro-table-action-header" }, item.label), [
            [measureActionContent, item.prop ?? item.label ?? ""]
          ]);
        }
        return renderHeaderLabel(item);
      }
    }
  );
};
</script>

<style lang="scss">
// 操作容器在函数式列的插槽中生成，不带本组件的 scopeId，使用专属类名限定样式。
.pro-table-actions {
  display: inline-flex;
  flex-wrap: nowrap;
  gap: 4px;
  align-items: center;
  width: max-content;
  vertical-align: middle;
  white-space: nowrap;
  .el-button {
    margin-left: 0;
  }
  .el-dropdown {
    display: inline-flex;
    align-items: center;
  }
}
.pro-table-action-header {
  display: inline-block;
  width: max-content;
  white-space: nowrap;
}
</style>
