"use strict";
// 本局过程的呈现集中在此模块；只读取确认视图，不维护第二份牌局。
function createHandRecord({panel, renderSettlement}) {
  const stages={start:"开局",preflop:"翻牌前",flop:"翻牌",turn:"转牌",river:"河牌",finished:"摊牌与结算"};
  const actions={check:"过牌",fold:"弃牌",bet:"下注",call:"跟注",allin:"全押"};
  const kinds={start:"本局开始",ante:"底注",deal:"发出私人手牌",flop:"翻牌 · 公开前三张公共牌",turn:"转牌 · 公开第四张公共牌",river:"河牌 · 公开第五张公共牌",showdown:"摊牌 · 比较最佳五张",settlement:"结算 · 分配单一底池",award:"获得奖项",refill:"筹码补给",reset:"钱包重设为 100",join:"进入房间",return:"返回牌桌",disconnect:"断线 · 保留宽限",leave:"离开房间",host:"接任房主"};
  const reasons={manual:"手动行动",robot:"机器人行动",timeout:"行动超时 · 自动处理",departure:"离房导致弃牌 · 已投入不退",grace:"连接宽限到期",reconnect:"重新入房 · 可用筹码设为 100",refill:"开局前归零 · 底注前补至 100"};
  let handKey="",identity="",version=-1,count=0,tab="process",following=true,unseen=0,savedTop=0,generation=0,updating=false;
  const opened=new Map();
  panel.setAttribute("aria-label","本局过程与结算");
  panel.innerHTML='<div class="record-header"><h2>本局</h2><span class="record-live"></span></div><div class="record-tabs" role="tablist" aria-label="本局内容"><button type="button" role="tab" id="tab-process" aria-controls="record-process" aria-selected="true">过程</button><button type="button" role="tab" id="tab-settlement" aria-controls="record-settlement" aria-selected="false">结算</button></div><p class="record-summary"></p><div id="record-process" class="record-scroll" role="tabpanel" aria-labelledby="tab-process" tabindex="0"></div><section id="record-settlement" class="record-settlement" role="tabpanel" aria-labelledby="tab-settlement" hidden></section><button type="button" class="record-follow" hidden></button><p class="record-footer">本局保留至下一局开始 · 刷新可恢复公开过程</p>';
  const scroll=panel.querySelector(".record-scroll"),settlement=panel.querySelector(".record-settlement"),follow=panel.querySelector(".record-follow");
  function node(tag,cls,text){const n=document.createElement(tag);n.className=cls;if(text!==undefined)n.textContent=text;return n;}
  function followHint(){follow.hidden=following||tab!=="process";follow.textContent=unseen?`${unseen} 条新记录 · 回到最新 ↓`:"正在回看 · 回到最新 ↓";}
  function position(token,top){requestAnimationFrame(()=>{if(token!==generation)return;if(scroll.clientHeight>0){scroll.scrollTop=following?scroll.scrollHeight:top;if(tab==="process")savedTop=scroll.scrollTop;}requestAnimationFrame(()=>{if(token===generation)updating=false;});});}
  new ResizeObserver(()=>{if(tab!=="process"||!scroll.clientHeight)return;updating=true;position(generation,savedTop);}).observe(scroll); // 手机详情显示或尺寸变化时恢复跟随／回看。
  function select(next){
    if(tab==="process"&&scroll.clientHeight>0)savedTop=scroll.scrollTop;
    updating=true;const token=++generation;tab=next;
    scroll.hidden=tab!=="process";settlement.hidden=tab!=="settlement";panel.classList.toggle("show-settlement",tab==="settlement");
    for(const b of panel.querySelectorAll("[role=tab]"))b.setAttribute("aria-selected",String(b.id===`tab-${tab}`));
    followHint();if(tab==="process")position(token,savedTop);else requestAnimationFrame(()=>{if(token===generation)updating=false;});
  }
  panel.querySelector("#tab-process").onclick=()=>select("process");panel.querySelector("#tab-settlement").onclick=()=>select("settlement");
  scroll.addEventListener("scroll",()=>{if(updating||tab!=="process"||!scroll.clientHeight)return;savedTop=scroll.scrollTop;following=scroll.scrollHeight-scroll.clientHeight-scroll.scrollTop<16;if(following)unseen=0;followHint();});
  follow.onclick=()=>{following=true;unseen=0;scroll.scrollTop=scroll.scrollHeight;followHint();};
  function participant(e,view){
    if(!e.participantID)return "牌桌";
    let name=e.participantID==="bot"?"机器人":`玩家 ${e.seat+1}`;
    if(e.participantID===view.you)name+=" · 你";
    const original=view.hand.players.some(p=>p.id===e.participantID);
    if(e.participantID!=="bot"&&view.seats[e.seat]?.id!==e.participantID)name+=original?" · 原参赛者":" · 已离房";
    else if(!original)name+=" · 待下一局";
    return name;
  }
  function entry(e,view){
    const row=node("article",`record-entry ${e.allIn?"allin":""} ${e.kind==="award"?"award":""}`);row.dataset.recordId=e.id;
    const top=node("div","entry-top");top.append(node("span","entry-no",`#${e.seq}`),node("time","",new Date(e.at).toLocaleTimeString("zh-CN",{hour12:false})));
    const title=node("div","entry-title");let action=e.kind==="action"?actions[e.action]||e.action:kinds[e.kind]||e.kind;
    if(["ante","action","award","refill"].includes(e.kind)&&e.amount!==undefined)action+=` ${e.amount}`;
    if(e.allIn&&e.action!=="allin")action+=" · 全押";
    title.append(node("strong","",participant(e,view)),node("span","entry-action",action));
    const money=node("div","entry-money");const pot=node("span","","底池 ");pot.append(node("b","",e.pot));money.append(pot);
    if(e.balance!==undefined){const balance=node("span","","可用 ");balance.append(node("b","",e.balance));money.append(balance);}
    row.append(top,title,money);if(e.reason)row.append(node("p","entry-reason",reasons[e.reason]||e.reason));return row;
  }
  function receive(view){
    const key=view.hand?.id||"";
    if(identity===view.you&&handKey===key&&version===view.version)return;
    const changed=identity!==view.you||handKey!==key;
    if(changed){identity=view.you;handKey=key;opened.clear();count=0;following=true;unseen=0;savedTop=0;select("process");}
    const previousTop=tab==="process"&&scroll.clientHeight>0?scroll.scrollTop:savedTop,records=view.hand?.record||[],added=Math.max(0,records.length-count);
    version=view.version;count=records.length;
    if(!following)unseen+=added;
    const token=++generation;updating=true;const fragment=document.createDocumentFragment(),groups=[];
    for(const e of records){if(groups.at(-1)?.stage!==e.stage)groups.push({stage:e.stage,entries:[]});groups.at(-1).entries.push(e);}
    for(const [i,g] of groups.entries()){
      const chapter=node("details","record-chapter");chapter.open=opened.has(g.stage)?opened.get(g.stage):i===groups.length-1;
      const summary=document.createElement("summary"),title=node("span","",stages[g.stage]||g.stage);title.append(node("small","",`${g.entries.length} 条记录 · 阶段末底池 ${g.entries.at(-1).pot}`));summary.append(title);chapter.append(summary);
      const content=node("div","chapter-content"),board=g.entries.find(e=>["flop","turn","river"].includes(e.kind))?.board;
      if(board?.length){const cards=node("div","record-board");appendCards(cards,board);content.append(cards);}
      for(const e of g.entries)content.append(entry(e,view));chapter.append(content);
      let knownOpen=chapter.open;
      chapter.addEventListener("toggle",()=>{if(chapter.isConnected&&chapter.open!==knownOpen){knownOpen=chapter.open;opened.set(g.stage,chapter.open);}});fragment.append(chapter); // 初始展开不冒充手动选择，标签／尺寸变化不使手动选择失效。
    }
    if(!records.length)fragment.append(node("p","record-empty","等待房主开始新一局 · 每人底注 1 chips"));
    scroll.replaceChildren(fragment);panel.querySelector(".record-live").textContent=view.hand?.stage==="finished"?"已结束":view.hand?"进行中":"待开局";
    panel.querySelector(".record-summary").textContent=view.hand?`${stages[records.at(-1)?.stage]||stages[view.hand.stage]} · ${records.length} 条记录 · 旧 → 新` : "过程将在开局后记录";
    renderSettlement(view,settlement);followHint();position(token,changed?0:previousTop);
  }
  return {receive};
}
