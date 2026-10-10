import assert from 'node:assert/strict';
import {writeFile} from 'node:fs/promises';
import {createBrowserClient} from './browser-client.mjs';
const client=await createBrowserClient(process.env.POKER_URL||'http://127.0.0.1:18082',{casino:true,newDocumentScript:`
 window.chipProof=[];
 const browserAnimate=Element.prototype.animate;
 Element.prototype.animate=function(frames,timing){
   const group=this.closest('.chip-transfer-group');
   if(group)chipProof.push({recordID:group.dataset.recordId,amount:group.dataset.amount,kind:group.dataset.kind,participantID:group.dataset.participantId,cls:this.className,text:this.textContent,frames,timing,at:Date.now()});
   return browserAnimate.call(this,frames,timing);
 };
`});
const pages=[],report={};
const query=p=>client.evaluate(p,'fetch("/api/state").then(r=>r.json())');
try{
 const own=await client.openPage(390);pages.push(own);await client.wait(own,"socket?.readyState===WebSocket.OPEN&&!connecting",'joined');
 assert.equal(await client.evaluate(own,"document.getElementById('motion-toggle')?.textContent"),'动效：开启','CMA05 default animation toggle must exist');
 for(let i=1;i<5;i++){const p=await client.openPage(1280);pages.push(p);await client.wait(p,"socket?.readyState===WebSocket.OPEN&&!connecting",'others joined');}
 await client.action(own,'开始新一局');
 await client.wait(own,"chipProof.some(p=>p.kind==='ante'&&p.cls.includes('chip-transfer'))",'ante flights');
 await client.action(own,'全押 99');
 const allin=(await query(own)).hand.record.find(e=>e.action==='allin');
 await client.wait(own,`chipProof.some(p=>p.recordID==='${allin.id}'&&p.cls.includes('chip-transfer'))`,'all-in flight');
 assert.equal(await client.evaluate(own,"document.querySelector('#seats .seat.mine .chips').textContent"),'0 chips');
 report.allin=await client.evaluate(own,`chipProof.filter(p=>p.recordID==='${allin.id}')`);assert.ok(report.allin.every(p=>p.amount==='99'));
 assert.ok(report.allin.filter(p=>p.cls==='chip-transfer').length<=7,'mobile particle budget');
 const picture=await client.send('Page.captureScreenshot',{format:'png'},own.session);await writeFile('artifacts/chip-motion-production-allin.png',Buffer.from(picture.data,'base64'));
 await client.evaluate(pages[2],"document.getElementById('motion-toggle').click()");
 await client.evaluate(own,"document.querySelector('[data-panel=results]').click()");
 const beforeDrawer=await client.evaluate(own,"chipProof.length");
 await client.action(pages[1],'跟注 99');
 await client.wait(own,"document.querySelector('#seats .place[data-seat=\"1\"] .chips').textContent==='0 chips'",'call displayed under drawer');
 assert.equal(await client.evaluate(own,"chipProof.length"),beforeDrawer,'CMA04 detail cover must suppress flights');
 assert.equal(await client.evaluate(own,"document.getElementById('mobile-sheet').hidden"),false);
 await client.evaluate(own,"document.getElementById('mobile-sheet-close').click()");
 for(let i=2;i<5;i++)await client.action(pages[i],'跟注 99');
 await client.wait(own,"document.getElementById('game-info').textContent.includes('本局结束')",'settled');
 await client.wait(own,"chipProof.some(p=>p.kind==='award'&&p.text.includes('+100'))",'six-way prize exact labels');
 const ended=await query(own);assert.equal(ended.hand.pot,0);assert.ok(ended.hand.players.every(p=>p.won===100));
 const awards=new Set(ended.hand.record.filter(e=>e.kind==='award').map(e=>e.id));
 report.payout=await client.evaluate(own,"chipProof.filter(p=>p.kind==='award')");
 assert.equal(new Set(report.payout.filter(p=>p.text.includes('+100')).map(p=>p.recordID)).size,6);
 assert.ok(report.payout.every(p=>awards.has(p.recordID)));
 const disabled=await client.evaluate(pages[2],"chipProof.filter(p=>p.kind==='award').length");assert.equal(disabled,0,'CMA05 disabled watcher must still receive results without flights');
 assert.equal((await query(pages[2])).hand.pot,0);
 const labels=report.payout.filter(p=>p.text.includes('+100'));assert.equal(labels.length,6,'CMA03 HTTP/WS must not duplicate prize labels');
 await client.send('Page.reload',{},pages[2].session);
 await client.wait(pages[2],"socket?.readyState===WebSocket.OPEN&&!connecting&&document.getElementById('motion-toggle').textContent==='动效：关闭'",'preference restored');
 assert.equal(await client.evaluate(pages[2],"chipProof.length"),0,'CMA03 reload must not replay history');
 await client.send('Emulation.setEmulatedMedia',{features:[{name:'prefers-reduced-motion',value:'reduce'}]},pages[3].session);
 await client.evaluate(pages[3],"chipProof=[]");
 await client.action(own,'开始新一局');
 await client.wait(pages[3],"chipProof.some(p=>p.kind==='ante')",'reduced static amount');
 assert.equal(await client.evaluate(pages[3],"document.querySelectorAll('.chip-transfer').length"),0,'CMA05 reduced motion must not fly chips');
 assert.ok(await client.evaluate(pages[3],"chipProof.filter(p=>p.kind==='ante').every(p=>!p.cls.includes('chip-transfer'))"));
 await client.wait(own,"document.querySelectorAll('.chip-motion-layer *').length===0",'natural visual cleanup');
 report.errors=client.errors;assert.deepEqual(client.errors,[]);await writeFile('artifacts/chip-motion-production-browser.json',JSON.stringify(report,null,2));
 console.log('PASS real antes/all-in, exact six-way payout, HTTP/WS dedupe, drawer suppression, preference reload, reduced motion and cleanup');
}finally{
 for(const p of pages)if(await client.evaluate(p,"!document.getElementById('leave').disabled"))await client.action(p,'退出房间');
 for(const id of client.contexts)await client.send('Target.disposeBrowserContext',{browserContextId:id});client.socket.close();
}
