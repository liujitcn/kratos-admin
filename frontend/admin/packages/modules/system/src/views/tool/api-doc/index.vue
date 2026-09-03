<template>
  <div v-loading="loading" class="app-container code-gen-sub-page">
    <div class="api-doc-page__layout">
      <el-card class="code-gen-sub-card api-doc-page__navigation" shadow="never">
        <div v-if="documents.length" class="api-doc-page__document-list">
          <button
            v-for="document in documents"
            :key="document.key"
            type="button"
            class="api-doc-page__document"
            :class="{ 'is-active': selectedDocumentKey === document.key }"
            @click="handleDocumentChange(document.key)"
          >
            <span class="api-doc-page__document-name">{{ document.name }}</span>
            <span class="api-doc-page__document-key">{{ document.key }}</span>
          </button>
        </div>
        <el-empty v-else :image-size="56" :description="t('system.base.api.doc.message.empty')" />
      </el-card>
      <el-card class="code-gen-sub-card api-doc-page__card" shadow="never">
        <div ref="swaggerRootRef" class="api-doc-page__swagger" />
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import SwaggerUIBundle from "swagger-ui-dist/swagger-ui-bundle.js";
import "swagger-ui-dist/swagger-ui.css";
import { DEFAULT_LOCALE, t, useLocaleStore } from "@liujitcn/kratos-admin-core";
import { getLocaleRequestHeaders, getRequestAccessToken } from "@liujitcn/kratos-admin-core/request";
import { defBaseApiService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_api";
import type { OpenApiServiceOption } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api";

const swaggerRootRef = ref<HTMLElement>();
const loading = ref(true);
const documents = ref<OpenApiServiceOption[]>([]);
const selectedDocumentKey = ref("");
const { locale } = useLocaleStore();

/** 初始化携带当前登录令牌的 Swagger UI。 */
function initializeSwaggerUI(documentKey: string) {
  const swaggerRoot = swaggerRootRef.value;
  if (!swaggerRoot) return;

  loading.value = true;
  swaggerRoot.replaceChildren();
  const localePath = locale.value === DEFAULT_LOCALE ? "" : `/${encodeURIComponent(locale.value)}`;
  SwaggerUIBundle({
    domNode: swaggerRoot,
    url: `/api/docs/openapi/${encodeURIComponent(documentKey)}${localePath}`,
    deepLinking: true,
    displayOperationId: true,
    displayRequestDuration: true,
    docExpansion: "list",
    defaultModelsExpandDepth: -1,
    filter: true,
    onComplete: () => {
      loading.value = false;
    },
    persistAuthorization: false,
    tryItOutEnabled: true,
    validatorUrl: null,
    requestInterceptor: async request => {
      const accessToken = await getRequestAccessToken();
      if (accessToken) {
        request.headers = {
          ...request.headers,
          ...getLocaleRequestHeaders(),
          Authorization: accessToken
        };
      } else {
        request.headers = { ...request.headers, ...getLocaleRequestHeaders() };
      }
      return request;
    }
  });
}

/** 切换当前展示的 OpenAPI 文档。 */
function handleDocumentChange(documentKey: string) {
  if (documentKey === selectedDocumentKey.value) return;
  selectedDocumentKey.value = documentKey;
  initializeSwaggerUI(documentKey);
}

/** 加载可访问的 OpenAPI 文档并初始化默认文档。 */
async function loadOpenAPIDocuments() {
  loading.value = true;
  try {
    const response = await defBaseApiService.OptionOpenApiService({});
    documents.value = response.list;
    const selectedDocument = response.list.find(document => document.key === selectedDocumentKey.value) ?? response.list[0];
    if (!selectedDocument) {
      selectedDocumentKey.value = "";
      loading.value = false;
      return;
    }
    selectedDocumentKey.value = selectedDocument.key;
    initializeSwaggerUI(selectedDocument.key);
  } catch {
    loading.value = false;
  }
}

onMounted(() => {
  loadOpenAPIDocuments();
});

watch(locale, () => {
  loadOpenAPIDocuments();
});

onBeforeUnmount(() => {
  swaggerRootRef.value?.replaceChildren();
});
</script>

<style scoped lang="scss">
.code-gen-sub-page {
  box-sizing: border-box;
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 0;
}
.code-gen-sub-card {
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}
.api-doc-page__layout {
  display: grid;
  flex: 1;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 10px;
  min-height: 0;
  overflow: hidden;
}
.api-doc-page__navigation,
.api-doc-page__card {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}
.api-doc-page__document-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.api-doc-page__document {
  display: flex;
  flex-direction: column;
  gap: 3px;
  width: 100%;
  min-width: 0;
  padding: 10px 12px;
  font: inherit;
  color: var(--admin-page-text-primary);
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: var(--admin-page-radius);
  transition: background-color 0.2s ease;
}
.api-doc-page__document:hover {
  background: var(--el-fill-color-light);
}
.api-doc-page__document.is-active {
  color: var(--admin-page-accent-soft-text);
  background: var(--admin-page-accent-soft-bg);
  box-shadow: inset 3px 0 0 var(--el-color-primary);
}
.api-doc-page__document-name,
.api-doc-page__document-key {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.api-doc-page__document-name {
  font-size: 14px;
  font-weight: 500;
}
.api-doc-page__document-key {
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}
.api-doc-page__swagger {
  height: 100%;
  min-height: 0;
  overflow: auto;
}
:deep(.api-doc-page__navigation .el-card__body),
:deep(.api-doc-page__card .el-card__body) {
  flex: 1;
  min-height: 0;
}
:deep(.api-doc-page__navigation .el-card__body) {
  padding: 8px;
  overflow-y: auto;
}
:deep(.api-doc-page__card .el-card__body) {
  height: 100%;
  padding: 0;
}
:deep(.api-doc-page__navigation .el-empty) {
  padding: 32px 0;
}
:deep(.swagger-ui) {
  color: var(--el-text-color-primary);
}
:deep(.swagger-ui .information-container) {
  display: none;
}
:deep(.swagger-ui .info .title),
:deep(.swagger-ui .info p),
:deep(.swagger-ui .opblock-tag),
:deep(.swagger-ui .opblock .opblock-summary-description),
:deep(.swagger-ui .opblock .opblock-summary-path),
:deep(.swagger-ui .parameter__name),
:deep(.swagger-ui .parameter__type),
:deep(.swagger-ui table thead tr td),
:deep(.swagger-ui table thead tr th) {
  color: var(--el-text-color-primary);
}
:deep(.swagger-ui .scheme-container),
:deep(.swagger-ui .opblock .opblock-section-header),
:deep(.swagger-ui .opblock-body pre),
:deep(.swagger-ui .model-box),
:deep(.swagger-ui .dialog-ux .modal-ux) {
  background: var(--el-fill-color-light);
  box-shadow: none;
}
:deep(.swagger-ui .opblock),
:deep(.swagger-ui .opblock-tag),
:deep(.swagger-ui .model-box),
:deep(.swagger-ui .dialog-ux .modal-ux),
:deep(.swagger-ui .dialog-ux .modal-ux-content) {
  border-color: var(--el-border-color-light);
}
:deep(.swagger-ui input[type="text"]),
:deep(.swagger-ui textarea),
:deep(.swagger-ui select) {
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border-color: var(--el-border-color);
}

@media (width <= 720px) {
  .code-gen-sub-page {
    height: auto;
    min-height: 100%;
  }
  .api-doc-page__layout {
    display: flex;
    flex-direction: column;
    overflow: visible;
  }
  .api-doc-page__navigation {
    flex: 0 0 auto;
    max-height: 240px;
  }
  .api-doc-page__card {
    min-height: 600px;
  }
}
</style>
