import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

test('SSE 解析器消费 activity 和 approval_required 事件', async () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-uni-stream-'))
  const output = resolve(root, 'stream.mjs')
  try {
    await build({
      entryPoints: [resolve(import.meta.dirname, '../src/views/pagesMember/ai/stream.ts')],
      bundle: true,
      platform: 'node',
      format: 'esm',
      outfile: output,
    })
    const stream = await import(`${pathToFileURL(output).href}?test=${Date.now()}`)
    const events = stream.parseAiEventStreamText(
      'event: activity\ndata: {"session_id":"s1","message_id":"m1","activity":{"id":"tool-1","phase":"tool","status":"running"}}\n\n' +
        'event: approval_required\ndata: {"session_id":"s1","message_id":"m1","messages":[]}\n\n',
    )

    assert.deepEqual(events, [
      {
        event: 'activity',
        payload: {
          session_id: 's1',
          message_id: 'm1',
          activity: { id: 'tool-1', phase: 'tool', status: 'running' },
        },
      },
      {
        event: 'approval_required',
        payload: { session_id: 's1', message_id: 'm1', messages: [] },
      },
    ])
    const { mergeAiToolActivity } = stream
    const tools = []
    assert.equal(
      mergeAiToolActivity(tools, { id: 'reasoning', phase: 'reasoning', status: 'running' }),
      tools,
    )
    assert.equal(
      mergeAiToolActivity(tools, { id: 'answer', phase: 'answering', status: 'running' }),
      tools,
    )
    const activity = { id: 'tool-1', phase: 'tool', status: 'running', tool_name: 'search' }
    const running = mergeAiToolActivity(tools, activity)
    const completed = mergeAiToolActivity(running, { ...activity, status: 'completed' })
    assert.deepEqual(
      completed.map(({ name, status }) => [name, status]),
      [['search', 'completed']],
    )
    assert.equal(mergeAiToolActivity(completed, { ...activity, id: 'tool-2' }).length, 2)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})

test('补读在 events 省略时继续轮询，去重乱序编号并以权威快照恢复终态', async () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-ai-recovery-'))
  const output = resolve(root, 'stream.mjs')
  try {
    await build({
      entryPoints: [resolve(import.meta.dirname, '../src/views/pagesMember/ai/stream.ts')],
      bundle: true,
      platform: 'node',
      format: 'esm',
      outfile: output,
    })
    const stream = await import(pathToFileURL(output).href)
    const received = []
    const consume = stream.createAiEventConsumer((event) => received.push(event))
    const item = (id, run = 'r1') => ({
      event: 'activity',
      payload: { session_id: '1', message_id: '2', event_id: id, run_id: run },
    })
    consume(item('9007199254740993'))
    consume(item('9007199254740993'))
    consume(item('9007199254740992'))
    consume(item('9007199254740994', 'r2'))
    consume(item('9007199254740993', 'r1'))
    assert.equal(received.length, 2)
    const queries = []
    const finished = []
    await stream.readAiMessageEvents(
      async (cursor) => {
        queries.push(cursor)
        return {
          next_cursor: '8',
          has_more: false,
          run_id: 'r2',
          messages: [{ id: '2', status: queries.length === 1 ? 1 : 3 }],
          session: { id: '1' },
        }
      },
      (event) => finished.push(event),
      () => false,
    )
    assert.deepEqual(queries, ['', '8'])
    assert.deepEqual(
      finished.map((event) => event.event),
      ['error'],
    )
    let queried = false
    await stream.readAiMessageEvents(
      async () => {
        queried = true
      },
      () => {},
      () => true,
    )
    assert.equal(queried, false)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})
