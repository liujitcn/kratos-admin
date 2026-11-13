type ConflictMessageTranslator = (key: string, params?: Record<string, string>) => string;

/** 格式化结构化冲突错误，优先显示资源和字段，缺少业务元数据时回退到接口路径。 */
export function formatConflictMessage(
  message: string,
  error: { reason?: string; metadata?: Record<string, string> } | undefined,
  translate: ConflictMessageTranslator
): string {
  if (error?.reason !== "CONFLICT") return message;

  const metadata = error.metadata ?? {};
  const resource = metadata.resource;
  const childResource = metadata.child_resource;
  const conflictType = metadata.conflict_type;
  const details: string[] = [];
  const fields = resource
    ? (metadata.field ?? "")
        .split(",")
        .map(field => field.trim())
        .filter(Boolean)
        .map(field => conflictFieldLabel(resource, field, translate))
    : [];

  if (resource && childResource && conflictType === "has_children") {
    details.push(
      translate("common.error.conflict.related_resource", {
        parent: conflictResourceLabel(resource, translate),
        child: conflictResourceLabel(childResource, translate)
      })
    );
  } else if (resource && conflictType === "unique_violation") {
    details.push(
      translate("common.error.conflict.unique_location", {
        resource: conflictResourceLabel(resource, translate),
        fields: fields.join(translate("common.error.conflict.field_separator")) || translate("common.error.conflict.unknown_field")
      })
    );
  } else if (resource) {
    details.push(
      fields.length
        ? translate("common.error.conflict.unique_location", {
            resource: conflictResourceLabel(resource, translate),
            fields: fields.join(translate("common.error.conflict.field_separator"))
          })
        : translate("common.error.conflict.resource", { resource: conflictResourceLabel(resource, translate) })
    );
  } else if (metadata.operation) {
    details.push(translate("common.error.conflict.operation", { operation: metadata.operation }));
  }

  if (!details.length) return message;

  if (!message || metadata.message_key === "common.error.conflict" || message === translate("common.error.conflict")) {
    return details.join(translate("common.error.conflict.detail_separator"));
  }

  return translate("common.error.conflict.with_details", {
    message,
    details: details.join(translate("common.error.conflict.detail_separator"))
  });
}

/** 将冲突资源标识翻译为可读名称，无对应词条时回退原始标识。 */
function conflictResourceLabel(resource: string, translate: ConflictMessageTranslator): string {
  const name = resource.startsWith("base_") ? resource.slice("base_".length) : resource;
  return translatedIdentifier([`system.base.${name}.resource`, `system.base.${name}.title`], resource, translate);
}

/** 将冲突字段名翻译为可读名称，优先取资源级词条再回退通用词条。 */
function conflictFieldLabel(resource: string, field: string, translate: ConflictMessageTranslator): string {
  const name = resource.startsWith("base_") ? resource.slice("base_".length) : resource;
  return translatedIdentifier([`system.base.${name}.field.${field}`, `common.field.${field}`], field, translate);
}

/** 依次尝试候选词条，全部未命中时回退到原始标识符。 */
function translatedIdentifier(keys: string[], fallback: string, translate: ConflictMessageTranslator): string {
  for (const key of keys) {
    const value = translate(key);
    if (value && value !== key) return value;
  }
  return fallback;
}
