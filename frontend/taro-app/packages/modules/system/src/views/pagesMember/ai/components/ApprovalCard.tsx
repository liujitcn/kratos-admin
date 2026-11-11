import { Button, Text, View } from '@tarojs/components'
import { t } from '@liujitcn/kratos-taro-app-core'
import type { AiToolApproval } from '../../../../rpc/base/v1/ai_tool'
import { AiMessageStatus } from '../../../../rpc/base/v1/ai_session'

/** ApprovalCardProps 展示服务端冻结的操作及本人的确认决定。 */
type ApprovalCardProps = {
  approval: AiToolApproval
  status: AiMessageStatus
  busy: boolean
  onDecision: (approved: boolean) => void
}

/** ApprovalCard 在共享对话中展示待执行参数和一次性确认操作。 */
export default function ApprovalCard({ approval, status, busy, onDecision }: ApprovalCardProps) {
  const expires = Number(approval.expires_at?.seconds || 0) * 1000
  const expired = expires > 0 && expires <= Date.now()
  return (
    <View className='approval-card'>
      <View>{t('system.ai.workspace.needs_confirmation')}</View>
      {approval.calls.map((call) => (
        <View className='approval-call' key={call.name}>
          <View>{call.title || call.name}</View>
          <Text selectable>{call.arguments}</Text>
        </View>
      ))}
      <View>{t('system.ai.workspace.expires')} · {expires ? new Date(expires).toLocaleString() : '—'}</View>
      {expired ? <View>{t('system.ai.workspace.approval_expired')}</View> :
        status === AiMessageStatus.AI_MESSAGE_STATUS_WAITING_APPROVAL ? (
          <View className='approval-actions'>
            <Button size='mini' disabled={busy} loading={busy} onClick={() => onDecision(true)}>
              {t('system.ai.workspace.approve')}
            </Button>
            <Button size='mini' disabled={busy} onClick={() => onDecision(false)}>
              {t('system.ai.workspace.reject')}
            </Button>
          </View>
        ) : null}
    </View>
  )
}
