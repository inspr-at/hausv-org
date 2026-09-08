#!/usr/bin/env node
// Local OpenAI-compatible triage fixture. Standalone:
// HV_QA_AI_MODE=langsam node scripts/snapshot/fake-ai.mjs 8299
// curl -X POST 'http://127.0.0.1:8299/control?mode=Fehler'
// Modes also work as ?mode=ok|langsam|Fehler on /v1/chat/completions.
import { createServer } from 'node:http';
import { pathToFileURL } from 'node:url';

const modes = new Set(['ok', 'langsam', 'Fehler']);

export async function startFakeAI(port, initialMode = process.env.HV_QA_AI_MODE || 'ok') {
  let mode = initialMode;
  let delay;
  const stats = { requests: 0, completed: 0, active: 0 };
  const timers = new Set();
  function setMode(next, delayMS) {
    if (!modes.has(next)) throw new Error('Mode must be ok, langsam or Fehler');
    mode = next;
    delay = delayMS;
  }
  setMode(initialMode);

  const server = createServer(async (request, response) => {
    response.setHeader('Content-Type', 'application/json');
    response.setHeader('Cache-Control', 'no-store');
    const url = new URL(request.url, 'http://127.0.0.1');
    if (request.method === 'GET' && url.pathname === '/healthz') {
      response.end(JSON.stringify({ mode, ...stats }));
      return;
    }
    if (request.method === 'POST' && url.pathname === '/control') {
      try {
        setMode(url.searchParams.get('mode'));
        response.end(JSON.stringify({ mode }));
      } catch {
        response.writeHead(400).end('{"error":"invalid mode"}');
      }
      return;
    }
    if (request.method !== 'POST' || url.pathname !== '/v1/chat/completions') {
      response.writeHead(404).end('{"error":"not found"}');
      return;
    }
    const requestMode = url.searchParams.get('mode') || mode;
    if (!modes.has(requestMode)) {
      response.writeHead(400).end('{"error":"invalid mode"}');
      return;
    }
    // Validate the actual transport/prompt shape instead of accepting any POST.
    let payload, catalogues;
    try {
      let body = '';
      for await (const chunk of request) {
        body += chunk;
        if (body.length > 1_048_576) throw new Error('request too large');
      }
      payload = JSON.parse(body);
      const system = payload.messages.find(message => message.role === 'system').content;
      catalogues = JSON.parse(system.split('\n\nKataloge:\n')[1]);
      if (!payload.model || payload.response_format?.type !== 'json_object' ||
          !payload.messages.some(message => message.role === 'user') || !catalogues.categories.length) {
        throw new Error('invalid completion request');
      }
    } catch {
      response.writeHead(400).end('{"error":"invalid triage completion request"}');
      return;
    }
    stats.requests++;
    stats.active++;
    response.once('close', () => { stats.active--; });
    const timer = setTimeout(() => {
      timers.delete(timer);
      stats.completed++;
      if (response.destroyed) return;
      if (requestMode === 'Fehler') {
        response.writeHead(503).end('{"error":{"message":"QA provider unavailable"}}');
        return;
      }
      const suggestion = {
        category: catalogues.categories.find(category => category.key === 'reparatur')?.key || catalogues.categories[0].key,
        priority: 'Hoch', house: catalogues.houses.find(house => house.slug === 'demo')?.slug || catalogues.houses[0]?.slug || '',
        unit: 'Top 1', assignee: '', template_key: '',
        reply: 'Wir prüfen die gemeldete Reparatur und melden uns bei Ihnen.',
        actions: ['Reparatur prüfen'],
        confidence: Object.fromEntries(['category', 'priority', 'house', 'unit', 'assignee', 'overall'].map(key => [key, 0.94])),
        reasoning: 'Die Testnachricht beschreibt einen Reparaturbedarf.',
      };
      response.end(JSON.stringify({
        id: 'qa-inbox-suggest', object: 'chat.completion', model: payload.model,
        choices: [{ index: 0, finish_reason: 'stop', message: { role: 'assistant', content: JSON.stringify(suggestion) } }],
      }));
    }, delay ?? (requestMode === 'langsam' ? 6000 : 1500));
    timers.add(timer);
  });
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(port, '127.0.0.1', resolve);
  });
  return {
    setMode,
    stats,
    port: server.address().port,
    async close() {
      for (const timer of timers) clearTimeout(timer);
      server.closeAllConnections();
      await new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve()));
    },
  };
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const fixture = await startFakeAI(Number(process.argv[2] || process.env.HV_QA_AI_PORT || 8299));
  process.stdout.write(`Fake AI listening on 127.0.0.1:${fixture.port}\n`);
  for (const signal of ['SIGINT', 'SIGTERM']) process.once(signal, async () => { await fixture.close(); });
}
