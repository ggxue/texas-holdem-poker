/* Shared harness for real HTTP/WS pages in isolated CDP browser contexts. */
export async function createBrowserClient(base, {casino=false, newDocumentScript=""}={}) {
  const debuggerURL = process.env.POKER_CDP_URL || 'http://127.0.0.1:9228';
  const info = await (await fetch(`${debuggerURL}/json/version`)).json();
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
    if (newDocumentScript) await send('Page.addScriptToEvaluateOnNewDocument', {source:newDocumentScript}, session);
    await send('Emulation.setDeviceMetricsOverride', {width, height:width < 600 ? 844 : casino ? 720 : 900, deviceScaleFactor:1, mobile:width < 600}, session);
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

  return {info,socket,errors,contexts,pausedRequests,send,evaluate,wait,openPage,state,action};
}
