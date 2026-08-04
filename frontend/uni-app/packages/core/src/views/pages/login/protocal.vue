<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { useSettingStore } from '../../../stores'
import { ref } from 'vue'
import { useI18n } from '../../../locales'

const settingStore = useSettingStore()
const { t } = useI18n()
const content = ref('')

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

/** 为协议富文本节点补充统一类名，确保小程序端也能应用页面样式。 */
const decorateProtocolContent = (html: string) => {
  return html.replace(
    /<(h2|h3|h4|p|ul|ol|li|strong|b|a|img|blockquote)(\s[^>]*)?>/gi,
    (match, tagName: string, attrs = '') => {
      const className = protocolClassMap[tagName.toLowerCase()]
      if (!className) return match

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

// 加载协议内容
const loadProtocol = async (type?: string) => {
  const isPrivacy = type === 'privacy'
  const title = isPrivacy ? t('core.protocol.privacy') : t('core.protocol.service')
  const key = isPrivacy ? 'privacyProtocol' : 'serviceProtocol'

  try {
    await settingStore.loadData()
    const protocol = settingStore.getData(key)
    if (!protocol) {
      throw new Error(t('core.protocol.not_configured', { title }))
    }

    await uni.setNavigationBarTitle({ title })
    content.value = decorateProtocolContent(protocol)
  } catch (error) {
    await uni.showToast({
      icon: 'none',
      title: error instanceof Error ? error.message : t('core.protocol.load_failed'),
    })
    setTimeout(() => {
      uni.navigateBack()
    }, 300)
  }
}

onLoad((query) => {
  void loadProtocol(query?.type)
})
</script>
<template>
  <view class="protocol-page">
    <view class="protocol-card">
      <rich-text class="protocol-content" :nodes="content" />
    </view>
  </view>
</template>

<style lang="scss">
page {
  background-color: #f5f7f6;
}

.protocol-page {
  min-height: 100vh;
  padding: 24rpx;
  box-sizing: border-box;
}

.protocol-card {
  overflow: hidden;
  padding: 40rpx 32rpx 56rpx;
  border: 1rpx solid #edf1ef;
  border-radius: 16rpx;
  background-color: #fff;
  box-shadow: 0 8rpx 24rpx rgba(33, 54, 48, 0.05);
}

.protocol-content {
  display: block;
  color: #53635d;
  font-size: 28rpx;
  line-height: 1.85;
  word-break: break-word;
}

.protocol-content .protocol-title {
  margin: 0 0 28rpx;
  color: #1f2d29;
  font-size: 42rpx;
  font-weight: 700;
  line-height: 1.35;
}

.protocol-content .protocol-section-title {
  margin: 38rpx 0 14rpx;
  padding-left: 16rpx;
  border-left: 6rpx solid #27ba9b;
  color: #263a34;
  font-size: 31rpx;
  font-weight: 600;
  line-height: 1.45;
}

.protocol-content .protocol-paragraph {
  margin: 0 0 22rpx;
  color: #53635d;
  font-size: 28rpx;
  line-height: 1.85;
}

.protocol-content .protocol-paragraph:last-child {
  margin-bottom: 0;
}

.protocol-content .protocol-list {
  margin: 0 0 22rpx;
  padding-left: 42rpx;
}

.protocol-content .protocol-list-item {
  margin-bottom: 10rpx;
  padding-left: 6rpx;
  line-height: 1.8;
}

.protocol-content .protocol-emphasis {
  color: #263a34;
  font-weight: 600;
}

.protocol-content .protocol-link {
  color: #16806d;
  text-decoration: underline;
}

.protocol-content .protocol-image {
  display: block;
  max-width: 100%;
  height: auto;
  margin: 24rpx auto;
}

.protocol-content .protocol-quote {
  margin: 24rpx 0;
  padding: 18rpx 22rpx;
  border-left: 6rpx solid #b8e8dc;
  color: #6c7b75;
  background-color: #f3fbf8;
}
</style>
