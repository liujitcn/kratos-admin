import { Button, Text, View, WebView } from '@tarojs/components'
import Taro, { useLoad } from '@tarojs/taro'
import { useState } from 'react'
import './webview.scss'

/** 统一外链承载页。 */
export default function WebViewPage() {
  const [url, setUrl] = useState('')

  useLoad((query) => {
    const nextUrl = decodeURIComponent(query?.url || '')
    const title = decodeURIComponent(query?.title || '')
    setUrl(nextUrl)
    if (title) void Taro.setNavigationBarTitle({ title })
  })

  if (!url) {
    return (
      <View className='webview-container'>
        <View className='webview-empty'>
          <Text className='webview-empty__title'>链接无法打开</Text>
          <Text className='webview-empty__desc'>缺少有效链接地址</Text>
        </View>
      </View>
    )
  }

  return (
    <View className='webview-container'>
      <WebView src={url} onMessage={(event) => console.log('收到H5消息:', event.detail)} />
      {process.env.TARO_ENV === 'h5' ? (
        <Button
          className='webview-open-button'
          onClick={() => window.open(url, '_blank', 'noopener,noreferrer')}
        >
          新窗口打开
        </Button>
      ) : null}
    </View>
  )
}
