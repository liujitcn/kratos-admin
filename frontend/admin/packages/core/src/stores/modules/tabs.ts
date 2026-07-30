import router from "@/routers";
import { defineStore } from "pinia";
import { getUrlWithParams } from "@/utils";
import { useKeepAliveStore } from "./keepAlive";
import { TabsState, TabsMenuProps } from "@/stores/interface";
import piniaPersistConfig from "@/stores/helper/persist";

export const useTabsStore = defineStore("admin-tabs", {
  state: (): TabsState => ({
    tabsMenuList: []
  }),
  actions: {
    /** 添加页签并同步 KeepAlive。 */
    async addTabs(tabItem: TabsMenuProps) {
      const keepAliveStore = useKeepAliveStore();
      if (this.tabsMenuList.every(item => item.path !== tabItem.path)) {
        this.tabsMenuList.push(tabItem);
      }
      // add keepalive
      if (!keepAliveStore.keepAliveName.includes(tabItem.name) && tabItem.isKeepAlive) {
        keepAliveStore.addKeepAliveName(tabItem.path);
      }
    },
    /** 删除页签并同步 KeepAlive。 */
    async removeTabs(tabPath: string, isCurrent: boolean = true) {
      const keepAliveStore = useKeepAliveStore();
      if (isCurrent) {
        this.tabsMenuList.forEach((item, index) => {
          if (item.path !== tabPath) return;
          const nextTab = this.tabsMenuList[index + 1] || this.tabsMenuList[index - 1];
          if (!nextTab) return;
          router.push(nextTab.path);
        });
      }
      // remove keepalive
      const tabItem = this.tabsMenuList.find(item => item.path === tabPath);
      tabItem?.isKeepAlive && keepAliveStore.removeKeepAliveName(tabItem.path);
      // set tabs
      this.tabsMenuList = this.tabsMenuList.filter(item => item.path !== tabPath);
    },
    /** 关闭指定页签一侧的其他页签。 */
    async closeTabsOnSide(path: string, type: "left" | "right") {
      const keepAliveStore = useKeepAliveStore();
      const currentIndex = this.tabsMenuList.findIndex(item => item.path === path);
      if (currentIndex !== -1) {
        const range = type === "left" ? [0, currentIndex] : [currentIndex + 1, this.tabsMenuList.length];
        this.tabsMenuList = this.tabsMenuList.filter((item, index) => {
          return index < range[0] || index >= range[1] || !item.close;
        });
      }
      // set keepalive
      const KeepAliveList = this.tabsMenuList.filter(item => item.isKeepAlive);
      keepAliveStore.setKeepAliveName(KeepAliveList.map(item => item.path));
    },
    /** 关闭除指定页签和固定页签外的其他页签。 */
    async closeMultipleTab(tabsMenuValue?: string) {
      const keepAliveStore = useKeepAliveStore();
      this.tabsMenuList = this.tabsMenuList.filter(item => {
        return item.path === tabsMenuValue || !item.close;
      });
      // set keepalive
      const KeepAliveList = this.tabsMenuList.filter(item => item.isKeepAlive);
      keepAliveStore.setKeepAliveName(KeepAliveList.map(item => item.path));
    },
    // Set Tabs
    async setTabs(tabsMenuList: TabsMenuProps[]) {
      this.tabsMenuList = tabsMenuList;
    },
    // Set Tabs Title
    async setTabsTitle(title: string) {
      this.tabsMenuList.forEach(item => {
        if (item.path == getUrlWithParams()) item.title = title;
      });
    }
  },
  persist: piniaPersistConfig("admin-tabs")
});
