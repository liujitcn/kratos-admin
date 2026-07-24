<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { useSettingStore } from '@/stores'
import { ref } from 'vue'

const settingStore = useSettingStore()
const content = ref('')

// 加载协议内容
const loadProtocol = async (type?: string) => {
  if (!settingStore.getData('serviceProtocol') && !settingStore.getData('privacyProtocol')) {
    await settingStore.loadData()
  }

  const isPrivacy = type === 'privacy'
  const title = isPrivacy ? '隐私协议' : '服务条款'
  const key = isPrivacy ? 'privacyProtocol' : 'serviceProtocol'

  uni.setNavigationBarTitle({
    title,
  })
  content.value = resolveProtocolContent(key, isPrivacy)
}

// 解析协议正文，隐私协议与服务条款完全一致时使用兜底隐私政策，避免展示错误协议。
const resolveProtocolContent = (key: string, isPrivacy: boolean) => {
  const targetContent = settingStore.getData(key) || ''
  if (!isPrivacy) return targetContent

  const serviceContent = settingStore.getData('serviceProtocol') || ''
  if (!targetContent || targetContent !== serviceContent) return targetContent

  return '<h2>隐私政策</h2><p>我们会按照法律法规要求收集、使用、保存和保护您的个人信息。我们仅在提供账号登录、应用服务、交易安全和客户服务所必需的范围内处理相关信息。</p><p>您可以依法查询、更正、删除个人信息，或撤回授权、注销账号。若您对个人信息处理有疑问，请通过应用内公布的客服渠道联系我们。</p>'
}

onLoad((query) => {
  loadProtocol(query?.type)
})
</script>
<template>
  <view class="container">
    <rich-text :nodes="content" />
  </view>
</template>
