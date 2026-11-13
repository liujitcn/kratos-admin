import { existsSync, mkdirSync, readFileSync, readdirSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";
import ts from "typescript";

const adminRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const coreRequire = createRequire(join(adminRoot, "packages/core/package.json"));
const { parse } = coreRequire("vue/compiler-sfc");
const packageRoots = ["packages/core", "packages/modules/system", "packages/modules/bi"];
const typeFormat = ts.TypeFormatFlags.NoTruncation | ts.TypeFormatFlags.UseAliasDefinedOutsideCurrentScope;

/** 提取组件公开 API，使用 TypeScript 解析导入、继承和工具类型。 */
export function extractComponentApis(files) {
  const configPath = join(adminRoot, "tsconfig.json");
  const config = ts.readConfigFile(configPath, ts.sys.readFile);
  if (config.error) throw new Error(ts.flattenDiagnosticMessageText(config.error.messageText, "\n"));
  const { options } = ts.parseJsonConfigFileContent(config.config, ts.sys, adminRoot);
  const vueModule = ts.resolveModuleName(
    "vue",
    join(adminRoot, "packages/core/src/component-docs.ts"),
    options,
    ts.sys
  ).resolvedModule;
  const components = new Map();
  const host = ts.createCompilerHost(options);
  const readFile = host.readFile;
  const fileExists = host.fileExists;
  host.readFile = filename => {
    if (!/\.vue\.tsx?$/.test(filename)) return readFile(filename);
    const componentFile = filename.replace(/\.tsx?$/, "");
    if (!components.has(componentFile)) components.set(componentFile, readComponent(componentFile));
    return components.get(componentFile).script;
  };
  host.fileExists = filename =>
    /\.vue\.tsx?$/.test(filename) ? fileExists(filename.replace(/\.tsx?$/, "")) : fileExists(filename);
  host.resolveModuleNames = (names, containingFile) =>
    names.map(name => {
      const module = ts.resolveModuleName(name, containingFile, options, host).resolvedModule;
      return module ?? (name === "vue" ? vueModule : undefined);
    });
  const virtualFiles = files.map(filename => {
    const component = readComponent(filename);
    components.set(filename, component);
    return `${filename}.${component.lang === "tsx" ? "tsx" : "ts"}`;
  });
  const globals = join(adminRoot, "packages/core/types/generated/auto-imports.d.ts");
  const program = ts.createProgram([...virtualFiles, ...(existsSync(globals) ? [globals] : [])], options, host);
  const checker = program.getTypeChecker();
  const result = new Map();
  for (let index = 0; index < files.length; index++) {
    const filename = files[index];
    try {
      result.set(filename, extractApi(program.getSourceFile(virtualFiles[index]), components.get(filename), checker));
    } catch (error) {
      throw new Error(`${filename}: ${error.message}`, { cause: error });
    }
  }
  return result;
}

/** 读取 Vue 脚本及模板，并让泛型约束参与类型推导。 */
function readComponent(filename) {
  const { descriptor, errors } = parse(readFileSync(filename, "utf8"), { filename });
  if (errors.length) throw new Error(`${filename}: ${errors.map(error => error.message ?? error).join("\n")}`);
  if (descriptor.script && !descriptor.scriptSetup)
    throw new Error(`${filename}: 组件 API 提取仅支持 script setup，请补充 Options API 解析后再生成`);
  if (descriptor.script?.src || descriptor.scriptSetup?.src) throw new Error(`${filename}: 暂不支持外部 script src`);
  let script = [descriptor.script?.content, descriptor.scriptSetup?.content].filter(Boolean).join("\n");
  const ast = ts.createSourceFile(`${filename}.tsx`, script, ts.ScriptTarget.Latest, true);
  const macros = new Set();
  let exposed = "{}";
  const visit = node => {
    if (
      ts.isCallExpression(node) &&
      ts.isIdentifier(node.expression) &&
      /^define(Props|Emits|Slots|Expose|Model)$|^withDefaults$/.test(node.expression.text)
    ) {
      macros.add(node.expression.text);
      if (node.expression.text === "defineExpose" && node.arguments[0]) exposed = node.arguments[0].getText(ast);
    }
    ts.forEachChild(node, visit);
  };
  visit(ast);
  const imported = new Set(
    ast.statements
      .filter(ts.isImportDeclaration)
      .flatMap(statement =>
        statement.importClause?.namedBindings && ts.isNamedImports(statement.importClause.namedBindings)
          ? statement.importClause.namedBindings.elements.map(element => element.name.text)
          : []
      )
  );
  const missingMacros = [...macros].filter(name => !imported.has(name));
  if (missingMacros.length) script += `\nimport { ${missingMacros.join(", ")} } from "vue";`;
  if (descriptor.scriptSetup) {
    script += `\nconst __componentDocsExposed = ${exposed};\ndeclare const __componentDocsDefault: new () => import("vue").ShallowUnwrapRef<typeof __componentDocsExposed>;\nexport default __componentDocsDefault;`;
  }
  const generic = descriptor.scriptSetup?.attrs.generic;
  if (typeof generic === "string") {
    const declaration = ts.createSourceFile(
      "generics.ts",
      `type ComponentGenerics<${generic}> = unknown;`,
      ts.ScriptTarget.Latest,
      true
    );
    for (const parameter of declaration.statements[0].typeParameters) {
      script += `\ntype ${parameter.name.text} = ${parameter.constraint?.getText(declaration) ?? "unknown"};`;
    }
  }
  return { script, lang: descriptor.scriptSetup?.lang ?? descriptor.script?.lang, template: descriptor.template?.ast };
}

/** 收集宏声明并将四类 API 转换为文档条目。 */
function extractApi(source, component, checker) {
  const calls = new Map();
  const visit = node => {
    if (
      ts.isCallExpression(node) &&
      ts.isIdentifier(node.expression) &&
      /^define(Props|Emits|Slots|Expose|Model)$|^withDefaults$/.test(node.expression.text)
    ) {
      const name = node.expression.text;
      calls.set(name, [...(calls.get(name) ?? []), node]);
    }
    ts.forEachChild(node, visit);
  };
  visit(source);
  const api = { props: [], events: [], slots: [], methods: [] };
  const defaults = new Map();
  for (const call of calls.get("withDefaults") ?? []) {
    for (const property of objectProperties(call.arguments[1], checker)) {
      defaults.set(propertyName(property.name), compact(property.initializer?.getText() ?? property.getText()));
    }
  }
  for (const call of calls.get("defineProps") ?? []) {
    if (call.typeArguments?.[0]) {
      const type = resolveApiType(call.typeArguments[0], checker);
      api.props.push(
        ...typeEntries(type, call, checker).map(entry => ({
          ...entry,
          required: entry.required && !defaults.has(entry.name),
          ...(defaults.has(entry.name) ? { default: defaults.get(entry.name) } : {})
        }))
      );
    } else {
      api.props.push(...runtimeProps(call.arguments[0], checker));
    }
  }
  for (const call of calls.get("defineEmits") ?? []) {
    if (call.typeArguments?.[0]) {
      const type = resolveApiType(call.typeArguments[0], checker);
      for (const signature of type.getCallSignatures()) {
        const [event, ...parameters] = signature.getParameters();
        if (!event) throw new Error("defineEmits 重载缺少事件名称");
        const eventType = checker.getTypeOfSymbolAtLocation(event, call);
        const names = eventType.isUnion() ? eventType.types : [eventType];
        const payload = `(${parameters.map(parameter => parameterText(parameter, call, checker)).join(", ")}) => void`;
        for (const name of names) {
          if (!name.isStringLiteral()) throw new Error(`无法解析 defineEmits 事件名称：${checker.typeToString(eventType)}`);
          addEntry(api.events, {
            name: name.value,
            description: ts.displayPartsToString(signature.getDocumentationComment(checker)),
            type: payload
          });
        }
      }
      for (const property of type.getProperties()) {
        const tuple = checker.getTypeOfSymbolAtLocation(property, call);
        const text = checker.typeToString(tuple, call, typeFormat);
        if (!checker.isTupleType(tuple)) throw new Error(`defineEmits.${property.name} 不是参数元组：${text}`);
        addEntry(api.events, {
          name: property.name,
          description: symbolDescription(property, checker),
          type: `(${text.slice(1, -1)}) => void`
        });
      }
    } else {
      api.events.push(...runtimeEvents(call.arguments[0], checker));
    }
  }
  for (const call of calls.get("defineModel") ?? []) {
    const named = call.arguments[0] && ts.isStringLiteral(call.arguments[0]);
    const name = named ? call.arguments[0].text : "modelValue";
    const modelOptions = objectProperties(call.arguments[named ? 1 : 0], checker);
    const type = call.typeArguments?.[0]
      ? compact(call.typeArguments[0].getText())
      : runtimePropType(modelOptions.find(property => propertyName(property.name) === "type")?.initializer, checker);
    const required =
      modelOptions.find(property => propertyName(property.name) === "required")?.initializer?.kind === ts.SyntaxKind.TrueKeyword;
    const defaultValue = modelOptions.find(property => propertyName(property.name) === "default")?.initializer;
    api.props.push({
      name,
      description: "双向绑定值",
      type,
      required,
      ...(defaultValue ? { default: compact(defaultValue.getText()) } : {})
    });
    api.events.push({ name: `update:${name}`, description: "更新双向绑定值", type: `(value: ${type}) => void` });
  }
  for (const call of calls.get("defineSlots") ?? []) {
    if (!call.typeArguments?.[0]) throw new Error("defineSlots 缺少类型声明");
    const type = resolveApiType(call.typeArguments[0], checker);
    api.slots.push(...typeEntries(type, call, checker).map(({ name, description, type }) => ({ name, description, type })));
    for (const index of checker.getIndexInfosOfType(type)) {
      api.slots.push({
        name: `[name: ${checker.typeToString(index.keyType)}]`,
        description: "动态具名插槽",
        type: checker.typeToString(index.type, call, typeFormat)
      });
    }
  }
  collectTemplateSlots(component.template, api.slots);
  for (const call of calls.get("defineExpose") ?? []) {
    if (!call.arguments[0]) continue;
    const type = checker.getTypeAtLocation(call.arguments[0]);
    api.methods.push(
      ...typeEntries(type, call, checker, true).map(({ name, description, type }) => ({ name, description, type }))
    );
  }
  return api;
}

/** 解析 API 对象类型，并拒绝丢失声明导致的空结果。 */
function resolveApiType(node, checker, seen = new Set()) {
  const type = checker.getTypeFromTypeNode(node);
  if (type.flags & (ts.TypeFlags.Any | ts.TypeFlags.Unknown)) throw new Error(`无法解析 API 类型：${node.getText()}`);
  const visit = child => {
    if (ts.isTypeReferenceNode(child) || ts.isExpressionWithTypeArguments(child)) {
      const name = child.typeName ?? child.expression;
      let symbol = checker.getSymbolAtLocation(name);
      if (!symbol) throw new Error(`无法解析 API 类型：${name.getText()}`);
      if (symbol.flags & ts.SymbolFlags.Alias) symbol = checker.getAliasedSymbol(symbol);
      if (!symbol.declarations?.length) throw new Error(`无法解析 API 类型：${name.getText()}`);
      for (const declaration of symbol.declarations) {
        if (seen.has(declaration)) continue;
        seen.add(declaration);
        if (ts.isInterfaceDeclaration(declaration)) {
          for (const clause of declaration.heritageClauses ?? []) for (const parent of clause.types) visit(parent);
        } else if (ts.isTypeAliasDeclaration(declaration)) {
          visit(declaration.type);
        }
      }
    }
    if (ts.isIntersectionTypeNode(child) || ts.isUnionTypeNode(child)) for (const part of child.types) visit(part);
    if (ts.isTypeReferenceNode(child)) for (const argument of child.typeArguments ?? []) visit(argument);
  };
  visit(node);
  return type;
}

/** 保留解析后的属性类型及源码中的中文说明。 */
function typeEntries(type, node, checker, unwrapRefs = false) {
  return type.getProperties().map(property => {
    let propertyType = checker.getTypeOfSymbolAtLocation(property, node);
    if (
      unwrapRefs &&
      /^(?:Ref|ShallowRef|ComputedRef|WritableComputedRef)$/.test(
        propertyType.aliasSymbol?.name ?? propertyType.symbol?.name ?? ""
      )
    ) {
      propertyType = checker.getTypeOfSymbolAtLocation(propertyType.getProperty("value"), node);
    }
    const optional = Boolean(property.flags & ts.SymbolFlags.Optional);
    let text = checker.typeToString(propertyType, node, typeFormat);
    if (optional) text = text.replace(/ \| undefined$/, "");
    if (unwrapRefs) {
      let declaration = property.valueDeclaration ?? property.declarations?.[0];
      if (declaration && ts.isShorthandPropertyAssignment(declaration))
        declaration = checker.getShorthandAssignmentValueSymbol(declaration)?.valueDeclaration;
      else if (declaration && ts.isPropertyAssignment(declaration) && ts.isIdentifier(declaration.initializer))
        declaration = checker.getSymbolAtLocation(declaration.initializer)?.valueDeclaration;
      const initializer = declaration && ts.isVariableDeclaration(declaration) ? declaration.initializer : undefined;
      if (
        initializer &&
        ts.isCallExpression(initializer) &&
        /^(?:ref|shallowRef|computed)$/.test(initializer.expression.getText()) &&
        initializer.typeArguments?.[0]
      ) {
        text = `${initializer.typeArguments[0].getText()}${initializer.arguments.length ? "" : " | undefined"}`;
      }
    }
    return { name: property.name, description: symbolDescription(property, checker), type: compact(text), required: !optional };
  });
}

/** 读取文档注释或字段末尾的行注释。 */
function symbolDescription(symbol, checker) {
  const description = ts.displayPartsToString(symbol.getDocumentationComment(checker));
  if (description) return compact(description);
  const declaration = symbol.valueDeclaration ?? symbol.declarations?.[0];
  if (!declaration) return "";
  if (ts.isShorthandPropertyAssignment(declaration)) {
    const target = checker.getShorthandAssignmentValueSymbol(declaration);
    if (target && target !== symbol) return symbolDescription(target, checker);
  }
  const source = declaration.getSourceFile();
  const leading = source.text.slice(declaration.pos, declaration.getStart()).match(/\/\*\*([\s\S]*?)\*\//g);
  if (leading?.length)
    return compact(
      leading
        .at(-1)
        .replace(/^\/\*\*|\*\/$/g, "")
        .replace(/^\s*\*\s?/gm, "")
    );
  const comments = ts.getTrailingCommentRanges(source.text, declaration.end) ?? [];
  return comments
    .map(comment => source.text.slice(comment.pos, comment.end).replace(/^\/\/\s?|^\/\*\*?\s?|\s?\*\/$/g, ""))
    .join(" ");
}

/** 格式化事件参数，保留参数名、可选标记和剩余参数。 */
function parameterText(parameter, node, checker) {
  const declaration = parameter.valueDeclaration ?? parameter.declarations?.[0];
  const optional = declaration?.questionToken || declaration?.initializer;
  let type = checker.typeToString(checker.getTypeOfSymbolAtLocation(parameter, node), node, typeFormat);
  if (optional) type = type.replace(/ \| undefined$/, "");
  return `${declaration?.dotDotDotToken ? "..." : ""}${parameter.name}${optional ? "?" : ""}: ${type}`;
}

/** 读取运行时 props 对象或数组声明。 */
function runtimeProps(node, checker) {
  node = resolveExpression(node, checker);
  if (!node) return [];
  if (ts.isArrayLiteralExpression(node))
    return node.elements.map(element => ({ name: propertyName(element), description: "", type: "unknown", required: false }));
  return objectProperties(node, checker).map(property => {
    const options =
      ts.isPropertyAssignment(property) && ts.isObjectLiteralExpression(property.initializer)
        ? objectProperties(property.initializer, checker)
        : [];
    const type = options.length
      ? options.find(option => propertyName(option.name) === "type")?.initializer
      : property.initializer;
    const defaultValue = options.find(option => propertyName(option.name) === "default")?.initializer;
    const symbol = checker.getSymbolAtLocation(property.name);
    return {
      name: propertyName(property.name),
      description: symbol ? symbolDescription(symbol, checker) : "",
      type: runtimePropType(type, checker),
      required: options.find(option => propertyName(option.name) === "required")?.initializer?.kind === ts.SyntaxKind.TrueKeyword,
      ...(defaultValue ? { default: compact(defaultValue.getText()) } : {})
    };
  });
}

/** 将 Vue 运行时构造器或 PropType 转换为可读类型。 */
function runtimePropType(node, checker) {
  if (!node || node.kind === ts.SyntaxKind.NullKeyword) return "unknown";
  if (ts.isAsExpression(node) || ts.isTypeAssertionExpression(node)) {
    if (ts.isTypeReferenceNode(node.type) && /(?:^|\.)PropType$/.test(node.type.typeName.getText()))
      return compact(node.type.typeArguments[0].getText());
    return runtimePropType(node.expression, checker);
  }
  node = resolveExpression(node, checker);
  if (ts.isArrayLiteralExpression(node)) return node.elements.map(element => runtimePropType(element, checker)).join(" | ");
  const constructors = {
    String: "string",
    Number: "number",
    Boolean: "boolean",
    Array: "unknown[]",
    Object: "Record<string, unknown>",
    Function: "(...args: any[]) => any",
    Date: "Date",
    Symbol: "symbol",
    BigInt: "bigint"
  };
  const type = constructors[node.getText()];
  if (!type) throw new Error(`无法解析运行时 prop 类型：${node.getText()}`);
  return type;
}

/** 读取运行时 emits 名称和校验函数参数。 */
function runtimeEvents(node, checker) {
  node = resolveExpression(node, checker);
  if (!node) return [];
  if (ts.isArrayLiteralExpression(node))
    return node.elements.map(element => ({ name: propertyName(element), description: "", type: "(...args: unknown[]) => void" }));
  return objectProperties(node, checker).map(property => {
    const type = checker.getTypeAtLocation(ts.isPropertyAssignment(property) ? property.initializer : property);
    const signature = type.getCallSignatures()[0];
    return {
      name: propertyName(property.name),
      description: symbolDescription(checker.getSymbolAtLocation(property.name), checker),
      type: signature
        ? `(${signature.parameters.map(parameter => parameterText(parameter, node, checker)).join(", ")}) => void`
        : "(...args: unknown[]) => void"
    };
  });
}

/** 解析常量引用，避免把非字面量声明误认为没有 API。 */
function resolveExpression(node, checker, seen = new Set()) {
  if (!node) return node;
  if (ts.isParenthesizedExpression(node) || ts.isAsExpression(node) || ts.isSatisfiesExpression(node))
    return resolveExpression(node.expression, checker, seen);
  if (ts.isIdentifier(node)) {
    let symbol = checker.getSymbolAtLocation(node);
    if (symbol?.flags & ts.SymbolFlags.Alias) symbol = checker.getAliasedSymbol(symbol);
    const declaration = symbol?.valueDeclaration;
    if (declaration && ts.isVariableDeclaration(declaration) && declaration.initializer && !seen.has(declaration)) {
      seen.add(declaration);
      return resolveExpression(declaration.initializer, checker, seen);
    }
  }
  return node;
}

/** 展开对象中的常量及展开项，无法解析时显式报错。 */
function objectProperties(node, checker) {
  node = resolveExpression(node, checker);
  if (!node) return [];
  if (!ts.isObjectLiteralExpression(node)) throw new Error(`无法解析 API 对象：${node.getText()}`);
  return node.properties.flatMap(property =>
    ts.isSpreadAssignment(property) ? objectProperties(property.expression, checker) : [property]
  );
}

/** 合并同名事件重载，保留全部可用参数。 */
function addEntry(entries, entry) {
  const existing = entries.find(item => item.name === entry.name);
  if (!existing) entries.push(entry);
  else {
    if (existing.type !== entry.type) existing.type = `(${existing.type}) | (${entry.type})`;
    existing.description ||= entry.description;
  }
}

/** 从 Vue 模板 AST 提取静态、默认及动态插槽。 */
function collectTemplateSlots(node, entries) {
  if (!node) return;
  if (node.type === 1 && node.tag === "slot") {
    const nameAttribute = node.props.find(prop => prop.type === 6 && prop.name === "name");
    const dynamicName = node.props.find(prop => prop.type === 7 && prop.name === "bind" && prop.arg?.content === "name");
    const name = dynamicName ? `[${dynamicName.exp?.content ?? "name"}]` : (nameAttribute?.value?.content ?? "default");
    if (!entries.some(entry => entry.name === name)) {
      const bindings = node.props.filter(prop => prop.type === 7 && prop.name === "bind" && prop !== dynamicName);
      const fields = bindings
        .filter(prop => prop.arg?.isStatic)
        .map(prop => `${/^[\w$]+$/.test(prop.arg.content) ? prop.arg.content : JSON.stringify(prop.arg.content)}: unknown`);
      const scope = [
        ...(bindings.some(prop => !prop.arg?.isStatic) ? ["Record<string, unknown>"] : []),
        ...(fields.length ? [`{ ${fields.join("; ")} }`] : [])
      ];
      entries.push({
        name,
        description: dynamicName ? "动态具名插槽" : "",
        type: scope.length ? `(scope: ${scope.join(" & ")}) => unknown` : "() => unknown"
      });
    }
  }
  for (const child of node.children ?? []) collectTemplateSlots(child, entries);
}

/** 获取稳定的属性名称，拒绝无法静态确定的计算属性。 */
function propertyName(node) {
  if (ts.isIdentifier(node) || ts.isStringLiteralLike(node) || ts.isNumericLiteral(node)) return node.text;
  throw new Error(`无法解析 API 名称：${node.getText()}`);
}

/** 折叠源码换行，保持文档表格可读。 */
function compact(text) {
  return text.replace(/("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|`(?:[^`\\]|\\.)*`)|\s+/g, (match, literal) => literal ?? " ").trim();
}

/** 检查组件清单与源码一一对应，忽略用法字符串中的示例字段。 */
export function validateComponentCatalog(filename, files, ids = new Set()) {
  const source = ts.createSourceFile(filename, readFileSync(filename, "utf8"), ts.ScriptTarget.Latest, true);
  const variables = new Map();
  const entries = [];
  const visit = node => {
    if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name)) variables.set(node.name.text, node.initializer);
    if (ts.isCallExpression(node) && ts.isIdentifier(node.expression) && node.expression.text === "componentDoc") {
      entries.push(
        Object.fromEntries(
          ["id", "label", "group", "file", "description", "usage", "code"].map((key, index) => [key, node.arguments[index]])
        )
      );
    }
    if (ts.isObjectLiteralExpression(node)) {
      const fields = Object.fromEntries(
        node.properties.filter(ts.isPropertyAssignment).map(property => [propertyName(property.name), property.initializer])
      );
      if (fields.file && fields.id) entries.push(fields);
    }
    ts.forEachChild(node, visit);
  };
  visit(source);
  const seen = new Set();
  for (const entry of entries) {
    const file = entry.file && ts.isStringLiteralLike(entry.file) ? entry.file.text : "未声明文件";
    for (const key of ["id", "label", "file", "description", "usage", "code"]) {
      let value = entry[key];
      const references = new Set();
      while (value && ts.isIdentifier(value) && !references.has(value.text)) {
        references.add(value.text);
        value = variables.get(value.text);
      }
      if (!value || (ts.isStringLiteralLike(value) && !value.text.trim()))
        throw new Error(`${filename}: ${file} 缺少有效 ${key}`);
    }
    if (seen.has(file)) throw new Error(`${filename}: 重复 file ${file}`);
    if (!files.includes(file)) throw new Error(`${filename}: 源码不存在 ${file}`);
    const id = entry.id.text;
    if (ids.has(id)) throw new Error(`${filename}: 重复 id ${id}`);
    seen.add(file);
    ids.add(id);
  }
  const missing = files.filter(file => !seen.has(file));
  if (missing.length) throw new Error(`${filename}: 缺少组件文档 ${missing.join(", ")}`);
}

/** 生成每个包的 API JSON，检查模式同时验证文档覆盖和内容新鲜度。 */
function generateComponentDocs(check) {
  const packages = packageRoots.map(path => {
    const root = join(adminRoot, path);
    const components = join(root, "src/components");
    const files = readdirSync(components, { recursive: true })
      .filter(path => path.endsWith(".vue"))
      .sort()
      .map(path => join(components, path));
    return { root, components, files, docs: join(root, path === "packages/core" ? "src/views/demo/docs/examples" : "src/demo") };
  });
  const ids = new Set();
  for (const item of packages)
    validateComponentCatalog(
      join(item.docs, "catalog.ts"),
      item.files.map(file => relative(item.components, file).split(sep).join("/")),
      ids
    );
  const apis = extractComponentApis(packages.flatMap(item => item.files));
  for (const item of packages) {
    const output = join(item.docs, "component-api.json");
    const json = `${JSON.stringify(Object.fromEntries(item.files.map(file => [relative(item.components, file).split(sep).join("/"), apis.get(file)])), null, 2)}\n`;
    if (check) {
      if (!existsSync(output) || readFileSync(output, "utf8") !== json)
        throw new Error(`${relative(adminRoot, output)} 已过期或组件覆盖不完整，请运行 node scripts/generate-component-docs.mjs`);
    } else {
      mkdirSync(dirname(output), { recursive: true });
      writeFileSync(output, json);
    }
    console.log(`${relative(adminRoot, item.root)}：${item.files.length} 个组件 API ${check ? "检查通过" : "已生成"}`);
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    if (process.argv.slice(2).some(argument => argument !== "--check"))
      throw new Error("用法：node scripts/generate-component-docs.mjs [--check]");
    generateComponentDocs(process.argv.includes("--check"));
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
