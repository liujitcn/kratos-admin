/**
 * v-copy
 * 复制某个值至剪贴板
 * 接收参数：string类型/Ref<string>类型/Reactive<string>类型
 */

import type { Directive, DirectiveBinding } from "vue";
import { ElMessage } from "element-plus";
/** 复制指令绑定后挂载复制内容的元素类型。 */
interface ElType extends HTMLElement {
  copyData: string | number;
}
const copy: Directive = {
  mounted(el: ElType, binding: DirectiveBinding) {
    el.copyData = binding.value;
    el.addEventListener("click", handleClick);
  },
  updated(el: ElType, binding: DirectiveBinding) {
    el.copyData = binding.value;
  },
  beforeUnmount(el: ElType) {
    el.removeEventListener("click", handleClick);
  }
};

/** 处理复制指令点击事件并写入剪贴板。 */
async function handleClick(this: any) {
  try {
    await navigator.clipboard.writeText(this.copyData);
    ElMessage({
      type: "success",
      message: "复制成功"
    });
  } catch (err) {
    console.error("复制操作不被支持或失败: ", err);
  }
}

export default copy;
