import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { test } from "node:test";
import { setImmediate as nextTask } from "node:timers/promises";
import { fileURLToPath } from "node:url";
import { runInNewContext } from "node:vm";
import ts from "typescript";
import * as vue from "vue";
import { buildCardChart, parseCardChart } from "../src/card-schema.ts";
import * as biResponse from "../src/bi-response.ts";
import * as metricEdit from "../src/metric-edit.ts";

const require = createRequire(import.meta.url);
const { parse, compileScript } = require("vue/compiler-sfc");

function setupComponent(name, props, mocks, emit = () => undefined) {
  const filename = fileURLToPath(new URL(`../src/components/${name}.vue`, import.meta.url));
  const { descriptor } = parse(readFileSync(filename, "utf8"), { filename });
  const script = compileScript(descriptor, { id: name, genDefaultAs: "component" });
  const module = { exports: {} };
  runInNewContext(
    ts.transpileModule(`${script.content}\nmodule.exports = component;`, {
      compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
    }).outputText,
    {
      module,
      exports: module.exports,
      Error,
      ElMessage: {info:()=>undefined,warning:()=>undefined,error:message=>mocks.errors?.push(message),success:()=>undefined},
      require: name => {
        if (name.endsWith(".vue") || name.endsWith(".css")) return {};
        assert.ok(Object.hasOwn(mocks, name), `未配置测试依赖: ${name}`);
        return mocks[name];
      }
    }
  );
  return module.exports.setup(props, { expose: () => undefined, emit });
}

test("工具响应省略空维度时，建卡弹窗按指标卡打开并保留来源记录", () => {
  let open;
  const query = { source_code: "test", measures: [{ code: "count" }], limit: 100 };
  const state = setupComponent(
    "BiCardFormDialog",
    {
      modelValue: false,
      sources: [],
      initialQuery: query,
      initialResult: { query_record_id: 7 },
      initialTitle: "用户总数"
    },
    {
      vue: {
        ...vue,
        watch: (_source, callback) => {
          open ??= callback;
        }
      },
      "@liujitcn/kratos-admin-core": { t: key => key },
      "../api/bi/admin/v1/bi_card": {defBiCardService:{PreviewBiCardSql:async()=>({})}},
      "../metric-edit": metricEdit,
      "../api/bi/admin/v1/bi_query": {},
      "../card-schema": { parseCardChart },
      "../bi-response": biResponse
    }
  );
  assert.doesNotThrow(() => open(true));
  assert.equal(state.form.chart_type, "metric");
  assert.equal(state.form.name, "用户总数");
  assert.equal(state.sourceRecordID.value, 7);
  assert.deepEqual(JSON.parse(JSON.stringify(state.baseQuery.value)), {
    ...query,
    dimensions: [],
    filters: [],
    orders: []
  });
  assert.equal(query.dimensions, undefined);
});

test("助手建卡读取来源记录，并补齐工具结果中省略的集合", async () => {
  let open;
  const guide = { query_record_id: 7, query: { source_code: "test", measures: [{ code: "count" }] } };
  const state = setupComponent(
    "../views/query/components/BiCardGuideDialog",
    { modelValue: false, guide },
    {
      vue: {
        ...vue,
        watch: (_source, callback) => {
          open = callback;
        }
      },
      "../../../api/bi/admin/v1/bi_query": {
        defBiQueryService: { ListBiSources: async () => ({ sources: [] }) }
      },
      "../../../api/bi/admin/v1/bi_query_record": {
        defBiQueryRecordService: {
          GetBiQuery: async request => {
            assert.equal(request.id, 7);
            return { result: { query_record_id: 7 } };
          }
        }
      },
      "../../../bi-response": biResponse
    }
  );
  await open([true, guide], undefined, () => undefined);
  assert.equal(state.visible.value, true);
  assert.deepEqual(JSON.parse(JSON.stringify(state.result.value)), {
    query_record_id: 7,
    columns: [],
    rows: [],
    truncated: false,
    elapsed_ms: 0
  });
});

test("关闭助手建卡后，迟到的来源响应不再打开弹窗", async () => {
  let open, cancel;
  const pending = Promise.withResolvers();
  const guide = { query_record_id: 7 };
  const state = setupComponent(
    "../views/query/components/BiCardGuideDialog",
    { modelValue: false, guide },
    {
      vue: {
        ...vue,
        watch: (_source, callback) => {
          open = callback;
        }
      },
      "../../../api/bi/admin/v1/bi_query": {
        defBiQueryService: { ListBiSources: async () => ({ sources: [] }) }
      },
      "../../../api/bi/admin/v1/bi_query_record": {
        defBiQueryRecordService: { GetBiQuery: () => pending.promise }
      },
      "../../../bi-response": biResponse
    }
  );
  const opening = open([true, guide], undefined, callback => {
    cancel = callback;
  });
  await Promise.resolve();
  cancel();
  await open([false, guide], undefined, () => undefined);
  pending.resolve({ result: { query_record_id: 7 } });
  await opening;
  assert.equal(state.visible.value, false);
  assert.equal(state.result.value, undefined);
});

test("助手建卡来源加载失败时同步关闭外层状态", async () => {
  let open;
  const events = [];
  const guide = { query_record_id: 7 };
  const state = setupComponent(
    "../views/query/components/BiCardGuideDialog",
    { modelValue: false, guide },
    {
      vue: {
        ...vue,
        watch: (_source, callback) => {
          open = callback;
        }
      },
      "../../../api/bi/admin/v1/bi_query": {
        defBiQueryService: {
          ListBiSources: async () => {
            throw new Error("offline");
          }
        }
      },
      "../../../api/bi/admin/v1/bi_query_record": {},
      "../../../bi-response": biResponse
    },
    (...event) => events.push(event)
  );
  await open([true, guide], undefined, () => undefined);
  assert.equal(state.visible.value, false);
  assert.deepEqual(events, [["update:modelValue", false]]);
});

test("缩略图可见后才按发布版本取数，错误可重试，卸载后不回填", async () => {
  let intersect,
    unmount,
    stopped = 0,
    calls = 0;
  const pending = Promise.withResolvers();
  const dashboard = { id: 7, published_version: 3 };
  const schema = { schemaVersion: 4, name: "published", cards: [{ id: "first" }, { id: "second" }, { id: "third" }] };
  const response = { schema_json: JSON.stringify(schema), cards: [{ item_id: "first", result: { rows: [{ values: ["42"] }] } }] };
  const state = setupComponent(
    "BiWorkbenchPreview",
    { dashboard },
    {
      vue: {
        ...vue,
        onBeforeUnmount: callback => {
          unmount = callback;
        }
      },
      "@vueuse/core": {
        useIntersectionObserver: (_host, callback) => {
          intersect = callback;
          return {
            stop: () => {
              stopped++;
            }
          };
        }
      },
      "@liujitcn/kratos-admin-core": { t: key => key },
      "../bi-dashboard-schema": {
        parseDashboardLayout: JSON.parse,
        dashboardResults: data => new Map(data.cards.map(item => [item.item_id, item.result]))
      },
      "../api/bi/admin/v1/bi_dashboard": {
        defBiDashboardService: {
          GetBiDashboardData: request => {
            assert.equal(request.id, 7);
            assert.equal(request.version, 3);
            calls++;
            if (calls === 1) return Promise.reject(new Error("offline"));
            return calls === 2 ? Promise.resolve(response) : pending.promise;
          }
        }
      }
    }
  );
  assert.equal(calls, 0);
  intersect([{ isIntersecting: false }]);
  assert.equal(calls, 0);
  intersect([{ isIntersecting: true }]);
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(stopped, 1);
  assert.equal(state.loading.value, false);
  assert.equal(state.error.value, "offline");
  await state.load();
  assert.equal(state.error.value, "");
  assert.equal(state.layout.value.cards.length, 3);
  assert.equal(state.results.value.get("first").rows[0].values[0], "42");
  const previous = state.layout.value;
  const loading = state.load();
  unmount();
  pending.resolve({ schema_json: JSON.stringify({ ...schema, name: "late" }), cards: [] });
  await loading;
  assert.equal(state.layout.value, previous);
});

test("缩略图保留真实数据和图表类型，隐藏密集控件且不改变完整卡片", () => {
  const result = {
    columns: [
      { code: "day", type: "string" },
      { code: "amount", type: "number" }
    ],
    rows: [{ values: ["Monday", "42"] }, { values: ["Tuesday", "81"] }]
  };
  for (const chart_type of ["bar", "line", "area", "scatter", "heatmap", "pie", "donut", "metric"]) {
    const card = {
      id: 1,
      chart_type,
      chart_json: JSON.stringify({ encoding: {}, style: { showLegend: true, showLabel: true, dataZoom: true } }),
      query: { dimensions: ["day"], measures: [{ code: "amount" }] }
    };
    const mocks = {
      vue,
      "@liujitcn/kratos-admin-core": { t: key => key },
      "@element-plus/icons-vue": { ArrowUp: {}, ArrowDown: {}, Minus: {} },
      "../card-schema": { buildCardChart },
      "../env": { useBiTheme: () => vue.ref({}) }
    };
    const original = setupComponent("BiCardVisualization", { card, result }, mocks).data.value;
    const preview = setupComponent("BiCardVisualization", { card, result, compact: true }, mocks).data.value;
    assert.deepEqual(preview.rows, original.rows);
    assert.deepEqual(preview.metrics, original.metrics);
    assert.equal(preview.chart.type, chart_type);
    assert.equal(preview.chart.style.showLegend, false);
    assert.equal(preview.chart.style.dataZoom, false);
    assert.equal(original.chart.options, undefined);
    assert.equal(original.chart.style.showLegend, true);
    assert.equal(original.chart.style.dataZoom, true);
    if (["bar", "line", "area", "scatter", "heatmap"].includes(chart_type)) {
      assert.equal(preview.chart.options.xAxis.show, false);
      assert.equal(preview.chart.options.yAxis.show, false);
    }
  }
});

test("公式编辑使预览失效，样式编辑保留预览，冲突刷新保留公式草稿", async () => {
 const metric={code:"sales",name:"销售额",type:"number",unit:"元",revision:5,expression:"SUM(amount)"};
 let revision=5;
 const errors=[];
 let gets=0;
 const card={id:3,version:1,query_record_id:7,query:{source_code:"sales",dimensions:[],measures:[{code:"sales"}],limit:100},chart_type:"metric",chart_json:""};
 const props=vue.reactive({modelValue:false,card,sources:[{code:"sales",version:5}]});
 const state=setupComponent("BiCardFormDialog",props,{
  vue, errors,"@liujitcn/kratos-admin-core":{t:key=>key},"../metric-edit":metricEdit,"../card-schema":{parseCardChart},"../bi-response":biResponse,
  "../api/bi/admin/v1/bi_card":{defBiCardService:{GetBiCard:async()=>{gets++;return ({...card,source_version:revision,can_edit_metrics:true,metric_definitions:[{...metric,revision}],sql:"SELECT ..."})},PreviewBiCardSql:async()=>({preview_query_record_id:10,source_version:revision,can_edit_metrics:true,result:{columns:[],rows:[]},sql:"SELECT new",impact:{changes:[]}})}}
 });
 props.modelValue=true;
 await vue.nextTick();
 await nextTask();
 assert.equal(gets,1);
 assert.equal(state.editedMetrics.value.length,1,JSON.stringify({errors,sourceRecordID:state.sourceRecordID.value,loading:state.loadingDetails.value}));
 state.editedMetrics.value[0].expression="SUM(amount-refund)";
 assert.equal(state.previewRecordID.value,0);
 assert.equal(state.sql.value,"");
 await state.preview();
 assert.equal(state.previewRecordID.value,10);
 state.form.name="净销售额";
 state.chartConfig.value.style.showLabel=true;
 assert.equal(state.previewRecordID.value,10);
 revision=6;
 await state.loadDetails(true);
 assert.equal(state.editedMetrics.value[0].expression,"SUM(amount-refund)");
 assert.equal(state.changes.value[0].expected_revision,6);
 assert.equal(state.previewRecordID.value,0);
 assert.equal(state.sql.value,"");
});

test("保存响应丢失后保留原请求，重试同一预览定位已保存卡片", async () => {
 const calls=[];
 const emitted=[];
 let previews=0;
 const saved={id:42,name:"原始名称",query_record_id:7};
 const props=vue.reactive({modelValue:false,sources:[{code:"sales",version:1}],initialQuery:{source_code:"sales",measures:[{code:"sales"}],limit:100},initialResult:{query_record_id:7},initialTitle:"原始名称"});
 const state=setupComponent("BiCardFormDialog",props,{
  vue,"@liujitcn/kratos-admin-core":{t:key=>key},"../metric-edit":metricEdit,"../card-schema":{parseCardChart},"../bi-response":biResponse,
  "../api/bi/admin/v1/bi_card":{defBiCardService:{PreviewBiCardSql:async()=>({preview_query_record_id:10+previews++,source_version:1,can_edit_metrics:false,result:{columns:[],rows:[]},sql:"SELECT ..."}),SaveBiCard:async req=>{calls.push(JSON.parse(JSON.stringify(req)));if(calls.length===1) throw new Error("Network Error");assert.equal(req.preview_query_record_id,10);throw {response:{status:409,data:{code:409,metadata:{card_id:"42"}}}}},GetBiCard:async req=>{assert.equal(req.id,42);return saved}}}
 },(...args)=>emitted.push(args));
 state.dialogRef.value={validate:async()=>true};
 props.modelValue=true;
 await vue.nextTick();
 await nextTask();
 await state.submit();
 state.form.name="尚未提交的新名称";
 await state.preview();
 assert.equal(previews,1,"结果未知时不能替换原一次性预览");
 await state.submit();
 assert.deepEqual(calls.map(req=>req.preview_query_record_id),[10,10]);
 assert.equal(calls[1].card.name,"原始名称","重试必须使用冻结的原请求");
 assert.equal(emitted.find(([event])=>event==="saved")?.[1].id,42);
});

test("数据源模型使用最新版本，检查影响后才能保存，刷新保留公式草稿", async () => {
 const stored=JSON.stringify({baseTable:"orders",tables:[{key:"orders"}],relations:[],dimensions:[],measures:[{code:"sales",name:"销售额",type:"number",expression:"SUM(amount)",unit:"元",revision:5}]});
 let version=8;
 const saved=[];
 const props=vue.reactive({modelValue:false,source:{id:1,version:7,code:"sales",schema_json:"{}"}});
 const state=setupComponent("BiSourceModelDialog",props,{
  vue:{...vue,useId:()=>"test"},"@liujitcn/kratos-admin-core":{t:key=>key},"@element-plus/icons-vue":{},
  "@vue-flow/core":{useVueFlow:()=>({fitView:()=>undefined,updateNodeInternals:()=>undefined,viewportHelper:vue.ref({viewportInitialized:false})}),Position:{Right:"right",Left:"left"}},
  "@vue-flow/background":{},"@vue-flow/controls":{},"@vue-flow/minimap":{},"./source-model-layout":{formatSourceModelLayout:()=>undefined},
  "../api/bi/admin/v1/bi_query":{defBiQueryService:{}},
  "../api/bi/admin/v1/bi_source":{defBiSourceService:{GetBiSource:async()=>({id:1,version,schema_json:stored,model_configured:true}),ListBiSourceTables:async()=>({tables:[{key:"orders",name:"orders",columns:[{name:"amount",data_type:"decimal"}]}]}),PreviewBiSourceModel:async req=>{assert.equal(req.expected_source_version,version);return {changes:[],affected_cards:2,affected_dashboards:1}},SaveBiSourceModel:async req=>{saved.push(req);return {version:++version,schema_json:stored,sync_summary:{affected_cards:2,affected_dashboards:1}}}}}
 });
 props.modelValue=true;
 await vue.nextTick();
 await nextTask();
 assert.equal(state.sourceVersion.value,8);
 state.model.value.measures[0].expression="SUM(amount-1)";
 await state.saveModel();
 assert.equal(saved.length,0);
 await state.analyzeModel();
 assert.equal(state.impactReady.value,true);
 state.model.value.measures[0].expression="SUM(amount-2)";
 assert.equal(state.impactReady.value,false);
 await state.analyzeModel();
 await state.saveModel();
 assert.equal(saved[0].expected_source_version,8);
 assert.equal(state.sourceVersion.value,9);
 state.model.value.measures[0].expression="SUM(amount-3)";
 version=10;
 await state.refreshVersion();
 assert.equal(state.model.value.measures[0].expression,"SUM(amount-3)");
 assert.equal(state.sourceVersion.value,10);
 assert.equal(state.impactReady.value,false);
});

for (const operation of ["loadModel", "refreshVersion", "saveModel"]) {
 test(`数据源切换后忽略 ${operation} 的迟到结果`, async () => {
  const pending=Promise.withResolvers();
  let hold=false;
  const stored=id=>JSON.stringify({baseTable:"orders",tables:[{key:"orders"}],relations:[],dimensions:[],measures:[{code:`sales${id}`,name:`销售额${id}`,type:"number",expression:"SUM(amount)",unit:"元",revision:1}]});
  const source=id=>({id,version:id,code:`source${id}`,schema_json:stored(id),model_configured:true});
  const props=vue.reactive({modelValue:false,source:source(1)});
  const state=setupComponent("BiSourceModelDialog",props,{
   vue:{...vue,useId:()=>"test"},"@liujitcn/kratos-admin-core":{t:key=>key},"@element-plus/icons-vue":{},
   "@vue-flow/core":{useVueFlow:()=>({fitView:()=>undefined,updateNodeInternals:()=>undefined,viewportHelper:vue.ref({viewportInitialized:false})}),Position:{Right:"right",Left:"left"}},
   "@vue-flow/background":{},"@vue-flow/controls":{},"@vue-flow/minimap":{},"./source-model-layout":{formatSourceModelLayout:()=>undefined},
   "../api/bi/admin/v1/bi_query":{defBiQueryService:{}},
   "../api/bi/admin/v1/bi_source":{defBiSourceService:{GetBiSource:async req=>hold&&req.id===1&&operation!=="saveModel"?pending.promise:source(req.id),ListBiSourceTables:async()=>({tables:[{key:"orders",name:"orders",columns:[{name:"amount",data_type:"decimal"}]}]}),PreviewBiSourceModel:async()=>({changes:[],affected_cards:0,affected_dashboards:0}),SaveBiSourceModel:async()=>pending.promise}}
  });
  let inFlight;
  if(operation==="loadModel") hold=true;
  props.modelValue=true;
  await vue.nextTick();
  await nextTask();
  if(operation!=="loadModel") {
   if(operation==="saveModel") await state.analyzeModel();
   hold=true;
   inFlight=state[operation]();
  }
  props.modelValue=false;
  await vue.nextTick();
  props.source=source(2);
  props.modelValue=true;
  await vue.nextTick();
  await nextTask();
  assert.equal(state.model.value.measures[0].code,"sales2");
  pending.resolve({...source(1),version:9});
  await inFlight;
  await nextTask();
  assert.equal(state.model.value.measures[0].code,"sales2","旧请求不能把其他源模型填入当前源");
  assert.equal(state.sourceVersion.value,2,"旧请求不能覆盖当前源并发版本");
 });
}

test("重复预览冲突只定位服务端明确给出的已保存卡片", () => {
 assert.equal(metricEdit.appliedCardID({response:{data:{code:409,metadata:{card_id:"42",card_version:"3"}}}}),42);
 for(const cause of [new Error("42"),{response:{data:{code:403,metadata:{card_id:"42"}}}},{response:{data:{code:409,metadata:{card_id:"NaN"}}}}]) assert.equal(metricEdit.appliedCardID(cause),0);
});
