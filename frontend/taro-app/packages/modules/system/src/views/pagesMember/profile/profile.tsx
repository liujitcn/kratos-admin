import {
  Button,
  Image,
  Input,
  Label,
  Radio,
  RadioGroup,
  Text,
  View,
} from '@tarojs/components'
import Taro, { useLoad } from '@tarojs/taro'
import { useState } from 'react'
import { defAuthService } from '@liujitcn/kratos-taro-app-core/api/system/auth'
import { defBaseDictService } from '@liujitcn/kratos-taro-app-core/api/system/base_dict'
import type { BaseDictForm_DictItem } from '@liujitcn/kratos-taro-app-core/rpc/system/app/v1/base_dict'
import type { UserProfileForm } from '@liujitcn/kratos-taro-app-core/rpc/system/app/v1/auth'
import {
  formatSrc,
  navigateToLogin,
  resolveBundledAsset,
  uploadFile,
  useUserStore,
} from '@liujitcn/kratos-taro-app-core'
import './profile.scss'

const IMG_MAX_SIZE = 1024 * 1024
const defaultAvatar = resolveBundledAsset('static/images/avatar.png')
const navigatorBackground = resolveBundledAsset('static/images/navigator_bg.png')

/** 用户个人资料编辑页。 */
export default function ProfilePage() {
  const [userInfo, setUserInfo] = useState<UserProfileForm>({
    user_name: '',
    nick_name: '',
    gender: 0,
    phone: '',
    avatar: '',
  })
  const [genderList, setGenderList] = useState<BaseDictForm_DictItem[]>([])
  const ensureAuthenticated = useUserStore((state) => state.ensureAuthenticated)
  const safeTop = Taro.getWindowInfo().safeArea?.top || 0

  const requireAuth = () => {
    if (ensureAuthenticated()) return true
    navigateToLogin()
    return false
  }

  const loadData = async () => {
    const [profile, gender] = await Promise.all([
      useUserStore.getState().getUserProfile(),
      defBaseDictService.GetBaseDict({ value: 'base_user_gender' }),
    ])
    setUserInfo(profile)
    setGenderList(gender.items || [])
  }

  useLoad(() => {
    if (requireAuth()) void loadData()
  })

  const uploadAvatar = async (filePath: string) => {
    if (!requireAuth()) return
    try {
      const fileInfo = await uploadFile('avatar', filePath)
      const nextProfile = { ...userInfo, avatar: fileInfo.url }
      await defAuthService.UpdateUserProfile(nextProfile)
      setUserInfo(nextProfile)
      await useUserStore.getState().getUserProfile()
      await Taro.showToast({ icon: 'success', title: '更新成功' })
    } catch {
      await Taro.showToast({ icon: 'error', title: '上传头像失败' })
    }
  }

  const onAvatarChange = async () => {
    if (!requireAuth()) return
    const selected =
      process.env.TARO_ENV === 'weapp'
        ? await Taro.chooseMedia({ count: 1, mediaType: ['image'] })
        : await Taro.chooseImage({ count: 1 })
    const file = selected.tempFiles[0]
    const filePath = 'tempFilePath' in file ? file.tempFilePath : file.path
    if (file.size > IMG_MAX_SIZE) {
      await Taro.showToast({ title: '请上传小于1M的照片', icon: 'none', duration: 1500 })
      return
    }
    await uploadAvatar(filePath)
  }

  const onGetPhoneNumber = async (event: { detail: { errMsg?: string; code?: string } }) => {
    if (event.detail.errMsg !== 'getPhoneNumber:ok' || !requireAuth()) return
    const response = await defAuthService.BindUserPhone({ code: event.detail.code || '' })
    setUserInfo((current) => ({ ...current, phone: response.phone }))
    await useUserStore.getState().getUserProfile()
    await Taro.showToast({ icon: 'success', title: '授权成功' })
  }

  const onSubmit = async () => {
    if (!requireAuth()) return
    await defAuthService.UpdateUserProfile(userInfo)
    await useUserStore.getState().getUserProfile()
    await Taro.showToast({ icon: 'success', title: '保存成功' })
    setTimeout(() => void Taro.navigateBack(), 400)
  }

  return (
    <View className='profile-viewport' style={{ backgroundImage: `url(${navigatorBackground})` }}>
      <View className='profile-navbar' style={{ paddingTop: `${safeTop}px` }}>
        <View className='profile-back' onClick={() => void Taro.navigateBack()}>‹</View>
        <View className='profile-navbar__title'>个人信息</View>
      </View>
      <View className='profile-avatar'>
        <View className='profile-avatar__content' onClick={() => void onAvatarChange()}>
          <Image className='profile-avatar__image' src={userInfo.avatar ? formatSrc(userInfo.avatar) : defaultAvatar} mode='aspectFill' />
          <Text className='profile-avatar__text'>点击修改头像</Text>
        </View>
      </View>
      <View className='profile-form'>
        <View className='profile-form__content'>
          {userInfo.user_name ? (
            <View className='profile-form__item'>
              <Text className='profile-label'>账号</Text>
              <Text className='profile-account profile-placeholder'>{userInfo.user_name}</Text>
            </View>
          ) : null}
          {process.env.TARO_ENV === 'weapp' ? (
            <View className='profile-form__item'>
              <Text className='profile-label'>手机号</Text>
              <View className='profile-input'>
                {userInfo.phone ? (
                  <Text className='profile-account'>{userInfo.phone}</Text>
                ) : (
                  <Button className='profile-auth-button' openType='getPhoneNumber' onGetPhoneNumber={onGetPhoneNumber}>微信授权手机号</Button>
                )}
              </View>
            </View>
          ) : null}
          <View className='profile-form__item'>
            <Text className='profile-label'>昵称</Text>
            <Input className='profile-input' placeholder='请填写昵称' value={userInfo.nick_name} onInput={(event) => setUserInfo((current) => ({ ...current, nick_name: event.detail.value }))} />
          </View>
          <View className='profile-form__item'>
            <Text className='profile-label'>性别</Text>
            <RadioGroup onChange={(event) => setUserInfo((current) => ({ ...current, gender: Number(event.detail.value) }))}>
              {genderList.map((item) => (
                <Label className='profile-radio' key={item.value}>
                  <Radio value={item.value} color='#27ba9b' checked={userInfo.gender === Number(item.value)} />
                  {item.label}
                </Label>
              ))}
            </RadioGroup>
          </View>
        </View>
        <Button className='profile-form__button' onClick={() => void onSubmit()}>保 存</Button>
      </View>
    </View>
  )
}
