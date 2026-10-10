// Public page behavior on the existing loopback fixture; no game implementation access.
import assert from 'node:assert/strict';
import {createBrowserClient} from './browser-client.mjs';
import {writeFile} from 'node:fs/promises';
const client=await createBrowserClient(process.env.POKER_URL||'http://127.0.0.1:18082');
let page;const pages=[],layouts=[];
try{
 page=await client.openPage(1280);
 pages.push(page);
 await client.wait(page,"confirmed?.seats.some(p=>p?.id===confirmed.you)&&socket?.readyState===WebSocket.OPEN&&!connecting",'connected');
 assert.equal(await client.evaluate(page,"document.getElementById('tab-process')?.textContent"),'过程','HRC10 current-hand process tab must exist');
 for(let i=1;i<5;i++){const joined=await client.openPage(390);pages.push(joined);await client.wait(joined,"socket?.readyState===WebSocket.OPEN&&!connecting",'other participant connected');}
 await client.action(page,'开始新一局');
 await client.wait(page,"document.querySelectorAll('.record-entry').length>=4",'current hand entries');
 await client.evaluate(page,"document.querySelector('.record-chapter summary').click()");
 const text=await client.evaluate(page,"document.getElementById('record-process').innerText");
 assert.match(text,/本局开始/);assert.match(text,/底注 1/);assert.match(text,/发出私人手牌/);
 assert.equal(await client.evaluate(page,"document.querySelectorAll('.record-chapter').length"),2);
 assert.equal(await client.evaluate(page,"document.getElementById('tab-process').getAttribute('aria-selected')"),'true');
 await client.action(page,'全押 99');
 for(let i=1;i<5;i++)await client.action(pages[i],'跟注 99');
 await client.wait(page,"document.getElementById('game-info').textContent.includes('本局结束')",'six-way settlement');
 for(const [width,height] of [[1280,720],[1920,1080],[390,720],[360,640],[320,568]]){
  await client.send('Emulation.setDeviceMetricsOverride',{width,height,deviceScaleFactor:1,mobile:width<700},page.session);
  await client.evaluate(page,"new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r)))");
  if(width<700)await client.evaluate(page,"if(document.getElementById('mobile-sheet').hidden)document.querySelector('[data-panel=results]').click()");
  await client.evaluate(page,"document.getElementById('tab-settlement').click()");
  const layout=await client.evaluate(page,`(()=>{const a=document.querySelector('.action-panel').getBoundingClientRect(),r=document.getElementById('record-settlement'),b=r.getBoundingClientRect(),rows=[...r.querySelectorAll('.result-player')];return {width:innerWidth,height:innerHeight,outerX:document.documentElement.scrollWidth-innerWidth,outerY:document.documentElement.scrollHeight-innerHeight,actionsVisible:a.bottom<=innerHeight&&a.top>=0,rows:rows.length,scroll:r.scrollHeight-r.clientHeight,allRowsInside:rows.every(n=>n.getBoundingClientRect().bottom<=b.bottom+1),drawerAboveActions:innerWidth>700||document.getElementById('mobile-sheet').getBoundingClientRect().bottom<=a.top};})()`);
  layouts.push(layout);assert.equal(layout.outerX,0);assert.equal(layout.outerY,0);assert.equal(layout.rows,6);assert.equal(layout.actionsVisible,true);assert.equal(layout.drawerAboveActions,true);
  if(width>700){assert.equal(layout.scroll,0,'full desktop settlement must not scroll');assert.equal(layout.allRowsInside,true);}
  const shot=await client.send('Page.captureScreenshot',{format:'png'},page.session);await writeFile(`artifacts/hand-record-production-${width}.png`,Buffer.from(shot.data,'base64'));
 }
 await writeFile('artifacts/hand-record-production-layouts.json',JSON.stringify(layouts,null,2));
 console.log('PASS current-hand C chapters, exact antes, public deal, six-person settlement and five sizes');
 assert.deepEqual(client.errors,[]);
}finally{
 for(const p of pages)if(await client.evaluate(p,"!document.getElementById('leave').disabled"))await client.action(p,'退出房间');
 for(const id of client.contexts)await client.send('Target.disposeBrowserContext',{browserContextId:id});client.socket.close();
}
