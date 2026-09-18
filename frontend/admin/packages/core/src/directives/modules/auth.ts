/**
 * v-hasPerm
 * 按钮权限指令
 */
import { useAuthStore } from "@/stores/modules/auth";
import type { Directive, DirectiveBinding } from "vue";

const GLOBAL_AUTH_BUTTON_KEY = "__global__";

const originalDisplayMap = new WeakMap<HTMLElement, string>();

/** 根据当前按钮权限更新元素可见性。 */
function updateAuthElement(el: HTMLElement, binding: DirectiveBinding) {
  const { value } = binding;
  const authStore = useAuthStore();
  const currentPageRoles =
    authStore.authButtonListGet[authStore.routeName] ?? authStore.authButtonListGet[GLOBAL_AUTH_BUTTON_KEY] ?? [];
  const hasPermission =
    value instanceof Array && value.length
      ? value.every(item => currentPageRoles.includes(item))
      : currentPageRoles.includes(value);
  if (!originalDisplayMap.has(el)) originalDisplayMap.set(el, el.style.display);
  el.style.display = hasPermission ? (originalDisplayMap.get(el) ?? "") : "none";
}

const auth: Directive = {
  mounted: updateAuthElement,
  updated: updateAuthElement,
  unmounted(el: HTMLElement) {
    originalDisplayMap.delete(el);
  }
};

export default auth;
