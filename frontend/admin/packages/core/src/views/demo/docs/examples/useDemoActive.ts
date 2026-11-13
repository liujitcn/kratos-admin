import { onActivated, onDeactivated, ref } from "vue";

/** 缓存示例停用时卸载内部弹层，避免抽屉或弹窗留在其他页面。 */
export function useDemoActive() {
  const active = ref(true);
  onActivated(() => {
    active.value = true;
  });
  onDeactivated(() => {
    active.value = false;
  });
  return active;
}
