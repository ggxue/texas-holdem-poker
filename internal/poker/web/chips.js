"use strict";
// 本地实体筹码：精确金额以席位数字为准，堆只示意余额规模。
function createChipStack(amount,scope='pot',label='底池'){
 if(!(Number(amount)>0))return null;
 const count=amount<10?1:amount<50?3:amount<100?5:7,towers=amount<10?1:amount<50?2:3;
 const colours=[['#f0d79e','#b38b4e','#60401f'],['#cf6b70','#7d2633','#39141d'],['#f2e9d3','#b7a582','#5e4f37']];
 const defs=colours.map((c,t)=>`<linearGradient id="chip-${scope}-${t}" x2="0" y2="1"><stop stop-color="${c[0]}"/><stop offset="1" stop-color="${c[1]}"/></linearGradient>`).join('');
 const disks=Array.from({length:towers},(_,t)=>Array.from({length:Math.max(1,count-t)},(_,i)=>{
  const x=19+t*28,y=35-i*3-(t===1?3:0),c=colours[t];
  return `<ellipse cx="${x}" cy="${y+3}" rx="17" ry="6" fill="${c[2]}"/><path d="M${x-17} ${y}v3a17 6 0 0 0 34 0v-3" fill="${c[1]}" stroke="${c[2]}" stroke-width=".7"/><ellipse cx="${x}" cy="${y}" rx="17" ry="6" fill="url(#chip-${scope}-${t})" stroke="${c[2]}" stroke-width="1"/><ellipse cx="${x}" cy="${y}" rx="15" ry="5" fill="none" stroke="#fff0cf" stroke-width="2" stroke-dasharray="4 6"/><ellipse cx="${x}" cy="${y}" rx="10" ry="3.5" fill="none" stroke="${c[2]}" stroke-width=".8"/><text x="${x}" y="${y+2}" text-anchor="middle" font-size="6" fill="${c[2]}">♠</text>`;
 }).join('')).join('');
 const markup = `<svg class="physical-chips" viewBox="0 0 96 46" role="img" aria-label="${label} ${amount} chips，筹码堆仅示意规模" data-amount="${amount}"><defs>${defs}</defs><ellipse cx="45" cy="42" rx="44" ry="3" fill="#000" opacity=".35"/>${disks}</svg>`;
 const container=document.createElement("div");
 container.innerHTML=markup; // 仅固定模板、服务器整数和内部固定scope，不接收用户HTML。
 return container.firstElementChild;
}
