<template>
  <a href="#" v-bind="$attrs" @click.prevent="download">{{ fileName || src }}</a>
</template>

<script setup lang="ts">
import { loadFileAssetSource } from "@/utils/fileAsset";

defineOptions({ name: "FileAsset", inheritAttrs: false });

interface FileAssetProps {
  /** 数据库存储的原始文件路径或公开文件地址。 */
  src?: string;
  /** 下载时使用的文件名。 */
  fileName?: string;
}

const props = withDefaults(defineProps<FileAssetProps>(), { src: "", fileName: "" });

/** 通过统一文件加载器下载文件并释放临时 URL。 */
async function download() {
  if (!props.src) return;
  try {
    const objectUrl = await loadFileAssetSource(props.src);
    const anchor = document.createElement("a");
    anchor.href = objectUrl;
    anchor.download = props.fileName || props.src.split(/[\\/]/).pop() || "download";
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
    if (objectUrl.startsWith("blob:")) URL.revokeObjectURL(objectUrl);
  } catch {
    return;
  }
}
</script>
