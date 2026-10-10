"use strict";
// 动效只呈现已经确认的转移；不控制余额、合法动作、期限或声音。
function createChipMotion({current, ready, refresh}) {
  const button=document.getElementById("motion-toggle"),layer=document.createElement("div");
  layer.className="chip-motion-layer";layer.setAttribute("aria-hidden","true");document.body.append(layer);
  const reduced=matchMedia("(prefers-reduced-motion:reduce)"),phone=matchMedia("(max-width:700px)");
  const animations=new Set(),positions=new Map();
  let enabled=true,identity="",handID=null,lastSeq=0,occupants="",recovering=true,epoch=0,sequence=0,lastLanding=0,lastBatch=0;
  try { enabled=localStorage.getItem("poker.motion")!=="off"; } catch { /* 偏好受限不阻止对局。 */ }
  function paint(){button.textContent=`动效：${enabled?"开启":"关闭"}`;button.setAttribute("aria-pressed",String(enabled));button.title=reduced.matches?"系统减少动态：显示静态金额提示":"筹码只示意规模，精确金额见数字";}
  function clear(){for(const a of animations)a.cancel();animations.clear();layer.replaceChildren();positions.clear();lastLanding=0;}
  function reset(){epoch++;recovering=true;clear();}
  button.addEventListener("click",()=>{enabled=!enabled;clear();try {localStorage.setItem("poker.motion",enabled?"on":"off");}catch{/* 当前页面仍可切换。 */}paint();});
  function point(id,view){
    const seat=id===view.bot.id?view.seats.length:view.seats.findIndex(p=>p?.id===id);
    if(seat<0)return null; // 原身份已离席时，不落到同席新人。
    const wallet=document.querySelector(`#seats .place[data-seat="${seat}"] .wallet`);
    if(!wallet)return null;const r=wallet.getBoundingClientRect();return {x:r.left+r.width/2,y:r.top+r.height/2};
  }
  function pool(){const r=document.querySelector("#seats .pot")?.getBoundingClientRect();return r?{x:r.left+r.width/2,y:r.top+r.height/2}:null;}
  function bounds(){
    const r=document.querySelector(".table").getBoundingClientRect(),a=document.querySelector(".action-panel").getBoundingClientRect();
    const b={left:Math.max(0,r.left),right:Math.min(innerWidth,r.right),top:Math.max(0,r.top),bottom:Math.min(innerHeight,r.bottom,phone.matches?a.top:r.bottom)};
    layer.style.clipPath=`inset(${b.top}px ${Math.max(0,innerWidth-b.right)}px ${Math.max(0,innerHeight-b.bottom)}px ${b.left}px)`;return b;
  }
  function group(e){
    const g=document.createElement("div");g.className="chip-transfer-group";
    if(e.id)g.dataset.recordId=e.id;g.dataset.kind=e.kind;g.dataset.amount=e.amount||"0";g.dataset.participantId=e.participantID||"";layer.append(g);return g;
  }
  function node(g,cls,p){const n=document.createElement("div");n.className=cls;n.style.left=`${p.x}px`;n.style.top=`${p.y}px`;g.append(n);return n;}
  function animate(n,frames,duration,delay=0){
    if(typeof n.animate!=="function"){n.remove();return;} // 不支持动画的浏览器仍正常游玩。
    const a=n.animate(frames,{duration,delay,easing:"cubic-bezier(.22,.65,.3,1)",fill:"both"});animations.add(a);
    a.finished.then(()=>{animations.delete(a);const parent=n.parentElement;n.remove();if(parent&&!parent.children.length)parent.remove();},()=>{}); // cancel拒绝finished时不产生未处理异常。
  }
  function label(g,p,text,winner=false,delay=0,staticMode=false){
    const b=bounds(),n=node(g,`chip-amount ${winner?"chip-winner":""}`,{x:Math.max(b.left+48,Math.min(b.right-48,p.x)),y:Math.max(b.top+20,Math.min(b.bottom-23,p.y-28))});n.textContent=text;
    animate(n,staticMode?[{opacity:1},{offset:.85,opacity:1},{opacity:0}]:[{opacity:0,transform:"translate(-50%,0)"},{offset:.18,opacity:1,transform:"translate(-50%,-7px)"},{offset:.8,opacity:1,transform:"translate(-50%,-10px)"},{opacity:0,transform:"translate(-50%,-12px)"}],1000,delay);
  }
  function pulse(g,p,strong,delay){const n=node(g,`chip-landing ${strong?"strong":""}`,p);animate(n,[{transform:"translate(-50%,-50%) scale(.4)",opacity:.7},{transform:`translate(-50%,-50%) scale(${strong?1.8:1.2})`,opacity:0}],strong?350:200,delay);}
  function fly(g,from,to,count,duration,payout,delay=0){
    const dx=to.x-from.x,dy=to.y-from.y;
    for(let i=0;i<count;i++){
      const spread=(i-(count-1)/2)*3,arc=Math.min(65,Math.hypot(dx,dy)*.3)+i%3*5;
      const n=node(g,"chip-transfer",{x:from.x+spread,y:from.y-i%3*2});n.append(createTransferChip(i));
      animate(n,[{transform:"translate(-50%,-50%) scale(.8) rotate(-12deg)",opacity:0},{offset:.1,transform:`translate(calc(-50% + ${dx*.05}px),calc(-50% + ${dy*.05-12}px)) scale(1) rotate(0deg)`,opacity:1},{offset:.52,transform:`translate(calc(-50% + ${dx*.52}px),calc(-50% + ${dy*.52-arc}px)) scale(1.1) rotate(${payout?90:45}deg)`,opacity:1},{offset:.92,transform:`translate(calc(-50% + ${dx}px),calc(-50% + ${dy}px)) scale(.8) rotate(${payout?180:80}deg)`,opacity:1},{transform:`translate(calc(-50% + ${dx}px),calc(-50% + ${dy+3}px)) scale(.5)`,opacity:0}],duration,delay+i*(payout?17:18));
    }
    pulse(g,to,payout,delay+duration+count*10-80);
  }
  function play(entries,view){
    if(!enabled||document.hidden||!ready()||phone.matches&&!document.getElementById("mobile-sheet").hidden)return;
    const pot=pool();if(!pot)return;bounds();
    const incoming=entries.filter(e=>["ante","action","award"].includes(e.kind)&&BigInt(e.amount||"0")>0n);
    if(!incoming.length)return;
    const staticMode=reduced.matches,busy=performance.now()-lastBatch<200;lastBatch=performance.now();
    if(layer.querySelectorAll(".chip-transfer").length>18)clear(); // 压缩旧视觉，完整事实仍在本局记录。
    const deposits=incoming.filter(e=>e.kind!=="award"),awards=incoming.filter(e=>e.kind==="award");
    for(const e of deposits){
      const target=point(e.participantID,view);if(!target)continue;
      const source=positions.get(e.participantID)||target,g=group(e);
      if(staticMode){label(g,source,`投入 ${e.amount}`,false,0,true);continue;}
      const amount=BigInt(e.amount),desired=deposits.length>3?(phone.matches?2:3):amount<10n?3:amount<50n?5:phone.matches?7:10;
      const count=Math.min(desired,28-layer.querySelectorAll(".chip-transfer").length),duration=busy?230:e.allIn?480:300;
      if(count>0)fly(g,source,pot,count,duration,false);
      label(g,source,`-${e.amount}`);lastLanding=Math.max(lastLanding,performance.now()+duration+count*18);
    }
    if(awards.length){
      const delay=staticMode?0:Math.max(0,Math.min(650,lastLanding-performance.now()))+30;
      const total=awards.reduce((sum,e)=>sum+BigInt(e.amount),0n);
      if(!staticMode){
        const g=group({kind:"visual-pool",amount:String(total)}),mound=node(g,"chip-visual-pool",pot);mound.append(createChipStack(total,`transfer-pool-${++sequence}`,"已确认的奖项总额"));
        animate(mound,[{opacity:0,transform:"translate(-50%,-50%) scale(1.15)"},{offset:.2,opacity:1,transform:"translate(-50%,-50%) scale(1)"},{offset:.65,opacity:1,transform:"translate(-50%,-50%) scale(.85)"},{opacity:0,transform:"translate(-50%,-50%) scale(.4)"}],290,delay);
      }
      for(const e of awards){
        const winner=point(e.participantID,view);if(!winner)continue;
        const g=group(e),count=Math.min(Math.max(1,Math.floor((phone.matches?10:16)/awards.length)),28-layer.querySelectorAll(".chip-transfer").length);
        if(!staticMode&&count>0)fly(g,pot,winner,count,520,true,delay+190);
        label(g,winner,`+${e.amount}`,true,staticMode?0:delay+590,staticMode);
      }
    }
  }
  function receive(view,live){
    if(view.you!==identity){reset();identity=view.you;handID=null;lastSeq=0;occupants="";}
    const key=view.hand?.id||null;
    if(key!==handID){clear();handID=key;lastSeq=0;}
    const nextOccupants=view.seats.map(p=>p?.id||"").join(":");
    if(occupants&&occupants!==nextOccupants)clear();occupants=nextOccupants;
    const records=view.hand?.record||[],fresh=records.filter(e=>e.seq>lastSeq);lastSeq=records.at(-1)?.seq||0;
    if(live&&!recovering)play(fresh,view);
    if(live&&ready()&&!document.hidden)recovering=false; // 初始WS视图只建立历史基线。
    for(const p of [...view.seats,view.bot])if(p){const anchor=point(p.id,view);if(anchor)positions.set(p.id,anchor);}
  }
  window.addEventListener("resize",clear);
  window.addEventListener("pagehide",reset);
  document.querySelector(".workspace").addEventListener("scroll",clear,true);
  new MutationObserver(()=>{if(!document.getElementById("mobile-sheet").hidden)clear();}).observe(document.getElementById("mobile-sheet"),{attributes:true,attributeFilter:["hidden"]});
  reduced.addEventListener("change",()=>{clear();paint();});
  document.addEventListener("visibilitychange",()=>{
    reset();if(document.hidden)return;const token=epoch;
    void refresh().then(()=>{if(token===epoch&&current()?.you===identity)recovering=false;}).catch(()=>{}); // 失败仍等待重连，绝不补播旧飞行。
  });
  paint();return {receive,reset};
}
