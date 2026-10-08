import {spawn} from 'node:child_process';
import {once} from 'node:events';
import net from 'node:net';
import http from 'node:http';
import {randomBytes, randomUUID} from 'node:crypto';
import fs from 'node:fs/promises';
import path from 'node:path';

const listener = net.createServer();
listener.listen(0, '127.0.0.1');
await once(listener, 'listening');
const port = listener.address().port;
await new Promise(resolve => listener.close(resolve));
const base = `http://127.0.0.1:${port}`;
let child;
let wire;
async function start() {
  child = spawn(path.resolve('artifacts/build/texas-poker.exe'), [], {env:{...process.env, PORT:String(port)}, windowsHide:true, stdio:['ignore','ignore','pipe']});
  let logs = '';
  child.stderr.on('data', data => {logs += data;});
  const end = Date.now() + 10000;
  while (Date.now() < end) {
    if (child.exitCode !== null) throw new Error(`server exited: ${logs}`);
    try { if ((await fetch(base + '/healthz')).ok) return; } catch { /* Process has not listened yet. */ }
    await new Promise(resolve => setTimeout(resolve, 25));
  }
  throw new Error(`server start timeout: ${logs}`);
}
async function stop() {
  if (wire) {wire.destroy(); wire = null;}
  if (child && child.exitCode === null) {const ended = once(child, 'exit'); child.kill(); await ended;}
}
async function read(cookie) {
  const response = await fetch(base + '/api/state', {headers:cookie ? {Cookie:cookie} : {}});
  return {state:await response.json(), cookie:response.headers.getSetCookie()[0]?.split(';')[0] || cookie};
}
async function command(cookie, pageID, state, action) {
  const response = await fetch(base + '/api/command', {method:'POST', headers:{Cookie:cookie, 'Content-Type':'application/json'},
    body:JSON.stringify({requestID:randomUUID(), version:state.version, control:state.control, pageID, action})});
  const result = await response.json();
  if (!response.ok) throw new Error(`command rejected ${JSON.stringify(result)}`);
  return result;
}
async function attach(cookie, pageID, generation) {
  const request = http.request({hostname:'127.0.0.1', port, path:`/api/ws?${new URLSearchParams({pageID, control:String(generation)})}`,
    headers:{Cookie:cookie, Connection:'Upgrade', Upgrade:'websocket', 'Sec-WebSocket-Key':randomBytes(16).toString('base64'), 'Sec-WebSocket-Version':'13'}});
  await new Promise((resolve, reject) => {
    let settled = false;
    const finish = error => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      if (error) { request.destroy(); reject(error); } else resolve();
    };
    const timer = setTimeout(() => finish(new Error('WebSocket handshake timeout')), 10000);
    request.once('error', finish);
    request.once('response', response => {
      response.destroy();
      finish(new Error(`WebSocket handshake rejected: ${response.statusCode}`));
    });
    request.once('close', () => finish(new Error('WebSocket handshake closed before upgrade')));
    request.once('upgrade', (_response, socket) => {
      wire = socket;
      wire.on('error', () => {});
      finish();
    });
    request.end();
  });
}
try {
  await start();
  const first = await read();
  const pageID = randomUUID();
  const joined = await command(first.cookie, pageID, first.state, 'join');
  await attach(first.cookie, pageID, joined.control);
  const ready = await read(first.cookie);
  const started = await command(first.cookie, pageID, ready.state, 'start');
  if (!started.hand || started.seats[0].chips !== 99) throw new Error('first process did not hold an active hand');
  await stop();
  await start();
  const fresh = await read(first.cookie);
  if (fresh.state.you === first.state.you || fresh.state.hand || fresh.state.host || fresh.state.seats.some(Boolean)) throw new Error('old cookie restored old process state');
  const newJoin = await command(fresh.cookie, randomUUID(), fresh.state, 'join');
  if (newJoin.seats[0].chips !== 100) throw new Error('new process did not issue 100 chips');
  const report = {passed:true, activeHandCleared:true, oldCookieCreatesNewIdentity:true, seatsAndHostCleared:true, newEntryChips:100};
  await fs.writeFile('artifacts/restart-report.json', JSON.stringify(report, null, 2));
  console.log(JSON.stringify(report));
} finally { await stop(); }
