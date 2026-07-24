<template>
  <div class="upload-file-box">
    <el-upload
      action="#"
      :class="['upload-file', selfDisabled ? 'disabled' : '']"
      :multiple="false"
      :disabled="selfDisabled"
      :show-file-list="false"
      :http-request="handleHttpUpload"
      :before-upload="beforeUpload"
      :on-success="uploadSuccess"
      :on-error="uploadError"
      :accept="fileType.join(',')"
    >
      <el-button type="primary" :disabled="selfDisabled">上传文件</el-button>
    </el-upload>

    <div v-if="fileInfo?.url" class="file-card">
      <div class="file-card__main">
        <el-icon class="file-card__icon"><Document /></el-icon>
        <div class="file-card__meta">
          <div class="file-card__name">{{ fileInfo.name || "未命名文件" }}</div>
          <div class="file-card__url">{{ fileInfo.url }}</div>
        </div>
      </div>
      <div class="file-card__action">
        <el-button link type="primary" :icon="Download" @click.stop="handleDownload">下载</el-button>
        <el-button v-if="!selfDisabled" link type="danger" :icon="Delete" @click.stop="handleDelete">删除</el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts" name="UploadFile">
import { computed, inject } from "vue";
import { Delete, Document, Download } from "@element-plus/icons-vue";
import { ElNotification, formContextKey, formItemContextKey } from "element-plus";
import type { UploadProps, UploadRequestOptions, UploadUserFile } from "element-plus";
import { defFileService } from "@/api/base/file";
import type { FileInfo } from "@/rpc/base/v1/file";

/** 单文件上传组件属性。 */
interface UploadFileProps {
  fileInfo?: UploadUserFile;
  api?: (file: File) => Promise<FileInfo>;
  disabled?: boolean;
  fileSize?: number;
  fileType?: string[];
  uploadType?: string;
}

const props = withDefaults(defineProps<UploadFileProps>(), {
  fileInfo: undefined,
  disabled: false,
  fileSize: 20,
  fileType: () => [],
  uploadType: "file"
});

const emit = defineEmits<{
  "update:fileInfo": [value: UploadUserFile | undefined];
}>();

const formContext = inject(formContextKey, void 0);
const formItemContext = inject(formItemContextKey, void 0);
/** Element Plus 上传错误回调要求的错误对象类型。 */
type UploadRequestError = Parameters<NonNullable<UploadRequestOptions["onError"]>>[0];

/** 兼容 Element Plus 上传组件要求的错误对象结构。 */
function buildUploadError(error: unknown): UploadRequestError {
  const uploadError = error instanceof Error ? error : new Error("文件上传失败");
  return Object.assign(uploadError, {
    status: 500,
    method: "POST",
    url: "#"
  }) as UploadRequestError;
}

/** 计算当前组件是否禁用。 */
const selfDisabled = computed(() => {
  return props.disabled || formContext?.disabled;
});

/** 上传前校验文件大小和格式。 */
const beforeUpload: UploadProps["beforeUpload"] = rawFile => {
  const fileSizeValid = rawFile.size / 1024 / 1024 < props.fileSize;
  const fileTypeValid = !props.fileType.length || props.fileType.includes(rawFile.type);

  if (!fileTypeValid) {
    ElNotification({
      title: "温馨提示",
      message: "上传文件不符合所需的格式！",
      type: "warning"
    });
  }

  if (!fileSizeValid) {
    ElNotification({
      title: "温馨提示",
      message: `上传文件大小不能超过 ${props.fileSize}M！`,
      type: "warning"
    });
  }

  return fileSizeValid && fileTypeValid;
};

/** 执行自定义文件上传。 */
const handleHttpUpload = async (options: UploadRequestOptions) => {
  try {
    const api = props.api ?? (file => defFileService.UploadFile(file, props.uploadType));
    const data = await api(options.file);
    options.onSuccess(data);
  } catch (error) {
    options.onError(buildUploadError(error));
  }
};

/** 处理单文件上传成功后的状态同步。 */
const uploadSuccess = (response: FileInfo | undefined) => {
  if (!response) return;
  emit("update:fileInfo", {
    name: response.name,
    url: response.url
  });
  formItemContext?.prop && formContext?.validateField([formItemContext.prop as string]);
  ElNotification({
    title: "温馨提示",
    message: "文件上传成功！",
    type: "success"
  });
};

/** 处理文件上传失败提示。 */
const uploadError = () => {
  ElNotification({
    title: "温馨提示",
    message: "文件上传失败，请您重新上传！",
    type: "error"
  });
};

/** 删除当前已上传文件。 */
function handleDelete() {
  emit("update:fileInfo", undefined);
  formItemContext?.prop && formContext?.validateField([formItemContext.prop as string]);
}

/** 下载当前已上传文件。 */
async function handleDownload() {
  if (!props.fileInfo?.url) return;
  await defFileService.DownloadFile(props.fileInfo.url, props.fileInfo.name ?? "download");
}
</script>

<style scoped lang="scss">
.upload-file-box {
  width: 100%;
}
.file-card {
  display: flex;
  gap: 16px;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  margin-top: 12px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
}
.file-card__main {
  display: flex;
  flex: 1;
  gap: 12px;
  min-width: 0;
}
.file-card__icon {
  margin-top: 2px;
  font-size: 18px;
  color: var(--el-color-primary);
}
.file-card__meta {
  min-width: 0;
}
.file-card__name {
  font-size: 14px;
  color: var(--el-text-color-primary);
}
.file-card__url {
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.file-card__action {
  display: flex;
  flex-shrink: 0;
  gap: 8px;
}
</style>
