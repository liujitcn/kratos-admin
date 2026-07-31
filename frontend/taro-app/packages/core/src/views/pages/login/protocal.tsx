import { RichText, View } from '@tarojs/components'
import Taro, { useLoad } from '@tarojs/taro'
import { useState } from 'react'
import { useSettingStore } from '../../../stores'
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
  const [content, setContent] = useState('')

  useLoad((query) => {
    const isPrivacy = query?.type === 'privacy'
    const title = isPrivacy ? '隐私协议' : '服务条款'
    const key = isPrivacy ? 'privacyProtocol' : 'serviceProtocol'
    const settingStore = useSettingStore.getState()
    void settingStore
      .loadData()
      .then(() => {
        const protocol = settingStore.getData(key)
        if (!protocol) throw new Error(`${title}未配置，暂时无法查看`)
        void Taro.setNavigationBarTitle({ title })
        setContent(decorateProtocolContent(protocol))
      })
      .catch(async (error: unknown) => {
        await Taro.showToast({
          icon: 'none',
          title: error instanceof Error ? error.message : '移动端配置加载失败',
        })
        setTimeout(() => void Taro.navigateBack(), 300)
      })
  })

  return (
    <View className='protocol-page'>
      <View className='protocol-card'>
        <RichText className='protocol-content' nodes={content} />
      </View>
    </View>
  )
}
