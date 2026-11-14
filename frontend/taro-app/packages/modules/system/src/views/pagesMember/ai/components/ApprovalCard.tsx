import { Text, View } from '@tarojs/components'
import { t } from '@liujitcn/kratos-taro-app-core'
import type { AiToolApproval } from '../../../../rpc/base/v1/ai_tool'
import { workspaceTimestamp } from '../workspace'

/** ApprovalCardProps 只展示服务端冻结的操作和确认有效期。 */
type ApprovalCardProps = {
  approval: AiToolApproval
  status: number
}

/** ApprovalCard 在移动端只读展示参数，确认操作留在管理端。 */
export default function ApprovalCard({ approval }: ApprovalCardProps) {
  const expires = workspaceTimestamp(approval.expires_at)
  const expired = expires > 0 && expires <= Date.now()
  return (
    <View className='approval-card'>
      <View>{t('system.ai.workspace.review_on_admin')}</View>
      {approval.calls.map((call) => (
        <View className='approval-call' key={call.name}>
          <View>{call.title || call.name}</View>
          <Text selectable>{call.arguments}</Text>
        </View>
      ))}
      <View>
        {t('system.ai.workspace.expires')} · {expires ? new Date(expires).toLocaleString() : '—'}
      </View>
      {expired ? <View>{t('system.ai.workspace.approval_expired')}</View> : null}
    </View>
  )
}
