import { computed } from "vue";
import { useRoute } from "vue-router";
import { useAuthStore } from "@/stores/modules/auth";

const GLOBAL_AUTH_BUTTON_KEY = "__global__";

/**
 * @description 页面按钮权限
 * */
export const useAuthButtons = () => {
  const route = useRoute();
  const authStore = useAuthStore();
  const authButtons =
    authStore.authButtonListGet[route.name as string] || authStore.authButtonListGet[GLOBAL_AUTH_BUTTON_KEY] || [];

  const BUTTONS = computed(() => {
    let currentPageAuthButton: { [key: string]: boolean } = {};
    authButtons.forEach(item => (currentPageAuthButton[item] = true));
    return currentPageAuthButton;
  });

  return {
    BUTTONS
  };
};
