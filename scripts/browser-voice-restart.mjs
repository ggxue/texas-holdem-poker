// Real browser survives a local process restart; no historical event replay.
import {spawn} from 'node:child_process';
import {once} from 'node:events';
import path from 'node:path';
import fs from 'node:fs/promises';
import {createBrowserClient} from './browser-client.mjs';
const executable=path.resolve(process.argv[2] || 'artifacts/build/browser-fixture.exe');
const base='http://localhost:18082';
let child, client;
async function start() {
  child=spawn(executable,[],{windowsHide:true,stdio:['ignore','ignore','pipe']});
  let logs='';child.stderr.on('data',chunk=>logs+=chunk);
  const end=Date.now()+10000;
  while(Date.now()<end) {
    if(child.exitCode!==null)throw new Error('fixture failed: '+logs);
    try {if((await fetch(base+'/healthz')).ok)return;}catch{/* Waiting for loopback listener. */}
    await new Promise(resolve=>setTimeout(resolve,25));
  }
  throw new Error('fixture health timeout');
}
async function stop(){if(child && child.exitCode===null){const ended=once(child,'exit');child.kill();await ended;}}
try {
  await start();
  client=await createBrowserClient(base,{casino:true,newDocumentScript:`
    window.played=[];const start=AudioBufferSourceNode.prototype.start;
    AudioBufferSourceNode.prototype.start=function(...args){played.push(document.getElementById('voice-status')?.textContent);return start.apply(this,args);};
  `});
  const {send,evaluate,wait,openPage,state,action}=client;
  const page=await openPage(1280);
  await wait(page,'confirmed?.you && socket?.readyState===WebSocket.OPEN','initial room');
  const first=await state(page);
  const pos=await evaluate(page,"(()=>{const r=document.getElementById('sound-toggle').getBoundingClientRect();return{x:r.x+r.width/2,y:r.y+r.height/2};})()");
  await send('Input.dispatchMouseEvent',{type:'mousePressed',button:'left',clickCount:1,...pos},page.session);
  await send('Input.dispatchMouseEvent',{type:'mouseReleased',button:'left',clickCount:1,...pos},page.session);
  await wait(page,"played.includes('中文播报 · 声音已启用。')",'initial audio');
  await action(page,'开始');
  await wait(page,"played.filter(p=>p==='中文播报 · 开始新一局。').length===1",'first-process start voice');
  await stop();
  await start();
  await wait(page,`confirmed?.you !== ${JSON.stringify(first.you)} && socket?.readyState===WebSocket.OPEN && !confirmed.hand`,'fresh identity and empty room after process restart');
  const fresh=await state(page);
  if(fresh.seats[0].chips!==100)throw new Error('new-process chips not 100');
  if(await evaluate(page,"document.getElementById('sound-toggle').textContent==='恢复声音'")) {
    await send('Input.dispatchMouseEvent',{type:'mousePressed',button:'left',clickCount:1,...pos},page.session);
    await send('Input.dispatchMouseEvent',{type:'mouseReleased',button:'left',clickCount:1,...pos},page.session);
    await wait(page,"document.getElementById('voice-status').dataset.voiceState==='ready'",'recover audio after temporary outage');
  }
  await action(page,'开始');
  await wait(page,"played.filter(p=>p==='中文播报 · 开始新一局。').length===2",'new-process event IDs must not collide with dedup history');
  await fs.writeFile('artifacts/browser-voice-restart-report.json',JSON.stringify({passed:true,newIdentity:true,newRoomChips:100,newProcessStartVoice:true,errors:client.errors},null,2));
  console.log('Browser voice and identity recover after real process restart.');
} finally {
  if(client){for(const context of client.contexts)await client.send('Target.disposeBrowserContext',{browserContextId:context});client.socket.close();}
  await stop();
}
