import GoCaptcha from 'go-captcha-react'
import 'go-captcha-react/dist/go-captcha-react.cjs.development.css'
import type { BehaviorCaptchaProps } from './types'

/** H5 行为验证码适配器。 */
export default function BehaviorCaptcha({
  type,
  data,
  config,
  onConfirm,
  onRefresh,
  onClose,
}: BehaviorCaptchaProps) {
  const events = { confirm: onConfirm, refresh: onRefresh, close: onClose }
  if (type === 'click') {
    return <GoCaptcha.Click data={{ image: data.image, thumb: data.thumb }} config={config} events={events} />
  }
  if (type === 'rotate') {
    return (
      <GoCaptcha.Rotate
        data={{
          image: data.image,
          thumb: data.thumb,
          thumbSize: data.thumbSize ?? 150,
          angle: data.angle,
        }}
        config={config}
        events={events}
      />
    )
  }
  return (
    <GoCaptcha.Slide
      data={{
        image: data.image,
        thumb: data.thumb,
        thumbX: data.thumbX ?? 0,
        thumbY: data.thumbY ?? 0,
        thumbWidth: data.thumbWidth ?? 60,
        thumbHeight: data.thumbHeight ?? 60,
      }}
      config={config}
      events={events}
    />
  )
}
