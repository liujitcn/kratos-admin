import { Button, Text, View, WebView } from '@tarojs/components'
import Taro, { useLoad } from '@tarojs/taro'
import { useEffect, useState } from 'react'
import './webview.scss'

/** 统一外链承载页。 */
export default function WebViewPage() {
  const [url, setUrl] = useState('')
  const [iframeLoaded, setIframeLoaded] = useState(false)
  const [iframeTimedOut, setIframeTimedOut] = useState(false)
  const isH5 = process.env.TARO_ENV === 'h5'

  useLoad((query) => {
    const nextUrl = decodeURIComponent(query?.url || '')
    const title = decodeURIComponent(query?.title || '')
    setUrl(nextUrl)
    if (title) void Taro.setNavigationBarTitle({ title })
  })

  useEffect(() => {
    if (!isH5 || !url) return undefined
    setIframeLoaded(false)
    setIframeTimedOut(false)
    const timer = window.setTimeout(() => setIframeTimedOut(true), 800)
    return () => window.clearTimeout(timer)
  }, [isH5, url])

  const showFallback = !url || (isH5 && iframeTimedOut && !iframeLoaded)
  const emptyDescription = url
    ? '当前 H5 页面可能被目标站点限制嵌入'
    : '缺少有效链接地址'

  return (
    <View className='webview-container'>
      {url ? (
        <WebView
          src={url}
          onLoad={() => setIframeLoaded(true)}
          onMessage={(event) => console.log('收到H5消息:', event.detail)}
        />
      ) : null}
      {showFallback ? (
        <View className='webview-empty'>
          <Text className='webview-empty__title'>链接无法打开</Text>
          <Text className='webview-empty__desc'>{emptyDescription}</Text>
        </View>
      ) : null}
      {isH5 && url && iframeTimedOut ? (
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
