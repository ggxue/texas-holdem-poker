import fs from 'node:fs/promises';

const base = (process.argv[2] || 'http://localhost:18080').replace(/\/$/, '');
const info = await (await fetch('http://127.0.0.1:9228/json/version')).json();
const socket = new WebSocket(info.webSocketDebuggerUrl);
await new Promise((resolve, reject) => { socket.addEventListener('open', resolve, {once:true}); socket.addEventListener('error', reject, {once:true}); });
let sequence = 0;
const pending = new Map();
const errors = [];
const contexts = [];
const pausedRequests = new Map();
socket.addEventListener('message', event => {
  const message = JSON.parse(event.data);
  if (message.id) {
    const request = pending.get(message.id);
    if (!request) return;
    clearTimeout(request.timer); pending.delete(message.id);
    if (message.error) request.reject(new Error(JSON.stringify(message.error))); else request.resolve(message.result);
  } else if (message.method === 'Runtime.exceptionThrown') errors.push(message.params.exceptionDetails);
  else if (message.method === 'Fetch.requestPaused') pausedRequests.set(message.sessionId, message.params.requestId);
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
  await send('Emulation.setDeviceMetricsOverride', {width, height:width < 600 ? 844 : process.argv.includes('--casino') ? 720 : 900, deviceScaleFactor:1, mobile:width < 600}, session);
  await send('Page.navigate', {url:base}, session);
  return page;
}
async function state(page) { return JSON.parse(await evaluate(page, 'JSON.stringify(confirmed)')); }
async function action(page, label) {
  const selector = `[...document.querySelectorAll('#actions button, #leave')].find(button => button.textContent.startsWith(${JSON.stringify(label)}) && !button.disabled)`;
  await wait(page, `!!(${selector})`, label);
  const before = (await state(page)).version;
  await evaluate(page, `(${selector}).click()`);
  await wait(page, `confirmed.version > ${before} && !sending`, `${label} acknowledged`);
}
async function pendingAction(page, label) {
  await wait(page, "confirmed?.hand?.actor === confirmed?.you && !sending", 'own action before request pause');
  await send('Fetch.enable', {patterns:[{urlPattern:'*/api/command', requestStage:'Request'}]}, page.session);
  const before = (await state(page)).version;
  let request;
  try {
    await evaluate(page, `[...document.querySelectorAll('#actions button')].find(button=>button.textContent.startsWith(${JSON.stringify(label)})).click()`);
    const end = Date.now()+12000;
    while (!pausedRequests.has(page.session) && Date.now()<end) await new Promise(resolve=>setTimeout(resolve,25));
    request = pausedRequests.get(page.session);
    assert(request, 'command request did not pause');
    assert(await evaluate(page, "sending && document.body.innerText.includes('操作结果待确认') && [...document.querySelectorAll('#actions button')].every(button=>button.disabled) && document.getElementById('leave').disabled"), 'pending command must show waiting confirmation and disable controls');
  } finally {
    if (request) await send('Fetch.continueRequest', {requestId:request}, page.session);
    await send('Fetch.disable', {}, page.session);
    pausedRequests.delete(page.session);
  }
  await wait(page, `confirmed.version > ${before} && !sending`, 'paused command acknowledged');
  report.cases.push('pending real HTTP command visibly awaits confirmation with own actions and exit disabled');
}
async function currentHuman(pages) {
  const end = Date.now() + 12000;
  while (Date.now() < end) {
    for (const page of pages) if (await evaluate(page, 'confirmed?.hand?.actor === confirmed?.you && !sending')) return page;
    await new Promise(resolve => setTimeout(resolve, 25));
  }
  throw new Error('No current human action became available');
}
function assert(condition, message) { if (!condition) throw new Error(message); }
async function screenshot(page, name) {
  assert(await evaluate(page, 'document.documentElement.scrollWidth <= innerWidth'), 'horizontal overflow');
  const picture = await send('Page.captureScreenshot', {format:'png', captureBeyondViewport:true}, page.session);
  const output = process.argv.includes('--tie-six') ? name.replace(/^poker-/, 'poker-tie-') : name;
  await fs.writeFile(`artifacts/${output}.png`, Buffer.from(picture.data, 'base64'));
}
const report = {browser:info.Browser, origin:base, cases:[], errors};
async function checkLayout(page, mobile) {
  const boxes = await evaluate(page, "[...document.querySelectorAll('#seats .seat')].map(seat => {const r=seat.getBoundingClientRect();return {x:r.x,y:r.y,right:r.right,bottom:r.bottom};})");
  assert(boxes.length === 6, 'six fixed seats missing');
  const [one,two,three,four,five,bot] = boxes;
  if (mobile) {
    assert(Math.abs(one.y-two.y)<2 && one.x<two.x && Math.abs(three.y-four.y)<2 && three.y>one.y && three.x<four.x && five.y>four.y && bot.y>five.y, 'phone fixed human rows and bottom bot');
    assert(await evaluate(page, "[...document.querySelectorAll('.place .wallet')].every(wallet => {const w=wallet.getBoundingClientRect(), s=wallet.parentElement.querySelector('.seat').getBoundingClientRect(); return w.top >= s.bottom && Math.abs((w.left+w.right)/2-(s.left+s.right)/2)<2;})"), 'phone wallet piles must sit below their own player, clear of player information');
  } else {
    assert(Math.abs(three.y-four.y)<2 && Math.abs(four.y-five.y)<2 && three.x<four.x && four.x<five.x && one.y>three.y && two.y>five.y && one.x<two.x && bot.y>one.y, 'desktop fixed top humans3/4/5, left1/right2, bottom bot');
  }
  assert(await evaluate(page, "document.getElementById('game-info').getBoundingClientRect().bottom <= document.getElementById('seats').getBoundingClientRect().top && document.getElementById('actions').getBoundingClientRect().top >= document.querySelectorAll('#seats .seat')[5].getBoundingClientRect().bottom && document.body.innerText.includes('你的操作') && document.getElementById('retry').getBoundingClientRect().bottom < document.getElementById('seats').getBoundingClientRect().top"), 'top status/connection controls or bottom own actions');
  assert(await evaluate(page, 'document.documentElement.scrollWidth <= innerWidth'), 'layout horizontal overflow');
}
async function checkCards(page, mobile) {
  assert(await evaluate(page, "document.querySelectorAll('.hand-reference .reference-row').length === 10 && [...document.querySelectorAll('.hand-reference .reference-row')].every(row => row.querySelectorAll('.playing-card').length === 5) && document.querySelector('.hand-reference').textContent.includes('同花大顺') && document.querySelector('.hand-reference').textContent.includes('High Card')"), 'complete ten hand categories with five examples');
  assert(await evaluate(page, "document.querySelectorAll('.board .playing-card').length === 5 && document.querySelector('.pot').getBoundingClientRect().bottom < document.querySelector('.board').getBoundingClientRect().top && document.querySelector('.pot').textContent.includes('chips')"), 'five board slots below chips pot');
  if (mobile) {
    assert(await evaluate(page, "!document.querySelector('.hand-reference').open"), 'phone reference must start collapsed');
    await evaluate(page, "document.querySelector('.hand-reference summary').click()");
    assert(await evaluate(page, "document.querySelector('.hand-reference').open && document.documentElement.scrollWidth <= innerWidth"), 'phone reference cannot expand without overflow');
    await screenshot(page, 'poker-reference-mobile');
    await evaluate(page, "document.querySelector('.hand-reference summary').click()");
  } else assert(await evaluate(page, "document.querySelector('.hand-reference').getBoundingClientRect().right <= document.querySelector('.table').getBoundingClientRect().left"), 'desktop reference must stay left');
}
async function checkResults(page, early = false) {
  const v = await state(page);
  assert(v.hand.stage === 'finished', 'result requires confirmed settlement');
  assert(await evaluate(page, "document.getElementById('results') && document.getElementById('results').classList.contains('settled') && (innerWidth > 700 ? document.getElementById('results').getBoundingClientRect().left >= document.querySelector('.workspace').getBoundingClientRect().right : document.getElementById('results').getBoundingClientRect().top >= document.getElementById('actions').getBoundingClientRect().bottom)"), 'independent results must be below own actions');
  const text = await evaluate(page, "document.getElementById('results').innerText");
  for (const player of v.hand.players) {
    const name = player.id === 'bot' ? '机器人' : `玩家 ${player.seat+1}`;
    assert(text.includes(name) && text.includes(`投入 ${player.invested}`) && text.includes(`结算余额 ${player.balance}`), 'settlement snapshot fields missing');
    if (player.won > 0) assert(text.includes(`获得 ${player.won} chips`) && text.includes('赢家'), 'winner/gain not prominent');
  }
  if (early) assert(!text.includes('Best Five') && text.includes('未强制亮牌'), 'early win forced a hand strength');
  else assert(await evaluate(page, "document.querySelectorAll('#results .best-five').length > 0"), 'showdown best five missing');
  assert(await evaluate(page, 'document.documentElement.scrollWidth <= innerWidth'), 'results horizontal overflow');
}
async function observeThinking(page) {
  await wait(page, 'confirmed?.hand?.botThinking', 'robot thinking');
  const start = await state(page);
  assert(start.hand.actor === 'bot' && start.hand.deadline - start.serverTime <= 30000 && start.hand.deadline - start.serverTime > 21000, 'robot display must use thirty real seconds');
  assert(await evaluate(page, "document.body.innerText.includes('思考中') && !document.querySelector('#actions button:not(:disabled)')?.textContent.match(/过牌|下注|跟注|弃牌/)"), 'thinking label or human controls');
  await evaluate(page, 'state()');
  const refreshed = await state(page);
  assert(refreshed.hand.turn === start.hand.turn && refreshed.hand.deadline === start.hand.deadline, 'query reset thinking display');
  await screenshot(page, 'poker-thinking');
  if (process.argv.includes('--tie-six')) {
    await wait(page, "confirmed.hand.botThinking && /剩余 2[78] 秒/.test(document.getElementById('countdown').textContent)", 'real thirty-second countdown after two seconds');
    assert((await state(page)).hand.turn === start.hand.turn, 'display countdown accelerated bot action');
    await screenshot(page, 'poker-thinking-elapsed');
  }
  await wait(page, `confirmed.hand.turn !== ${start.hand.turn}`, 'actual robot action');
  const next = await state(page);
  const remaining = next.hand.deadline - next.serverTime;
  assert(next.hand.actor !== 'bot' && remaining <= 30000 && remaining >= 29900, `next human did not get fresh thirty seconds: ${remaining}ms`);
  report.cases.push('robot thinking label, thirty-second display basis retained across query, action before display expires, next human fresh thirty seconds');
}
try {
  const health = await fetch(base + '/healthz');
  assert(health.ok && (await health.text()).trim() === 'ok', 'health probe failed');
  report.cases.push('health probe without room identity');
  const first = await openPage(1280);
  await wait(first, "confirmed?.seats[0]?.id === confirmed?.you && socket?.readyState === WebSocket.OPEN", 'first entry');
  if (process.argv.includes('--casino')) {
    assert(await evaluate(first, "document.documentElement.scrollWidth === 1280 && document.documentElement.scrollHeight === 720"), 'B2 desktop must fit the 1280x720 viewport without scrolling');
    assert(await evaluate(first, "document.querySelectorAll('.wallet svg').length === 2 && document.querySelector('.seat.mine').textContent.includes('玩家 1') && !document.body.innerText.includes('真人')"), 'occupied player and bot need stacks and player naming');
  }
  if (process.argv.includes('--layout')) await checkLayout(first, false);
  if (process.argv.includes('--cards')) await checkCards(first, false);
  await action(first, '开始');
  assert((await state(first)).hand.players.length === 2, 'solo game missing bot');
  assert(await evaluate(first, "document.getElementById('countdown').textContent.includes('当前行动剩余')"), 'action countdown missing');
  const second = await openPage(390);
  await wait(second, "confirmed?.seats[1]?.id === confirmed?.you && socket?.readyState === WebSocket.OPEN", 'second entry');
  const waiting = await state(second);
  if (process.argv.includes('--layout')) await checkLayout(second, true);
  if (process.argv.includes('--cards')) { await checkCards(second, true); assert(await evaluate(second, "document.querySelectorAll('.seat.mine .playing-card').length === 0"), 'waiting entrant must not have dealt cards'); }
  assert(!waiting.hand.players.some(player => player.id === waiting.you), 'mid-hand entrant was dealt');
  for (let i = 0; i < 4; i++) {
    if (i === 0 && process.argv.includes('--pending')) await pendingAction(first, '过牌');
    else await action(first, '过牌');
    if (i === 0) await observeThinking(first);
    if (process.argv.includes('--cards')) {
      await wait(first, `document.querySelectorAll('.board .card-face').length === ${[3,4,5,5][i]}`, 'revealed board slots');
      assert(await evaluate(first, "[...document.querySelectorAll('.board .playing-card')].every(card => Math.abs(card.getBoundingClientRect().y-document.querySelector('.board .playing-card').getBoundingClientRect().y)<2)"), 'five board cards must stay in one row');
    }
  }
  await wait(first, "confirmed?.hand?.stage === 'finished'", 'solo settlement after bot');
  if (process.argv.includes('--results')) await checkResults(first);
  assert((await state(first)).hand.stage === 'finished', 'solo did not settle');
  report.cases.push('one human + bot, mid-hand waiting entrant, four check streets and result');
  if (process.argv.includes('--cards')) report.cases.push('ten reference categories, phone expand/collapse, native card faces/backs, waiting privacy, 0/3/4/5 board progression and chips');
  await action(first, '开始');
  if (process.argv.includes('--results')) assert(await evaluate(first, "!document.getElementById('results').classList.contains('settled')"), 'next hand did not replace old results');
  assert((await state(first)).hand.players.length === 3, 'next hand did not include second human');
  const third = await openPage(390);
  await wait(third, "confirmed?.seats[2]?.id === confirmed?.you && socket?.readyState === WebSocket.OPEN", 'third entry');
  assert(await evaluate(third, "document.querySelectorAll('#seats h2')[2].textContent === '玩家 3' && document.querySelectorAll('#seats h2')[5].textContent === '机器人'"), 'human 3 and bot seat labels');
  assert(await evaluate(third, "document.getElementById('seats').textContent.includes('空位')"), 'empty seats must remain visible');
  const fourth = await openPage(1280);
  await wait(fourth, "confirmed?.seats[3]?.id === confirmed?.you && socket?.readyState === WebSocket.OPEN", 'fourth entry');
  const fifth = await openPage(390);
  await wait(fifth, "confirmed?.seats[4]?.id === confirmed?.you && socket?.readyState === WebSocket.OPEN", 'fifth entry');
  if (process.argv.includes('--layout')) { await checkLayout(fourth, false); await checkLayout(fifth, true); report.cases.push('fixed desktop and phone seat positions, top status/connection controls, own actions below bot, empty and full rooms'); }
  for (const entrant of [third, fourth, fifth]) {
    const waiting = await state(entrant);
    assert(!waiting.hand.players.some(player => player.id === waiting.you) && waiting.hand.players.every(player => !player.hole?.length), 'waiting entrant inherited private cards/participation');
    assert(await evaluate(entrant, "document.querySelectorAll('#seats .mine').length === 1 && document.querySelector('#seats .mine').textContent.includes('你') && document.querySelector('#seats .mine').textContent.includes('等待下一局')"), 'own/waiting marker missing');
  }
  const sixth = await openPage(390);
  await wait(sixth, "document.getElementById('status').textContent.includes('房间已满')", 'room full');
  report.cases.push('five human seats, sixth identity room-full, empty/own/waiting markers, fixed bot identity');
  const bettor = await currentHuman([first, second]);
  assert(await evaluate(bettor, "document.getElementById('actions').textContent.includes('下注 10')"), 'bet amount missing');
  await action(bettor, '下注');
  const caller = await currentHuman([first, second]);
  await wait(caller, "document.getElementById('actions').textContent.includes('跟注 10')", 'call amount');
  await action(caller, '跟注');
  for (let i = 0; i < 6; i++) await action(await currentHuman([first, second]), '过牌');
  await wait(first, "confirmed?.hand?.stage === 'finished'", 'two human settlement after bot');
  if (process.argv.includes('--results')) { await checkResults(first); await checkResults(second); }
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
  if (process.argv.includes('--results')) { assert(JSON.stringify(after.hand.players.map(p=>[p.won,p.balance]))===JSON.stringify(before.hand.players.map(p=>[p.won,p.balance])), 'takeover rewrote historical awards'); await checkResults(replacement); }
  await send('Target.closeTarget', {targetId:first.target});
  await action(replacement, '开始');
  assert((await state(replacement)).hand.players.length === 6, 'five humans + fixed bot did not start');
  if (process.argv.includes('--full-hand')) {
    const fullStart = await state(replacement);
    for (let i = 0; i < 20; i++) await action(await currentHuman([replacement,second,third,fourth,fifth]), '过牌');
    await wait(replacement, "confirmed?.hand?.stage === 'finished'", 'full six-player showdown');
    await checkResults(replacement); await checkResults(fifth);
    if (process.argv.includes('--tie-six')) {
      const full = await state(replacement);
      assert(full.hand.players.every(player=>player.won===1 && player.strength.category==='同花大顺' && player.balance === (player.id === 'bot' ? fullStart.bot.chips : fullStart.seats[player.seat].chips)+1), 'six-way royal board should split six chips as one each');
      assert(await evaluate(replacement, "document.querySelectorAll('#results .winner').length === 6"), 'six winners not all highlighted');
      assert(await evaluate(fifth, "document.querySelectorAll('#results .winner').length === 6"), 'phone missing tied winners');
    }
    if (process.argv.includes('--casino')) {
      assert(await evaluate(replacement, "document.documentElement.scrollHeight === 720 && document.querySelectorAll('.wallet svg').length === 6 && !document.querySelector('.pot svg') && document.querySelectorAll('#results .result-player').length === 6"), 'full settlement must fit one screen with six balances and empty pot');
      const overflow = await evaluate(replacement, "[...document.querySelectorAll('.seat,.result-player,.results-panel,.reference-row,.app-header,.app-footer')].filter(e=>e.scrollWidth>e.clientWidth+1||e.scrollHeight>e.clientHeight+1).map(e=>e.className)");
      assert(overflow.length === 0, 'full settlement internal overflow: '+JSON.stringify(overflow));
      assert(await evaluate(replacement, "[...document.querySelectorAll('.wallet svg')].every(e=>{const a=e.getBoundingClientRect(),b=document.querySelector('.community').getBoundingClientRect();return !(a.left<b.right&&a.right>b.left&&a.top<b.bottom&&a.bottom>b.top)})"), 'wallet chips overlap public cards or pot');
      for (const [width,height] of [[1920,720],[1920,900]]) {
        await send('Emulation.setDeviceMetricsOverride', {width,height,deviceScaleFactor:1,mobile:false}, replacement.session);
        await new Promise(resolve=>setTimeout(resolve,150));
        assert(await evaluate(replacement, "document.documentElement.scrollWidth===innerWidth && document.documentElement.scrollHeight===innerHeight && document.getElementById('seats').getBoundingClientRect().bottom<=document.querySelector('.action-panel').getBoundingClientRect().top"), 'wide desktop table must fit above actions');
        assert(await evaluate(replacement, "[...document.querySelectorAll('.seat,.result-player,.results-panel,.reference-row,.app-header,.app-footer')].every(e=>e.scrollWidth<=e.clientWidth+1&&e.scrollHeight<=e.clientHeight+1)"), 'wide desktop internal overflow');
      }
      await send('Emulation.setDeviceMetricsOverride', {width:1280,height:720,deviceScaleFactor:1,mobile:false}, replacement.session);
      report.cases.push('B2 at 1280x720,1920x720,1920x900: complete six-player settlement, six wallet piles, zero pot pile, no page/panel overflow or public-area overlap');
    }
    await screenshot(replacement, 'poker-full-results-desktop');
    await screenshot(fifth, 'poker-full-results-mobile');
    report.cases.push('five humans + bot full four-street showdown, desktop and phone multi-winner results and next hand');
    await action(replacement, '开始');
  }
  for (let i = 0; i < 5; i++) {
    const actor = await currentHuman([replacement, second, third, fourth, fifth]);
    const actorState = await state(actor);
    const seat = actorState.hand.players.find(player => player.id === actorState.you).seat;
    assert(await evaluate(actor, `document.getElementById('game-info').textContent.includes('当前行动：玩家 ${seat+1}')`), 'human action label missing');
    await action(actor, '弃牌');
  }
  assert((await state(second)).hand.stage === 'finished', 'fold game not finished');
  if (process.argv.includes('--results')) { await checkResults(second, true); assert(await evaluate(second, "document.querySelectorAll('#results .card-face').length === 2"), 'early win leaked others holes or concealed own allowed hole'); report.cases.push('B2 settlement panel, winners/gains, Best Five, historical balance after takeover100, result replacement and early-win privacy'); }
  report.cases.push('same-cookie takeover, old page stopped reconnect, old close harmless, manual next hand/fold');
  await action(replacement, '退出');
  await wait(second, 'confirmed.host === confirmed.you', 'host transfer');
  await evaluate(sixth, "document.getElementById('retry').click()");
  await wait(sixth, "confirmed?.seats[0]?.id === confirmed?.you && socket?.readyState === WebSocket.OPEN", 'fill low seat');
  await action(second, '退出');
  await wait(third, 'confirmed.host === confirmed.you', 'entry-order host transfer');
  assert((await state(sixth)).host !== (await state(sixth)).you, 'later low-seat entrant stole host priority');
  await screenshot(third, 'poker-capacity-mobile');
  await screenshot(fourth, 'poker-capacity-desktop');
  report.cases.push('five humans + bot manual next hand/fold, seat reuse and host transfer by entry order');
  assert(errors.length === 0, 'uncaught browser errors');
  report.passed = true;
  await fs.writeFile(process.argv.includes('--tie-six') ? 'artifacts/browser-tie-report.json' : 'artifacts/browser-report.json', JSON.stringify(report, null, 2));
  console.log(JSON.stringify(report));
} finally {
  for (const context of contexts) await send('Target.disposeBrowserContext', {browserContextId:context});
  socket.close();
}
