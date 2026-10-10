import assert from 'node:assert/strict';
import {writeFile} from 'node:fs/promises';
import {createBrowserClient} from './browser-client.mjs';
const client=await createBrowserClient(process.env.POKER_URL||'http://127.0.0.1:18082',{casino:true}),pages=[],report={};
const query=p=>client.evaluate(p,'fetch("/api/state").then(r=>r.json())');
try{
 for(let i=0;i<5;i++){const p=await client.openPage(i===0?1280:390);pages.push(p);await client.wait(p,"socket?.readyState===WebSocket.OPEN&&!connecting",'joined');}
 await client.action(pages[0],'开始新一局');await client.action(pages[0],'全押 99');
 for(let i=1;i<5;i++)await client.action(pages[i],'跟注 99');
 await client.wait(pages[0],"document.getElementById('game-info').textContent.includes('本局结束')",'settled');
 const before=await query(pages[0]),allin=before.hand.record.find(e=>e.action==='allin');
 await client.evaluate(pages[0],"document.getElementById('record-process').scrollTop=0");
 await client.wait(pages[0],"!document.querySelector('.record-follow').hidden",'lookback');
 await client.evaluate(pages[0],"document.getElementById('retry').click()");
 await client.wait(pages[0],`socket?.readyState===WebSocket.OPEN&&!connecting&&document.querySelectorAll('.record-entry').length>${before.hand.record.length}`,'reconnected history');
 report.lookback=await client.evaluate(pages[0],`({top:document.getElementById('record-process').scrollTop,hint:document.querySelector('.record-follow').textContent})`);
 assert.equal(report.lookback.top,0);assert.match(report.lookback.hint,/条新记录/);
 const restored=await query(pages[0]);assert.deepEqual(restored.hand.record.slice(0,before.hand.record.length),before.hand.record);
 assert.equal(restored.hand.record.find(e=>e.id===allin.id).balance,'0');
 await client.evaluate(pages[0],"document.getElementById('tab-settlement').click();document.getElementById('tab-process').click()");
 assert.equal(await client.evaluate(pages[0],"document.getElementById('record-process').scrollTop"),0);
 await client.evaluate(pages[0],"document.querySelector('.record-follow').click()");
 assert.equal(await client.evaluate(pages[0],"document.querySelector('.record-follow').hidden"),true);
 await client.action(pages[0],'退出房间');
 const newcomer=await client.openPage(390);pages.push(newcomer);await client.wait(newcomer,"socket?.readyState===WebSocket.OPEN&&!connecting",'new occupant');
 await client.evaluate(newcomer,"document.querySelector('[data-panel=results]').click()");
 const current=await query(newcomer);assert.notEqual(current.you,allin.participantID);assert.equal(current.seats[0].id,current.you);
 const oldRow=await client.evaluate(newcomer,`document.querySelector('[data-record-id="${allin.id}"]').textContent`);
 assert.match(oldRow,/原参赛者/);assert.match(oldRow,/可用 0/);report.original=oldRow;
 assert.equal(current.hand.players.find(p=>p.id===allin.participantID).hole.length,2); // Already public at this showdown, unchanged visibility.
 await client.action(pages[1],'开始新一局');
 await client.wait(newcomer,`document.querySelectorAll('[data-record-id="${allin.id}"]').length===0`,'new hand replaces history');
 const next=await query(newcomer);assert.notEqual(next.hand.id,current.hand.id);assert.equal(await client.evaluate(newcomer,"document.getElementById('tab-process').getAttribute('aria-selected')"),'true');
 report.handIDs=[current.hand.id,next.hand.id];report.errors=client.errors;assert.deepEqual(client.errors,[]);
 await writeFile('artifacts/hand-continuity-browser.json',JSON.stringify(report,null,2));console.log('PASS current-hand restore, immutable all-in balance, preserved lookback/tabs, original identity and next-hand reset');
}finally{
 for(const p of pages)if(await client.evaluate(p,"!document.getElementById('leave').disabled"))await client.action(p,'退出房间');
 for(const id of client.contexts)await client.send('Target.disposeBrowserContext',{browserContextId:id});client.socket.close();
}
