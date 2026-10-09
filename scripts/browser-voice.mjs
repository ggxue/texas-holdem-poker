import fs from 'node:fs/promises';
import {createBrowserClient} from './browser-client.mjs';
const base=(process.argv[2]||'http://localhost:18080').replace(/\/$/,'');
const {info,socket,errors,contexts,pausedRequests,send,evaluate,wait,openPage,state,action}=await createBrowserClient(base,{casino:process.argv.includes('--casino'),newDocumentScript:`
    window.mediaProof=[];
    const start=AudioBufferSourceNode.prototype.start, stop=AudioBufferSourceNode.prototype.stop;
    AudioBufferSourceNode.prototype.start=function(...args){ window.mediaProof.push({kind:'start',duration:this.buffer?.duration,state:this.context.state,text:document.getElementById('voice-status')?.textContent,at:Date.now()}); return start.apply(this,args); };
    AudioBufferSourceNode.prototype.stop=function(...args){ window.mediaProof.push({kind:'stop',text:document.getElementById('voice-status')?.textContent,at:Date.now()}); return stop.apply(this,args); };
`});
function assert(condition,message){if(!condition)throw new Error(message);}

try {
  const page=await openPage(1280);
  await wait(page,"confirmed?.you && socket?.readyState===WebSocket.OPEN",'entry');
  assert(await evaluate(page,"!document.getElementById('sound-toggle').disabled && !document.getElementById('volume').disabled"),'production voice controls must be enabled');
  async function click(selector) {
    const pos=await evaluate(page,`(()=>{const r=document.querySelector(${JSON.stringify(selector)}).getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2};})()`);
    await send('Input.dispatchMouseEvent',{type:'mousePressed',button:'left',clickCount:1,...pos},page.session);
    await send('Input.dispatchMouseEvent',{type:'mouseReleased',button:'left',clickCount:1,...pos},page.session);
  }
  assert(await evaluate(page,"document.getElementById('volume').value==='60' && document.getElementById('sound-toggle').textContent==='启用声音'"),'default sound preferences');
  await click('#sound-toggle');
  await wait(page,"mediaProof.some(p=>p.kind==='start' && p.state==='running' && p.duration>0) && document.getElementById('sound-toggle').textContent==='静音'",'real Mandarin audio decoded and played after trusted click');
  const manifest=await (await fetch(base+'/audio/manifest.json')).json();
  assert(manifest.locale==='zh-CN' && manifest.voice==='Microsoft Huihui','fixed Mandarin voice');
  for (const key of Object.keys(manifest.phrases)) {
    const response=await fetch(base+`/audio/${key}.wav`), bytes=new Uint8Array(await response.arrayBuffer());
    assert(response.ok && bytes.length>44 && new TextDecoder().decode(bytes.slice(0,4))==='RIFF',`voice asset ${key} missing`);
  }
  await action(page,'开始');
  await wait(page,"confirmed.hand.actor===confirmed.you",'own first opportunity');
  await wait(page,"mediaProof.some(p=>p.kind==='start' && p.text==='中文播报 · 轮到你出牌。')",'own priority reminder');
  const firstReminder=await evaluate(page,"mediaProof.filter(p=>p.kind==='start' && p.text==='中文播报 · 轮到你出牌。').length");
  await evaluate(page,'state()');
  assert(await evaluate(page,"mediaProof.filter(p=>p.kind==='start' && p.text==='中文播报 · 轮到你出牌。').length")===firstReminder,'query repeated reminder');
  // 前台刷新HTTP在途时，真实WS快照先到；同一机会只允许一个恢复提醒。
  const recoveryStart=await evaluate(page,'mediaProof.length');
  await send('Fetch.enable',{patterns:[{urlPattern:'*/api/state',requestStage:'Request'}]},page.session);
  await evaluate(page,"document.dispatchEvent(new Event('visibilitychange'))");
  const pauseEnd=Date.now()+5000;
  while(!pausedRequests.has(page.session) && Date.now()<pauseEnd) await new Promise(resolve=>setTimeout(resolve,25));
  assert(pausedRequests.has(page.session),'foreground state refresh did not pause');
  const visitor=await openPage(390);
  await wait(visitor,"confirmed?.you && socket?.readyState===WebSocket.OPEN",'visitor delivers live room change');
  await wait(page,`mediaProof.slice(${recoveryStart}).some(p=>p.kind==='start' && /轮到你出牌，还剩/.test(p.text))`,'live recovery reminder before HTTP');
  await send('Fetch.continueRequest',{requestId:pausedRequests.get(page.session)},page.session);
  await send('Fetch.disable',{},page.session);
  pausedRequests.delete(page.session);
  await new Promise(resolve=>setTimeout(resolve,600));
  assert(await evaluate(page,`mediaProof.slice(${recoveryStart}).filter(p=>p.kind==='start' && /轮到你出牌，还剩/.test(p.text)).length===1`),'foreground HTTP/WS race repeated current reminder');
  await action(visitor,'退出');
  await click('#sound-toggle');
  const mutedCount=await evaluate(page,"mediaProof.filter(p=>p.kind==='start').length");
  await action(page,'过牌');
  assert(await evaluate(page,"mediaProof.filter(p=>p.kind==='start').length")===mutedCount,'mute must suppress all voices');
  await wait(page,"confirmed.hand.actor===confirmed.you",'next own opportunity while muted');
  await click('#sound-toggle');
  await wait(page,"mediaProof.some(p=>p.kind==='start' && /轮到你出牌，还剩\\d+ 秒。/.test(p.text))",'unmute current actual remaining only');
  await evaluate(page,"document.getElementById('volume').value='35';document.getElementById('volume').dispatchEvent(new Event('input'))");
  assert(await evaluate(page,"JSON.parse(localStorage.getItem('poker.voice')).volume===.35"),'volume preference not remembered');
  // 让真实服务器期限推进到十秒；媒体播放不暂停牌局。
  await new Promise(resolve=>setTimeout(resolve,20500));
  await wait(page,"mediaProof.some(p=>p.kind==='start' && p.text==='中文播报 · 请尽快行动，还剩十秒。')",'single ten-second reminder');
  await evaluate(page,'state()');
  assert(await evaluate(page,"mediaProof.filter(p=>p.kind==='start' && p.text==='中文播报 · 请尽快行动，还剩十秒。').length===1"),'ten-second warning duplicated');
  await wait(page,"/剩余 [1-9] 秒/.test(document.getElementById('countdown').textContent)",'less than ten seconds before late recovery');
  await click('#sound-toggle');
  await click('#sound-toggle');
  await wait(page,"mediaProof.some(p=>p.kind==='start' && /轮到你出牌，还剩[1-9] 秒。/.test(p.text))",'late unmute only actual remaining');
  assert(await evaluate(page,"mediaProof.filter(p=>p.kind==='start' && p.text==='中文播报 · 请尽快行动，还剩十秒。').length===1"),'late unmute added stale ten-second warning');
  await action(page,'下注');
  await wait(page,"mediaProof.some(p=>p.kind==='start' && p.text==='中文播报 · 玩家一下注十筹码。')",'real exact bet voice');
  await wait(page,"confirmed.hand.actor===confirmed.you",'new opportunity after bot call');
  const proof=await evaluate(page,'mediaProof');
  assert(proof.some(p=>p.kind==='stop'),'priority did not stop old audio');
  const restored=await openPage(1280,page.context);
  await wait(restored,"confirmed?.you && socket?.readyState===WebSocket.OPEN",'same-browser new page reconnect');
  assert(await evaluate(restored,"document.getElementById('volume').value==='35' && document.getElementById('sound-toggle').textContent==='恢复声音' && mediaProof.length===0"),'restored preferences should require interaction without historical speech');
  await send('Network.enable',{},restored.session);
  await send('Network.setBlockedURLs',{urls:['*audio/*']},restored.session);
  const pos=await evaluate(restored,"(()=>{const r=document.getElementById('sound-toggle').getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2};})()");
  await send('Input.dispatchMouseEvent',{type:'mousePressed',button:'left',clickCount:1,...pos},restored.session);
  await send('Input.dispatchMouseEvent',{type:'mouseReleased',button:'left',clickCount:1,...pos},restored.session);
  await wait(restored,"document.getElementById('voice-status').dataset.voiceState==='blocked' && document.getElementById('voice-status').textContent.includes('游戏可继续')",'real media fetch failure visible');
  await action(restored,'过牌');
  await send('Network.setBlockedURLs',{urls:[]},restored.session);
  await wait(restored,"confirmed.hand.actor===confirmed.you",'game clock continues after voice failure');
  const recoverPos=await evaluate(restored,"(()=>{const r=document.getElementById('sound-toggle').getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2};})()");
  await send('Input.dispatchMouseEvent',{type:'mousePressed',button:'left',clickCount:1,...recoverPos},restored.session);
  await send('Input.dispatchMouseEvent',{type:'mouseReleased',button:'left',clickCount:1,...recoverPos},restored.session);
  await wait(restored,"mediaProof.some(p=>p.kind==='start' && p.state==='running' && /轮到你出牌，还剩/.test(p.text))",'media failure recovers current reminder only');
  // 旧页动作仍在网络中时被新页接管：WS和HTTP同故障不能截断第一次播报。
  await send('Fetch.enable',{patterns:[{urlPattern:'*/api/command',requestStage:'Request'}]},restored.session);
  await evaluate(restored,"[...document.querySelectorAll('#actions button')].find(b=>b.textContent.startsWith('过牌')).click()");
  const takeoverPause=Date.now()+5000;
  while(!pausedRequests.has(restored.session) && Date.now()<takeoverPause) await new Promise(resolve=>setTimeout(resolve,25));
  assert(pausedRequests.has(restored.session),'old command not held before takeover');
  const takeover=await openPage(1280,restored.context);
  await wait(takeover,"confirmed?.you && socket?.readyState===WebSocket.OPEN",'new control page');
  await wait(restored,"mediaProof.some(p=>p.kind==='start' && p.text==='中文播报 · 已在其他页面打开，请使用新页面。')",'first takeover voice');
  await send('Fetch.continueRequest',{requestId:pausedRequests.get(restored.session)},restored.session);
  await send('Fetch.disable',{},restored.session);
  pausedRequests.delete(restored.session);
  await wait(restored,'!sending','duplicate takeover HTTP completed');
  assert(await evaluate(restored,"mediaProof.filter(p=>p.kind==='start' && p.text==='中文播报 · 已在其他页面打开，请使用新页面。').length===1 && !mediaProof.some(p=>p.kind==='stop' && p.text==='中文播报 · 已在其他页面打开，请使用新页面。')"),'same takeover fault must be heard once without truncation');
  assert(errors.length===0,'uncaught voice browser errors');
  await fs.writeFile('artifacts/browser-voice-report.json',JSON.stringify({passed:true,browser:info.Browser,assets:Object.keys(manifest.phrases).length,cases:['trusted gesture real WAV playback','local fixed voice assets','one reminder per opportunity and silent query','foreground HTTP/WS recovery race deduplicated','mute suppresses all voices','unmute actual remaining','volume remembered','ten seconds once with unchanged real timer','late unmute suppresses ten-second repeat','exact bet amount speech','priority stops queued audio','same-browser reconnect remembers preferences without replay','real media failure visible, game continues and sound recovers','duplicate takeover fault heard once without truncation'],proof,restoredProof:await evaluate(restored,'mediaProof'),errors},null,2));
  console.log('Voice browser checks passed; real media proof saved.');
} finally {
  for (const context of contexts) await send('Target.disposeBrowserContext',{browserContextId:context});
  socket.close();
}
