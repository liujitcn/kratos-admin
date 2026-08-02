<script setup lang="ts">
import { defAuthService } from '@liujitcn/kratos-uni-app-core/api/system/auth'
import { useUserStore } from '@liujitcn/kratos-uni-app-core/stores'
import type { UserProfileForm } from '@liujitcn/kratos-uni-app-core/rpc/system/app/v1/auth'
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import type { BaseDictForm_DictItem } from '@liujitcn/kratos-uni-app-core/rpc/system/app/v1/base_dict'
import { defBaseDictService } from '@liujitcn/kratos-uni-app-core/api/system/base_dict'
import { formatSrc } from '@liujitcn/kratos-uni-app-core/utils/index'
import { uploadFile } from '@liujitcn/kratos-uni-app-core/utils/file'
import { navigateToLogin } from '@liujitcn/kratos-uni-app-core/utils/navigation'
import defaultAvatar from '@liujitcn/kratos-uni-app-core/static/images/avatar.png'
import navigatorBackground from '@liujitcn/kratos-uni-app-core/static/images/navigator_bg.png'
import { useI18n } from '@liujitcn/kratos-uni-app-core'

const userStore = useUserStore()
const { t } = useI18n()

// 获取屏幕边界到安全区域距离
const { safeAreaInsets } = uni.getSystemInfoSync()

const imgMaxSize = ref(1024 * 1024)

// 获取个人信息，修改个人信息需提供初始值
const userInfo = ref({} as UserProfileForm)
const syncUserStoreProfile = (profile: UserProfileForm) => {
  if (!userStore.userInfo) {
    return
  }

  userStore.userInfo.user_name = profile.user_name
  userStore.userInfo.nick_name = profile.nick_name
  userStore.userInfo.gender = profile.gender
  userStore.userInfo.phone = profile.phone
  userStore.userInfo.avatar = profile.avatar
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
    value: genderCode,
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
          title: t('system.profile.photoLimit'),
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
          title: t('system.profile.photoLimit'),
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
    await defAuthService.UpdateUserProfile(userInfo.value)
    syncUserStoreProfile(userInfo.value)
    await uni.showToast({ icon: 'success', title: t('system.profile.updateSuccess') })
  } catch {
    await uni.showToast({ icon: 'error', title: t('system.profile.avatarUploadFailed') })
  }
}

// 修改性别
const onGenderChange: UniHelper.RadioGroupOnChange = (ev) => {
  userInfo.value.gender = Number(ev.detail.value)
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
  await uni.showToast({ icon: 'success', title: t('system.profile.phoneAuthorizationSuccess') })
}

// #endif

// 点击保存提交表单
const onSubmit = async () => {
  if (!userStore.ensureAuthenticated()) {
    navigateToLogin()
    return
  }

  const { nick_name, gender } = userInfo.value
  await defAuthService.UpdateUserProfile({
    nick_name: nick_name,
    gender: gender,
    avatar: userInfo.value.avatar,
    phone: userInfo.value.phone,
    user_name: userInfo.value.user_name,
  })
  // 更新Store昵称
  syncUserStoreProfile(userInfo.value)
  await uni.showToast({ icon: 'success', title: t('system.profile.saveSuccess') })
  setTimeout(() => {
    uni.navigateBack()
  }, 400)
}
</script>

<template>
  <view class="viewport" :style="{ backgroundImage: `url(${navigatorBackground})` }">
    <!-- 导航栏 -->
    <view class="navbar" :style="{ paddingTop: safeAreaInsets?.top + 'px' }">
      <navigator open-type="navigateBack" class="back icon-left" hover-class="none"></navigator>
      <view class="title">{{ t('system.profile.title') }}</view>
    </view>
    <view class="avatar">
      <view @tap="onAvatarChange" class="avatar-content">
        <image
          v-if="userInfo?.avatar"
          class="image"
          :src="formatSrc(userInfo?.avatar)"
          mode="aspectFill"
        />
        <image v-else class="image" :src="defaultAvatar" mode="aspectFill"></image>
        <text class="text">{{ t('system.profile.avatarChange') }}</text>
      </view>
    </view>
    <!-- 表单 -->
    <view class="form">
      <!-- 表单内容 -->
      <view class="form-content">
        <view class="form-item" v-if="userInfo?.user_name">
          <text class="label">{{ t('system.profile.account') }}</text>
          <text class="account placeholder">{{ userInfo?.user_name }}</text>
        </view>
        <!-- #ifdef MP-WEIXIN -->
        <!-- 手机号 -->
        <view class="form-item">
          <text class="label">{{ t('system.profile.mobile') }}</text>
          <view class="input">
            <text v-if="userInfo.phone" class="account">{{ userInfo.phone }}</text>
            <button
              v-else
              class="auth-button"
              open-type="getPhoneNumber"
              @getphonenumber="onGetPhoneNumber"
            >
              {{ t('system.profile.phoneAuthorization') }}
            </button>
          </view>
        </view>
        <!-- #endif -->
        <view class="form-item">
          <text class="label">{{ t('system.profile.nickName') }}</text>
          <input
            class="input"
            type="text"
            :placeholder="t('system.profile.nickNamePlaceholder')"
            v-model="userInfo.nick_name"
          />
        </view>
        <view class="form-item">
          <text class="label">{{ t('system.profile.gender') }}</text>
          <radio-group @change="onGenderChange">
            <label class="radio" v-for="(item, index) in genderList" :key="index">
              <radio
                :value="item.value"
                color="#27ba9b"
                :checked="userInfo?.gender === Number(item.value)"
              />
              {{ item.label }}
            </label>
          </radio-group>
        </view>
      </view>
      <!-- 提交按钮 -->
      <button @tap="onSubmit" class="form-button">{{ t('common.action.save') }}</button>
    </view>
  </view>
</template>

<style lang="scss">
page {
  background-color: #f4f4f4;
}

.viewport {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-size: auto 420rpx;
  background-repeat: no-repeat;
}

// 导航栏
.navbar {
  position: relative;

  .title {
    height: 40px;
    display: flex;
    justify-content: center;
    align-items: center;
    font-size: 16px;
    font-weight: 500;
    color: #fff;
  }

  .back {
    position: absolute;
    height: 40px;
    width: 40px;
    left: 0;
    font-size: 20px;
    color: #fff;
    display: flex;
    justify-content: center;
    align-items: center;

    &::before {
      content: '‹';
    }
  }
}

// 头像
.avatar {
  text-align: center;
  width: 100%;
  height: 260rpx;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;

  .image {
    width: 160rpx;
    height: 160rpx;
    border-radius: 50%;
    background-color: #eee;
  }

  .text {
    display: block;
    padding-top: 20rpx;
    line-height: 1;
    font-size: 26rpx;
    color: #fff;
  }
}

// 表单
.form {
  background-color: #f4f4f4;

  &-content {
    margin: 20rpx 20rpx 0;
    padding: 0 20rpx;
    border-radius: 10rpx;
    background-color: #fff;
  }

  &-item {
    display: flex;
    height: 96rpx;
    line-height: 46rpx;
    padding: 25rpx 10rpx;
    background-color: #fff;
    font-size: 28rpx;
    border-bottom: 1rpx solid #ddd;

    &:last-child {
      border: none;
    }

    .label {
      width: 180rpx;
      color: #333;
    }

    .account {
      color: #666;
    }

    .input {
      flex: 1;
      display: block;
      height: 46rpx;
    }

    .radio {
      margin-right: 20rpx;
    }

    .picker {
      flex: 1;
    }
    .placeholder {
      color: #808080;
    }
  }

  &-button {
    height: 80rpx;
    text-align: center;
    line-height: 80rpx;
    margin: 30rpx 20rpx;
    color: #fff;
    border-radius: 80rpx;
    font-size: 30rpx;
    background-color: #27ba9b;
  }
}
.auth-button {
  height: 60rpx;
  line-height: 60rpx;
  margin: 0;
  padding: 0 20rpx;
  font-size: 26rpx;
  color: #27ba9b;
  border: 1rpx solid #27ba9b;
  border-radius: 30rpx;
  background: none;

  &::after {
    border: none;
  }
}
</style>
