import assert from 'node:assert/strict';
import {writeFile} from 'node:fs/promises';
import {createBrowserClient} from './browser-client.mjs';
const client=await createBrowserClient(process.env.POKER_URL||'http://localhost:18082',{casino:true,newDocumentScript:`
 window.chipProof=[];
 const nativeAnimate=Element.prototype.animate;
 Element.prototype.animate=function(frames,timing){
  const group=this.closest('.chip-transfer-group');
  if(group)chipProof.push({id:group.dataset.recordId,kind:group.dataset.kind,amount:group.dataset.amount,owner:group.dataset.participantId,cls:this.className,text:this.textContent,timing,at:performance.now()});
  return nativeAnimate.call(this,frames,timing);
 };
`});
const pages=[];
const query=p=>client.evaluate(p,'fetch("/api/state").then(r=>r.json())');
try{
 for(let i=0;i<5;i++){const p=await client.openPage(i===0?390:1280);pages.push(p);await client.wait(p,'socket?.readyState===WebSocket.OPEN&&!connecting','joined');}
 const own=pages[0];assert.equal((await query(own)).bot.chips,100,'run this scenario with a freshly started browser fixture');
 await client.action(own,'开始');
 await client.action(own,'下注 10');
 for(let i=1;i<5;i++)await client.action(pages[i],'弃牌');
 await client.wait(own,"confirmed?.hand?.stage==='flop'&&confirmed.hand.actor===confirmed.you&&!sending",'two eligible players with26 in pot');
 assert.equal((await query(own)).hand.pot,26);
 await client.evaluate(own,"document.getElementById('retry').click()");
 await client.wait(own,"socket?.readyState===WebSocket.OPEN&&!connecting&&document.querySelector('#seats .mine .chips').textContent==='100 chips'",'actual memory reset100');
 await client.action(own,'全押 100');
 await client.wait(own,"confirmed?.hand?.stage==='finished'",'odd single pool215');
 const ended=await query(own),awards=ended.hand.record.filter(e=>e.kind==='award');
 assert.deepEqual(awards.map(e=>[e.participantID===ended.you?'you':e.participantID,e.amount,e.pot]),[['bot','108','107'],['you','107','0']]);
 assert.equal(ended.hand.pot,0);
 const lastCall=ended.hand.record.findLast(e=>e.action==='call');assert.equal(lastCall.participantID,'bot');assert.equal(lastCall.amount,'89');
 await client.wait(own,`chipProof.some(e=>e.id==='${awards[0].id}'&&e.text==='+108')&&chipProof.some(e=>e.id==='${awards[1].id}'&&e.text==='+107')`,'exact unequal winner labels');
 const proof=await client.evaluate(own,'chipProof');
 for(const e of awards){const label=proof.filter(p=>p.id===e.id&&p.text==='+'+e.amount);assert.equal(label.length,1);assert.equal(label[0].owner,e.participantID);}
 const deposit=proof.filter(p=>p.id===lastCall.id&&p.cls==='chip-transfer');assert.ok(deposit.length>0);
 const arrives=Math.max(...deposit.map(p=>p.at+p.timing.delay+p.timing.duration));
 const firstPrize=Math.min(...proof.filter(p=>awards.some(e=>e.id===p.id)&&p.cls==='chip-transfer').map(p=>p.at+p.timing.delay));
 assert.ok(firstPrize>=arrives,'CMA02 final bot contribution lands before prize bundles depart');
 assert.deepEqual(client.errors,[]);await writeFile('artifacts/chip-motion-production-odd-payout.json',JSON.stringify({awards,lastCall,proof,arrives,firstPrize,errors:client.errors},null,2));
 console.log('PASS legal unequal all-ins: pool215, exact +107/+108, bot actual89 at zero wallet, HTTP/WS dedupe and final-deposit-before-payout');
}finally{
 for(const p of pages)if(await client.evaluate(p,"!document.getElementById('leave').disabled"))await client.action(p,'退出房间');
 for(const id of client.contexts)await client.send('Target.disposeBrowserContext',{browserContextId:id});client.socket.close();
}
