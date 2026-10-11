import { RichText, View } from '@tarojs/components'
import Taro, { useLoad } from '@tarojs/taro'
import { useState } from 'react'
import { useSettingStore, useThemeStore } from '../../../stores'
import { useI18n } from '../../../locales'
import { homeTabPage } from '../../../utils/navigation'
import './protocal.scss'

const protocolClassMap: Record<string, string> = {
  h2: 'protocol-title',
  h3: 'protocol-section-title',
  h4: 'protocol-section-title',
  p: 'protocol-paragraph',
  ul: 'protocol-list',
  ol: 'protocol-list',
  li: 'protocol-list-item',
  strong: 'protocol-emphasis',
  b: 'protocol-emphasis',
  a: 'protocol-link',
  img: 'protocol-image',
  blockquote: 'protocol-quote',
}

function decorateProtocolContent(html: string): string {
  return html.replace(
    /<(h2|h3|h4|p|ul|ol|li|strong|b|a|img|blockquote)(\s[^>]*)?>/gi,
    (match, tagName: string, attrs = '') => {
      const className = protocolClassMap[tagName.toLowerCase()]
      const classAttribute = attrs.match(/\bclass\s*=\s*(["'])(.*?)\1/i)
      if (classAttribute) {
        const nextAttrs = attrs.replace(
          classAttribute[0],
          `class="${classAttribute[2]} ${className}"`,
        )
        return `<${tagName}${nextAttrs}>`
      }
      return `<${tagName} class="${className}"${attrs}>`
    },
  )
}

/** 服务条款与隐私协议页。 */
export default function ProtocolPage() {
  const { t } = useI18n()
  const themeStore = useThemeStore()
  const [content, setContent] = useState('')
  const [pageTitle, setPageTitle] = useState(t('core.navigation.protocol'))

  useLoad((query) => {
    const isPrivacy = query?.type === 'privacy'
    const title = isPrivacy ? t('core.protocol.privacy') : t('core.protocol.service')
    setPageTitle(title)
    const key = isPrivacy ? 'privacyProtocol' : 'serviceProtocol'
    const settingStore = useSettingStore.getState()
    void settingStore
      .loadData()
      .then(() => {
        const protocol = settingStore.getData(key)
        if (!protocol) throw new Error(t('core.protocol.not_configured', { title }))
        setContent(decorateProtocolContent(protocol))
      })
      .catch(async (error: unknown) => {
        await Taro.showToast({
          icon: 'none',
          title: error instanceof Error ? error.message : t('core.protocol.load_failed'),
        })
        setTimeout(() => void Taro.navigateBack(), 300)
      })
  })

  /** 返回来源页，页面栈为空时回到首页。 */
  const onBack = () => {
    void Taro.navigateBack({ fail: () => void Taro.reLaunch({ url: homeTabPage }) })
  }

  return (
    <View className={`protocol-page ${themeStore.theme === 'light' ? 'theme-light' : 'theme-dark'}`}>
      <View className={`k-page-nav ${process.env.TARO_ENV === 'weapp' ? 'k-page-nav--weapp' : ''}`}>
        <View className='k-page-nav__back' aria-label={t('common.action.back')} onClick={onBack} />
        <View className='k-page-nav__title'>{pageTitle}</View>
        <View className='k-page-nav__side' aria-hidden />
      </View>
      <View className='protocol-card'>
        <RichText className='protocol-content' nodes={content} />
      </View>
    </View>
  )
}
