// ? 系统全局字典
import { t } from "@/locales";

/**
 * @description：用户性别
 */
export const genderType = [
  {
    get label() {
      return t("common.value.male");
    },
    value: 1
  },
  {
    get label() {
      return t("common.value.female");
    },
    value: 2
  }
];

/**
 * @description：用户状态
 */
export const userStatus = [
  {
    get label() {
      return t("common.status.enabled");
    },
    value: 1,
    tagType: "success"
  },
  {
    get label() {
      return t("common.status.disabled");
    },
    value: 0,
    tagType: "danger"
  }
];
