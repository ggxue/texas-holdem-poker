import fs from 'node:fs/promises';
import {createBrowserClient} from './browser-client.mjs';

const base = (process.argv[2] || 'http://localhost:18082').replace(/\/$/, '');
const client = await createBrowserClient(base, {casino:true,newDocumentScript:`
  window.mobileMedia=[];
  const start=AudioBufferSourceNode.prototype.start;
  AudioBufferSourceNode.prototype.start=function(...args){mobileMedia.push({state:this.context.state,duration:this.buffer?.duration});return start.apply(this,args);};
`});
const {send, evaluate, wait, openPage, action, contexts, socket, errors} = client;
const report = {browser:client.info.Browser, origin:base, cases:[], layouts:[], errors};
function assert(value, message) { if (!value) throw new Error(message); }
async function size(page, width, height) {
  await send('Emulation.setDeviceMetricsOverride', {width, height, deviceScaleFactor:1, mobile:width<=700}, page.session);
  await evaluate(page, 'new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)))');
}
async function click(page, selector) {
  const point=await evaluate(page, `(() => {
    const n=document.querySelector(${JSON.stringify(selector)}), r=n.getBoundingClientRect();
    const x=r.x+r.width/2,y=r.y+r.height/2;
    if(!r.width||!r.height||!n.contains(document.elementFromPoint(x,y)))throw new Error('Control is hidden or covered: '+n.textContent);
    return {x,y};
  })()`);
  await send('Input.dispatchMouseEvent',{type:'mousePressed',button:'left',clickCount:1,...point},page.session);
  await send('Input.dispatchMouseEvent',{type:'mouseReleased',button:'left',clickCount:1,...point},page.session);
}
async function publicView(page) { return evaluate(page, 'fetch("/api/state",{cache:"no-store"}).then(r=>r.json())'); }
async function screenshot(page, name) {
  const shot = await send('Page.captureScreenshot', {format:'png'}, page.session);
  await fs.writeFile(`artifacts/mobile-${name}.png`, Buffer.from(shot.data, 'base64'));
}
async function layout(page, label, compact=true) {
  const result = await evaluate(page, `(() => {
    const box = n => { const r=n.getBoundingClientRect(); return {x:r.x,y:r.y,right:r.right,bottom:r.bottom,width:r.width,height:r.height}; };
    const visible = r => r.x>=0 && r.y>=0 && r.right<=innerWidth+1 && r.bottom<=innerHeight+1;
    const panel = box(document.querySelector('.action-panel'));
    const buttons = [...document.querySelectorAll('#actions button')].map(n=>({...box(n),text:n.textContent,disabled:n.disabled}));
    const places = [...document.querySelectorAll('#seats .seat')].map(box);
    const own = document.querySelector('#seats .seat.mine .hole-cards');
    const board = box(document.querySelector('.community'));
    return {width:innerWidth,height:innerHeight,documentWidth:document.documentElement.scrollWidth,
      panel,buttons,places,own:own?box(own):null,board,documentHeight:document.documentElement.scrollHeight,
      actionsVisible:visible(panel)&&buttons.every(b=>visible(b)&&b.height>=44),
      coreVisible:places.every(r=>visible(r)&&r.bottom<=panel.y)&&visible(board)&&board.bottom<=panel.y&&(!own||visible(box(own))&&box(own).bottom<=panel.y)};
  })()`);
  report.layouts.push({label,...result});
  await screenshot(page, label);
  assert(result.documentWidth<=result.width, `${label}: horizontal overflow`);
  assert(result.documentHeight<=result.height, `${label}: outer page requires scrolling`);
  assert(result.actionsVisible, `${label}: current action or buttons outside viewport`);
  if (compact) assert(result.coreVisible, `${label}: table or own cards require scrolling`);
}
try {
  const pages=[];
  for (let i=0;i<5;i++) {
    const page=await openPage(390);
    await wait(page, 'document.querySelectorAll("#seats .seat:not(.empty)").length>=2 && !document.getElementById("leave").disabled', 'player joined');
    pages.push(page);
    if(i===0) { await size(page,390,720); await layout(page,'empty-390x720'); }
  }
  await action(pages[0], '开始');
  await wait(pages[0], '[...document.querySelectorAll("#actions button")].some(b=>b.textContent==="全押 99"&&!b.disabled)', 'own first opportunity');
  await size(pages[0],360,640);
  await layout(pages[0],'own-360x640');
  report.cases.push('MAC01: real full table and own actions visible at 360x640');
  assert(await evaluate(pages[0], '[...document.querySelectorAll("button")].some(b=>b.textContent==="更多")'), 'MAC10: mobile more controls entry missing');
  await click(pages[0], '[data-panel="more"]');
  await wait(pages[0], 'document.getElementById("retry").getBoundingClientRect().height>0', 'more controls visible');
  assert(await evaluate(pages[0], 'document.getElementById("sound-toggle").getBoundingClientRect().height>0 && document.getElementById("volume").getBoundingClientRect().height>0 && document.getElementById("leave").getBoundingClientRect().height>0'), 'MAC10: original controls inaccessible');
  assert(await evaluate(pages[0], 'document.getElementById("volume").value==="60"'), 'MAC10: default volume changed');
  await click(pages[0], '#sound-toggle');
  await wait(pages[0], 'mobileMedia.some(p=>p.state==="running"&&p.duration>0)', 'real mobile sound playback');
  await click(pages[0], '#sound-toggle');
  await evaluate(pages[0], 'document.getElementById("volume").value="35";document.getElementById("volume").dispatchEvent(new Event("input",{bubbles:true}))');
  await layout(pages[0],'more-360x640');
  await evaluate(pages[0], 'document.querySelector(".hand-reference").open=true');
  assert(await evaluate(pages[0], 'document.querySelectorAll(".reference-row").length===10 && document.querySelector(".hand-reference").getBoundingClientRect().height>0'), 'MAC06: full reference unavailable');
  await click(pages[0], '#mobile-sheet-close');
  report.cases.push('MAC06/10: more controls accessible while actions remain visible');
  for(const [width,height] of [[390,720],[412,780],[320,568]]) {
    await size(pages[0],width,height);await layout(pages[0],`own-${width}x${height}`,width>=360);
  }
  const beforeDetails=await publicView(pages[0]);
  await click(pages[0], '[data-panel="players"]');
  assert(await evaluate(pages[0], 'document.querySelectorAll(".mobile-roster .seat").length===6 && document.querySelectorAll(".mobile-roster .seat:not(.mine) .card-face").length===0 && document.querySelectorAll(".mobile-roster .mine .card-face").length===2'), 'MAC06/08: roster incomplete or private cards exposed');
  await screenshot(pages[0],'players-320x568');
  await click(pages[0], '#mobile-sheet-close');
  const afterDetails=await publicView(pages[0]);
  assert(afterDetails.version===beforeDetails.version && afterDetails.hand.deadline===beforeDetails.hand.deadline, 'MAC06: detail navigation mutated game/expiry');
  await size(pages[0],390,720);
  // Pause a real HTTP request; public waiting feedback must remain usable above an open detail sheet.
  await click(pages[0], '[data-panel="players"]');
  await send('Fetch.enable',{patterns:[{urlPattern:'*/api/command',requestStage:'Request'}]},pages[0].session);
  await click(pages[0], '#actions button.allin');
  await wait(pages[0], 'document.getElementById("command-status").textContent.includes("待确认") && !document.getElementById("command-status").hidden', 'real command pending');
  assert(await evaluate(pages[0], '[...document.querySelectorAll("#actions button")].every(b=>b.disabled)'), 'MAC05: pending buttons remain enabled');
  await layout(pages[0],'pending-390x720');
  const end=Date.now()+5000;
  while(!client.pausedRequests.has(pages[0].session)&&Date.now()<end)await new Promise(r=>setTimeout(r,25));
  assert(client.pausedRequests.has(pages[0].session), 'real command pause missing');
  await send('Fetch.continueRequest',{requestId:client.pausedRequests.get(pages[0].session)},pages[0].session);
  await send('Fetch.disable',{},pages[0].session);client.pausedRequests.delete(pages[0].session);
  await wait(pages[1], '[...document.querySelectorAll("#actions button")].some(b=>b.textContent==="跟注 99"&&!b.disabled)', 'confirmed call opportunity');
  await click(pages[0], '#mobile-sheet-close');
  report.cases.push('MAC04/05/06: actual all-in hit target through open details, real pending disables controls');
  for(let i=1;i<5;i++) {
    if(i===3) { await size(pages[i],412,780);await layout(pages[i],'player4-call-412x780'); }
    await action(pages[i], '跟注');
  }
  await wait(pages[0], 'document.getElementById("game-info").textContent.includes("本局结束")', 'six-player runout');
  await click(pages[0], '[data-panel="results"]');
  assert(await evaluate(pages[0], 'document.querySelectorAll("#results .result-player").length===6 && document.querySelectorAll("#results .best-five .card-face").length===30'), 'MAC07: incomplete six-player public settlement');
  const settled=await publicView(pages[0]);
  assert(settled.hand.players.every(p=>p.won===100&&p.balance===100), 'MAC04: six-way single pool600 must split100 each');
  await layout(pages[0],'results-390x720');
  await size(pages[0],320,568);await layout(pages[0],'results-320x568',false);
  await click(pages[0], '#mobile-sheet-close');
  for(const [width,height] of [[1280,720],[1920,1080],[390,720]]) {
    await size(pages[0],width,height);
    if(width>700) {
      assert(await evaluate(pages[0], 'document.documentElement.scrollWidth===innerWidth && document.documentElement.scrollHeight===innerHeight && [...document.querySelectorAll("#results .result-player")].every(p=>p.getBoundingClientRect().bottom<=innerHeight) && document.querySelector(".hand-reference").open'), 'MAC11: desktop layout or live panels not restored');
      await screenshot(pages[0],`desktop-${width}x${height}`);
    } else await layout(pages[0],'settled-390x720');
  }
  report.cases.push('MAC03/07/11: fixed player4 perspective, full public results, desktop restored across breakpoints');
  // Build unequal wallets through legal play rather than inject online balances.
  await action(pages[0],'开始');await action(pages[0],'全押');await action(pages[1],'跟注');
  for(let i=2;i<5;i++)await action(pages[i],'弃牌');
  await wait(pages[0], 'document.getElementById("game-info").textContent.includes("本局结束")', 'unequal wallets');
  await action(pages[0],'开始');await action(pages[0],'全押');await action(pages[1],'跟注');
  await wait(pages[2], '[...document.querySelectorAll("#actions button")].some(b=>b.textContent==="全押 98"&&!b.disabled)', 'short stack response');
  await size(pages[2],360,640);await layout(pages[2],'short-360x640');
  await action(pages[2],'跟注');
  await send('Page.reload',{},pages[2].session);
  await wait(pages[2], 'document.querySelector("#seats .mine .chips").textContent==="100 chips" && document.querySelector("#seats .mine .seat-status").textContent.includes("全押")', 'memory refill retains all-in');
  assert(await evaluate(pages[2], '!document.querySelector("#actions button.allin")'), 'MAC09: reload created illegal new all-in');
  await layout(pages[2],'rejoin-allin-360x640');
  await action(pages[3],'全押');await action(pages[4],'跟注');
  await wait(pages[0], 'document.getElementById("game-info").textContent.includes("本局结束")', 'short stack single-pool settlement');
  const replacement=await openPage(390,pages[0].context);
  await wait(pages[0], 'document.getElementById("status").textContent.includes("其他页面")', 'control takeover visible');
  assert(await evaluate(pages[0], '!document.querySelector("#actions button") && document.getElementById("retry").disabled && document.getElementById("leave").disabled'), 'MAC10: old page can still act');
  await layout(pages[0],'taken-over-390x720');
  await wait(replacement, '!document.getElementById("leave").disabled', 'replacement joined');
  await click(replacement,'[data-panel="more"]');
  assert(await evaluate(replacement, 'document.getElementById("volume").value==="35" && document.getElementById("sound-toggle").textContent==="启用声音"'), 'MAC10: mobile preferences not retained');
  await click(replacement,'#leave');
  await wait(replacement, 'document.getElementById("status").textContent.includes("已退出房间")', 'actual exit through more');
  assert(await evaluate(replacement, 'document.getElementById("retry").textContent==="重新入房"'), 'MAC10: reentry unavailable');
  report.cases.push('MAC08/09/10: legitimate short response98, reload wallet100 with retained all-in, control takeover, real exit');
  await send('Target.closeTarget',{targetId:pages[3].target});
  await wait(pages[1], 'document.querySelectorAll("#seats .place.offline").length>0', 'disconnected seat');
  await layout(pages[1],'disconnected-390x844');
  report.cases.push('MAC08/10: visible real disconnect, actual mobile WAV playback and remembered settings');
  assert(await evaluate(pages[1], '!document.querySelector(".prototype-switcher, .prototype-review, [data-prototype]")'), 'MAC12: prototype shipped');
  assert(errors.length===0, 'browser runtime errors');
  console.log('Mobile production browser checks passed.');
} catch (error) {
  report.failure=error.message;
  throw error;
} finally {
  await fs.writeFile('artifacts/mobile-browser-report.json', JSON.stringify(report,null,2));
  for (const context of contexts) await send('Target.disposeBrowserContext',{browserContextId:context});
  socket.close();
}
