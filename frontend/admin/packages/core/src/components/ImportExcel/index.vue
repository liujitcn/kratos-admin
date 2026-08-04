<template>
  <el-dialog
    v-model="dialogVisible"
    :title="t('core.upload.batch_add', { resource: parameter.title })"
    :destroy-on-close="true"
    width="580px"
    draggable
  >
    <el-form class="drawer-multiColumn-form" label-width="100px">
      <el-form-item :label="t('core.upload.template_download')">
        <el-button type="primary" :icon="Download" @click="downloadTemp">{{ t("common.action.download") }}</el-button>
      </el-form-item>
      <el-form-item :label="t('core.upload.file')">
        <el-upload
          action="#"
          class="upload"
          :drag="true"
          :limit="excelLimit"
          :multiple="true"
          :show-file-list="true"
          :http-request="uploadExcel"
          :before-upload="beforeExcelUpload"
          :on-exceed="handleExceed"
          :on-success="excelUploadSuccess"
          :on-error="excelUploadError"
          :accept="parameter.fileType!.join(',')"
        >
          <slot name="empty">
            <el-icon class="el-icon--upload">
              <upload-filled />
            </el-icon>
            <div class="el-upload__text">
              {{ t("core.upload.drag_or_click") }}<em>{{ t("core.upload.click") }}</em>
            </div>
          </slot>
          <template #tip>
            <slot name="tip">
              <div class="el-upload__tip">{{ t("core.upload.excel_format", { size: parameter.fileSize ?? 5 }) }}</div>
            </slot>
          </template>
        </el-upload>
      </el-form-item>
      <el-form-item :label="t('core.upload.data_cover')">
        <el-switch v-model="isCover" />
      </el-form-item>
    </el-form>
  </el-dialog>
</template>

<script setup lang="ts" name="ImportExcel">
import { ref } from "vue";
import { useDownload } from "@/hooks/useDownload";
import { Download } from "@element-plus/icons-vue";
import { ElNotification, UploadRequestOptions, UploadRawFile } from "element-plus";
import { useLocaleStore } from "@/locales";

const { t } = useLocaleStore();

/** Excel 导入弹窗参数。 */
export interface ExcelParameterProps {
  title: string; // 标题
  fileSize?: number; // 上传文件的大小
  fileType?: File.ExcelMimeType[]; // 上传文件的类型
  tempApi?: (params: any) => Promise<any>; // 下载模板的Api
  importApi?: (params: any) => Promise<any>; // 批量导入的Api
  getTableList?: () => void; // 获取表格数据的Api
}

// 是否覆盖数据
const isCover = ref(false);
// 最大文件上传数
const excelLimit = ref(1);
// dialog状态
const dialogVisible = ref(false);
// 父组件传过来的参数
const parameter = ref<ExcelParameterProps>({
  title: "",
  fileSize: 5,
  fileType: ["application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"]
});

// 接收父组件参数
const acceptParams = (params: ExcelParameterProps) => {
  parameter.value = { ...parameter.value, ...params };
  dialogVisible.value = true;
};

// Excel 导入模板下载
const downloadTemp = () => {
  if (!parameter.value.tempApi) return;
  useDownload(parameter.value.tempApi, t("core.upload.template", { resource: parameter.value.title }));
};

// 文件上传
const uploadExcel = async (param: UploadRequestOptions) => {
  let excelFormData = new FormData();
  excelFormData.append("file", param.file);
  excelFormData.append("isCover", isCover.value as unknown as Blob);
  await parameter.value.importApi!(excelFormData);
  parameter.value.getTableList && parameter.value.getTableList();
  dialogVisible.value = false;
};

/**
 * @description 文件上传之前判断
 * @param file 上传的文件
 * */
const beforeExcelUpload = (file: UploadRawFile) => {
  const isExcel = parameter.value.fileType!.includes(file.type as File.ExcelMimeType);
  const fileSize = file.size / 1024 / 1024 < parameter.value.fileSize!;
  if (!isExcel)
    ElNotification({
      title: t("common.title.warning"),
      message: t("core.upload.file_format_invalid"),
      type: "warning"
    });
  if (!fileSize)
    setTimeout(() => {
      ElNotification({
        title: t("common.title.warning"),
        message: t("core.upload.file_size_exceeded", { size: parameter.value.fileSize ?? 5 }),
        type: "warning"
      });
    }, 0);
  return isExcel && fileSize;
};

// 文件数超出提示
const handleExceed = () => {
  ElNotification({
    title: t("common.title.warning"),
    message: t("core.upload.single_file_only"),
    type: "warning"
  });
};

// 上传错误提示
const excelUploadError = () => {
  ElNotification({
    title: t("common.title.warning"),
    message: t("core.upload.batch_failed", { resource: parameter.value.title }),
    type: "error"
  });
};

// 上传成功提示
const excelUploadSuccess = () => {
  ElNotification({
    title: t("common.title.notice"),
    message: t("core.upload.batch_success", { resource: parameter.value.title }),
    type: "success"
  });
};

defineExpose({
  acceptParams
});
</script>
<style lang="scss" scoped>
@use "./index.scss" as *;
</style>
