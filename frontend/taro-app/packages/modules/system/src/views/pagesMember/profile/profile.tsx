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
import { useEffect, useRef, useState } from 'react'
import defaultAvatar from '@liujitcn/kratos-taro-app-core/static/images/avatar.png'
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
  useThemeStore,
  useUserStore,
} from '@liujitcn/kratos-taro-app-core'
import './profile.scss'

const IMG_MAX_SIZE = 1024 * 1024
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const ID_CODE_PATTERN = /^[A-Za-z0-9][A-Za-z0-9-]{0,63}$/

/** 用户个人资料编辑页。 */
export default function ProfilePage() {
  const { t } = useI18n()
  const themeStore = useThemeStore()
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
  const submittingRef = useRef(false)
  const navigateBackTimerRef = useRef<ReturnType<typeof setTimeout>>()
  const ensureAuthenticated = useUserStore((state) => state.ensureAuthenticated)
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
    return () => {
      if (navigateBackTimerRef.current) clearTimeout(navigateBackTimerRef.current)
    }
  }, [])

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
    if (submittingRef.current) return
    if (!requireAuth()) return
    if (userInfo.email && !EMAIL_PATTERN.test(userInfo.email)) {
      await Taro.showToast({ icon: 'none', title: t('system.profile.email_invalid') })
      return
    }
    if (userInfo.id_code && !ID_CODE_PATTERN.test(userInfo.id_code)) {
      await Taro.showToast({ icon: 'none', title: t('system.profile.id_code_invalid') })
      return
    }
    submittingRef.current = true
    try {
      await defAuthService.UpdateUserProfile({ user_profile: userInfo })
      await useUserStore.getState().getUserProfile()
      await Taro.showToast({ icon: 'success', title: t('system.profile.save_success') })
      if (navigateBackTimerRef.current) clearTimeout(navigateBackTimerRef.current)
      navigateBackTimerRef.current = setTimeout(() => void Taro.navigateBack(), 400)
    } finally {
      submittingRef.current = false
    }
  }

  return (
    <View
      className={`profile-viewport ${themeStore.theme === 'light' ? 'theme-light' : 'theme-dark'}`}
    >
      <View className='profile-content'>
        <View className={`k-page-nav ${process.env.TARO_ENV === 'weapp' ? 'k-page-nav--weapp' : ''}`}>
          <View
            className='k-page-nav__back'
            aria-label={t('common.action.back')}
            onClick={() => void Taro.navigateBack()}
          />
          <View className='k-page-nav__title'>{t('system.profile.title')}</View>
          <View className='k-page-nav__side' aria-hidden />
        </View>

        <View className='k-glass k-glass--lit k-pfx profile-id' onClick={() => void onAvatarChange()}>
          <View className='profile-id__main'>
            <View className='k-pfx__av'>
              <Image className='profile-av-img' src={userInfo.avatar ? formatSrc(userInfo.avatar) : defaultAvatar} mode='aspectFill' />
            </View>
            <View className='profile-id__meta'>
              <View className='k-pfx__nm'>{userInfo.nick_name || t('system.profile.title')}</View>
              <View className='k-pfx__ac'>{userInfo.user_name}</View>
            </View>
          </View>
          <View className='k-pfx__ent'>{t('system.profile.avatar_change')}</View>
        </View>

        <View className='k-glass profile-form'>
          <View className='k-phd'>
            <View className='k-phd__title'>{t('system.profile.title')}</View>
          </View>

          {userInfo.user_name ? (
            <View className='k-field'>
              <Text className='k-field__k'>{t('system.profile.account')}</Text>
              <Text className='k-field__v'>{userInfo.user_name}</Text>
            </View>
          ) : null}
          {process.env.TARO_ENV === 'weapp' ? (
            <View className='k-field'>
              <Text className='k-field__k'>{t('system.profile.mobile')}</Text>
              {userInfo.phone ? (
                <Text className='k-field__v'>{userInfo.phone}</Text>
              ) : (
                <Button className='profile-auth-button' openType='getPhoneNumber' onGetPhoneNumber={onGetPhoneNumber}>{t('system.profile.phone_authorization')}</Button>
              )}
            </View>
          ) : null}
          <View className='k-field'>
            <Text className='k-field__k'>{t('system.profile.nick_name')}</Text>
            <Input className='k-field__v profile-input' placeholder={t('system.profile.nick_name_placeholder')} value={userInfo.nick_name} onInput={(event) => setUserInfo((current) => ({ ...current, nick_name: event.detail.value }))} />
          </View>
          <View className='k-field'>
            <Text className='k-field__k'>{t('system.profile.email')}</Text>
            <Input className='k-field__v profile-input' type='text' placeholder={t('system.profile.email_placeholder')} value={userInfo.email} onInput={(event) => setUserInfo((current) => ({ ...current, email: event.detail.value }))} />
          </View>
          <Picker
            mode='selector'
            range={idTypeOptions.map((item) => item.label)}
            value={idTypeIndex}
            onChange={(event) => setUserInfo((current) => ({ ...current, id_type: idTypeOptions[Number(event.detail.value)]?.value ?? BaseUserIDType.BASE_USER_ID_TYPE_UNSPECIFIED }))}
          >
            <View className='k-field profile-field--sub'>
              <Text className='k-field__k'>{t('system.profile.id_type')}</Text>
              <Text className='k-field__v'>{idTypeOptions[idTypeIndex]?.label}</Text>
            </View>
          </Picker>
          <View className='k-field'>
            <Text className='k-field__k'>{t('system.profile.id_code')}</Text>
            <Input className='k-field__v profile-input' type='text' placeholder={t('system.profile.id_code_placeholder')} value={userInfo.id_code} onInput={(event) => setUserInfo((current) => ({ ...current, id_code: event.detail.value }))} />
          </View>
          <View className='k-field'>
            <Text className='k-field__k'>{t('system.profile.gender')}</Text>
            <RadioGroup className='profile-radios' onChange={(event) => setUserInfo((current) => ({ ...current, gender: Number(event.detail.value) }))}>
              {genderList.map((item) => (
                <Label className='profile-radio' key={item.value}>
                  <Radio
                    value={item.value}
                    color={themeStore.theme === 'light' ? '#039e8c' : '#28f0d4'}
                    checked={userInfo.gender === Number(item.value)}
                  />
                  {item.label}
                </Label>
              ))}
            </RadioGroup>
          </View>

          <Button className='k-save' onClick={() => void onSubmit()}>{t('common.action.save')}</Button>
        </View>
      </View>
    </View>
  )
}
