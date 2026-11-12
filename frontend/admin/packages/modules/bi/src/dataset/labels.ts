/** 按日期分组值生成短标签，只改变显示文本，保留原始分类和提示数据。 */
export function createDimensionLabelFormatter(values: unknown[], field = ""): (value: unknown) => string {
  const pattern = /^(\d{4})-(\d{2})(?:-(\d{2})(?:[ T](\d{2})(?::(\d{2}))?(?::(\d{2})(?:\.\d+)?)?(?:Z|[+-]\d{2}:?\d{2})?)?)?$/;
  const dates = values.map(value => pattern.exec(String(value ?? ""))).filter(value => value !== null);
  const month = /(?:^|_)(?:month|monthly|ym)(?:_|$)|月/i.test(field);
  const dayCounts = new Map<string, number>();
  for (const date of dates) {
    const day = date[0].slice(0, 10);
    dayCounts.set(day, (dayCounts.get(day) ?? 0) + 1);
  }
  const repeatedDay = [...dayCounts.values()].some(count => count > 1);
  const includeDate = dayCounts.size > 1 && repeatedDay;
  const hour =
    /(?:^|_)(?:hour|hourly)(?:_|$)|小时/i.test(field) ||
    includeDate ||
    (dates.length > 0 &&
      dates.every(date => date[3] && date[0].slice(0, 10) === dates[0]![0].slice(0, 10)) &&
      dates.some(date => Number(date[4]) || Number(date[5]) || Number(date[6])));
  return value => {
    const text = String(value ?? "");
    const date = pattern.exec(text);
    if (!date) return text;
    if (month || !date[3]) return `${date[2]}月`;
    if (hour && date[4]) return includeDate ? `${date[2]}/${date[3]} ${date[4]}时` : `${date[4]}时`;
    return `${date[2]}/${date[3]}`;
  };
}
