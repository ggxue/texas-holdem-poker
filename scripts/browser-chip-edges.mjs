import assert from 'node:assert/strict';
import {writeFile} from 'node:fs/promises';
import {createBrowserClient} from './browser-client.mjs';
const client=await createBrowserClient(process.env.POKER_URL||'http://127.0.0.1:18082',{casino:true,newDocumentScript:`
 window.chipProof=[];
 const nativeAnimate=Element.prototype.animate;
 Element.prototype.animate=function(frames,timing){
  const group=this.closest('.chip-transfer-group');
  if(group)chipProof.push({id:group.dataset.recordId,kind:group.dataset.kind,amount:group.dataset.amount,owner:group.dataset.participantId,cls:this.className,text:this.textContent,frames,timing});
  return nativeAnimate.call(this,frames,timing);
 };
 // Simulate only the agreed preference-storage failure; gameplay session storage remains native.
 const get=Storage.prototype.getItem,set=Storage.prototype.setItem;
 Storage.prototype.getItem=function(key){if(key==='poker.motion')throw new DOMException('Blocked','SecurityError');return get.call(this,key);};
 Storage.prototype.setItem=function(key,value){if(key==='poker.motion')throw new DOMException('Blocked','SecurityError');return set.call(this,key,value);};
`});
const pages=[],report={};
const query=p=>client.evaluate(p,'fetch("/api/state").then(r=>r.json())');
try{
 for(let i=0;i<5;i++){const p=await client.openPage(1280);pages.push(p);await client.wait(p,'socket?.readyState===WebSocket.OPEN&&!connecting','ready despite blocked preference storage');}
 const own=pages[0];
 await client.evaluate(own,"document.getElementById('motion-toggle').click();document.getElementById('motion-toggle').click()");
 assert.equal(await client.evaluate(own,"document.getElementById('motion-toggle').textContent"),'动效：开启');
 await client.action(own,'开始');
 await client.action(own,'下注 10');
 const bet=(await query(own)).hand.record.find(e=>e.action==='bet');
 await client.wait(own,`chipProof.some(e=>e.id==='${bet.id}'&&e.cls==='chip-transfer')`,'ordinary wager flight');
 report.bet=await client.evaluate(own,`chipProof.filter(e=>e.id==='${bet.id}')`);
 assert.ok(report.bet.every(e=>e.amount==='10'));
 assert.ok(report.bet.find(e=>e.cls==='chip-transfer').frames.some(f=>f.offset===.52),'A must use an arc through a raised midpoint');
 await client.send('Emulation.setDeviceMetricsOverride',{width:390,height:720,deviceScaleFactor:1,mobile:true},own.session);
 await client.wait(own,"document.querySelectorAll('.chip-motion-layer *').length===0",'resize cancels displaced flights');
 await client.action(pages[1],'跟注 10');
 const call=(await query(own)).hand.record.find(e=>e.action==='call');
 await client.wait(own,`chipProof.some(e=>e.id==='${call.id}'&&e.cls==='chip-transfer')`,'other player exact call flight');
 const proof=await client.evaluate(own,'chipProof.length');
 await client.evaluate(own,"document.getElementById('retry').click()");
 await client.wait(own,"socket?.readyState===WebSocket.OPEN&&!connecting&&document.querySelector('#seats .seat.mine .chips').textContent==='100 chips'",'actual reconnect resets wallet');
 assert.equal(await client.evaluate(own,'chipProof.length'),proof,'reconnect must seed history without replaying transfers');
 assert.equal((await query(own)).hand.record.find(e=>e.id===bet.id).balance,'89','original debit snapshot survives reconnect');
 await client.action(pages[2],'弃牌');
 const beforeBackground=await client.evaluate(own,'chipProof.length');
 await client.send('Page.setWebLifecycleState',{state:'frozen'},own.session);
 await client.action(pages[3],'跟注 10');
 await client.action(pages[4],'跟注 10');
 await client.wait(pages[1],"confirmed?.hand?.stage==='flop'",'others continue while page is frozen');
 await client.send('Page.setWebLifecycleState',{state:'active'},own.session);
 await client.send('Page.bringToFront',{},own.session);
 await client.wait(own,"!document.hidden&&socket?.readyState===WebSocket.OPEN&&!connecting",'visible and synchronized after actual background return');
 await client.wait(own,"confirmed?.hand?.stage==='flop'",'public flop after bot actual call');
 assert.equal(await client.evaluate(own,'chipProof.length'),beforeBackground,'background return must not replay buffered transfers');
 await client.action(own,'弃牌');
 await client.action(pages[1],'弃牌');
 await client.action(pages[3],'弃牌');
 await client.action(pages[4],'弃牌');
 await client.wait(own,"confirmed?.hand?.stage==='finished'",'single bot winner');
 const ended=await query(own),award=ended.hand.record.find(e=>e.kind==='award');
 assert.equal(award.participantID,'bot');assert.equal(award.amount,'56');assert.equal(award.pot,'0');
 assert.equal(ended.hand.record.filter(e=>e.kind==='award').length,1);
 assert.ok(!ended.hand.record.some(e=>e.kind==='showdown'));
 await client.wait(own,`chipProof.some(e=>e.id==='${award.id}'&&e.text==='+56')`,'entire pool reaches the sole winner');
 report.award=await client.evaluate(own,`chipProof.filter(e=>e.id==='${award.id}')`);
 assert.ok(report.award.every(e=>e.owner==='bot'&&e.amount==='56'));
 assert.equal(report.award.filter(e=>e.text==='+56').length,1);
 const recordedIDs=new Set(ended.hand.record.filter(e=>['ante','action','award'].includes(e.kind)&&BigInt(e.amount||'0')>0n).map(e=>e.id));
 assert.ok(await client.evaluate(own,`chipProof.filter(e=>e.id).every(e=>${JSON.stringify([...recordedIDs])}.includes(e.id))`),'check/fold/reset must not manufacture chip transfers');
 await client.wait(own,"document.querySelectorAll('.chip-motion-layer *').length===0",'single-pool visual cleanup');
 report.errors=client.errors;assert.deepEqual(client.errors,[]);
 await writeFile('artifacts/chip-motion-production-edges.json',JSON.stringify(report,null,2));
 console.log('PASS exact ordinary bet/call, arc midpoint, resize cleanup, reconnect/background without replay, storage blocked, single winner full pool56 and no fold/reset flights');
}finally{
 for(const p of pages)if(await client.evaluate(p,"!document.getElementById('leave').disabled"))await client.action(p,'退出房间');
 for(const id of client.contexts)await client.send('Target.disposeBrowserContext',{browserContextId:id});client.socket.close();
}
