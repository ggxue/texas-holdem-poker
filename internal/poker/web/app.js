"use strict";
const statusLine = document.getElementById("status");
const seats = document.getElementById("seats");
const retryButton = document.getElementById("retry");
const gameInfo = document.getElementById("game-info");
const actions = document.getElementById("actions");
const countdown = document.getElementById("countdown");
const leaveButton = document.getElementById("leave"); // 顶部退出入口保持服务端占座权限。
let clockOffset = 0;
let lastServerTime = 0;
let sending = false;
let leftRoom = false;
let takenOver = false;
const pageID = crypto.randomUUID();
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
  taken_over: "已在其他页面打开，请使用新页面。",
  connection_grace: "有玩家断线，等待重连或宽限到期后再开局。",
  chips_overflow: "筹码数值超出可表示范围，此操作没有生效。",
};
function render(view) {
  if (view.error === "taken_over") {
    takenOver = true;
    clearTimeout(reconnectTimer);
    if (socket) { socket.onclose = null; socket.close(); socket = null; }
    actions.replaceChildren();
    retryButton.disabled = true;
    leaveButton.disabled = true; // 旧控制页不能通过顶部入口继续操作。
    statusLine.textContent = messages.taken_over;
    return;
  }
  if (takenOver) return;
  if (view.error) { statusLine.textContent = messages[view.error] || "暂时无法完成操作。"; return; }
  if (confirmed && view.you === confirmed.you && view.version < confirmed.version) return;
  confirmed = view;
  if (view.serverTime > lastServerTime) { lastServerTime = view.serverTime; clockOffset = view.serverTime - Date.now(); }
  seats.replaceChildren();
  [...view.seats, view.bot].forEach((player, i) => {
    const card = document.createElement("article");
    card.dataset.seat = String(i); // 固定布局只依座位编号，不随观看身份旋转。
    card.className = "seat" + (player && player.id === view.you ? " mine" : "") + (player && view.hand?.actor === player.id ? " current" : ""); // 本人和当前行动分别突出。
    const name = document.createElement("h2");
    name.textContent = i === view.seats.length ? "机器人" : `真人 ${i + 1}`;
    card.append(name);
    if (player) {
      const badge = document.createElement("div");
      badge.className = "badge";
      badge.textContent = [player.id === view.you ? "你" : "", player.id === view.host ? "房主" : ""].filter(Boolean).join(" · ");
      card.append(badge);
      const participant = view.hand?.players.find(item => item.id === player.id);
      const hole = document.createElement("div"); // 两张手牌区域位于名称身份之后。
      hole.className = "hole-cards"; // 空位和待局者不画虚假暗牌。
      const chips = document.createElement("div"); // 余额仅展示确认视图。
      chips.className = "chips"; // 金色区分可用筹码。
      chips.textContent = `${player.chips} 筹码`; // 牌面票再统一chips与图标。
      const investment = document.createElement("div"); // 投入和状态位于余额之后。
      investment.className = "investment"; // 与可用余额分别显示。
      const detail = document.createElement("div"); // 座位状态不和名称混排。
      detail.className = "seat-status"; // 保留读屏文字。
      const labels = []; // 各状态来自当前确认视图。
      if (player.disconnectedUntil) labels.push("断线宽限中"); // 展示原离房计时状态。
      if (player.connectingUntil) labels.push("正在建立控制连接"); // 握手期间不伪装在线。
      if (participant) { // 仅本局参赛身份有手牌和投入。
        if (participant.folded) labels.push("弃牌"); // 弃牌状态仍可辨认。
        else if (participant.allIn) labels.push("全押"); // 全押标记与余额分开。
        if (view.hand.actor === player.id) labels.push("当前行动"); // 当前行动与本人高亮同时存在。
        if (player.id === "bot" && view.hand.botThinking) labels.push("思考中"); // 保留真实等待提示。
        hole.textContent = participant.hole?.length ? cardsText(participant.hole) : "暗牌 · 暗牌"; // 不从隐藏DOM恢复他人暗牌。
        investment.textContent = `本局投入 ${participant.invested}`; // 本局投入由服务端确认。
        if (participant.strength) labels.push(`${participant.strength.category}：${cardsText(participant.strength.cards)}`); // 结果票前仍保留公开结果。
        if (view.hand.stage === "finished") labels.push(`获得 ${participant.won} · 结算余额 ${participant.balance}`); // 不自行重新派奖。
      } else { // 没有本局资格不显示已发手牌。
        labels.push(view.hand && view.hand.stage !== "finished" ? "等待下一局" : "准备开局"); // 空闲成员与待局者分别提示。
      }
      detail.textContent = labels.join(" · "); // 用短标签呈现状态。
      card.append(hole, chips, investment, detail); // 按约定的信息顺序追加。
    } else { // 空席仍保留固定位置。
      const empty = document.createElement("div"); // 不制造暗牌或余额。
      empty.className = "empty"; // 空位采用次要文字。
      empty.textContent = "空位 · 等待玩家入房"; // 清楚标明可入房的位置。
      card.append(empty); // 保持六席稳定。
    }
    seats.append(card);
  });
  const hand = view.hand;
  const community = document.createElement("div"); // 中央区域只呈现确认的底池与公共牌。
  community.className = "community"; // 桌面居中，手机在五名真人之后。
  const pot = document.createElement("div"); // 底池始终在公共牌上方。
  pot.className = "pot"; // 独立显示底池数值。
  pot.textContent = `底池 ${hand?.pot || 0} 筹码`; // 不使用参考图示例数额。
  const board = document.createElement("div"); // 牌面票会替换成固定五位卡片。
  board.className = "board"; // 保留公共牌区域。
  board.textContent = cardsText(hand?.board || []) || "公共牌 · 尚未发牌"; // 仅展示已公开牌。
  community.append(pot, board); // 维持底池与公共牌层级。
  seats.append(community); // 与六个固定席共用区域关系。
  gameInfo.textContent = hand ? stageNames[hand.stage] : "等待房主开始 · 每人底注 1"; // 状态栏在桌面上方。
  if (hand?.actor) {
    const actor = hand.players.find(player => player.id === hand.actor);
    gameInfo.textContent += ` · 当前行动：${actor?.id === view.bot.id ? "机器人" : `真人 ${(actor?.seat ?? 0) + 1}`}`;
    if (hand.botThinking) gameInfo.textContent += " · 思考中";
  }
  actions.replaceChildren();
  if (view.host === view.you && (!hand || hand.stage === "finished") && !view.seats.some(player => player?.disconnectedUntil || player?.connectingUntil)) addAction("start", "开始新一局");
  for (const action of hand?.legal || []) {
    const labels = {check: "过牌", fold: "弃牌", bet: `下注 ${hand.betAmount}${hand.betAmount < 10 ? "（不足全押）" : ""}`, call: `跟注 ${hand.callAmount}${hand.callAmount < hand.target ? "（不足全押）" : ""}`};
    addAction(action, labels[action] || action);
  }
  leaveButton.disabled = sending || !view.seats.some(player => player?.id === view.you); // 顶部退出入口跟随占座与提交状态。
  if (!actions.children.length) { // 没有合法动作也说明等待原因。
    const waiting = document.createElement("span"); // 不生成可操作的伪按钮。
    waiting.className = "action-empty"; // 等待说明保持可读。
    waiting.textContent = sending ? "操作结果待确认…" : hand?.stage !== "finished" && hand ? "等待当前玩家行动" : "等待房主开局"; // 不推定动作成功。
    actions.append(waiting); // 留住清楚的操作入口。
  }
  updateCountdown();
}
function updateCountdown() {
  if (!confirmed || takenOver || leftRoom) { countdown.textContent = ""; return; }
  const now = Date.now() + clockOffset;
  const parts = [];
  if (confirmed.hand?.deadline) parts.push(`当前行动剩余 ${Math.max(0, Math.ceil((confirmed.hand.deadline - now) / 1000))} 秒`);
  confirmed.seats.forEach((player, seat) => {
    if (player?.disconnectedUntil) parts.push(`真人 ${seat + 1} 重连宽限 ${Math.max(0, Math.ceil((player.disconnectedUntil - now) / 1000))} 秒`);
  });
  countdown.textContent = parts.join(" · ");
}
setInterval(updateCountdown, 1000);
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
    handID: confirmed.hand?.id || 0, turnID: confirmed.hand?.turn || 0, pageID, control: confirmed.control};
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
  if (saved && saved.you === view.you && saved.command?.pageID === pageID) return saved.command;
  const command = { requestID: crypto.randomUUID(), version: view.version, action: "join", pageID, control: view.control };
  sessionStorage.setItem("poker.pendingJoin", JSON.stringify({ you: view.you, command }));
  return command;
}
async function connect() {
  if (connecting || takenOver) return;
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
  } finally { connecting = false; retryButton.disabled = takenOver; }
}
function openSocket() {
  const query = new URLSearchParams({pageID, control: String(confirmed.control)});
  const connection = new WebSocket(`${location.protocol === "https:" ? "wss:" : "ws:"}//${location.host}/api/ws?${query}`);
  socket = connection;
  connection.onopen = () => { if (socket !== connection) return; retryDelay = 1000; statusLine.textContent = "已入房 · 筹码仅保存在本次服务内存中"; };
  connection.onmessage = event => {
    if (socket !== connection) return;
    try { render(JSON.parse(event.data)); } catch { statusLine.textContent = "状态读取失败，请重新连接。"; }
  };
  connection.onclose = () => {
    if (socket !== connection || leftRoom || takenOver) return;
    statusLine.textContent = "连接或唤醒中…";
    reconnectTimer = setTimeout(connect, retryDelay);
    retryDelay = Math.min(retryDelay * 2, 15000);
  };
}
retryButton.addEventListener("click", connect);
leaveButton.addEventListener("click", () => sendAction("leave")); // 退出仍经原公开命令处理。
connect();
