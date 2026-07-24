import { defineStore } from "pinia";
import pinia from "@/stores";
import { defBaseDictService } from "@/api/system/base_dict";
import type { DictState } from "@/stores/interface";
import piniaPersistConfig from "@/stores/helper/persist";
import type { OptionBaseDictResponse_BaseDict, OptionBaseDictResponse_BaseDictItem } from "@/rpc/system/admin/v1/base_dict";

export const useDictStore = defineStore("admin-dict", {
  state: (): DictState => ({
    dictionary: {}
  }),
  getters: {},
  actions: {
    /** 设置单个字典缓存 */
    setDictionary(dict: OptionBaseDictResponse_BaseDict) {
      if (!dict.code) return;
      this.dictionary[dict.code] = dict.items ?? [];
    },
    /** 从服务端加载全部字典缓存 */
    async loadDictionaries(forceRefresh = false) {
      if (!forceRefresh && Object.keys(this.dictionary).length) return this.dictionary;

      const dictRes = await defBaseDictService.OptionBaseDict({});
      const nextDictionary: Record<string, OptionBaseDictResponse_BaseDictItem[]> = {};

      const baseDicts = dictRes.base_dicts ?? [];
      baseDicts.forEach(dict => {
        if (!dict.code) return;
        nextDictionary[dict.code] = dict.items ?? [];
      });

      this.dictionary = nextDictionary;
      return this.dictionary;
    },
    /** 获取指定字典缓存 */
    getDictionary(dictCode: string): OptionBaseDictResponse_BaseDictItem[] {
      return this.dictionary[dictCode] ?? [];
    },
    /** 确保指定字典编码已加载，避免持久化旧缓存缺少新增字典时下拉为空 */
    async ensureDictionary(dictCode: string) {
      const cachedDict = this.getDictionary(dictCode);
      if (cachedDict.length) return cachedDict;

      await this.loadDictionaries(true);
      return this.getDictionary(dictCode);
    },
    /** 清空字典缓存 */
    clearDictionaryCache() {
      this.dictionary = {};
    },
    /** 强制刷新字典缓存 */
    async updateDictionaryCache() {
      this.clearDictionaryCache();
      await this.loadDictionaries(true);
    }
  },
  persist: piniaPersistConfig("admin-dict")
});

/**
 * 在非 setup 场景使用 Dict Store。
 */
export function useDictStoreHook() {
  return useDictStore(pinia);
}
