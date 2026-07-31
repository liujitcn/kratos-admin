declare namespace UniHelper {
  type UniPopupType =
    | 'top'
    | 'right'
    | 'bottom'
    | 'left'
    | 'center'
    | 'message'
    | 'dialog'
    | 'share'

  interface SwiperOnChangeEvent {
    detail: {
      current: number
    }
  }

  type SwiperOnChange = (ev: SwiperOnChangeEvent) => void
  type SelectorPickerOnChange = (ev: { detail: { value: number } }) => void
  type RadioGroupOnChange = (ev: { detail: { value: string } }) => void
  type SwitchOnChange = (ev: any) => void
  type RegionPickerOnChange = (ev: { detail: { value: string[]; code: string[] } }) => void
  type UniDataPickerOnChange = (ev: {
    detail: {
      value: Array<{ text?: string; value: string }>
    }
  }) => void
  type ButtonOnGetphonenumber = (ev: {
    detail: {
      errMsg?: string
      code?: string
    }
  }) => void | Promise<void>

  interface UniFormsRuleItem {
    rules?: Array<Record<string, unknown>>
    validateTrigger?: 'bind' | 'submit' | string
    errorMessage?: string
  }

  type UniFormsRules = Record<string, UniFormsRuleItem>

  interface UniFormsInstance {
    validate: () => Promise<unknown>
    validateField?: (props: string | string[]) => Promise<unknown>
    clearValidate?: (props?: string | string[]) => void
    setRules?: (rules: UniFormsRules) => void
  }

  interface UniPopupInstance {
    open: (type?: UniPopupType) => void
    close: () => void
  }
}
