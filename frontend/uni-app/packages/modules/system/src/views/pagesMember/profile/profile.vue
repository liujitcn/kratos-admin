<script setup lang="ts">
import { defAuthService } from '@liujitcn/kratos-uni-app-core/api/system/app/v1/auth'
import { useThemeStore, useUserStore } from '@liujitcn/kratos-uni-app-core/stores'
import type { UserProfileForm } from '@liujitcn/kratos-uni-app-core/rpc/system/app/v1/auth'
import { onLoad } from '@dcloudio/uni-app'
import { computed, onBeforeUnmount, ref } from 'vue'
import type { BaseDictForm_DictItem } from '@liujitcn/kratos-uni-app-core/rpc/system/app/v1/base_dict'
import { BaseUserIDType } from '@liujitcn/kratos-uni-app-core/rpc/system/common/v1/common'
import { defBaseDictService } from '@liujitcn/kratos-uni-app-core/api/system/app/v1/base_dict'
import { formatSrc } from '@liujitcn/kratos-uni-app-core/utils/index'
import { uploadFile } from '@liujitcn/kratos-uni-app-core/utils/file'
import { navigateToLogin } from '@liujitcn/kratos-uni-app-core/utils/navigation'
import defaultAvatar from '@liujitcn/kratos-uni-app-core/static/images/avatar.png'
import { useI18n } from '@liujitcn/kratos-uni-app-core'

const userStore = useUserStore()
const themeStore = useThemeStore()
const { t } = useI18n()

/** 返回上一页。 */
const onBack = () => uni.navigateBack()

const imgMaxSize = ref(1024 * 1024)
const submitting = ref(false)
let navigateBackTimer: ReturnType<typeof setTimeout> | undefined

onBeforeUnmount(() => {
  if (navigateBackTimer) clearTimeout(navigateBackTimer)
  navigateBackTimer = undefined
})

// 获取个人信息，修改个人信息需提供初始值
const userInfo = ref({} as UserProfileForm)
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const idCodePattern = /^[A-Za-z0-9][A-Za-z0-9-]{0,63}$/
const idTypeOptions = computed(() => [
  {
    label: t('system.profile.id_type.unspecified'),
    value: BaseUserIDType.BASE_USER_ID_TYPE_UNSPECIFIED,
  },
  { label: t('system.profile.id_type.id_card'), value: BaseUserIDType.BASE_USER_ID_TYPE_ID_CARD },
  { label: t('system.profile.id_type.passport'), value: BaseUserIDType.BASE_USER_ID_TYPE_PASSPORT },
  {
    label: t('system.profile.id_type.hk_macao_permit'),
    value: BaseUserIDType.BASE_USER_ID_TYPE_HK_MACAO_PERMIT,
  },
  {
    label: t('system.profile.id_type.taiwan_permit'),
    value: BaseUserIDType.BASE_USER_ID_TYPE_TAIWAN_PERMIT,
  },
  {
    label: t('system.profile.id_type.foreign_permanent_residence'),
    value: BaseUserIDType.BASE_USER_ID_TYPE_FOREIGN_PERMANENT_RESIDENCE,
  },
  { label: t('system.profile.id_type.other'), value: BaseUserIDType.BASE_USER_ID_TYPE_OTHER },
])
const idTypeIndex = computed(() => {
  const index = idTypeOptions.value.findIndex((item) => item.value === userInfo.value.id_type)
  return index >= 0 ? index : 0
})
const idTypeLabel = computed(
  () => idTypeOptions.value[idTypeIndex.value]?.label || idTypeOptions.value[0].label,
)
const syncUserStoreProfile = (profile: UserProfileForm) => {
  if (!userStore.userInfo) {
    return
  }

  userStore.userInfo.user_name = profile.user_name
  userStore.userInfo.nick_name = profile.nick_name
  userStore.userInfo.gender = profile.gender
  userStore.userInfo.phone = profile.phone
  userStore.userInfo.avatar = profile.avatar
  userStore.userInfo.email = profile.email
  userStore.userInfo.id_type = profile.id_type
  userStore.userInfo.id_code = profile.id_code
}

const getUserData = async () => {
  const res = await defAuthService.GetUserProfile({})
  userInfo.value = res
  // 同步 Store 的头像和昵称，用于我的页面展示
  syncUserStoreProfile(res)
}

const genderList = ref<BaseDictForm_DictItem[]>([])

const getDictData = async () => {
  const genderCode = 'base_user_gender'
  const res = await defBaseDictService.GetBaseDict({
    code: genderCode,
  })
  genderList.value = res.items || []
}

onLoad(() => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }

  Promise.all([getUserData(), getDictData()])
})
// 修改头像
const onAvatarChange = async () => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }

  // 调用拍照/选择图片
  // 选择图片条件编译
  // #ifdef H5 || APP-PLUS
  // 微信小程序从基础库 2.21.0 开始， wx.chooseImage 停止维护，请使用 uni.chooseMedia 代替
  uni.chooseImage({
    count: 1,
    success: async (res: any) => {
      const { path, size } = res.tempFiles[0]
      if (size > imgMaxSize.value) {
        await uni.showToast({
          title: t('system.profile.photo_limit'),
          icon: 'none',
          duration: 1500,
        })
        return
      }
      // 上传
      await uploadAvatar(path)
    },
  })
  // #endif

  // #ifdef MP-WEIXIN
  // uni.chooseMedia 仅支持微信小程序端
  uni.chooseMedia({
    // 文件个数
    count: 1,
    // 文件类型
    mediaType: ['image'],
    success: async (res: any) => {
      // 本地路径
      const { tempFilePath, size } = res.tempFiles[0]
      if (size > imgMaxSize.value) {
        await uni.showToast({
          title: t('system.profile.photo_limit'),
          icon: 'none',
          duration: 1500,
        })
        return
      }
      await uploadAvatar(tempFilePath)
    },
  })
  // #endif
}

// 上传头像并同步个人资料与用户 Store。
const uploadAvatar = async (file: string) => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }

  try {
    const fileInfo = await uploadFile('avatar', file)
    userInfo.value.avatar = fileInfo.url
    await defAuthService.UpdateUserProfile({ user_profile: userInfo.value })
    syncUserStoreProfile(userInfo.value)
    await uni.showToast({ icon: 'success', title: t('system.profile.update_success') })
  } catch {
    await uni.showToast({ icon: 'error', title: t('system.profile.avatar_upload_failed') })
  }
}

// 修改性别
const onGenderChange: UniHelper.RadioGroupOnChange = (ev) => {
  userInfo.value.gender = Number(ev.detail.value)
}

// 选择证件类型。
const onIdTypeChange: UniHelper.SelectorPickerOnChange = (ev) => {
  userInfo.value.id_type =
    idTypeOptions.value[Number(ev.detail.value)]?.value ??
    BaseUserIDType.BASE_USER_ID_TYPE_UNSPECIFIED
}

// #ifdef MP-WEIXIN
// 新增授权手机号处理
const onGetPhoneNumber: UniHelper.ButtonOnGetphonenumber = async (e) => {
  if (e.detail.errMsg !== 'getPhoneNumber:ok') return
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }

  const res = await defAuthService.BindUserPhone({ code: e.detail.code || '' })
  userInfo.value.phone = res.phone
  syncUserStoreProfile(userInfo.value)
  await uni.showToast({ icon: 'success', title: t('system.profile.phone_authorization_success') })
}

// #endif

// 点击保存提交表单
const onSubmit = async () => {
  if (submitting.value) return
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }

  if (userInfo.value.email && !emailPattern.test(userInfo.value.email)) {
    await uni.showToast({ icon: 'none', title: t('system.profile.email_invalid') })
    return
  }
  if (userInfo.value.id_code && !idCodePattern.test(userInfo.value.id_code)) {
    await uni.showToast({ icon: 'none', title: t('system.profile.id_code_invalid') })
    return
  }

  submitting.value = true
  try {
    const { nick_name, gender, email, id_type, id_code } = userInfo.value
    await defAuthService.UpdateUserProfile({
      user_profile: {
        nick_name: nick_name,
        gender: gender,
        avatar: userInfo.value.avatar,
        phone: userInfo.value.phone,
        user_name: userInfo.value.user_name,
        email,
        id_type,
        id_code,
      },
    })
    // 更新Store昵称
    syncUserStoreProfile(userInfo.value)
    await uni.showToast({ icon: 'success', title: t('system.profile.save_success') })
    if (navigateBackTimer) clearTimeout(navigateBackTimer)
    navigateBackTimer = setTimeout(() => {
      navigateBackTimer = undefined
      uni.navigateBack()
    }, 400)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <view class="profile-viewport" :class="themeStore.themeClass">
    <view class="profile-content">
      <!-- 导航栏：统一子页头（单一返回 + 标题） -->
      <view class="k-page-nav">
        <view class="k-page-nav__back" :aria-label="t('common.action.back')" @tap="onBack"></view>
        <view class="k-page-nav__title">{{ t('system.profile.title') }}</view>
        <view class="k-page-nav__side" aria-hidden="true"></view>
      </view>

      <!-- 头像卡：点击更换头像 -->
      <view class="k-glass k-glass--lit k-pfx profile-id" @tap="onAvatarChange">
        <view class="profile-id__main">
          <view class="k-pfx__av">
            <image
              class="profile-av-img"
              :src="userInfo?.avatar ? formatSrc(userInfo?.avatar) : defaultAvatar"
              mode="aspectFill"
            />
          </view>
          <view class="profile-id__meta">
            <view class="k-pfx__nm">{{ userInfo?.nick_name || t('system.profile.title') }}</view>
            <view class="k-pfx__ac">{{ userInfo?.user_name }}</view>
          </view>
        </view>
        <view class="k-pfx__ent">{{ t('system.profile.avatar_change') }}</view>
      </view>

      <!-- 表单 -->
      <view class="k-glass profile-form">
        <view class="k-phd">
          <view class="k-phd__title">{{ t('system.profile.title') }}</view>
        </view>

        <view class="k-field" v-if="userInfo?.user_name">
          <text class="k-field__k">{{ t('system.profile.account') }}</text>
          <text class="k-field__v">{{ userInfo?.user_name }}</text>
        </view>
        <!-- #ifdef MP-WEIXIN -->
        <!-- 手机号 -->
        <view class="k-field">
          <text class="k-field__k">{{ t('system.profile.mobile') }}</text>
          <text v-if="userInfo.phone" class="k-field__v">{{ userInfo.phone }}</text>
          <button
            v-else
            class="profile-auth-button"
            open-type="getPhoneNumber"
            @getphonenumber="onGetPhoneNumber"
          >
            {{ t('system.profile.phone_authorization') }}
          </button>
        </view>
        <!-- #endif -->
        <view class="k-field">
          <text class="k-field__k">{{ t('system.profile.nick_name') }}</text>
          <input
            class="k-field__v profile-input"
            type="text"
            :placeholder="t('system.profile.nick_name_placeholder')"
            v-model="userInfo.nick_name"
          />
        </view>
        <view class="k-field">
          <text class="k-field__k">{{ t('system.profile.email') }}</text>
          <input
            class="k-field__v profile-input"
            type="text"
            :placeholder="t('system.profile.email_placeholder')"
            v-model="userInfo.email"
          />
        </view>
        <picker
          mode="selector"
          :range="idTypeOptions"
          range-key="label"
          :value="idTypeIndex"
          @change="onIdTypeChange"
        >
          <view class="k-field profile-field--sub">
            <text class="k-field__k">{{ t('system.profile.id_type') }}</text>
            <view class="k-field__v">{{ idTypeLabel }}</view>
          </view>
        </picker>
        <view class="k-field">
          <text class="k-field__k">{{ t('system.profile.id_code') }}</text>
          <input
            class="k-field__v profile-input"
            type="text"
            :placeholder="t('system.profile.id_code_placeholder')"
            v-model="userInfo.id_code"
          />
        </view>
        <view class="k-field">
          <text class="k-field__k">{{ t('system.profile.gender') }}</text>
          <radio-group class="profile-radios" @change="onGenderChange">
            <label class="profile-radio" v-for="(item, index) in genderList" :key="index">
              <radio
                :value="item.value"
                :color="themeStore.theme === 'light' ? '#039e8c' : '#28f0d4'"
                :checked="userInfo?.gender === Number(item.value)"
              />
              {{ item.label }}
            </label>
          </radio-group>
        </view>

        <!-- 提交按钮 -->
        <button @tap="onSubmit" class="k-save">{{ t('common.action.save') }}</button>
      </view>
    </view>
  </view>
</template>

<style lang="scss">
.profile-viewport {
  min-height: 100vh;
  box-sizing: border-box;
  background: var(--kratos-bg-gradient);
}

/* k-page-nav 自带安全区顶部内边距，这里只补水平与底部 */
.profile-content {
  padding: 0 32rpx calc(40rpx + env(safe-area-inset-bottom));
}

/* 头像卡 */
.profile-id {
  margin-top: 8rpx;
  padding: 36rpx 30rpx;
}

.profile-id__main {
  display: flex;
  align-items: center;
  gap: 24rpx;
  min-width: 0;
}

.profile-id__meta {
  min-width: 0;
}

.profile-av-img {
  width: 100%;
  height: 100%;
  border-radius: 22rpx;
}

/* 表单卡 */
.profile-form {
  margin-top: 24rpx;
  padding: 10rpx 28rpx 28rpx;
}

/* 子项字段包在 picker 中会命中 :first-child，恢复分隔线 */
.profile-field--sub {
  border-top: 1rpx solid var(--kratos-color-border);
}

.profile-input {
  flex: 1;
  height: 46rpx;
}

.profile-radios {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 24rpx;
}

.profile-radio {
  display: flex;
  align-items: center;
  color: var(--kratos-color-text);
  font-size: 25rpx;
}

.profile-auth-button {
  height: 60rpx;
  margin: 0;
  padding: 0 24rpx;
  color: var(--kratos-color-primary);
  font-size: 24rpx;
  line-height: 58rpx;
  border: 1rpx solid var(--kratos-color-border-accent);
  border-radius: 999rpx;
  background: var(--kratos-color-primary-soft);

  &::after {
    border: none;
  }
}
</style>
