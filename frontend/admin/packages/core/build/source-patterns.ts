/** 为 API 和组件自动导入创建跨平台源码匹配规则，包含 npm 包内源码。 */
export function createSourcePatterns(sourceRoots: string[]): RegExp[] {
  return [...new Set(sourceRoots.map(sourceRoot => sourceRoot.replaceAll("\\", "/")))].map(sourceRoot => {
    const escapedRoot = sourceRoot.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    return new RegExp(`^${escapedRoot.replaceAll("/", "[/\\\\]")}[/\\\\].*\\.(?:vue|[jt]sx?)(?:\\?.*)?$`);
  });
}
