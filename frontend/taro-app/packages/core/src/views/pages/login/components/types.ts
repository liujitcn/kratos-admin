/** 行为验证码展示数据。 */
export interface BehaviorCaptchaData {
  image: string
  thumb: string
  thumbX?: number
  thumbY?: number
  thumbWidth?: number
  thumbHeight?: number
  thumbSize?: number
  angle?: number
}

/** 行为验证码点位。 */
export interface BehaviorCaptchaPoint {
  x: number
  y: number
  key?: number
  index?: number
}

/** 行为验证码组件参数。 */
export interface BehaviorCaptchaProps {
  type: string
  data: BehaviorCaptchaData
  config: Record<string, string | number | boolean>
  onConfirm: (
    value: BehaviorCaptchaPoint | BehaviorCaptchaPoint[] | number,
    reset: () => void,
  ) => void
  onRefresh: () => void
  onClose: () => void
}
