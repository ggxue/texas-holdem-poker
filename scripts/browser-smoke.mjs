import fs from 'node:fs/promises';

const base = (process.argv[2] || 'http://localhost:18080').replace(/\/$/, '');
const info = await (await fetch('http://127.0.0.1:9228/json/version')).json();
const socket = new WebSocket(info.webSocketDebuggerUrl);
await new Promise((resolve, reject) => { socket.addEventListener('open', resolve, {once:true}); socket.addEventListener('error', reject, {once:true}); });
let sequence = 0;
const pending = new Map();
const errors = [];
const contexts = [];
socket.addEventListener('message', event => {
  const message = JSON.parse(event.data);
  if (message.id) {
    const request = pending.get(message.id);
    if (!request) return;
    clearTimeout(request.timer); pending.delete(message.id);
    if (message.error) request.reject(new Error(JSON.stringify(message.error))); else request.resolve(message.result);
  } else if (message.method === 'Runtime.exceptionThrown') errors.push(message.params.exceptionDetails);
});
function send(method, params = {}, sessionId) {
  const id = ++sequence;
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {pending.delete(id); reject(new Error(`CDP timeout ${method}`));}, 10000);
    pending.set(id, {resolve, reject, timer});
    socket.send(JSON.stringify({id, method, params, ...(sessionId ? {sessionId} : {})}));
  });
}
async function evaluate(page, expression) {
  const result = await send('Runtime.evaluate', {expression, returnByValue:true, awaitPromise:true}, page.session);
  if (result.exceptionDetails) throw new Error(JSON.stringify(result.exceptionDetails));
  return result.result.value;
}
async function wait(page, expression, label) {
  const end = Date.now() + 12000;
  while (Date.now() < end) {
    if (await evaluate(page, `(() => { try { return Boolean(${expression}); } catch { return false; } })()`)) return;
    await new Promise(resolve => setTimeout(resolve, 25));
  }
  throw new Error(`UI timeout ${label}: ${await evaluate(page, 'document.body.innerText')}`);
}
async function openPage(width, context) {
  if (!context) { context = (await send('Target.createBrowserContext')).browserContextId; contexts.push(context); }
  const target = (await send('Target.createTarget', {url:'about:blank', browserContextId:context})).targetId;
  const session = (await send('Target.attachToTarget', {targetId:target, flatten:true})).sessionId;
  const page = {target, session, context};
  await send('Runtime.enable', {}, session);
  await send('Page.enable', {}, session);
  await send('Emulation.setDeviceMetricsOverride', {width, height:width < 600 ? 844 : 900, deviceScaleFactor:1, mobile:width < 600}, session);
  await send('Page.navigate', {url:base}, session);
  return page;
}
async function state(page) { return JSON.parse(await evaluate(page, 'JSON.stringify(confirmed)')); }
async function action(page, label) {
  const selector = `[...document.querySelectorAll('#actions button')].find(button => button.textContent.startsWith(${JSON.stringify(label)}) && !button.disabled)`;
  await wait(page, `!!(${selector})`, label);
  const before = (await state(page)).version;
  await evaluate(page, `(${selector}).click()`);
  await wait(page, `confirmed.version > ${before} && !sending`, `${label} acknowledged`);
}
function assert(condition, message) { if (!condition) throw new Error(message); }
async function screenshot(page, name) {
  assert(await evaluate(page, 'document.documentElement.scrollWidth <= innerWidth'), 'horizontal overflow');
  const picture = await send('Page.captureScreenshot', {format:'png', captureBeyondViewport:true}, page.session);
  await fs.writeFile(`artifacts/${name}.png`, Buffer.from(picture.data, 'base64'));
}
const report = {browser:info.Browser, origin:base, cases:[], errors};
try {
  const health = await fetch(base + '/healthz');
  assert(health.ok && (await health.text()).trim() === 'ok', 'health probe failed');
  report.cases.push('health probe without room identity');
  const first = await openPage(1280);
  await wait(first, "confirmed?.seats[0]?.id === confirmed?.you && socket?.readyState === WebSocket.OPEN", 'first entry');
  await action(first, '开始');
  assert((await state(first)).hand.players.length === 2, 'solo game missing bot');
  assert(await evaluate(first, "document.getElementById('countdown').textContent.includes('当前行动剩余')"), 'action countdown missing');
  const second = await openPage(390);
  await wait(second, "confirmed?.seats[1]?.id === confirmed?.you && socket?.readyState === WebSocket.OPEN", 'second entry');
  const waiting = await state(second);
  assert(!waiting.hand.players.some(player => player.id === waiting.you), 'mid-hand entrant was dealt');
  for (let i = 0; i < 4; i++) await action(first, '过牌');
  assert((await state(first)).hand.stage === 'finished', 'solo did not settle');
  report.cases.push('one human + bot, mid-hand waiting entrant, four check streets and result');
  await action(first, '开始');
  assert((await state(first)).hand.players.length === 3, 'next hand did not include second human');
  const third = await openPage(390);
  await wait(third, "document.getElementById('status').textContent.includes('房间已满')", 'room full');
  report.cases.push('third identity sees room-full message');
  assert(await evaluate(first, "document.getElementById('actions').textContent.includes('下注 10')"), 'bet amount missing');
  await action(first, '下注');
  await wait(second, "document.getElementById('actions').textContent.includes('跟注 10')", 'call amount');
  await action(second, '跟注');
  for (let i = 0; i < 3; i++) { await action(first, '过牌'); await action(second, '过牌'); }
  assert((await state(first)).hand.stage === 'finished', 'three player game did not settle');
  await screenshot(first, 'poker-desktop');
  await screenshot(second, 'poker-mobile');
  report.cases.push('two human + bot, fixed bet/call, showdown, desktop/mobile no horizontal overflow');
  const before = await state(first);
  const replacement = await openPage(1280, first.context);
  await wait(replacement, "confirmed?.you && socket?.readyState === WebSocket.OPEN", 'replacement entry');
  await wait(first, "document.getElementById('status').textContent.includes('已在其他页面打开') && takenOver", 'old page takeover');
  const after = await state(replacement);
  assert(after.you === before.you && after.seats[0].chips === 100 && after.hand.id === before.hand.id, 'takeover changed identity/hand or did not reset100');
  await send('Target.closeTarget', {targetId:first.target});
  await action(replacement, '开始');
  await action(replacement, '弃牌');
  await action(second, '弃牌');
  assert((await state(second)).hand.stage === 'finished', 'fold game not finished');
  report.cases.push('same-cookie takeover, old page stopped reconnect, old close harmless, manual next hand/fold');
  await action(replacement, '退出');
  await wait(second, 'confirmed.host === confirmed.you', 'host transfer');
  await action(second, '退出');
  report.cases.push('explicit departure stops reconnect and transfers host');
  assert(errors.length === 0, 'uncaught browser errors');
  report.passed = true;
  await fs.writeFile('artifacts/browser-report.json', JSON.stringify(report, null, 2));
  console.log(JSON.stringify(report));
} finally {
  for (const context of contexts) await send('Target.disposeBrowserContext', {browserContextId:context});
  socket.close();
}
