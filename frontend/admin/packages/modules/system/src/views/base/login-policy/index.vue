<template>
  <div class="login-policy-page">
    <el-card shadow="never">
      <template #header>
        <div class="policy-header">
          <div>
            <h3>{{ t("system.base.login_policy.title") }}</h3>
            <p>{{ t("system.base.login_policy.description") }}</p>
          </div>
          <el-switch v-model="form.enabled" :active-text="t('system.base.login_policy.enabled')" />
        </div>
      </template>

      <el-alert :title="t('system.base.login_policy.hint')" type="info" :closable="false" show-icon />
      <el-form label-position="top" class="policy-form" @submit.prevent="save">
        <el-form-item :label="t('system.base.login_policy.field.ip_blacklist')">
          <el-input v-model="form.ip_blacklist" type="textarea" :rows="3" :placeholder="t('system.base.login_policy.placeholder.ip')" />
        </el-form-item>
        <el-form-item :label="t('system.base.login_policy.field.ip_whitelist')">
          <el-input v-model="form.ip_whitelist" type="textarea" :rows="3" :placeholder="t('system.base.login_policy.placeholder.ip')" />
        </el-form-item>
        <el-form-item :label="t('system.base.login_policy.field.time_windows')">
          <el-input v-model="form.time_windows" type="textarea" :rows="2" :placeholder="t('system.base.login_policy.placeholder.time')" />
        </el-form-item>
        <el-form-item :label="t('system.base.login_policy.field.device_blacklist')">
          <el-input v-model="form.device_blacklist" type="textarea" :rows="2" :placeholder="t('system.base.login_policy.placeholder.device')" />
        </el-form-item>
        <el-form-item :label="t('system.base.login_policy.field.device_whitelist')">
          <el-input v-model="form.device_whitelist" type="textarea" :rows="2" :placeholder="t('system.base.login_policy.placeholder.device')" />
        </el-form-item>
        <el-divider />
        <div class="rules-header">
          <div>
            <h4>{{ t("system.base.login_policy.rules.title") }}</h4>
            <p>{{ t("system.base.login_policy.rules.description") }}</p>
          </div>
          <el-button type="primary" plain @click="addRule">{{ t("system.base.login_policy.rules.add") }}</el-button>
        </div>
        <el-table v-if="form.rules.length" :data="form.rules" border class="rules-table">
          <el-table-column :label="t('system.base.login_policy.rules.target_type')" width="130">
            <template #default="{ row }">
              <el-select v-model="row.target_type" class="rule-control">
                <el-option label="TENANT" value="TENANT" />
                <el-option label="USER" value="USER" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column :label="t('system.base.login_policy.rules.target_value')" min-width="160">
            <template #default="{ row }"><el-input v-model="row.target_value" class="rule-control" /></template>
          </el-table-column>
          <el-table-column :label="t('system.base.login_policy.field.ip_blacklist')" min-width="180">
            <template #default="{ row }"><el-input v-model="row.ip_blacklist" :placeholder="t('system.base.login_policy.placeholder.ip')" class="rule-control" /></template>
          </el-table-column>
          <el-table-column :label="t('system.base.login_policy.field.ip_whitelist')" min-width="180">
            <template #default="{ row }"><el-input v-model="row.ip_whitelist" :placeholder="t('system.base.login_policy.placeholder.ip')" class="rule-control" /></template>
          </el-table-column>
          <el-table-column :label="t('system.base.login_policy.field.time_windows')" min-width="160">
            <template #default="{ row }"><el-input v-model="row.time_windows" :placeholder="t('system.base.login_policy.placeholder.time')" class="rule-control" /></template>
          </el-table-column>
          <el-table-column :label="t('system.base.login_policy.rules.enabled')" width="90">
            <template #default="{ row }"><el-switch v-model="row.enabled" /></template>
          </el-table-column>
          <el-table-column :label="t('common.field.operation')" width="90" fixed="right">
            <template #default="{ $index }"><el-button link type="danger" @click="removeRule($index)">{{ t("common.action.delete") }}</el-button></template>
          </el-table-column>
        </el-table>
        <div class="policy-actions">
          <el-button type="primary" :loading="saving" @click="save">{{ t("common.action.save") }}</el-button>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseLoginPolicyService } from "@liujitcn/kratos-admin-system/api/system/base_login_policy";
import type { BaseLoginPolicy, BaseLoginPolicyRule } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_login_policy";

const saving = ref(false);
/** 登录来源定向规则表单。 */
type RuleForm = { target_type: string; target_value: string; enabled: boolean; ip_blacklist: string; ip_whitelist: string; time_windows: string; device_blacklist: string; device_whitelist: string };
const form = reactive({
  enabled: false,
  ip_blacklist: "",
  ip_whitelist: "",
  time_windows: "",
  device_blacklist: "",
  device_whitelist: "",
  rules: [] as RuleForm[]
});

/** 将接口数组转换为逐行编辑文本。 */
function assignForm(policy: BaseLoginPolicy) {
  form.enabled = policy.enabled;
  form.ip_blacklist = policy.ip_blacklist.join("\n");
  form.ip_whitelist = policy.ip_whitelist.join("\n");
  form.time_windows = policy.time_windows.join("\n");
  form.device_blacklist = policy.device_blacklist.join("\n");
  form.device_whitelist = policy.device_whitelist.join("\n");
  form.rules = (policy.rules ?? []).map(rule => ({
    target_type: rule.target_type,
    target_value: rule.target_value,
    enabled: rule.enabled,
    ip_blacklist: rule.ip_blacklist.join(","),
    ip_whitelist: rule.ip_whitelist.join(","),
    time_windows: rule.time_windows.join(","),
    device_blacklist: rule.device_blacklist.join(","),
    device_whitelist: rule.device_whitelist.join(",")
  }));
}

/** 将逐行编辑文本转换为接口数组。 */
function splitLines(value: string) {
  return value.split(/[\n,]/).map(item => item.trim()).filter(Boolean);
}

/** 加载登录来源策略。 */
async function load() {
  assignForm(await defBaseLoginPolicyService.GetBaseLoginPolicy({}));
}

/** 保存登录来源策略。 */
async function save() {
  saving.value = true;
  try {
    const policy = await defBaseLoginPolicyService.UpdateBaseLoginPolicy({
      policy: {
        enabled: form.enabled,
        ip_blacklist: splitLines(form.ip_blacklist),
        ip_whitelist: splitLines(form.ip_whitelist),
        time_windows: splitLines(form.time_windows),
        device_blacklist: splitLines(form.device_blacklist),
        device_whitelist: splitLines(form.device_whitelist),
        rules: form.rules.map(rule => ({
          target_type: rule.target_type,
          target_value: rule.target_value,
          enabled: rule.enabled,
          ip_blacklist: splitLines(rule.ip_blacklist),
          ip_whitelist: splitLines(rule.ip_whitelist),
          time_windows: splitLines(rule.time_windows),
          device_blacklist: splitLines(rule.device_blacklist),
          device_whitelist: splitLines(rule.device_whitelist)
        } as BaseLoginPolicyRule))
      }
    });
    assignForm(policy);
    ElMessage.success(t("common.message.update_success", { resource: t("system.base.login_policy.title") }));
  } finally {
    saving.value = false;
  }
}

/** 新增一条定向登录策略。 */
function addRule() {
  form.rules.push({ target_type: "TENANT", target_value: "", enabled: true, ip_blacklist: "", ip_whitelist: "", time_windows: "", device_blacklist: "", device_whitelist: "" });
}

/** 删除一条定向登录策略。 */
function removeRule(index: number) {
  form.rules.splice(index, 1);
}

onMounted(load);
</script>

<style scoped lang="scss">
.login-policy-page {
  padding: 20px;
}
.policy-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.policy-header h3 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 18px;
}
.policy-header p {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.policy-form {
  max-width: 920px;
  margin-top: 20px;
}
.policy-actions {
  display: flex;
  justify-content: flex-end;
}
.rules-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin: 20px 0 12px;
}
.rules-header h4 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 15px;
}
.rules-header p {
  margin: 4px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.rules-table {
  width: 100%;
}
.rule-control {
  width: 100%;
}
</style>
