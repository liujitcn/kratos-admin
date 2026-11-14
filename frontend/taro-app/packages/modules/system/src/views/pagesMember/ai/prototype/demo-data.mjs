/** 创建原型的内存数据，刷新或重置后恢复。 */
export function createDemoData() {
  const now = Date.now()
  const tasks = [
    { id: 'weekly', name: '每周项目周报', prompt: '汇总本周项目进展、关键数据和风险，生成一份可下载的工作周报。', status: 'active', frequency: 'weekly', weekdays: [5], time: '18:00', timezone: 'Asia/Shanghai', model: 'GPT 6.1 Sol', knowledge: ['项目资料'], activeRun: '' },
    { id: 'daily', name: '每日运营数据汇总', prompt: '分析昨日运营数据，列出异常指标和建议。', status: 'active', frequency: 'daily', weekdays: [], time: '09:00', timezone: 'Asia/Shanghai', model: 'GPT 6.1 Sol', knowledge: [], activeRun: 'run-running' },
    { id: 'reminder', name: '周会准备提醒', prompt: '在每周例会前创建一条日历提醒，附上需要准备的资料。', status: 'attention', frequency: 'weekdays', weekdays: [], time: '08:30', timezone: 'Asia/Shanghai', model: 'GPT 6.1 Sol', knowledge: ['团队知识库'], activeRun: 'run-approval' },
    { id: 'notes', name: '会议纪要整理', prompt: '整理最近一次项目会议纪要。', status: 'paused', frequency: 'weekly', weekdays: [1], time: '17:00', timezone: 'Asia/Shanghai', model: 'GPT 6.1 Sol', knowledge: [], activeRun: '' },
    { id: 'old', name: '上月项目复盘', prompt: '整理上月项目复盘结论。', status: 'archived', frequency: 'once', onceAt: '2026-10-31T18:00', weekdays: [], time: '18:00', timezone: 'Asia/Shanghai', model: 'GPT 6.1 Sol', knowledge: [], activeRun: '' },
  ]
  const snapshot = id => structuredClone(tasks.find(task => task.id === id))
  const runs = [
    { id: 'run-success', taskId: 'weekly', status: 'success', createdAt: now - 3600000, summary: '已整理项目进展、3 项风险与下周行动计划。', snapshot: snapshot('weekly'), output: '本周完成接口联调与核心页面验收。\n\n需要关注的事项：\n1. 外部接口响应延迟。\n2. 测试数据覆盖不足。\n3. 下周发布窗口待确认。\n\n建议先完成回归测试，再确认发布计划。', fileId: 'generated-weekly' },
    { id: 'run-running', taskId: 'daily', status: 'running', createdAt: now - 45000, summary: '正在读取运营数据并汇总指标。', snapshot: snapshot('daily'), output: '' },
    { id: 'run-approval', taskId: 'reminder', status: 'waiting_approval', createdAt: now - 120000, summary: '创建日历提醒前需在管理端确认。', snapshot: snapshot('reminder'), output: '', approval: { tool: '创建日历提醒', arguments: { title: '项目周会准备', time: '下周一 08:30', description: '准备项目进度和风险清单' }, expiresAt: now + 600000 } },
    { id: 'run-failed', taskId: 'daily', status: 'failed', createdAt: now - 86400000, summary: '数据源暂时不可用。', error: 'SOURCE_UNAVAILABLE', snapshot: snapshot('daily'), output: '' },
    { id: 'run-unknown', taskId: 'reminder', status: 'unknown', createdAt: now - 172800000, summary: '外部连接在返回结果前中断，请先核对日历中是否已有提醒。', error: 'EXTERNAL_RESULT_UNKNOWN', snapshot: snapshot('reminder'), output: '' },
  ]
  const weeklyText = '# 每周项目周报\n\n## 进展\n完成接口联调与核心页面验收。\n\n## 风险\n- 外部接口响应延迟\n- 测试数据覆盖不足\n\n## 下周计划\n完成回归测试，确认发布窗口。\n'
  const files = [
    { id: 'uploaded-csv', name: '运营数据.csv', size: 8240, source: 'uploaded', createdAt: now - 300000, trashed: false, type: 'text/csv', content: '日期,项目,访问量,成交数\n2026-11-10,A项目,1250,86\n2026-11-11,A项目,1380,94\n2026-11-12,A项目,1420,103\n' },
    { id: 'generated-weekly', name: '每周项目周报.md', size: 2048, source: 'generated', createdAt: now - 3600000, trashed: false, type: 'text/markdown', content: weeklyText, sessionId: 'session-weekly' },
    { id: 'uploaded-guide', name: '项目说明.md', size: 4280, source: 'uploaded', createdAt: now - 86400000, trashed: false, type: 'text/markdown', content: '# 项目说明\n\n本项目包含用户管理、数据分析与消息提醒功能。\n\n验收重点：权限范围、数据一致性与多端体验。' },
    { id: 'generated-summary', name: '会议行动项.txt', size: 1890, source: 'generated', createdAt: now - 86400000, trashed: false, type: 'text/plain', content: '会议行动项\n1. 完成接口回归测试。\n2. 确认发布负责人。\n3. 周五前更新项目风险清单。' },
    { id: 'trashed-note', name: '旧版会议纪要.md', size: 3100, source: 'uploaded', createdAt: now - 604800000, trashed: true, type: 'text/markdown', content: '# 旧版会议纪要\n\n此版本已被新版会议纪要替代。' },
  ]
  const tools = [
    { id: 'query', pluginId: 'business', name: '查询项目数据', description: '查询当前身份可见的项目指标和趋势。', effect: 'read', allowed: true, schema: { type: 'object', properties: { question: { type: 'string', description: '需要分析的问题' } }, required: ['question'] }, prompts: '先确认时间范围和项目范围。' },
    { id: 'read-file', pluginId: 'files', name: '读取文件', description: '读取对话附件和有权访问的文件内容。', effect: 'read', allowed: true, schema: { type: 'object', properties: { file_id: { type: 'string' } }, required: ['file_id'] }, prompts: '' },
    { id: 'create-file', pluginId: 'files', name: '生成文件', description: '根据结果生成可下载的文档。', effect: 'write', allowed: true, schema: { type: 'object', properties: { name: { type: 'string' }, content: { type: 'string' } }, required: ['name', 'content'] }, prompts: '保存前明确文件名称。' },
    { id: 'calendar-read', pluginId: 'calendar', name: '查看日历', description: '读取已连接账号的日程。', effect: 'read', allowed: true, schema: { type: 'object', properties: { date: { type: 'string' } } }, prompts: '' },
    { id: 'calendar-create', pluginId: 'calendar', name: '创建日历提醒', description: '在已连接日历账号中创建提醒。', effect: 'write', allowed: true, schema: { type: 'object', properties: { title: { type: 'string' }, time: { type: 'string' } }, required: ['title', 'time'] }, prompts: '执行前确认提醒标题和时间。' },
    { id: 'docs-search', pluginId: 'docs', name: '检索团队文档', description: '搜索账号有权访问的团队文档。', effect: 'read', allowed: true, schema: { type: 'object', properties: { query: { type: 'string' } }, required: ['query'] }, prompts: '' },
  ]
  const plugins = [
    { id: 'business', code: 'business-data', name: '业务数据', source: 'NATIVE', visibility: 'system', description: '查询项目数据，分析指标和业务趋势。', enabled: true, installed: true, terminals: ['app', 'admin'], endpoint: '' },
    { id: 'files', code: 'file-tools', name: '文件处理', source: 'NATIVE', visibility: 'system', description: '读取附件、整理内容并生成结果文件。', enabled: true, installed: true, terminals: ['app', 'admin'], endpoint: '' },
    { id: 'calendar', code: 'calendar', name: '团队日历', source: 'MCP', visibility: 'tenant', description: '读取团队日程，创建会议和提醒。', enabled: true, installed: true, terminals: ['app', 'admin'], endpoint: 'https://calendar.example.com/mcp' },
    { id: 'docs', code: 'team-docs', name: '团队文档', source: 'MCP', visibility: 'private', description: '在已授权的团队资料中搜索文档。', enabled: true, installed: true, terminals: ['app', 'admin'], endpoint: 'https://docs.example.com/mcp' },
    { id: 'desktop', code: 'desktop-report', name: '报表导出', source: 'NATIVE', visibility: 'tenant', description: '导出完整业务报表。', enabled: true, installed: true, terminals: ['admin'], endpoint: '' },
  ]
  const connections = [
    { id: 'calendar-account', pluginId: 'calendar', label: '团队工作日历', state: 'connected', sharing: 'tenant', auth: 'oauth', scopes: 'calendar.read, calendar.write', error: '', updatedAt: now - 1800000 },
    { id: 'docs-account', pluginId: 'docs', label: '我的文档账号', state: 'expired', sharing: 'personal', auth: 'token', scopes: 'documents.read', error: '凭据已过期，需要在管理端更新连接。', updatedAt: now - 86400000 },
  ]
  const sessions = [
    { id: 'session-intro', title: '验证去reasoning', summary: '用一句话介绍你自己', updatedAt: now, model: 'GPT 6.1 Sol', knowledge: [], messages: [{ id: 'intro-user', role: 'user', text: '用一句话介绍你自己' }, { id: 'intro-ai', role: 'ai', text: '我是 Codex，一名专注于协作编程、代码分析与问题解决的 AI 助手。', model: 'GPT 6.1 Sol', activities: [], files: [] }] },
    { id: 'session-plan', title: '本周工作计划', summary: '帮我整理本周的工作安排', updatedAt: now - 86400000, model: 'GPT 6.1 Sol', knowledge: [], messages: [{ id: 'plan-user', role: 'user', text: '帮我整理本周的工作安排' }, { id: 'plan-ai', role: 'ai', text: '建议优先完成接口回归测试，再安排页面验收和发布准备。', model: 'GPT 6.1 Sol', activities: [], files: [] }] },
    { id: 'session-weekly', title: '每周项目周报', summary: '已生成每周项目周报', updatedAt: now - 3600000, model: 'GPT 6.1 Sol', knowledge: ['项目资料'], messages: [{ id: 'weekly-ai', role: 'ai', text: runs[0].output, model: 'GPT 6.1 Sol', activities: [{ name: '生成文件', description: '已保存每周项目周报.md' }], files: ['generated-weekly'] }] },
  ]
  return { tasks, runs, files, plugins, connections, tools, sessions }
}

/** 预览下一批触发时间，按任务的 IANA 时区计算。 */
export function previewSchedule(task, now = Date.now()) {
  const formatter = new Intl.DateTimeFormat('en-CA', { timeZone: task.timezone, year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23' })
  const partsAt = timestamp => Object.fromEntries(formatter.formatToParts(timestamp).filter(part => part.type !== 'literal').map(part => [part.type, Number(part.value)]))
  const toInstant = (year, month, day, hour, minute) => {
    const wallTime = Date.UTC(year, month - 1, day, hour, minute)
    let instant = wallTime
    for (let index = 0; index < 3; index++) {
      const parts = partsAt(instant)
      const difference = wallTime - Date.UTC(parts.year, parts.month - 1, parts.day, parts.hour, parts.minute, parts.second)
      instant += difference
      if (!difference) break
    }
    return instant
  }
  if (task.frequency === 'once') {
    const match = task.onceAt?.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/)
    if (!match) return []
    const instant = toInstant(...match.slice(1).map(Number))
    return instant > now ? [instant] : []
  }
  if (!/^([01]\d|2[0-3]):[0-5]\d$/.test(task.time)) return []
  const [hour, minute] = task.time.split(':').map(Number)
  const today = partsAt(now)
  const times = []
  for (let index = 0; index < 35 && times.length < 3; index++) {
    const date = new Date(Date.UTC(today.year, today.month - 1, today.day + index))
    const weekday = date.getUTCDay() || 7
    if (task.frequency === 'weekdays' && weekday > 5) continue
    if (task.frequency === 'weekly' && !task.weekdays.includes(weekday)) continue
    const instant = toInstant(date.getUTCFullYear(), date.getUTCMonth() + 1, date.getUTCDate(), hour, minute)
    if (instant > now) times.push(instant)
  }
  return times
}
