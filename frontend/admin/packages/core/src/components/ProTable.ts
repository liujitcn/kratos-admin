import type { DefineComponent } from "vue";
import type { ProTableInstance, ProTableProps } from "./ProTable/interface";
import ProTableImplementation from "./ProTable/index.vue";

/** ProTable 公共组件 Adapter。 */
const ProTable = ProTableImplementation as unknown as DefineComponent<ProTableProps, ProTableInstance>;

export default ProTable;
