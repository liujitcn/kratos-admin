<template>
  <div class="profile-page">
    <section class="profile-shell">
      <aside class="profile-nav">
        <button
          v-for="tab in profileTabs"
          :key="tab.value"
          type="button"
          class="nav-item"
          :class="{ 'nav-item--active': activeTab === tab.value }"
          @click="handleTabChange(tab.value)"
        >
          <div class="nav-item__content">
            <strong>{{ tab.label }}</strong>
            <span>{{ tab.description }}</span>
          </div>
          <el-icon><ArrowRight /></el-icon>
        </button>
      </aside>

      <main class="profile-content">
        <ProfileBase v-if="activeTab === 'account'" :profile="userProfileForm" @refreshed="loadUserProfile" />
        <ProfileSecurity
          v-else-if="activeTab === 'security'"
          :profile="userProfileForm"
          @refreshed="loadUserProfile"
          @switch-tab="handleTabChange"
        />
        <ProfileSession v-else-if="activeTab === 'session'" />
        <ProfilePassword v-else />
      </main>
    </section>
  </div>
</template>

<script setup lang="ts">
defineOptions({
  name: "Profile",
  inheritAttrs: false
});

import { computed, onMounted, reactive, ref } from "vue";
import { useRoute } from "vue-router";
import { t } from "@liujitcn/kratos-admin-core";
import { defProfileAuthService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/auth";
import type { UserProfileForm } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/auth";
import { useUserStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import ProfileBase from "./components/base.vue";
import ProfileSecurity from "./components/security.vue";
import ProfilePassword from "./components/password.vue";
import ProfileSession from "./components/session.vue";

/** 个人中心标签页。 */
type ProfileTab = "account" | "security" | "password" | "session";

/** 左侧导航项结构。 */
interface ProfileTabOption {
  /** 标签值。 */
  value: ProfileTab;
  /** 导航标题。 */
  label: string;
  /** 导航描述。 */
  description: string;
}

const userStore = useUserStore();
const route = useRoute();
const activeTab = ref<ProfileTab>("account");
const profileTabs = computed<ProfileTabOption[]>(() => {
  const tabs: ProfileTabOption[] = [
    {
      value: "account",
      label: t("system.profile.account.title"),
      description: t("system.profile.account.nav_description")
    },
    {
      value: "security",
      label: t("system.profile.security.title"),
      description: t("system.profile.security.nav_description")
    },
    {
      value: "password",
      label: t("system.profile.password.title"),
      description: t("system.profile.password.nav_description")
    }
  ];
  if (userStore.userInfo.role_code === "super") {
    tabs.push({
      value: "session",
      label: t("system.profile.session.title"),
      description: t("system.profile.session.nav_description")
    });
  }
  return tabs;
});
const userProfileForm = reactive<UserProfileForm>({
  user_name: "",
  nick_name: "",
  avatar: "",
  gender: 3,
  phone: "",
  email: "",
  id_type: 0,
  id_code: "",
  role_name: "",
  dept_name: "",
  created_at: ""
});

/** 切换当前显示标签，仅更新本地视图状态，不触发路由变化。 */
function handleTabChange(tab: ProfileTab) {
  activeTab.value = tab;
}

/** 拉取当前登录用户的个人中心资料。 */
async function loadUserProfile() {
  const profile = await defProfileAuthService.GetUserProfile({});
  Object.assign(userProfileForm, profile);
  // 个人中心资料更新后，同步刷新头部头像和昵称展示，避免页面内外信息不一致。
  userStore.setUserInfo({
    ...userStore.userInfo,
    user_name: profile.user_name,
    nick_name: profile.nick_name,
    phone: profile.phone,
    email: profile.email,
    id_type: profile.id_type,
    id_code: profile.id_code,
    avatar: profile.avatar,
    role_name: profile.role_name,
    dept_name: profile.dept_name
  });
}

onMounted(async () => {
  if (route.query.tab === "password") {
    activeTab.value = "password";
  } else if (route.query.oauth_bind_success || route.query.oauth_bind_error) {
    // OAuth 绑定回跳时优先打开安全设置，让子组件消费绑定结果并刷新状态。
    activeTab.value = "security";
  }
  await loadUserProfile();
});
</script>

<style scoped lang="scss">
.profile-page {
  min-width: 0;
}
.profile-shell {
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  gap: 10px;
  align-items: start;
}
.profile-nav {
  position: sticky;
  top: 10px;
  padding: 16px;
  background: #ffffff;
  border: 1px solid #ebeef5;
  border-radius: var(--admin-page-radius);
  box-shadow: 0 2px 12px rgb(0 0 0 / 4%);
}
.nav-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 14px 12px;
  color: #606266;
  text-align: left;
  cursor: pointer;
  background: #ffffff;
  border: 1px solid transparent;
  border-radius: var(--admin-page-radius);
  transition: all 0.2s ease;
}
.nav-item + .nav-item {
  margin-top: 8px;
}
.nav-item__content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.nav-item strong {
  display: block;
  font-size: 14px;
  font-weight: 600;
}
.nav-item span {
  font-size: 12px;
  color: #909399;
}
.nav-item:hover,
.nav-item--active {
  color: #409eff;
  background: #f5f9ff;
  border-color: #d9ecff;
}
.profile-content {
  min-width: 0;
}

@media screen and (width <= 1080px) {
  .profile-shell {
    grid-template-columns: 1fr;
  }
  .profile-nav {
    position: static;
  }
}

</style>
