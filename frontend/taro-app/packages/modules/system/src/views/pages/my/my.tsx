import KratosMyPage from '@liujitcn/kratos-taro-app-system/components/KratosMyPage'
import MyExtensions from './components/MyExtensions'

/** MyPage 组合公共个人中心与项目业务扩展。 */
export default function MyPage() {
  return (
    <KratosMyPage>
      <MyExtensions />
    </KratosMyPage>
  )
}
