import fs from 'node:fs/promises';
import {createBrowserClient} from './browser-client.mjs';
const base=(process.argv[2]||'http://localhost:18082').replace(/\/$/,'');
const client=await createBrowserClient(base,{casino:true,newDocumentScript:`
  window.mediaProof=[];
  const original=AudioBufferSourceNode.prototype.start;
  AudioBufferSourceNode.prototype.start=function(...args){mediaProof.push({state:this.context.state,duration:this.buffer?.duration,text:document.getElementById('voice-status')?.textContent});return original.apply(this,args);};
`});
const {send,evaluate,wait,openPage,state,action,socket,contexts,errors,pausedRequests}=client;
function assert(condition,message){if(!condition)throw new Error(message);}
async function screenshot(page,name){const result=await send('Page.captureScreenshot',{format:'png',captureBeyondViewport:true},page.session);await fs.writeFile(`artifacts/${name}.png`,Buffer.from(result.data,'base64'));}
async function click(page,selector){const pos=await evaluate(page,`(()=>{const r=document.querySelector(${JSON.stringify(selector)}).getBoundingClientRect();return{x:r.x+r.width/2,y:r.y+r.height/2};})()`);await send('Input.dispatchMouseEvent',{type:'mousePressed',button:'left',clickCount:1,...pos},page.session);await send('Input.dispatchMouseEvent',{type:'mouseReleased',button:'left',clickCount:1,...pos},page.session);}
try{
 const pages=[];
 for(let i=0;i<5;i++){const page=await openPage(i===2?390:1280);await wait(page,'confirmed?.you && socket?.readyState===WebSocket.OPEN','independent player entry');pages.push(page);}
 const observer=pages[4];await click(observer,'#sound-toggle');
 await wait(observer,"mediaProof.some(p=>p.state==='running'&&p.duration>0)",'actual voice enabled');
 await action(pages[0],'开始');
 await wait(pages[0],'confirmed.hand.actor===confirmed.you','first human after bot thinking');
 assert(await evaluate(pages[0],"[...document.querySelectorAll('#actions button')].some(b=>b.textContent==='全押 99'&&!b.disabled)"),'active all-in button with actual remaining 99 missing');
 assert(await evaluate(pages[0],"!document.getElementById('actions').textContent.includes('不足全押')"),'button labels must stay short');
 await screenshot(pages[0],'allin-production-before');
 await send('Fetch.enable',{patterns:[{urlPattern:'*/api/command',requestStage:'Request'}]},pages[0].session);
 const before=await state(pages[0]);
 await click(pages[0],'#actions button.allin');
 await wait(pages[0],"sending && document.getElementById('command-status').textContent.includes('待确认')",'all-in visibly pending');
 assert(await evaluate(pages[0],"[...document.querySelectorAll('#actions button')].every(b=>b.disabled)"),'pending all-in buttons remain usable');
 const pendingDeadline=Date.now()+5000;
 while(!pausedRequests.has(pages[0].session)&&Date.now()<pendingDeadline)await new Promise(resolve=>setTimeout(resolve,25));
 assert(pausedRequests.has(pages[0].session),'pending all-in request was not intercepted');
 await send('Fetch.continueRequest',{requestId:pausedRequests.get(pages[0].session)},pages[0].session);
 await send('Fetch.disable',{},pages[0].session);pausedRequests.delete(pages[0].session);
 await wait(pages[0],`confirmed.version>${before.version}&&!sending`,'all-in confirmation');
 await wait(observer,"mediaProof.some(p=>p.state==='running'&&p.duration>0&&p.text==='中文播报 · 玩家一全押九十九筹码。')",'actual full all-in audio without duplicated all-in');
 const pushed=await state(pages[1]);
 assert(pushed.hand.target===99&&pushed.hand.pot===105&&pushed.hand.players[0].allIn&&pushed.seats[0].chips===0,'real six-player transfer must be 6+99=105');
 assert(await evaluate(pages[1],"document.getElementById('actions').textContent.includes('跟注 99')&&!document.getElementById('actions').textContent.includes('不足全押')"),'response must offer plain call 99');
 assert(await evaluate(pages[1],"confirmed.hand.players.every(p=>p.id===confirmed.you?(p.hole||[]).length===2:(p.hole||[]).length===0)"),'all-in WS exposed opponent hole cards');
 await screenshot(pages[1],'allin-production-call');
 for(let i=1;i<5;i++)await action(pages[i],'跟注');
 await wait(pages[0],"confirmed.hand.stage==='finished'",'real all-in runout and six-way tie');
 const finished=await state(pages[0]);
 assert(finished.hand.pot===0&&finished.hand.board.length===5&&finished.hand.players.every(p=>p.invested===100&&p.won===100&&p.balance===100),'six-player pool 600 must split 100 each');
 await evaluate(pages[0],"document.getElementById('tab-settlement').click()");
 assert(await evaluate(pages[0],"document.documentElement.scrollHeight===innerHeight&&document.documentElement.scrollWidth===innerWidth&&[...document.querySelectorAll('#results .result-player')].every(p=>p.getBoundingClientRect().bottom<=innerHeight)"),'desktop six-player results overflow');
 await screenshot(pages[0],'allin-production-six-results');await screenshot(pages[2],'allin-production-mobile');
 assert(await evaluate(pages[2],"document.documentElement.scrollWidth===innerWidth"),'mobile horizontal overflow');
 await action(pages[0],'开始');await wait(pages[0],'confirmed.hand.actor===confirmed.you','next hand after all-in');
 assert((await state(pages[0])).hand.pot===6,'next hand resets pool and charges one ante per player');
 // 用真实投入和弃牌形成不同钱包，随后验收短额回应；不注入余额或局内状态。
 await action(pages[0],'全押');await action(pages[1],'跟注');
 for(let i=2;i<5;i++)await action(pages[i],'弃牌');
 await wait(pages[0],"confirmed.hand.stage==='finished'",'three-way tie creates differing wallets');
 const unequal=await state(pages[0]);
 assert(unequal.hand.players[0].won===101&&unequal.hand.players[1].won===101&&unequal.hand.players[5].won===101&&unequal.seats[2].chips===99,'pool 303 must split 101 to three survivors');
 await action(pages[0],'开始');await wait(pages[0],'confirmed.hand.actor===confirmed.you','unequal stack hand starts');
 await action(pages[0],'全押');await action(pages[1],'跟注');
 await wait(pages[2],'confirmed.hand.actor===confirmed.you','mobile short response');
 assert(await evaluate(pages[2],"[...document.querySelectorAll('#actions button')].some(b=>b.textContent==='跟注 98')&&[...document.querySelectorAll('#actions button')].some(b=>b.textContent==='全押 98')&&!document.getElementById('actions').textContent.includes('不足全押')&&document.getElementById('action-hint').textContent==='需跟 100 · 本次跟注 98 后全押'"),'short response needs plain professional labels and auxiliary explanation');
 assert(await evaluate(pages[2],"document.documentElement.scrollWidth===innerWidth"),'short response mobile horizontal overflow');
 await screenshot(pages[2],'allin-production-short-mobile');
 await action(pages[2],'跟注');
 const short=await state(pages[3]);assert(short.hand.target===100&&short.hand.players[2].allIn&&short.hand.players[2].invested===99&&short.seats[2].chips===0,'short call must keep target 100 and enter all-in state');
 await action(pages[3],'全押');await action(pages[4],'跟注');
 await wait(pages[0],"confirmed.hand.stage==='finished'",'unequal all-ins still split whole single pool');
 assert((await state(pages[0])).hand.players.every(p=>p.won===100),'small stacks share full pool 600 equally');
 if(process.argv.includes('--icon')){
  const icon=await evaluate(pages[0],"document.querySelector('link[rel=icon]')?.href");assert(icon&&new URL(icon).origin===new URL(base).origin,'local favicon link missing');
  const response=await fetch(icon);assert(response.ok&&response.headers.get('content-type').startsWith('image/svg+xml'),'favicon fetch or SVG type failed');
 }
 assert(errors.length===0,'browser runtime errors');
 await fs.writeFile('artifacts/allin-browser-report.json',JSON.stringify({browser:client.info.Browser,cases:['six independent seats and exact all-in button','real pending HTTP disables all controls','actual Mandarin all-in audio reads amount once','confirmed real HTTP/WS transfer and privacy','call 99 without parenthetical label','six-way runout and pot 600 split','1280x720 and 390px layout','next hand reset','legitimate wallet differences and real mobile short call/all-in labels','short 98 responds to target 100 and shares whole pool',...(process.argv.includes('--icon')?['local SVG favicon']:[])],errors},null,2));
 console.log('All-in production browser checks passed.');
}finally{
 for(const context of contexts)await send('Target.disposeBrowserContext',{browserContextId:context});socket.close();
}
