import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

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
