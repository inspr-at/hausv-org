#!/usr/bin/env node

import net from 'node:net';
import http from 'node:http';

const smtpPort = Number(process.argv[2]);
const apiPort = Number(process.argv[3]);
if (!Number.isInteger(smtpPort) || !Number.isInteger(apiPort)) {
  console.error('usage: fake-smtp.mjs <smtp-port> <api-port>');
  process.exit(1);
}

const messages = [];

function reply(socket, line) {
  socket.write(`${line}\r\n`);
}

const smtpServer = net.createServer((socket) => {
  socket.setEncoding('utf8');
  let buffer = '';
  let dataMode = false;
  let dataLines = [];
  reply(socket, '220 hausv.test ESMTP local-fixture');

  socket.on('data', (chunk) => {
    buffer += chunk;
    let newline;
    while ((newline = buffer.indexOf('\n')) !== -1) {
      const rawLine = buffer.slice(0, newline);
      buffer = buffer.slice(newline + 1);
      const line = rawLine.replace(/\r$/, '');
      if (dataMode) {
        if (line === '.') {
          messages.push({ body: dataLines.map((item) => item.startsWith('..') ? item.slice(1) : item).join('\n') });
          dataMode = false;
          dataLines = [];
          reply(socket, '250 2.0.0 accepted');
        } else {
          dataLines.push(line);
        }
        continue;
      }
      const command = line.trim().toUpperCase();
      if (command.startsWith('EHLO') || command.startsWith('HELO')) reply(socket, '250 hausv.test');
      else if (command.startsWith('MAIL FROM:')) reply(socket, '250 2.1.0 sender ok');
      else if (command.startsWith('RCPT TO:')) reply(socket, '250 2.1.5 recipient ok');
      else if (command === 'DATA') {
        dataMode = true;
        reply(socket, '354 end with <CRLF>.<CRLF>');
      } else if (command === 'RSET' || command === 'NOOP') reply(socket, '250 2.0.0 ok');
      else if (command === 'QUIT') {
        reply(socket, '221 2.0.0 bye');
        socket.end();
      } else reply(socket, '502 5.5.2 unsupported');
    }
  });
});

const apiServer = http.createServer((request, response) => {
  if (request.method === 'GET' && request.url === '/healthz') {
    response.writeHead(204).end();
    return;
  }
  if (request.method === 'GET' && request.url === '/messages') {
    response.writeHead(200, { 'Content-Type': 'application/json', 'Cache-Control': 'no-store' });
    response.end(JSON.stringify(messages));
    return;
  }
  response.writeHead(404).end();
});

smtpServer.listen(smtpPort, '127.0.0.1', () => {
  apiServer.listen(apiPort, '127.0.0.1', () => process.stdout.write('fake smtp ready\n'));
});

function shutdown() {
  apiServer.close();
  smtpServer.close();
}
process.on('SIGTERM', shutdown);
process.on('SIGINT', shutdown);
