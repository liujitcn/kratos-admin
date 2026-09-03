<template>
  <div v-loading="pageLoading" class="table-box runtime-config-page">
    <main class="runtime-config-shell">
      <aside class="runtime-config-navigation" :aria-label="t('system.base.runtime_config.navigation')">
        <header class="runtime-config-navigation__header">
          <el-input
            v-model="keyword"
            clearable
            :placeholder="t('system.base.runtime_config.placeholder.search')"
            :prefix-icon="Search"
          />
          <span class="runtime-config-count">{{ filteredDefinitions.length }} / {{ runtimeConfigDefinitions.length }}</span>
        </header>

        <nav
          v-if="filteredDefinitions.length"
          class="runtime-config-list"
          :aria-label="t('system.base.runtime_config.navigation')"
        >
          <button
            v-for="definition in filteredDefinitions"
            :key="definition.key"
            type="button"
            class="runtime-config-item"
            :class="{ 'is-active': definition.key === selectedKey }"
            @click="selectConfig(definition.key)"
          >
            <el-icon class="runtime-config-item__icon"><component :is="definition.icon" /></el-icon>
            <span class="runtime-config-item__text">
              <strong>{{ t(definition.titleKey) }}</strong>
              <small>{{ definition.key }}</small>
            </span>
          </button>
        </nav>
        <el-empty v-else :description="t('system.base.runtime_config.message.no_match')" :image-size="72" />
      </aside>

      <section class="runtime-config-content">
        <div v-if="!selectedDefinition" class="runtime-config-empty">
          <el-empty :description="t('system.base.runtime_config.message.select')" />
        </div>
        <template v-else>
          <header class="runtime-config-header">
            <div class="runtime-config-header__identity">
              <span class="runtime-config-header__icon">
                <el-icon><component :is="selectedDefinition.icon" /></el-icon>
              </span>
              <div>
                <h1>{{ t(selectedDefinition.titleKey) }}</h1>
                <p>{{ t(selectedDefinition.descriptionKey) }}</p>
              </div>
            </div>
            <time v-if="updatedAt" :datetime="updatedAt" :title="updatedAt">
              {{ t("system.base.runtime_config.value.updated_at", { time: formatUpdatedAt(updatedAt) }) }}
            </time>
          </header>

          <div v-loading="configLoading" class="runtime-config-form-scroll">
            <el-skeleton v-if="configLoading && !loadedKeys.has(selectedDefinition.key)" :rows="8" animated />
            <ProForm
              v-else
              :key="selectedDefinition.key"
              ref="formRef"
              :model="selectedModel"
              :fields="localizedFields"
              label-width="220px"
            />
          </div>

          <footer v-if="canUpdateRuntimeConfig" class="runtime-config-footer">
            <el-button type="primary" :icon="Check" :loading="configSaving" @click="saveCurrent">
              {{ t("common.action.save") }}
            </el-button>
          </footer>
        </template>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { Check, Search } from "@element-plus/icons-vue";
import { ElMessage } from "element-plus";
import { getCurrentLocale, t, useLocaleStore } from "@liujitcn/kratos-admin-core";
import ProForm from "@liujitcn/kratos-admin-core/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { defBaseConfigService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_config";
import {
  runtimeConfigDefinitions,
  type RuntimeConfigDefinition,
  type RuntimeConfigField,
  type RuntimeConfigMessageArg,
  type RuntimeConfigModel
} from "@liujitcn/kratos-admin-system/config";

defineOptions({ name: "BaseRuntimeConfig", inheritAttrs: false });

const { locale } = useLocaleStore();
const { BUTTONS } = useAuthButtons();
/** 判断当前用户是否拥有运行配置更新权限。 */
const canUpdateRuntimeConfig = computed(() => !!BUTTONS.value["base:runtime-config:update"]);
const pageLoading = ref(false);
const configLoading = ref(false);
const configSaving = ref(false);
const keyword = ref("");
const selectedKey = ref("");
const updatedAt = ref("");
const loadedKeys = reactive(new Set<string>());
const models = reactive<Record<string, RuntimeConfigModel>>({});
const formRef = ref<ProFormInstance>();
let requestToken = 0;

const sortedDefinitions = computed(() => {
  const currentLocale = locale.value;
  return [...runtimeConfigDefinitions].sort((left, right) => {
    const titleOrder = t(left.titleKey).localeCompare(t(right.titleKey), currentLocale || getCurrentLocale(), {
      sensitivity: "base"
    });
    return titleOrder || left.key.localeCompare(right.key);
  });
});

const filteredDefinitions = computed(() => {
  const search = keyword.value.trim().toLocaleLowerCase();
  if (!search) return sortedDefinitions.value;
  return sortedDefinitions.value.filter(definition => {
    return [t(definition.titleKey), t(definition.descriptionKey), definition.key].some(value =>
      value.toLocaleLowerCase().includes(search)
    );
  });
});

const selectedDefinition = computed<RuntimeConfigDefinition | undefined>(() => {
  return runtimeConfigDefinitions.find(definition => definition.key === selectedKey.value);
});

const selectedModel = computed(() => {
  return selectedKey.value ? (models[selectedKey.value] ?? {}) : {};
});

const localizedFields = computed<ProFormField[]>(() => {
  if (!selectedDefinition.value) return [];
  return selectedDefinition.value.fields.map(localizeField);
});

/** 加载指定配置并恢复默认模型。 */
async function loadConfig(key: string) {
  const definition = runtimeConfigDefinitions.find(item => item.key === key);
  if (!definition) return;
  const currentToken = requestToken + 1;
  requestToken = currentToken;
  configLoading.value = true;
  if (!models[key]) models[key] = definition.createModel();
  try {
    const result = await defBaseConfigService.GetBaseConfigByKey({ key });
    if (requestToken !== currentToken || selectedKey.value !== key) return;
    Object.assign(models[key], JSON.parse(result.value_json) as RuntimeConfigModel);
    updatedAt.value = result.updated_at;
    loadedKeys.add(key);
  } catch {
    if (requestToken === currentToken) ElMessage.error(t("system.base.runtime_config.message.load_failed"));
  } finally {
    if (requestToken === currentToken) configLoading.value = false;
  }
}

/** 切换当前配置表单。 */
function selectConfig(key: string) {
  if (selectedKey.value === key) return;
  selectedKey.value = key;
  updatedAt.value = "";
  formRef.value?.clearValidate();
  void loadConfig(key);
}

/** 保存当前配置并刷新服务端缓存。 */
async function saveCurrent() {
  const definition = selectedDefinition.value;
  if (!canUpdateRuntimeConfig.value || !definition || !selectedKey.value) return;
  const valid = await formRef.value?.validate();
  if (valid !== true) return;
  configSaving.value = true;
  try {
    await defBaseConfigService.UpdateBaseConfigByKey({
      key: definition.key,
      value_json: JSON.stringify(selectedModel.value)
    });
    loadedKeys.add(definition.key);
    ElMessage.success(t("system.base.runtime_config.message.save_success"));
  } catch {
    ElMessage.error(t("system.base.runtime_config.message.save_failed"));
  } finally {
    configSaving.value = false;
  }
}

/** 将配置字段定义本地化为 ProForm 字段。 */
function localizeField(field: RuntimeConfigField): ProFormField {
  const fieldProps = field.props;
  return {
    prop: field.prop,
    label: t(field.labelKey),
    component: field.component,
    props:
      typeof fieldProps === "function"
        ? model => {
            const props = fieldProps(model);
            return { ...props, disabled: !canUpdateRuntimeConfig.value || Boolean(props.disabled) };
          }
        : { ...(fieldProps ?? {}), disabled: !canUpdateRuntimeConfig.value || Boolean(fieldProps?.disabled) },
    itemProps: field.itemProps,
    options: field.options,
    visible: field.visible,
    labelTooltip: field.labelTooltipKey ? t(field.labelTooltipKey) : undefined,
    rules: field.rules?.map(rule => ({
      ...rule,
      message: t(rule.messageKey, localizeArgs(rule.messageArgs))
    }))
  };
}

/** 将配置校验规则中的消息键参数转换为当前语言文本。 */
function localizeArgs(args?: Record<string, RuntimeConfigMessageArg>) {
  if (!args) return undefined;
  const result: Record<string, string | number> = {};
  for (const [key, value] of Object.entries(args)) {
    result[key] = typeof value === "object" ? t(value.key) : typeof value === "boolean" ? String(value) : value;
  }
  return result;
}

/** 格式化配置更新时间。 */
function formatUpdatedAt(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat(getCurrentLocale(), {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23"
  })
    .format(date)
    .replaceAll("/", "-");
}

watch(locale, () => {
  if (!selectedKey.value) return;
  void loadConfig(selectedKey.value);
});

onMounted(async () => {
  pageLoading.value = true;
  try {
    const firstDefinition = sortedDefinitions.value[0];
    if (firstDefinition) {
      selectedKey.value = firstDefinition.key;
      await loadConfig(firstDefinition.key);
    }
  } finally {
    pageLoading.value = false;
  }
});
</script>

<style scoped lang="scss">
.runtime-config-page {
  box-sizing: border-box;
  color: var(--admin-page-text-primary);
  background: var(--el-bg-color-page);
}

.runtime-config-shell {
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  min-width: 0;
  min-height: 560px;
  overflow: hidden;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}

.runtime-config-navigation,
.runtime-config-content {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.runtime-config-navigation {
  border-right: 1px solid var(--admin-page-divider);
}

.runtime-config-navigation__header {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid var(--admin-page-divider);
}

.runtime-config-count {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--admin-page-text-secondary);
  white-space: nowrap;
}

.runtime-config-list {
  flex: 1;
  min-height: 0;
  padding: 8px;
  overflow-y: auto;
}

.runtime-config-item {
  display: flex;
  gap: 12px;
  align-items: center;
  width: 100%;
  min-height: 64px;
  padding: 10px 12px;
  color: var(--admin-page-text-primary);
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 6px;
  transition:
    background-color 0.2s ease,
    color 0.2s ease;
}

.runtime-config-item:hover {
  background: var(--el-fill-color-light);
}

.runtime-config-item.is-active {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.runtime-config-item__icon {
  flex: 0 0 auto;
  font-size: 18px;
}

.runtime-config-item__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 4px;
}

.runtime-config-item__text strong,
.runtime-config-item__text small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.runtime-config-item__text strong {
  font-size: 14px;
  font-weight: 600;
}

.runtime-config-item__text small {
  color: var(--admin-page-text-secondary);
  font-size: 12px;
}

.runtime-config-content {
  background: var(--el-bg-color-page);
}

.runtime-config-header {
  display: flex;
  gap: 24px;
  align-items: flex-start;
  justify-content: space-between;
  padding: 24px 28px;
  background: var(--admin-page-card-bg);
  border-bottom: 1px solid var(--admin-page-divider);
}

.runtime-config-header__identity {
  display: flex;
  gap: 14px;
  align-items: flex-start;
  min-width: 0;
}

.runtime-config-header__icon {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-radius: 8px;
}

.runtime-config-header h1 {
  margin: 0;
  color: var(--admin-page-text-primary);
  font-size: 20px;
  font-weight: 650;
}

.runtime-config-header p {
  max-width: 720px;
  margin: 8px 0 0;
  color: var(--admin-page-text-secondary);
  line-height: 1.6;
}

.runtime-config-header time {
  flex: 0 0 auto;
  color: var(--admin-page-text-secondary);
  font-size: 12px;
  white-space: nowrap;
}

.runtime-config-form-scroll {
  flex: 1;
  min-height: 0;
  padding: 28px;
  overflow-y: auto;
  background: var(--admin-page-card-bg);
}

.runtime-config-footer {
  display: flex;
  justify-content: flex-end;
  padding: 16px 28px;
  background: var(--admin-page-card-bg);
  border-top: 1px solid var(--admin-page-divider);
}

.runtime-config-empty {
  display: grid;
  flex: 1;
  place-items: center;
  background: var(--admin-page-card-bg);
}

@media (max-width: 800px) {
  .runtime-config-shell {
    grid-template-columns: 1fr;
  }

  .runtime-config-navigation {
    max-height: 280px;
    border-right: 0;
    border-bottom: 1px solid var(--admin-page-divider);
  }

  .runtime-config-header {
    flex-direction: column;
    padding: 20px;
  }

  .runtime-config-form-scroll {
    padding: 20px;
  }

  .runtime-config-footer {
    padding: 14px 20px;
  }
}
</style>
