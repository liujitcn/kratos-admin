/** 支持的申请表单字段类型。 */
export const FORM_SCHEMA_FIELD_TYPES = ["text", "textarea", "number", "date", "select"] as const;

/** 申请表单字段类型。 */
export type FormSchemaFieldType = (typeof FORM_SCHEMA_FIELD_TYPES)[number];

/** 可视化申请表单字段。 */
export interface FormSchemaField {
  /** 提交数据中的字段标识。 */
  key: string;
  /** 表单展示标签。 */
  label: string;
  /** 字段组件类型。 */
  type: FormSchemaFieldType;
  /** 是否必填。 */
  required: boolean;
  /** 下拉选项，使用逗号分隔，仅 select 类型使用。 */
  options?: string;
}

/** 创建一个可编辑的申请表单字段。 */
export function createFormSchemaField(type: FormSchemaFieldType, index: number): FormSchemaField {
  return {
    key: `field_${index + 1}`,
    label: "",
    type,
    required: false,
    ...(type === "select" ? { options: "" } : {})
  };
}

/** 将后端保存的 JSON 表单定义解析为可视化字段。 */
export function parseFormSchema(raw: string): FormSchemaField[] {
  if (!raw.trim()) return [];
  const parsed: unknown = JSON.parse(raw);
  if (!Array.isArray(parsed)) throw new Error("form schema must be an array");
  return parsed.map((item, index) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) {
      throw new Error(`form field ${index} must be an object`);
    }
    const record = item as Record<string, unknown>;
    const type = record.type;
    if (typeof type !== "string" || !isFormSchemaFieldType(type)) {
      throw new Error(`form field ${index} has an unsupported type`);
    }
    const options = type === "select" ? normalizeOptions(record.options) : undefined;
    return {
      key: typeof record.key === "string" ? record.key : String(record.key ?? ""),
      label: typeof record.label === "string" ? record.label : String(record.label ?? ""),
      type,
      required: record.required === true,
      ...(options !== undefined ? { options } : {})
    };
  });
}

/** 将可视化字段序列化为后端保存的 JSON 结构。 */
export function serializeFormSchema(fields: readonly FormSchemaField[]): string {
  return JSON.stringify(
    fields.map(field => ({
      key: field.key,
      label: field.label,
      type: field.type,
      required: field.required,
      ...(field.type === "select" ? { options: field.options ?? "" } : {})
    })),
    null,
    2
  );
}

/** 校验可视化申请表单并返回国际化错误键。 */
export function validateFormSchema(fields: readonly FormSchemaField[]): string {
  const keys = new Set<string>();
  for (const field of fields) {
    if (!field.key) return "system.workflow.definition.error.form_field_key_required";
    if (!/^[A-Za-z][A-Za-z0-9_]*$/.test(field.key)) return "system.workflow.definition.error.form_field_key_invalid";
    if (keys.has(field.key)) return "system.workflow.definition.error.form_field_key_duplicate";
    keys.add(field.key);
    if (!field.label) return "system.workflow.definition.error.form_field_label_required";
    if (field.type === "select" && !getFormSchemaOptions(field.options).length) {
      return "system.workflow.definition.error.form_field_options_required";
    }
  }
  return "";
}

/** 将下拉选项转换为申请页可直接渲染的列表。 */
export function getFormSchemaOptions(options?: string): string[] {
  return options ? options.split(",").map(option => option.trim()).filter(Boolean) : [];
}

/** 判断字段类型是否受当前申请表单协议支持。 */
function isFormSchemaFieldType(value: string): value is FormSchemaFieldType {
  return (FORM_SCHEMA_FIELD_TYPES as readonly string[]).includes(value);
}

/** 兼容历史版本将数组保存为下拉选项字符串。 */
function normalizeOptions(value: unknown): string {
  if (Array.isArray(value)) return value.map(item => String(item)).join(",");
  return typeof value === "string" ? value : "";
}
