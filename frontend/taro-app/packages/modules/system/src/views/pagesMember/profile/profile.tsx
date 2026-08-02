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
import { useEffect, useState } from 'react'
import defaultAvatar from '@liujitcn/kratos-taro-app-core/static/images/avatar.png'
import navigatorBackground from '@liujitcn/kratos-taro-app-core/static/images/navigator_bg.png'
import { defAuthService } from '@liujitcn/kratos-taro-app-core/api/system/auth'
import { defBaseDictService } from '@liujitcn/kratos-taro-app-core/api/system/base_dict'
import type { BaseDictForm_DictItem } from '@liujitcn/kratos-taro-app-core/rpc/system/app/v1/base_dict'
import type { UserProfileForm } from '@liujitcn/kratos-taro-app-core/rpc/system/app/v1/auth'
import {
  formatSrc,
  navigateToLogin,
  uploadFile,
  useI18n,
  useUserStore,
} from '@liujitcn/kratos-taro-app-core'
import './profile.scss'

const IMG_MAX_SIZE = 1024 * 1024

/** 用户个人资料编辑页。 */
export default function ProfilePage() {
  const { locale, t } = useI18n()
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

  useEffect(() => {
    void Taro.setNavigationBarTitle({ title: t('system.profile.title') })
  }, [locale, t])

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
      await Taro.showToast({ icon: 'success', title: t('system.profile.updateSuccess') })
    } catch {
      await Taro.showToast({ icon: 'error', title: t('system.profile.avatarUploadFailed') })
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
      await Taro.showToast({ title: t('system.profile.photoLimit'), icon: 'none', duration: 1500 })
      return
    }
    await uploadAvatar(filePath)
  }

  const onGetPhoneNumber = async (event: { detail: { errMsg?: string; code?: string } }) => {
    if (event.detail.errMsg !== 'getPhoneNumber:ok' || !requireAuth()) return
    const response = await defAuthService.BindUserPhone({ code: event.detail.code || '' })
    setUserInfo((current) => ({ ...current, phone: response.phone }))
    await useUserStore.getState().getUserProfile()
    await Taro.showToast({ icon: 'success', title: t('system.profile.phoneAuthorizationSuccess') })
  }

  const onSubmit = async () => {
    if (!requireAuth()) return
    await defAuthService.UpdateUserProfile(userInfo)
    await useUserStore.getState().getUserProfile()
    await Taro.showToast({ icon: 'success', title: t('system.profile.saveSuccess') })
    setTimeout(() => void Taro.navigateBack(), 400)
  }

  return (
    <View className='profile-viewport' style={{ backgroundImage: `url(${navigatorBackground})` }}>
      <View className='profile-navbar' style={{ paddingTop: `${safeTop}px` }}>
        <View className='profile-back' onClick={() => void Taro.navigateBack()}>‹</View>
        <View className='profile-navbar__title'>{t('system.profile.title')}</View>
      </View>
      <View className='profile-avatar'>
        <View className='profile-avatar__content' onClick={() => void onAvatarChange()}>
          <Image className='profile-avatar__image' src={userInfo.avatar ? formatSrc(userInfo.avatar) : defaultAvatar} mode='aspectFill' />
          <Text className='profile-avatar__text'>{t('system.profile.avatarChange')}</Text>
        </View>
      </View>
      <View className='profile-form'>
        <View className='profile-form__content'>
          {userInfo.user_name ? (
            <View className='profile-form__item'>
              <Text className='profile-label'>{t('system.profile.account')}</Text>
              <Text className='profile-account profile-placeholder'>{userInfo.user_name}</Text>
            </View>
          ) : null}
          {process.env.TARO_ENV === 'weapp' ? (
            <View className='profile-form__item'>
              <Text className='profile-label'>{t('system.profile.mobile')}</Text>
              <View className='profile-input'>
                {userInfo.phone ? (
                  <Text className='profile-account'>{userInfo.phone}</Text>
                ) : (
                  <Button className='profile-auth-button' openType='getPhoneNumber' onGetPhoneNumber={onGetPhoneNumber}>{t('system.profile.phoneAuthorization')}</Button>
                )}
              </View>
            </View>
          ) : null}
          <View className='profile-form__item'>
            <Text className='profile-label'>{t('system.profile.nickName')}</Text>
            <Input className='profile-input' placeholder={t('system.profile.nickNamePlaceholder')} value={userInfo.nick_name} onInput={(event) => setUserInfo((current) => ({ ...current, nick_name: event.detail.value }))} />
          </View>
          <View className='profile-form__item'>
            <Text className='profile-label'>{t('system.profile.gender')}</Text>
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
        <Button className='profile-form__button' onClick={() => void onSubmit()}>{t('common.action.save')}</Button>
      </View>
    </View>
  )
}
