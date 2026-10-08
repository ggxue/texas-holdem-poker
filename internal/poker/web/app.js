"use strict";
const statusLine = document.getElementById("status");
const seats = document.getElementById("seats");
const retryButton = document.getElementById("retry");
const gameInfo = document.getElementById("game-info");
const actions = document.getElementById("actions");
let sending = false;
let leftRoom = false;
const ranks = {11: "J", 12: "Q", 13: "K", 14: "A"};
const suitNames = ["♦", "♣", "♥", "♠"];
const stageNames = {preflop: "翻牌前", flop: "翻牌", turn: "转牌", river: "河牌", finished: "本局结束"};
function cardsText(cards) { return cards.map(card => `${suitNames[card.suit]}${ranks[card.rank] || card.rank}`).join(" "); }
let confirmed = null;
let socket = null;
let reconnectTimer = null;
let connecting = false;
let retryDelay = 1000;
const messages = {
  room_full: "房间已满，请稍后再试。",
  unavailable: "服务暂不可用，请稍后重新连接。",
  stale_state: "房间状态已更新，正在同步…",
  request_conflict: "请求标识冲突，请重新连接。",
  identity_required: "身份凭证已失效，请重新连接。",
  invalid_command: "操作无效，请重新连接。",
  origin_denied: "此页面不能操作该房间。",
  host_only: "只有房主可以开局。",
  hand_active: "请先完成当前牌局。",
  stale_turn: "行动机会已更新，请按当前状态操作。",
  not_your_turn: "尚未轮到你。",
  no_active_hand: "请等待房主开始新一局。",
  invalid_action: "当前不能执行该操作。",
  not_in_room: "你已离开房间，可重新入房。",
};
function render(view) {
  if (view.error) { statusLine.textContent = messages[view.error] || "暂时无法完成操作。"; return; }
  if (confirmed && view.you === confirmed.you && view.version < confirmed.version) return;
  confirmed = view;
  seats.replaceChildren();
  [...view.seats, view.bot].forEach((player, i) => {
    const card = document.createElement("article");
    card.className = "seat" + (player && player.id === view.you ? " mine" : "");
    const name = document.createElement("h2");
    name.textContent = i === 2 ? "机器人" : `真人 ${i + 1}`;
    card.append(name);
    const chips = document.createElement("div");
    chips.className = player ? "chips" : "empty";
    chips.textContent = player ? `${player.chips} 筹码` : "等待玩家入房";
    card.append(chips);
    if (player) {
      const badge = document.createElement("div");
      badge.className = "badge";
      badge.textContent = [player.id === view.you ? "你" : "", player.id === view.host ? "房主" : ""].filter(Boolean).join(" · ");
      card.append(badge);
      const participant = view.hand?.players.find(item => item.id === player.id);
      const detail = document.createElement("p");
      if (participant) {
        if (participant.folded) badge.textContent += " · 弃牌";
        else if (participant.allIn) badge.textContent += " · 全押";
        if (view.hand.actor === player.id) badge.textContent += " · 当前行动";
        detail.textContent = `本局投入 ${participant.invested} · ${participant.hole?.length ? cardsText(participant.hole) : "🂠 🂠"}`;
        if (participant.strength) detail.textContent += ` · ${participant.strength.category}：${cardsText(participant.strength.cards)}`;
        if (view.hand.stage === "finished") detail.textContent += ` · 获得 ${participant.won} · 结算余额 ${participant.balance}`;
      } else detail.textContent = view.hand && view.hand.stage !== "finished" ? "等待下一局" : "准备开局";
      card.append(detail);
    }
    seats.append(card);
  });
  const hand = view.hand;
  gameInfo.textContent = hand ? `${stageNames[hand.stage]} · 底池 ${hand.pot} · 公共牌 ${cardsText(hand.board) || "尚未发牌"}` : "等待房主开始 · 每人底注 1";
  if (hand?.actor) {
    const actor = hand.players.find(player => player.id === hand.actor);
    gameInfo.textContent += ` · 当前行动：${actor?.seat === 2 ? "机器人" : `真人 ${(actor?.seat ?? 0) + 1}`}`;
  }
  actions.replaceChildren();
  if (view.host === view.you && (!hand || hand.stage === "finished")) addAction("start", "开始新一局");
  for (const action of hand?.legal || []) {
    const labels = {check: "过牌", fold: "弃牌", bet: `下注 ${hand.betAmount}${hand.betAmount < 10 ? "（不足全押）" : ""}`, call: `跟注 ${hand.callAmount}${hand.callAmount < hand.target ? "（不足全押）" : ""}`};
    addAction(action, labels[action] || action);
  }
  if (view.seats.some(player => player?.id === view.you)) addAction("leave", "退出房间");
}
function addAction(action, label) {
  const button = document.createElement("button");
  button.type = "button";
  button.textContent = label;
  button.disabled = sending;
  button.addEventListener("click", () => sendAction(action));
  actions.append(button);
}
async function sendAction(action) {
  if (sending || !confirmed) return;
  sending = true;
  const command = {requestID: crypto.randomUUID(), version: confirmed.version, action,
    handID: confirmed.hand?.id || 0, turnID: confirmed.hand?.turn || 0};
  render(confirmed);
  try {
    const response = await fetch("/api/command", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify(command)});
    const result = await response.json();
    render(result);
    if (response.ok && action === "leave") {
      leftRoom = true;
      clearTimeout(reconnectTimer);
      if (socket) { socket.onclose = null; socket.close(); socket = null; }
      statusLine.textContent = "已退出房间 · 已投入筹码不退";
      retryButton.textContent = "重新入房";
    }
    if (result.error) await state();
  } catch { statusLine.textContent = "操作结果待确认，请重新连接查看服务器状态。"; }
  finally { sending = false; if (confirmed) render(confirmed); }
}
async function state() {
  const response = await fetch("/api/state", { cache: "no-store" });
  const view = await response.json();
  if (!response.ok) throw new Error(messages[view.error] || "服务暂不可用。");
  render(view);
  return view;
}
function pendingJoin(view) {
  let saved = null;
  try { saved = JSON.parse(sessionStorage.getItem("poker.pendingJoin")); } catch { /* Ignore malformed local UI data. */ }
  if (saved && saved.you === view.you && saved.command) return saved.command;
  const command = { requestID: crypto.randomUUID(), version: view.version, action: "join" };
  sessionStorage.setItem("poker.pendingJoin", JSON.stringify({ you: view.you, command }));
  return command;
}
async function connect() {
  if (connecting) return;
  connecting = true;
  leftRoom = false;
  retryButton.textContent = "重新连接";
  retryButton.disabled = true;
  clearTimeout(reconnectTimer);
  if (socket) { socket.onclose = null; socket.close(); socket = null; }
  statusLine.textContent = "正在连接房间…";
  try {
    for (let attempt = 0; attempt < 3; attempt++) {
      const current = await state();
      const command = pendingJoin(current);
      const response = await fetch("/api/command", {
        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(command),
      });
      const result = await response.json();
      if (response.status === 503) throw new Error("服务暂不可用，请重新连接。");
      sessionStorage.removeItem("poker.pendingJoin");
      if (result.error === "stale_state") continue;
      if (!response.ok) { render(result); return; }
      await state();
      openSocket();
      return;
    }
    throw new Error("房间状态正在变化，请重新连接。");
  } catch (error) {
    statusLine.textContent = error.message || "连接中断，入房结果待确认。请重新连接。";
  } finally { connecting = false; retryButton.disabled = false; }
}
function openSocket() {
  const connection = new WebSocket(`${location.protocol === "https:" ? "wss:" : "ws:"}//${location.host}/api/ws`);
  socket = connection;
  connection.onopen = () => { if (socket !== connection) return; retryDelay = 1000; statusLine.textContent = "已入房 · 筹码仅保存在本次服务内存中"; };
  connection.onmessage = event => {
    if (socket !== connection) return;
    try { render(JSON.parse(event.data)); } catch { statusLine.textContent = "状态读取失败，请重新连接。"; }
  };
  connection.onclose = () => {
    if (socket !== connection || leftRoom) return;
    statusLine.textContent = "连接或唤醒中…";
    reconnectTimer = setTimeout(connect, retryDelay);
    retryDelay = Math.min(retryDelay * 2, 15000);
  };
}
retryButton.addEventListener("click", connect);
connect();
