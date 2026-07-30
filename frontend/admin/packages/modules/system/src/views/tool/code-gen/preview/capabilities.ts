/** 预览能力判断所需的 Proto 检查项。 */
export interface CodeGenPreviewProtoCheck {
  /** Proto 方法名。 */
  method_name: string;
  /** 接口是否已经存在。 */
  exists: boolean;
  /** 接口缺失时是否已选择生成。 */
  generate_when_missing: boolean;
}

/** 代码生成页面预览支持的维护能力。 */
export interface CodeGenPreviewCapabilities {
  /** 是否支持新增。 */
  create: boolean;
  /** 是否支持更新。 */
  update: boolean;
  /** 是否支持删除。 */
  delete: boolean;
}

/** 根据当前实体可用的 Proto 方法解析预览维护能力。 */
export function resolveCodeGenPreviewCapabilities(
  entityName: string,
  protoChecks: CodeGenPreviewProtoCheck[]
): CodeGenPreviewCapabilities {
  const availableMethods = new Set(
    protoChecks.filter(item => item.exists || item.generate_when_missing).map(item => item.method_name)
  );
  return {
    create: availableMethods.has(`Create${entityName}`),
    update: availableMethods.has(`Update${entityName}`),
    delete: availableMethods.has(`Delete${entityName}`)
  };
}
