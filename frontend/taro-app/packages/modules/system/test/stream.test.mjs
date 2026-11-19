import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

test('SSE 解析器消费 activity 和 approval_required 事件', async () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-stream-extra-'))
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
    assert.deepEqual(
      stream.parseAiEventStreamText(
        'event: activity\ndata: {"session_id":"s1","message_id":"m1","activity":{"id":"tool-1","phase":"tool","status":"running"}}\n\n' +
          'event: approval_required\ndata: {"session_id":"s1","message_id":"m1","messages":[]}\n\n',
      ),
      [
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
      ],
    )
    const { mergeAiToolActivity } = stream
    const tools = []
    assert.equal(mergeAiToolActivity(tools, { id: 'reasoning', phase: 'reasoning', status: 'running' }), tools)
    assert.equal(mergeAiToolActivity(tools, { id: 'answer', phase: 'answering', status: 'running' }), tools)
    const activity = { id: 'tool-1', phase: 'tool', status: 'running', tool_name: 'search' }
    const running = mergeAiToolActivity(tools, activity)
    const completed = mergeAiToolActivity(running, { ...activity, status: 'completed' })
    assert.deepEqual(completed.map(({ name, status }) => [name, status]), [['search', 'completed']])
    assert.equal(mergeAiToolActivity(completed, { ...activity, id: 'tool-2' }).length, 2)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})


test('SSE 解析器支持分块、CRLF、多行 data 并忽略无效事件', async () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-stream-'))
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
    const events = []
    const parser = stream.createAiEventStreamTextParser((event) => events.push(event))
    parser.push(': keep-alive\r\nevent: delta\r\ndata: {"session_id":"s1",')
    parser.push('"message_id":"m1","delta":"你"}\r\n\r\n')
    parser.push('event: unknown\ndata: {}\n\n')
    parser.push('event: finish\ndata: {"session_id":"s1",\n')
    parser.push('data: "message_id":"m1"}\n')
    parser.flush()

    assert.deepEqual(events, [
      {
        event: 'delta',
        payload: { session_id: 's1', message_id: 'm1', delta: '你' },
      },
      {
        event: 'finish',
        payload: { session_id: 's1', message_id: 'm1' },
      },
    ])
    assert.deepEqual(
      stream.parseAiEventStreamText(
        'event: error\ndata: {"session_id":"s2","message_id":"m2"}\n\n',
      ),
      [{ event: 'error', payload: { session_id: 's2', message_id: 'm2' } }],
    )
    assert.equal(stream.normalizeAiStreamItem({ event: 'delta', data: 'not-json' }), null)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})
