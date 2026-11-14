import { createDemoData, previewSchedule } from './demo-data.mjs'

const $ = selector => document.querySelector(selector)
const query = new URLSearchParams(window.location.search)
const labels = { active: '运行中', paused: '已暂停', attention: '需处理', archived: '已归档', queued: '排队中', running: '执行中', waiting_approval: '待确认', success: '已完成', failed: '执行失败', canceled: '已停止', unknown: '结果未知', connected: '已连接', expired: '已过期', disconnected: '已断开', system: '系统', tenant: '租户共享', private: '个人', personal: '个人', uploaded: '上传文件', generated: '生成文件' }
const icons = { calendar: '\ue6c0', files: '\ue644', plugin: '\ue6ce', search: '\ue654', more: '\ue64d', plus: '\ue67b', arrow: '\ue6b5', check: '\ue62c', refresh: '\ue657' }
let data = createDemoData()
const state = { view: 'chat', page: 'list', target: '', activeSession: 'session-intro', search: '', filter: '', fileTab: 'all', pluginTab: 'catalog', scope: '', sessionSearch: '', drafts: {}, back: [], timers: new Map(), uploadTarget: 'chat', sheet: null, lastFocus: null }
let toastTimer

/** 转义用户输入及文件内容，只以文本渲染。 */
function escape(value = '') {
  return String(value).replace(/[&<>"']/g, character => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[character])
}

function icon(name, iconSize = 20) {
  return `<span class="kratos-uni-icon" aria-hidden="true" style="font-size:${iconSize}px">${icons[name] || icons.files}</span>`
}

function button(label, action, id = '', { primary = false, danger = false, disabled = false } = {}) {
  return `<button type="button" class="preview-btn${primary ? ' is-primary' : ''}${danger ? ' is-danger' : ''}" data-action="${action}" data-id="${escape(id)}"${disabled ? ' disabled' : ''}>${escape(label)}</button>`
}

function badge(value, text = labels[value] || value) {
  return `<span class="preview-badge ${['failed', 'expired', 'unknown', 'attention', 'waiting_approval'].includes(value) ? 'is-warning' : ''}">${escape(text)}</span>`
}

function time(value, timezone = 'Asia/Shanghai') {
  return value ? new Intl.DateTimeFormat('zh-CN', { timeZone: timezone, month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).format(new Date(value)) : '—'
}

function size(value) {
  return value < 1024 ? `${value} B` : value < 1048576 ? `${(value / 1024).toFixed(1)} KB` : `${(value / 1048576).toFixed(1)} MB`
}

function empty(text = '没有找到匹配的内容') {
  return `<div class="preview-empty">${escape(text)}</div>`
}

function tabs(items, action, selected) {
  return `<div class="k-chips preview-tabs" role="tablist">${items.map(([value, name]) => `<button type="button" role="tab" aria-selected="${value === selected}" class="k-ch${value === selected ? ' k-ch--pri' : ''}" data-action="${action}" data-id="${value}">${escape(name)}</button>`).join('')}</div>`
}

function search(placeholder) {
  return `<label class="session-search preview-search">${icon('search', 16)}<input id="workspace-search" class="session-search-input" type="search" aria-label="${placeholder}" placeholder="${placeholder}" value="${escape(state.search)}" /></label>`
}

function currentSession() {
  return data.sessions.find(session => session.id === state.activeSession)
}

function draft() {
  return state.drafts[state.activeSession] ||= { text: '', files: [] }
}

function availablePlugin(plugin) {
  if (!plugin.enabled || !plugin.installed) return '未启用'
  if (!plugin.terminals.includes('app')) return '未开放应用端'
  if (plugin.source === 'MCP' && !data.connections.some(connection => connection.pluginId === plugin.id && connection.state === 'connected')) return '需要连接账号'
  if (!data.tools.some(tool => tool.pluginId === plugin.id && tool.allowed)) return '暂无可用工具'
  return ''
}

function setDrawer(open) {
  $('#drawer').classList.toggle('is-open', open)
  $('#drawer').inert = !open
  $('#drawer').setAttribute('aria-hidden', String(!open))
  $('#mask').hidden = !open
}

function setTheme(theme) {
  const resolved = theme === 'dark' ? 'dark' : 'light'
  document.documentElement.className = `theme-${resolved}`
  document.querySelectorAll('[data-theme]').forEach(item => item.setAttribute('aria-pressed', String(item.dataset.theme === resolved)))
  const params = new URLSearchParams(window.location.search)
  params.set('theme', resolved)
  window.history.replaceState(null, '', `?${params}`)
}

function navigate(view, page = 'list', target = '', remember = true) {
  if (remember) state.back.push({ view: state.view, page: state.page, target: state.target })
  Object.assign(state, { view, page, target, search: '', filter: '' })
  setDrawer(false)
  render()
  $('#content').scrollTop = 0
  const params = new URLSearchParams(window.location.search)
  params.set('view', view)
  params.delete('drawer')
  window.history.replaceState(null, '', `?${params}`)
}

function renderSessions() {
  $('#session-list').innerHTML = data.sessions.filter(session => `${session.title} ${session.summary}`.includes(state.sessionSearch)).sort((left, right) => right.updatedAt - left.updatedAt).map(session => `<div class="session-item${session.id === state.activeSession ? ' is-active' : ''}" data-action="select-session" data-id="${session.id}" role="button" tabindex="0"><div class="session-content"><div class="session-row"><div class="session-title">${escape(session.title)}</div><div class="session-time">${time(session.updatedAt)}</div></div><div class="session-summary">${escape(session.summary || '新对话')}</div></div></div>`).join('') || empty('没有找到会话')
  const pending = data.runs.filter(run => run.status === 'waiting_approval').length
  $('#pending-count').textContent = pending || ''
  $('#pending-count').hidden = !pending
}

function render() {
  const focused = document.activeElement
  const selection = focused?.id === 'workspace-search' ? focused.selectionStart : null
  const isChat = state.view === 'chat'
  $('#content').classList.toggle('is-workspace', !isChat)
  $('#page-title').textContent = isChat ? 'AI 助手' : state.view === 'files' ? '文件' : state.view === 'plugins' ? '插件' : state.page === 'runs' ? '执行记录' : state.page === 'run' ? '执行详情' : '定时任务'
  $('#composer').hidden = !isChat || Boolean(currentSession()?.readOnly)
  $('#content').innerHTML = isChat ? renderChat() : state.view === 'automations' ? renderAutomations() : state.view === 'files' ? renderFiles() : renderPlugins()
  document.querySelectorAll('[data-view]').forEach(item => item.classList.toggle('is-active', item.dataset.view === state.view))
  renderSessions()
  if (isChat && currentSession()) {
    $('#message-input').value = draft().text
    $('#model-label').textContent = currentSession().model
    $('#knowledge-label').textContent = currentSession().knowledge.length ? `知识库 · ${currentSession().knowledge.length}` : '知识库'
    $('#message-attachments').innerHTML = draft().files.map(id => data.files.find(file => file.id === id)).filter(Boolean).map(file => `<button type="button" class="composer-attachment" data-action="remove-attachment" data-id="${file.id}">${escape(file.name)} ×</button>`).join('')
    updateSendButton()
  }
  if (selection !== null && $('#workspace-search')) {
    $('#workspace-search').focus()
    $('#workspace-search').setSelectionRange(selection, selection)
  }
}

function scheduleLabel(task) {
  const frequency = { daily: '每天', weekdays: '工作日', weekly: `每周${(task.weekdays || []).map(day => '一二三四五六日'[day - 1]).join('、')}`, once: '仅一次' }[task.frequency]
  return `${frequency} ${task.frequency === 'once' ? task.onceAt?.replace('T', ' ') || '未设置' : task.time}`
}

function renderAutomations() {
  if (state.page === 'runs') return renderRuns()
  if (state.page === 'run') return renderRun()
  const tasks = data.tasks.filter(task => (!state.filter || task.status === state.filter) && `${task.name} ${task.prompt}`.includes(state.search))
  return `<div class="preview-management">${tabs([['', '全部'], ['active', '运行中'], ['paused', '已暂停'], ['attention', '需处理'], ['archived', '已归档']], 'task-filter', state.filter)}${search('搜索定时任务')}<div class="preview-list">${tasks.map(task => {
    const next = task.status === 'active' ? previewSchedule(task)[0] : null
    return `<article class="session-item workspace-record"><div class="preview-record-head"><button class="preview-record-title" type="button" data-action="task-detail" data-id="${task.id}">${escape(task.name)}</button>${badge(task.status)}</div><p class="preview-description">${escape(task.prompt)}</p><div class="preview-meta">${escape(scheduleLabel(task))} · ${escape(task.timezone)}</div><div class="preview-meta">下次执行：${next ? time(next, task.timezone) : '—'}</div>${task.status === 'attention' ? '<div class="preview-hint">待管理端处理确认请求。</div>' : ''}<div class="preview-row-actions">${button('查看详情', 'task-detail', task.id)}${button('执行记录', 'runs', task.id)}</div></article>`
  }).join('') || empty()}</div></div>`
}

function renderRuns() {
  const runs = data.runs.filter(run => !state.target || run.taskId === state.target).sort((left, right) => right.createdAt - left.createdAt)
  return `<div class="preview-management"><div class="preview-toolbar"><span class="preview-meta">${state.target ? escape(data.tasks.find(task => task.id === state.target)?.name) : '全部任务'} · ${runs.length} 条记录</span>${button('刷新', 'refresh')}</div><div class="preview-list">${runs.map(run => `<article class="session-item workspace-record"><div class="preview-record-head"><strong>${escape(run.snapshot.name)}</strong>${badge(run.status)}</div><div class="preview-meta">${time(run.createdAt, run.snapshot.timezone)}</div><p class="preview-description">${escape(run.summary)}</p><div class="preview-row-actions">${button('详情', 'run-detail', run.id)}${button('查看结果', 'run-result', run.id, { primary: true, disabled: ['running', 'queued', 'failed', 'canceled'].includes(run.status) })}</div></article>`).join('') || empty('还没有执行记录')}</div></div>`
}

function renderRun() {
  const run = data.runs.find(item => item.id === state.target)
  if (!run) return empty('执行记录不存在')
  return `<div class="preview-management"><article class="session-item workspace-record"><div class="preview-record-head"><strong>${escape(run.snapshot.name)}</strong>${badge(run.status)}</div><p class="preview-description">${escape(run.summary)}</p>${run.error ? `<div class="preview-hint is-warning">${escape(run.error)}</div>` : ''}<div class="preview-meta">执行时间：${time(run.createdAt, run.snapshot.timezone)}</div><div class="preview-meta">模型：${escape(run.snapshot.model)}</div><div class="preview-meta">时区：${escape(run.snapshot.timezone)}</div><div class="preview-meta">知识库：${escape(run.snapshot.knowledge.join('、') || '未指定')}</div><h3>本次执行指令</h3><pre class="preview-pre">${escape(run.snapshot.prompt)}</pre><div class="preview-row-actions">${button('查看结果', 'run-result', run.id, { primary: true, disabled: ['running', 'queued', 'failed', 'canceled'].includes(run.status) })}</div></article>${run.status === 'waiting_approval' ? approvalCard(run) : ''}${run.status === 'unknown' ? '<div class="preview-hint is-warning">外部执行结果尚不确定，需在管理端核对处理。</div>' : ''}</div>`
}

function renderFiles() {
  const totals = { all: data.files.filter(file => !file.trashed).length, uploaded: data.files.filter(file => !file.trashed && file.source === 'uploaded').length, generated: data.files.filter(file => !file.trashed && file.source === 'generated').length, trash: data.files.filter(file => file.trashed).length }
  const files = data.files.filter(file => state.fileTab === 'trash' ? file.trashed : !file.trashed && (state.fileTab === 'all' || file.source === state.fileTab)).filter(file => file.name.includes(state.search)).sort((left, right) => right.createdAt - left.createdAt)
  return `<div class="preview-management">${tabs([['all', `全部 ${totals.all}`], ['uploaded', `上传 ${totals.uploaded}`], ['generated', `生成 ${totals.generated}`], ['trash', `回收站 ${totals.trash}`]], 'file-tab', state.fileTab)}${search('搜索文件')}<div class="preview-list">${files.map(file => `<article class="session-item"><div class="preview-file-icon">${icon('files', 23)}</div><div class="session-content"><button type="button" class="session-title preview-file-title" data-action="file-preview" data-id="${file.id}">${escape(file.name)}</button><div class="session-summary">${escape(size(file.size))} · ${labels[file.source]}</div><div class="preview-meta">${time(file.createdAt)} · 个人空间</div></div><button class="session-more" type="button" aria-label="查看${escape(file.name)}" data-action="file-preview" data-id="${file.id}">${icon('arrow', 20)}</button></article>`).join('') || empty(state.fileTab === 'trash' ? '回收站为空' : '没有找到文件')}</div></div>`
}

function renderPlugins() {
  let body
  if (state.pluginTab === 'catalog') {
    const plugins = data.plugins.filter(plugin => (!state.scope || plugin.visibility === state.scope) && `${plugin.name} ${plugin.description}`.includes(state.search))
    body = `${search('搜索插件')}<div class="preview-toolbar"><label class="preview-inline-field">范围<select id="plugin-scope"><option value="">全部范围</option>${['system', 'tenant', 'private'].map(value => `<option value="${value}"${state.scope === value ? ' selected' : ''}>${labels[value]}</option>`).join('')}</select></label></div><div class="preview-list">${plugins.map(plugin => {
      const reason = availablePlugin(plugin)
      return `<article class="session-item workspace-record"><div class="preview-record-head"><button type="button" class="preview-record-title" data-action="plugin-detail" data-id="${plugin.id}">${escape(plugin.name)}</button>${badge(plugin.source, plugin.source)}</div><p class="preview-description">${escape(plugin.description)}</p><div class="preview-record-head"><span class="preview-meta">${labels[plugin.visibility]}</span>${badge(reason ? 'attention' : 'active', reason || '可自动调用')}</div><div class="preview-row-actions">${button('详情', 'plugin-detail', plugin.id)}${plugin.source === 'MCP' ? button('账号', 'plugin-accounts', plugin.id) : ''}${button('工具', 'plugin-tools', plugin.id)}</div></article>`
    }).join('') || empty()}</div>`
  } else if (state.pluginTab === 'connections') {
    const connections = data.connections.filter(connection => (!state.target || connection.pluginId === state.target) && `${connection.label} ${data.plugins.find(plugin => plugin.id === connection.pluginId)?.name}`.includes(state.search))
    body = `${search('搜索账号')}<div class="preview-list">${connections.map(connection => `<article class="session-item workspace-record"><div class="preview-record-head"><strong>${escape(connection.label)}</strong>${badge(connection.state)}</div><div class="preview-meta">${escape(data.plugins.find(plugin => plugin.id === connection.pluginId)?.name)} · ${labels[connection.sharing]}</div><div class="preview-meta">授权范围：${escape(connection.scopes || '—')}</div>${connection.error ? `<div class="preview-hint is-warning">${escape(connection.error)}</div>` : ''}<div class="preview-row-actions">${button('查看详情', 'connection-detail', connection.id)}</div></article>`).join('') || empty('暂时没有连接账号')}</div>`
  } else {
    const tools = data.tools.filter(tool => (!state.target || tool.pluginId === state.target) && `${tool.name} ${tool.description}`.includes(state.search))
    body = `${search('搜索工具')}<div class="preview-list">${tools.map(tool => `<article class="session-item workspace-record"><div class="preview-record-head"><strong>${escape(tool.name)}</strong>${badge(tool.allowed ? 'active' : 'paused', tool.allowed ? '已启用' : '已禁用')}</div><div class="preview-meta">${escape(data.plugins.find(plugin => plugin.id === tool.pluginId)?.name)} · ${tool.effect === 'read' ? '只读' : '执行前确认'}</div><p class="preview-description">${escape(tool.description)}</p><div class="preview-row-actions">${button('参数与策略', 'tool-detail', tool.id)}</div></article>`).join('') || empty()}</div>`
  }
  return `<div class="preview-management">${tabs([['catalog', '插件目录'], ['connections', '连接账号'], ['tools', '工具']], 'plugin-tab', state.pluginTab)}<p class="preview-meta">对话根据语意自动调用已启用且可用的工具。</p>${body}</div>`
}

function approvalCard(run) {
  const expired = run.approval.expiresAt <= Date.now()
  return `<section class="preview-approval"><div class="preview-record-head"><strong>操作确认信息</strong>${badge('waiting_approval', expired ? '已过期' : '待管理端确认')}</div><p>${escape(run.approval.tool)}</p><pre class="preview-pre">${escape(JSON.stringify(run.approval.arguments, null, 2))}</pre><div class="preview-meta">有效期至 ${time(run.approval.expiresAt)}</div></section>`
}

function renderChat() {
  const session = currentSession()
  if (!session) return empty('请选择或新建会话')
  const run = session.runId ? data.runs.find(item => item.id === session.runId) : null
  const banner = session.readOnly ? '<div class="preview-result-banner"><span>定时任务结果 · 只读</span></div>' : ''
  if (!session.messages.length) return `<div class="welcome-panel"><div class="welcome-bubble is-intro">你好！今天有什么需要我协助的吗？</div><div class="prompt-card"><div class="prompt-card__head"><div><div class="prompt-card__eyebrow">快捷操作</div><div class="prompt-card__title">您可以这样问</div></div></div>${['帮我总结一段内容', '帮我分析一份数据', '帮我制定一个计划', '给我一些灵感'].map((prompt, index) => `<button type="button" class="prompt-item" data-action="prompt" data-id="${index}"><span class="prompt-index">${index + 1}</span><span class="prompt-content"><span class="prompt-text">${prompt}</span><span class="prompt-meta">${index < 2 ? '文本与数据' : '效率助手'}</span></span>${icon('arrow')}</button>`).join('')}</div></div>`
  return `${banner}<div class="chat-list">${session.messages.map(message => `<div class="message-row ${message.role === 'user' ? 'is-user' : 'is-ai'}"><div class="bubble ${message.role === 'user' ? 'user-bubble' : 'ai-bubble'}">${message.role === 'ai' ? `<div class="reply-meta"><span class="reply-tag">模型回复</span><span>${escape(message.model || session.model)}</span></div>` : ''}${message.activities?.length ? `<details class="preview-activities"><summary>已自动调用 ${message.activities.length} 项工具</summary>${message.activities.map(activity => `<div class="preview-activity"><strong>${escape(activity.name)}</strong><span>${escape(activity.description)}</span></div>`).join('')}</details>` : ''}<div class="bubble-content">${escape(message.text)}</div>${message.status === 'generating' ? '<div class="preview-meta" role="status">正在生成…</div>' : ''}${message.role === 'ai' && run?.status === 'waiting_approval' ? approvalCard(run) : ''}${message.files?.length ? `<div class="attachment-list">${message.files.map(id => data.files.find(file => file.id === id)).filter(file => file && !file.trashed).map(file => `<button class="attachment-card" type="button" data-action="file-preview" data-id="${file.id}"><span class="attachment-icon">文件</span><span class="attachment-info"><span class="attachment-name">${escape(file.name)}</span><span class="attachment-meta">${size(file.size)}</span></span></button>`).join('')}</div>` : ''}<div class="preview-message-actions">${button('复制', 'message-copy', message.id)}${button('更多', 'message-menu', message.id)}</div></div></div>`).join('')}</div>`
}

function updateSendButton() {
  const session = currentSession()
  if (!session) return
  const busy = session.messages.some(message => message.status === 'generating')
  const disabled = !busy && !draft().text.trim() && !draft().files.length
  const send = $('#message-send')
  send.disabled = disabled
  send.classList.toggle('is-disabled', disabled)
  send.dataset.action = busy ? 'message-stop' : 'message-send'
  send.setAttribute('aria-label', busy ? '停止生成' : '发送')
  send.innerHTML = busy ? '<span class="preview-stop-icon"></span>' : `<span class="kratos-uni-icon" style="font-size:28px;color:var(--kratos-color-${disabled ? 'text-muted' : 'primary'})">&#xe672;</span>`
}

function newSession({ title = '新会话', messages = [], runId = '', readOnly = false } = {}) {
  const session = { id: crypto.randomUUID(), title, summary: messages.at(-1)?.text.slice(0, 100) || '', updatedAt: Date.now(), model: currentSession()?.model || 'GPT 6.1 Sol', knowledge: [...(currentSession()?.knowledge || [])], messages, runId, readOnly }
  data.sessions.unshift(session)
  state.activeSession = session.id
  return session
}

/** 消息和工具调用只在内存演示，不发送请求。 */
function sendMessage(text = draft().text) {
  const session = currentSession()
  if (!session || session.readOnly || session.messages.some(message => message.status === 'generating')) return
  const files = [...draft().files]
  if (!text.trim() && !files.length) return
  session.messages.push({ id: crypto.randomUUID(), role: 'user', text, files })
  if (session.title === '新会话') session.title = text.slice(0, 20) || '文件分析'
  session.summary = text.slice(0, 100) || '分析文件'
  session.updatedAt = Date.now()
  state.drafts[session.id] = { text: '', files: [] }
  generateReply(session, text, files)
}

function generateReply(session, text, files = []) {
  const activities = []
  const candidate = files.length || /文件|附件|总结/.test(text) ? 'files' : /数据|指标|报表|趋势/.test(text) ? 'business' : /日程|日历|提醒/.test(text) ? 'calendar' : ''
  const plugin = data.plugins.find(item => item.id === candidate)
  const usable = plugin && !availablePlugin(plugin)
  const tool = usable ? data.tools.find(item => item.pluginId === plugin.id && item.allowed && item.effect === 'read') : null
  if (tool) activities.push({ name: tool.name, description: `根据问题自动选择${plugin.name}，已完成读取。` })
  const message = { id: crypto.randomUUID(), role: 'ai', text: '正在整理信息…', model: session.model, status: 'generating', activities, files: [] }
  session.messages.push(message)
  render()
  $('#content').scrollTop = $('#content').scrollHeight
  const timer = setTimeout(() => {
    if (message.status !== 'generating') return
    message.status = 'success'
    if (files.length) message.text = `已查看${files.map(id => data.files.find(file => file.id === id)?.name).filter(Boolean).join('、')}。\n\n建议先确认目标和时间范围，再整理关键结论与下一步行动。`
    else if (/数据|指标|报表|趋势/.test(text)) message.text = '近期访问量与成交数持续增长。\n\n1. 优先关注增长明显的项目。\n2. 对异常波动补充原因说明。\n3. 将重点指标加入后续跟踪。'
    else if (/计划|工作安排/.test(text)) message.text = '建议按优先级安排：\n\n1. 先完成接口回归测试。\n2. 再安排页面验收。\n3. 最后确认发布计划与负责人。'
    else message.text = '可以。我会先提炼核心信息，再整理成清晰的结论和行动项。你可以继续补充内容、文件或具体要求。'
    if (plugin && !usable) message.text += `\n\n${plugin.name}当前${availablePlugin(plugin)}，可在侧栏的插件页面查看。`
    state.timers.delete(message.id)
    if (state.view === 'chat' && state.activeSession === session.id) {
      render()
      $('#content').scrollTop = $('#content').scrollHeight
    }
  }, 900)
  state.timers.set(message.id, timer)
}

function openResult(run) {
  let session = data.sessions.find(item => item.runId === run.id)
  if (!session) session = newSession({ title: run.snapshot.name, runId: run.id, readOnly: true, messages: [{ id: crypto.randomUUID(), role: 'user', text: run.snapshot.prompt }, { id: crypto.randomUUID(), role: 'ai', text: run.output || run.summary, model: run.snapshot.model, activities: [], files: run.fileId ? [run.fileId] : [] }] })
  session.model = run.snapshot.model
  session.knowledge = [...run.snapshot.knowledge]
  state.activeSession = session.id
  navigate('chat')
}

function showToast(message) {
  $('#preview-toast').textContent = message
  $('#preview-toast').hidden = false
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { $('#preview-toast').hidden = true }, 2400)
}

function openSheet(title, body, footer = '', context = {}) {
  state.lastFocus = document.activeElement
  state.sheet = context
  $('#sheet-title').textContent = title
  $('#sheet-body').innerHTML = body
  $('#sheet-footer').innerHTML = footer
  $('#sheet-footer').hidden = !footer
  $('#sheet-overlay').hidden = false
  $('#drawer').inert = true
  $('#sheet-close').focus()
}

function closeSheet() {
  $('#sheet-overlay').hidden = true
  $('#sheet-body').innerHTML = ''
  state.sheet = null
  $('#drawer').inert = !$('#drawer').classList.contains('is-open')
  if (state.lastFocus?.isConnected && !state.lastFocus.closest('[inert]')) state.lastFocus.focus()
}

function actionSheet(title, actions) {
  openSheet(title, `<div class="preview-action-list">${actions.map(([label, action, id, danger]) => button(label, action, id, { danger })).join('')}</div>`)
}

function confirmAction(title, description, action, id, danger = false) {
  openSheet(title, `<p class="preview-description">${escape(description)}</p>`, `${button('取消', 'close-sheet')}${button('确认', action, id, { primary: !danger, danger })}`)
}

function openPluginDetail(plugin) {
  const tools = data.tools.filter(tool => tool.pluginId === plugin.id)
  openSheet(plugin.name, `<p class="preview-description">${escape(plugin.description)}</p><div class="preview-meta">来源：${plugin.source} · 范围：${labels[plugin.visibility]}</div><div class="preview-meta">标识：${escape(plugin.code)}</div>${plugin.endpoint ? `<div class="preview-meta">地址：${escape(plugin.endpoint)}</div>` : ''}<div class="preview-meta">启用状态：${plugin.enabled && plugin.installed ? '已启用' : '未启用'}</div><div class="preview-meta">开放终端：${plugin.terminals.map(terminal => terminal === 'app' ? '应用端' : '管理端').join('、') || '无'}</div>${availablePlugin(plugin) ? `<div class="preview-hint is-warning">${availablePlugin(plugin)}</div>` : ''}<h3>工具能力 · ${tools.length}</h3>${tools.map(tool => `<details class="preview-tool"><summary>${escape(tool.name)} ${badge(tool.effect === 'read' ? 'active' : 'attention', tool.effect === 'read' ? '只读' : '执行前确认')}</summary><p>${escape(tool.description)}</p><div class="preview-meta">调用状态：${tool.allowed ? '已启用' : '已禁用'}</div><div class="preview-meta">调用提示：${escape(tool.prompts || '未设置')}</div><pre class="preview-pre">${escape(JSON.stringify(tool.schema, null, 2))}</pre></details>`).join('') || empty('尚未发现工具')}`, `${plugin.source === 'MCP' ? button('查看账号', 'plugin-accounts', plugin.id) : ''}${button('查看工具', 'plugin-tools', plugin.id)}`)
}

function previewFile(file) {
  if (!file) return showToast('文件不存在')
  openSheet(file.name, `<div class="preview-meta">${size(file.size)} · ${labels[file.source]} · ${time(file.createdAt)}</div>${file.trashed ? '<p class="preview-hint">文件位于回收站，仅可查看文件信息。</p>' : `<pre class="preview-pre preview-document">${escape(file.content || '当前文件没有可直接展示的文本内容，可下载查看。')}</pre>`}`, file.trashed ? '' : button('下载', 'file-download', file.id, { primary: true }))
}

function downloadFile(file) {
  const blob = file.blob || new Blob([file.content || ''], { type: file.type || 'text/plain' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = file.name
  link.click()
  setTimeout(() => URL.revokeObjectURL(link.href), 1000)
  showToast('已开始下载')
}

async function uploadFiles(files) {
  const target = state.uploadTarget
  const uploaded = []
  for (const file of files) {
    if (file.size > 20 * 1024 * 1024) {
      showToast(`${file.name} 超过 20 MB`)
      continue
    }
    const text = /\.(csv|md|txt|json|log|svg)$/i.test(file.name) || file.type.startsWith('text/')
    const item = { id: crypto.randomUUID(), name: file.name, type: file.type, size: file.size, source: 'uploaded', createdAt: Date.now(), trashed: false, blob: file, content: text ? (await file.text()).slice(0, 100000) : '' }
    data.files.unshift(item)
    uploaded.push(item.id)
  }
  if (target === 'chat') draft().files.push(...uploaded.slice(0, Math.max(0, 6 - draft().files.length)))
  render()
  if (uploaded.length) showToast(`已添加 ${uploaded.length} 个文件`)
}

async function handleAction(action, id) {
  const task = data.tasks.find(item => item.id === id)
  const run = data.runs.find(item => item.id === id)
  const file = data.files.find(item => item.id === id)
  const plugin = data.plugins.find(item => item.id === id)
  const connection = data.connections.find(item => item.id === id)
  const tool = data.tools.find(item => item.id === id)
  const message = currentSession()?.messages.find(item => item.id === id)
  switch (action) {
    case 'close-sheet': closeSheet(); break
    case 'refresh': render(); showToast('已刷新'); break
    case 'task-filter': state.filter = id; render(); break
    case 'runs': navigate('automations', 'runs', id); break
    case 'run-detail': navigate('automations', 'run', id); break
    case 'task-detail':
      openSheet(task.name, `<div class="preview-record-head">${badge(task.status)}<span class="preview-meta">${escape(scheduleLabel(task))}</span></div><p class="preview-description">${escape(task.prompt)}</p><div class="preview-meta">时区：${escape(task.timezone)}</div><div class="preview-meta">模型：${escape(task.model)}</div><div class="preview-meta">知识库：${escape(task.knowledge.join('、') || '未指定')}</div><div class="preview-meta">工具选择：按执行指令自动匹配</div>`, button('查看执行记录', 'task-open-runs', id))
      break
    case 'task-open-runs': closeSheet(); navigate('automations', 'runs', id); break
    case 'run-result': openResult(run); break
    case 'file-tab': state.fileTab = id; render(); break
    case 'file-preview': previewFile(file); break
    case 'file-download': if (!file.trashed) downloadFile(file); break
    case 'attach': actionSheet('添加附件', [['上传本地文件', 'upload-chat'], ['从文件中选择', 'choose-chat-files']]); break
    case 'upload-chat': state.uploadTarget = 'chat'; closeSheet(); $('#upload-input').click(); break
    case 'choose-chat-files':
      openSheet('选择文件', `<div class="preview-selection-list">${data.files.filter(item => !item.trashed).map(item => `<label class="knowledge-picker__row"><span class="knowledge-picker__row-main"><span class="knowledge-picker__row-name">${escape(item.name)}</span><span class="knowledge-picker__row-meta">${size(item.size)}</span></span><input type="checkbox" name="chat-file" value="${item.id}"${draft().files.includes(item.id) ? ' checked' : ''} /></label>`).join('') || empty('暂无可用文件')}</div>`, button('确定', 'confirm-chat-files', '', { primary: true }))
      break
    case 'confirm-chat-files': {
      const selected = [...document.querySelectorAll('[name="chat-file"]:checked')].map(item => item.value)
      if (selected.length > 6) return showToast('每次最多选择 6 个附件。')
      draft().files = selected; closeSheet(); render(); break
    }
    case 'remove-attachment': draft().files = draft().files.filter(value => value !== id); render(); break
    case 'plugin-tab': state.pluginTab = id; state.target = ''; state.search = ''; render(); break
    case 'plugin-detail': openPluginDetail(plugin); break
    case 'plugin-accounts': closeSheet(); state.pluginTab = 'connections'; navigate('plugins', 'list', id); break
    case 'plugin-tools': closeSheet(); state.pluginTab = 'tools'; navigate('plugins', 'list', id); break
    case 'connection-detail':
      openSheet(connection.label, `<div class="preview-record-head">${badge(connection.state)}${badge('personal', labels[connection.sharing])}</div><div class="k-field"><span class="k-field__k">插件</span><span class="k-field__v">${escape(data.plugins.find(item => item.id === connection.pluginId)?.name)}</span></div><div class="k-field"><span class="k-field__k">认证方式</span><span class="k-field__v">${escape(connection.auth)}</span></div><div class="preview-meta">授权范围：${escape(connection.scopes)}</div><div class="preview-meta">状态更新时间：${time(connection.updatedAt)}</div><p class="preview-description">${escape(connection.error || '连接正常，可用于自动工具调用。')}</p>`)
      break
    case 'tool-detail':
      openSheet(tool.name, `<p class="preview-description">${escape(tool.description)}</p><div>${badge(tool.effect === 'read' ? 'active' : 'attention', tool.effect === 'read' ? '只读工具' : '执行前确认')}${badge(tool.allowed ? 'active' : 'paused', tool.allowed ? '已启用' : '已禁用')}</div><h3>输入参数</h3><pre class="preview-pre">${escape(JSON.stringify(tool.schema, null, 2))}</pre><div class="preview-meta">调用提示：${escape(tool.prompts || '未设置')}</div>`)
      break
    case 'select-session': state.activeSession = id; navigate('chat'); break
    case 'new-session': newSession(); navigate('chat'); break
    case 'prompt': sendMessage(['帮我总结一段内容', '帮我分析一份数据', '帮我制定一个计划', '给我一些灵感'][Number(id)]); break
    case 'message-send': sendMessage(); break
    case 'message-stop':
      currentSession().messages.filter(item => item.status === 'generating').forEach(item => { clearTimeout(state.timers.get(item.id)); state.timers.delete(item.id); item.status = 'canceled'; item.text = '已停止生成。' })
      render(); break
    case 'message-copy':
      try { await navigator.clipboard.writeText(message.text); showToast('已复制') } catch { openSheet('复制内容', `<textarea class="preview-copy-text" readonly>${escape(message.text)}</textarea>`) }
      break
    case 'message-menu': actionSheet(message.role === 'ai' ? '回复信息' : '消息信息', [['复制', 'message-copy', id], ...(message.role === 'ai' ? [['回复信息', 'message-usage', id]] : [])]); break
    case 'message-usage': openSheet('回复信息', `<div class="k-field"><span class="k-field__k">模型</span><span class="k-field__v">${escape(message.model)}</span></div><div class="k-field"><span class="k-field__k">状态</span><span class="k-field__v">${labels[message.status] || '已完成'}</span></div><div class="k-field"><span class="k-field__k">工具调用</span><span class="k-field__v">${message.activities?.length || 0} 项</span></div><p class="preview-meta">Token 与耗时在接入真实响应后显示。</p>`); break
    case 'open-model':
      openSheet('选择模型', `<div class="preview-selection-list">${['GPT 6.1 Sol', '默认模型'].map(name => `<button type="button" class="knowledge-picker__row" data-action="select-model" data-id="${name}"><span class="knowledge-picker__row-name">${name}</span><span class="knowledge-picker__check${currentSession().model === name ? ' is-checked' : ''}">${currentSession().model === name ? '✓' : ''}</span></button>`).join('')}</div>`)
      break
    case 'select-model': currentSession().model = id; closeSheet(); render(); break
    case 'open-knowledge':
      openSheet('选择知识库', `<div class="preview-selection-list">${['项目资料', '团队知识库'].map(name => `<label class="knowledge-picker__row"><span class="knowledge-picker__row-main"><span class="knowledge-picker__row-name">${name}</span><span class="knowledge-picker__row-meta">${name === '项目资料' ? 12 : 28} 篇文档</span></span><input type="checkbox" name="knowledge-choice" value="${name}"${currentSession().knowledge.includes(name) ? ' checked' : ''} /></label>`).join('')}</div>`, button('完成', 'confirm-knowledge', '', { primary: true }))
      break
    case 'confirm-knowledge': currentSession().knowledge = [...document.querySelectorAll('[name="knowledge-choice"]:checked')].map(item => item.value); closeSheet(); render(); break
    case 'voice': openSheet('语音输入', '<form id="workspace-form" data-kind="voice"><label class="preview-field" for="voice-value"><span>语音识别内容</span><textarea id="voice-value" name="value" rows="4" required></textarea><small>原型通过文本模拟识别结果，不调用麦克风。</small></label></form>', '<button class="preview-btn is-primary" type="submit" form="workspace-form">加入输入框</button>'); break
    case 'reset-demo-confirm': confirmAction('重置演示', '将恢复初始示例任务、文件、插件与会话。', 'reset-demo'); break
    case 'reset-demo':
      state.timers.forEach(timer => clearTimeout(timer)); state.timers.clear(); data = createDemoData()
      Object.assign(state, { view: 'chat', page: 'list', target: '', activeSession: 'session-intro', search: '', filter: '', fileTab: 'all', pluginTab: 'catalog', scope: '', sessionSearch: '', drafts: {}, back: [] })
      closeSheet(); setDrawer(false); $('#session-search').value = ''; navigate('chat', 'list', '', false); showToast('演示数据已重置'); break
    default: throw new Error(`未接入的原型操作：${action}`)
  }
}

document.addEventListener('click', event => {
  const target = event.target.closest('[data-action]')
  if (target && !target.disabled) void handleAction(target.dataset.action, target.dataset.id || '').catch(error => { showToast(error.message); console.error(error) })
})
document.addEventListener('input', event => {
  if (event.target.id === 'message-input') { draft().text = event.target.value; updateSendButton() }
  if (event.target.id === 'workspace-search') { state.search = event.target.value; render() }
  if (event.target.id === 'session-search') { state.sessionSearch = event.target.value; renderSessions() }
})
document.addEventListener('change', event => {
  if (event.target.id === 'plugin-scope') { state.scope = event.target.value; render() }
})
document.addEventListener('submit', event => {
  if (event.target.id !== 'workspace-form') return
  event.preventDefault()
  if (event.target.dataset.kind === 'voice') {
    draft().text = new FormData(event.target).get('value'); closeSheet(); render(); return
  }
})
document.addEventListener('keydown', event => {
  if (event.key === 'Escape') { if (!$('#sheet-overlay').hidden) closeSheet(); else setDrawer(false) }
  if (event.key === 'Enter' && event.target.matches('[data-action="select-session"]')) void handleAction('select-session', event.target.dataset.id)
  if (event.key !== 'Tab' || $('#sheet-overlay').hidden) return
  const focusable = [...$('#sheet-overlay').querySelectorAll('button:not(:disabled), input, textarea, select')].filter(item => item.getClientRects().length)
  const first = focusable[0]
  const last = focusable.at(-1)
  if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus() }
  if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus() }
})
$('#drawer-button').addEventListener('click', () => setDrawer(true))
$('#preview-drawer').addEventListener('click', () => setDrawer(true))
$('#mask').addEventListener('click', () => setDrawer(false))
$('#new-chat').addEventListener('click', () => { newSession(); navigate('chat') })
$('#back').addEventListener('click', () => {
  const previous = state.back.pop() || { view: 'chat', page: 'list', target: '' }
  navigate(previous.view, previous.page, previous.target, false)
})
$('#upload-input').addEventListener('change', event => {
  void uploadFiles([...event.target.files]).catch(error => showToast(error.message))
  event.target.value = ''
})
document.querySelectorAll('[data-view]').forEach(item => item.addEventListener('click', () => navigate(item.dataset.view)))
document.querySelectorAll('[data-theme]').forEach(item => item.addEventListener('click', () => setTheme(item.dataset.theme)))
setTheme(query.get('theme') || 'light')
if (['automations', 'files', 'plugins'].includes(query.get('view'))) state.view = query.get('view')
render()
if (query.get('drawer') === '1') setDrawer(true)
