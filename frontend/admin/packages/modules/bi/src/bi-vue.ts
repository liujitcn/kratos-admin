/**
 * Vue 3 BI 仪表盘：BiDashboard 只读渲染器 + BiDesigner 画布设计器。
 * 数据查询与卡片配置由服务端提供，设计器仅调整卡片布局。
 */
import "./styles.scss";

export { default as BiDashboard } from "./components/BiDashboard.vue";
export { default as BiDesigner } from "./components/BiDesigner.vue";
export { default as BiChart } from "./components/BiChart.vue";

// 环境注入（主题跟随 / resolvers / 项目 i18n）
export { provideBiEnv, useBiEnv, useBiTheme, useBiI18n } from "./env";
export type { BiEnv } from "./env";
