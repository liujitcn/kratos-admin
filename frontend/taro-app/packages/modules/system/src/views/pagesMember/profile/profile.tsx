import {
  Button,
  Image,
  Input,
  Label,
  Picker,
  Radio,
  RadioGroup,
  Text,
  View,
} from '@tarojs/components'
import Taro, { useLoad } from '@tarojs/taro'
import { useEffect, useState } from 'react'
import defaultAvatar from '@liujitcn/kratos-taro-app-core/static/images/avatar.png'
import navigatorBackground from '@liujitcn/kratos-taro-app-core/static/images/navigator_bg.png'
import { defAuthService } from '@liujitcn/kratos-taro-app-core/api/system/app/v1/auth'
import { defBaseDictService } from '@liujitcn/kratos-taro-app-core/api/system/app/v1/base_dict'
import type { BaseDictForm_DictItem } from '@liujitcn/kratos-taro-app-core/rpc/system/app/v1/base_dict'
import type { UserProfileForm } from '@liujitcn/kratos-taro-app-core/rpc/system/app/v1/auth'
import { BaseUserIDType } from '@liujitcn/kratos-taro-app-core/rpc/system/common/v1/common'
import {
  formatSrc,
  navigateToLogin,
  uploadFile,
  useI18n,
  useUserStore,
} from '@liujitcn/kratos-taro-app-core'
import './profile.scss'

const IMG_MAX_SIZE = 1024 * 1024
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const ID_CODE_PATTERN = /^[A-Za-z0-9][A-Za-z0-9-]{0,63}$/

/** 用户个人资料编辑页。 */
export default function ProfilePage() {
  const { locale, t } = useI18n()
  const [userInfo, setUserInfo] = useState<UserProfileForm>({
    user_name: '',
    nick_name: '',
    gender: 0,
    phone: '',
    email: '',
    id_type: BaseUserIDType.BASE_USER_ID_TYPE_UNSPECIFIED,
    id_code: '',
    avatar: '',
  })
  const [genderList, setGenderList] = useState<BaseDictForm_DictItem[]>([])
  const ensureAuthenticated = useUserStore((state) => state.ensureAuthenticated)
  const safeTop = Taro.getWindowInfo().safeArea?.top || 0
  const idTypeOptions = [
    { label: t('system.profile.id_type.unspecified'), value: BaseUserIDType.BASE_USER_ID_TYPE_UNSPECIFIED },
    { label: t('system.profile.id_type.id_card'), value: BaseUserIDType.BASE_USER_ID_TYPE_ID_CARD },
    { label: t('system.profile.id_type.passport'), value: BaseUserIDType.BASE_USER_ID_TYPE_PASSPORT },
    { label: t('system.profile.id_type.hk_macao_permit'), value: BaseUserIDType.BASE_USER_ID_TYPE_HK_MACAO_PERMIT },
    { label: t('system.profile.id_type.taiwan_permit'), value: BaseUserIDType.BASE_USER_ID_TYPE_TAIWAN_PERMIT },
    {
      label: t('system.profile.id_type.foreign_permanent_residence'),
      value: BaseUserIDType.BASE_USER_ID_TYPE_FOREIGN_PERMANENT_RESIDENCE,
    },
    { label: t('system.profile.id_type.other'), value: BaseUserIDType.BASE_USER_ID_TYPE_OTHER },
  ]
  const idTypeIndex = Math.max(0, idTypeOptions.findIndex((item) => item.value === userInfo.id_type))

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
      defBaseDictService.GetBaseDict({ code: 'base_user_gender' }),
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
      await defAuthService.UpdateUserProfile({ user_profile: nextProfile })
      setUserInfo(nextProfile)
      await useUserStore.getState().getUserProfile()
      await Taro.showToast({ icon: 'success', title: t('system.profile.update_success') })
    } catch {
      await Taro.showToast({ icon: 'error', title: t('system.profile.avatar_upload_failed') })
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
      await Taro.showToast({ title: t('system.profile.photo_limit'), icon: 'none', duration: 1500 })
      return
    }
    await uploadAvatar(filePath)
  }

  const onGetPhoneNumber = async (event: { detail: { errMsg?: string; code?: string } }) => {
    if (event.detail.errMsg !== 'getPhoneNumber:ok' || !requireAuth()) return
    const response = await defAuthService.BindUserPhone({ code: event.detail.code || '' })
    setUserInfo((current) => ({ ...current, phone: response.phone }))
    await useUserStore.getState().getUserProfile()
    await Taro.showToast({ icon: 'success', title: t('system.profile.phone_authorization_success') })
  }

  const onSubmit = async () => {
    if (!requireAuth()) return
    if (userInfo.email && !EMAIL_PATTERN.test(userInfo.email)) {
      await Taro.showToast({ icon: 'none', title: t('system.profile.email_invalid') })
      return
    }
    if (userInfo.id_code && !ID_CODE_PATTERN.test(userInfo.id_code)) {
      await Taro.showToast({ icon: 'none', title: t('system.profile.id_code_invalid') })
      return
    }
    await defAuthService.UpdateUserProfile({ user_profile: userInfo })
    await useUserStore.getState().getUserProfile()
    await Taro.showToast({ icon: 'success', title: t('system.profile.save_success') })
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
          <Text className='profile-avatar__text'>{t('system.profile.avatar_change')}</Text>
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
                  <Button className='profile-auth-button' openType='getPhoneNumber' onGetPhoneNumber={onGetPhoneNumber}>{t('system.profile.phone_authorization')}</Button>
                )}
              </View>
            </View>
          ) : null}
          <View className='profile-form__item'>
            <Text className='profile-label'>{t('system.profile.nick_name')}</Text>
            <Input className='profile-input' placeholder={t('system.profile.nick_name_placeholder')} value={userInfo.nick_name} onInput={(event) => setUserInfo((current) => ({ ...current, nick_name: event.detail.value }))} />
          </View>
          <View className='profile-form__item'>
            <Text className='profile-label'>{t('system.profile.email')}</Text>
            <Input className='profile-input' type='text' placeholder={t('system.profile.email_placeholder')} value={userInfo.email} onInput={(event) => setUserInfo((current) => ({ ...current, email: event.detail.value }))} />
          </View>
          <Picker
            mode='selector'
            range={idTypeOptions.map((item) => item.label)}
            value={idTypeIndex}
            onChange={(event) => setUserInfo((current) => ({ ...current, id_type: idTypeOptions[Number(event.detail.value)]?.value ?? BaseUserIDType.BASE_USER_ID_TYPE_UNSPECIFIED }))}
          >
            <View className='profile-form__item'>
              <Text className='profile-label'>{t('system.profile.id_type')}</Text>
              <Text className='profile-input profile-account'>{idTypeOptions[idTypeIndex]?.label}</Text>
            </View>
          </Picker>
          <View className='profile-form__item'>
            <Text className='profile-label'>{t('system.profile.id_code')}</Text>
            <Input className='profile-input' type='text' placeholder={t('system.profile.id_code_placeholder')} value={userInfo.id_code} onInput={(event) => setUserInfo((current) => ({ ...current, id_code: event.detail.value }))} />
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
