<template>
  <img v-if="resolvedSrc" :src="resolvedSrc" v-bind="$attrs" @click="openPreview" @error="handleError" />
  <el-image-viewer v-if="previewVisible" :url-list="[previewUrl]" @close="previewVisible = false" />
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from "vue";
import { loadFileAssetSource } from "@/utils/fileAsset";

defineOptions({ name: "ImageAsset", inheritAttrs: false });

interface ImageAssetProps {
  /** 数据库存储的原始图片路径或公开图片地址。 */
  src?: string;
  /** 加载失败时使用的备用地址。 */
  fallback?: string;
  /** 是否允许点击预览。 */
  preview?: boolean;
  /** 预览时使用的独立文件地址。 */
  previewSrc?: string;
}

const props = withDefaults(defineProps<ImageAssetProps>(), { src: "", fallback: "", preview: false, previewSrc: "" });
const resolvedSrc = ref("");
const previewUrl = ref("");
const previewVisible = ref(false);
let objectUrl = "";
let previewObjectUrl = "";
let loadVersion = 0;

/** 释放当前组件创建的 Blob URL。 */
function revokeObjectUrl() {
  if (!objectUrl) return;
  URL.revokeObjectURL(objectUrl);
  objectUrl = "";
}

/** 释放当前预览创建的 Blob URL。 */
function revokePreviewObjectUrl() {
  if (!previewObjectUrl || previewObjectUrl === objectUrl) return;
  URL.revokeObjectURL(previewObjectUrl);
  previewObjectUrl = "";
}

/** 加载图片并在请求过期时忽略旧结果。 */
async function refreshSource(source: string) {
  const version = ++loadVersion;
  revokeObjectUrl();
  revokePreviewObjectUrl();
  previewVisible.value = false;
  resolvedSrc.value = "";
  try {
    const nextSource = await loadFileAssetSource(source, props.fallback);
    if (version !== loadVersion) {
      if (nextSource.startsWith("blob:")) URL.revokeObjectURL(nextSource);
      return;
    }
    if (nextSource.startsWith("blob:")) objectUrl = nextSource;
    resolvedSrc.value = nextSource;
  } catch {
    if (version === loadVersion) resolvedSrc.value = props.fallback;
  }
}

/** 加载预览地址并打开大图查看器。 */
async function openPreview() {
  if (!props.preview) return;
  try {
    revokePreviewObjectUrl();
    const nextSource = await loadFileAssetSource(props.previewSrc || props.src, props.fallback);
    if (nextSource.startsWith("blob:") && nextSource !== objectUrl) previewObjectUrl = nextSource;
    previewUrl.value = nextSource;
    previewVisible.value = true;
  } catch {
    previewVisible.value = false;
  }
}

/** 图片请求失败时切换到备用地址。 */
function handleError() {
  revokeObjectUrl();
  revokePreviewObjectUrl();
  resolvedSrc.value = props.fallback;
}

watch(() => props.src, refreshSource, { immediate: true });
onBeforeUnmount(() => {
  revokeObjectUrl();
  revokePreviewObjectUrl();
});
</script>
