/** 通过当前平台能力复制文本，并在 H5 非安全上下文中降级到选区复制。 */
export async function copyText(value: string): Promise<void> {
  if (!value) throw new Error('clipboard value is empty')

  // #ifdef H5
  if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value)
      return
    } catch {
      // 局域网 HTTP 等非安全上下文继续使用同步选区复制。
    }
  }
  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  textarea.style.pointerEvents = 'none'
  document.body.appendChild(textarea)
  textarea.select()
  const copied = document.execCommand('copy')
  document.body.removeChild(textarea)
  if (!copied) throw new Error('clipboard copy failed')
  return undefined
  // #endif

  // #ifndef H5
  await new Promise<void>((resolve, reject) => {
    uni.setClipboardData({ data: value, success: () => resolve(), fail: reject })
  })
  // #endif
}
