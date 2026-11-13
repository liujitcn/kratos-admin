<template>
  <div v-if="active">
    <p class="example-tip">选择的图片只在浏览器中预览，刷新页面后清空。</p>
    <template v-if="componentId === 'UploadImg'">
      <UploadImg v-model:image-url="imageUrl" :api="previewImage" width="180px" height="150px">
        <template #tip>支持 JPG、PNG、GIF，单张不超过 5 MB。</template>
      </UploadImg>
    </template>
    <UploadImgs v-else v-model:file-list="images" :api="previewImage" :limit="3" width="140px" height="140px">
      <template #tip>最多 3 张，可预览或移除。</template>
    </UploadImgs>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onDeactivated, ref } from "vue";
import type { UploadUserFile } from "element-plus";
import UploadImg from "@/components/Upload/Img.vue";
import UploadImgs from "@/components/Upload/Imgs.vue";
import type { FileInfo } from "@/rpc/base/v1/file";
import avatar from "@/assets/images/avatar.png";
import { useDemoActive } from "./useDemoActive";

defineProps<{ componentId: string }>();
const active = useDemoActive();
const imageUrl = ref(avatar);
const images = ref<UploadUserFile[]>([{ name: "示例头像.png", url: avatar }]);
const objectUrls = new Set<string>();

/** 用浏览器对象地址完成图片选择回填，不发送上传请求。 */
async function previewImage(file: File): Promise<FileInfo> {
  const url = URL.createObjectURL(file);
  objectUrls.add(url);
  return { name: file.name, url, extname: file.name.split(".").at(-1) ?? "" };
}

/** 释放预览占用的浏览器资源。 */
function clearPreviews() {
  objectUrls.forEach(url => URL.revokeObjectURL(url));
  objectUrls.clear();
  imageUrl.value = avatar;
  images.value = [{ name: "示例头像.png", url: avatar }];
}

onDeactivated(clearPreviews);
onBeforeUnmount(clearPreviews);
</script>

<style scoped lang="scss">
.example-tip {
  margin: 0 0 18px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}
</style>
