"use strict";
const statusLine = document.getElementById("status");
const seats = document.getElementById("seats");
const retryButton = document.getElementById("retry");
const gameInfo = document.getElementById("game-info");
const actions = document.getElementById("actions");
const countdown = document.getElementById("countdown");
const leaveButton = document.getElementById("leave"); // 顶部退出入口保持服务端占座权限。
const results = document.getElementById("results"); // 结算在独立区域呈现，保留服务器快照。
const commandStatus = document.getElementById("command-status"); // 提交确认与连接状态分开，避免覆盖故障信息。
const actionHint = document.getElementById("action-hint"); // 简短说明放在按钮旁，不附加括号状态。
let clockOffset = 0;
let lastServerTime = 0;
let sending = false;
let leftRoom = false;
let takenOver = false;
const pageID = crypto.randomUUID();
const stageNames = {preflop: "翻牌前", flop: "翻牌", turn: "转牌", river: "河牌", finished: "本局结束"};
let confirmed = null;
let socket = null;
let reconnectTimer = null;
let connecting = false;
let retryDelay = 1000;
const voice = createPokerVoice({current:()=>confirmed, now:()=>Date.now()+clockOffset,
  ready:()=>socket?.readyState===WebSocket.OPEN && !connecting && !takenOver && !leftRoom,
  refresh:()=>state()}); // 声音读取确认快照，不能修改计时或余额。
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
function render(view, live = false) {
  commandStatus.hidden = !sending || takenOver; // 只有未被接管的待响应命令显示提交提示。
  commandStatus.textContent = sending ? "操作结果待确认…" : ""; // 按钮禁用时仍有明确确认状态。
  if (view.error === "taken_over") {
    if (takenOver) return; // HTTP与WS重复接管故障不得截断第一次本人播报。
    voice.reset();
    voice.fault("taken_over");
    takenOver = true;
    clearTimeout(reconnectTimer);
    if (socket) { socket.onclose = null; socket.close(); socket = null; }
    actions.replaceChildren();
    retryButton.disabled = true;
    leaveButton.disabled = true; // 旧控制页不能通过顶部入口继续操作。
    commandStatus.hidden = true; // 接管提示替代旧命令的等待状态。
    statusLine.textContent = messages.taken_over;
    return;
  }
  if (takenOver) return;
  if (view.error) { statusLine.textContent = messages[view.error] || "暂时无法完成操作。"; voice.fault(view.error); return; }
  if (confirmed && view.you === confirmed.you && view.version < confirmed.version) return;
  confirmed = view;
  if (view.serverTime > lastServerTime) { lastServerTime = view.serverTime; clockOffset = view.serverTime - Date.now(); }
  renderSeats(view);
  const hand = view.hand;
  const community = document.createElement("section");
  community.className = "community";
  community.setAttribute("aria-label", "公共牌及底池");
  const pot = document.createElement("div");
  pot.className = "pot";
  const stack = createChipStack(hand?.pot || 0, "pot", "底池");
  const value = document.createElement("div");
  value.className = "pot-value";
  const label = document.createElement("span");
  label.textContent = hand?.stage === "finished" ? "底池 · 已分配" : "底池";
  const amount = document.createElement("strong");
  amount.textContent = `${hand?.pot || 0} chips`;
  value.append(label, amount);
  if (stack) pot.append(stack);
  pot.append(value);
  const board = document.createElement("div");
  board.className = "board";
  board.setAttribute("aria-label", "公共牌");
  appendCards(board, hand?.board || [], 5);
  community.append(pot, board);
  seats.append(community);
  gameInfo.textContent = hand ? stageNames[hand.stage] : "等待房主开始 · 每人底注 1"; // 状态栏在桌面上方。
  if (hand?.actor) {
    const actor = hand.players.find(player => player.id === hand.actor);
    gameInfo.textContent += ` · 当前行动：${actor?.id === view.bot.id ? "机器人" : `玩家 ${(actor?.seat ?? 0) + 1}`}`;
    if (hand.botThinking) gameInfo.textContent += " · 思考中";
  }
  actions.replaceChildren();
  if (view.host === view.you && (!hand || hand.stage === "finished") && !view.seats.some(player => player?.disconnectedUntil || player?.connectingUntil)) addAction("start", "开始新一局");
  for (const action of hand?.legal || []) {
    const labels = {check: "过牌", fold: "弃牌", bet: `下注 ${hand.betAmount}`, call: `跟注 ${hand.callAmount}`, allin: `全押 ${hand.allInAmount}`};
    addAction(action, labels[action] || action);
  }
  actionHint.hidden = sending || connecting || takenOver || !hand?.legal?.length;
  actionHint.textContent = "";
  if (!actionHint.hidden) {
    const remaining = BigInt(hand.allInAmount), required = BigInt(hand.callRequired);
    actionHint.textContent = required > 0n
      ? remaining <= required ? `需跟 ${hand.callRequired} · 本次跟注 ${hand.callAmount} 后全押` : `需跟 ${hand.callRequired} · 跟注后剩 ${remaining - BigInt(hand.callAmount)}`
      : "过牌继续参与 · 弃牌放弃本局 · 全押投入全部余款";
  }
  leaveButton.disabled = sending || !view.seats.some(player => player?.id === view.you); // 顶部退出入口跟随占座与提交状态。
  if (!actions.children.length) { // 没有合法动作也说明等待原因。
    const waiting = document.createElement("span"); // 不生成可操作的伪按钮。
    waiting.className = "action-empty"; // 等待说明保持可读。
    waiting.textContent = sending ? "操作结果待确认…" : hand?.stage !== "finished" && hand ? "等待当前玩家行动" : "等待房主开局"; // 不推定动作成功。
    actions.append(waiting); // 留住清楚的操作入口。
  }
  updateCountdown();
  renderResults(view); // 只在确认结算后展示结果，新局替换旧结果。
  voice.receive(view, live); // HTTP与推送共享事件编号，查询与重绘不追播。
}
function renderSeats(view) {
  seats.replaceChildren();
  [...view.seats, view.bot].forEach((player, i) => {
    const place = document.createElement("div");
    place.className = "place";
    place.dataset.seat = String(i);
    if (player?.disconnectedUntil) place.classList.add("offline");
    const seat = document.createElement("article");
    seat.dataset.seat = String(i);
    seat.className = "seat";
    seat.classList.toggle("mine", !!player && player.id === view.you);
    seat.classList.toggle("current", !!player && player.id === view.hand?.actor);
    const title = document.createElement("div");
    title.className = "seat-title";
    const name = document.createElement("h2");
    name.textContent = i === view.seats.length ? "机器人" : `玩家 ${i + 1}`;
    title.append(name);
    seat.append(title);
    if (player) {
      const badge = document.createElement("span");
      badge.className = "badge";
      badge.textContent = [player.id === view.you ? "你" : "", player.id === view.host ? "房主" : ""].filter(Boolean).join(" · ");
      title.append(badge);
      const p = view.hand?.players.find(item => item.id === player.id);
      const hole = document.createElement("div");
      hole.className = "hole-cards";
      if (p) appendCards(hole, p.hole || [], 2, !p.hole?.length);
      const money = document.createElement("div");
      money.className = "seat-money";
      const chips = document.createElement("strong");
      chips.className = "chips";
      chips.textContent = `${player.chips} chips`;
      const investment = document.createElement("span");
      investment.className = "investment";
      investment.textContent = `投入 ${p?.invested || 0} chips`;
      money.append(chips, investment);
      const detail = document.createElement("div");
      detail.className = "seat-status";
      const labels = [];
      if (player.disconnectedUntil) { detail.dataset.grace = String(player.disconnectedUntil); labels.push("断线宽限中"); }
      else if (player.connectingUntil) labels.push("建立连接中");
      if (p) {
        if (p.folded) labels.push("弃牌");
        else if (p.allIn) labels.push("全押");
        if (view.hand.actor === player.id) labels.push(view.hand.botThinking ? "思考中" : "当前行动");
        else if (view.hand.stage === "finished") labels.push(p.won > 0 ? "赢家 · 已派奖" : "已结束");
      } else labels.push(view.hand && view.hand.stage !== "finished" ? "等待下一局" : "准备开局");
      detail.dataset.labels = labels.slice(player.disconnectedUntil ? 1 : 0).join(" · ");
      detail.textContent = labels.join(" · ");
      seat.append(hole, money, detail);
      const wallet = document.createElement("div");
      wallet.className = "wallet";
      wallet.dataset.owner = String(i);
      const stack = createChipStack(player.chips, "wallet-" + i, name.textContent + "可用筹码");
      if (stack) wallet.append(stack);
      place.append(seat, wallet);
    } else {
      seat.classList.add("empty");
      const empty = document.createElement("div");
      empty.className = "seat-status";
      empty.textContent = "空位 · 等待入房";
      seat.append(empty);
      place.append(seat);
    }
    seats.append(place);
  });
}
function renderResults(view) {
  results.replaceChildren();
  const finished = view.hand?.stage === "finished";
  results.classList.toggle("settled", finished);
  const heading = document.createElement("h2");
  heading.textContent = finished ? "本局结算" : "牌局进行中";
  results.append(heading);
  if (!finished) {
    const waiting = document.createElement("p");
    waiting.className = "result-waiting";
    waiting.textContent = view.hand ? "结算后，在这里核对全部参赛者的奖项、牌型和最佳五张。其他玩家暗牌保持隐藏。" : "等待房主开始新一局 · 每人底注 1 chips";
    results.append(waiting);
    return;
  }
  const total = document.createElement("p");
  total.className = "result-total";
  total.textContent = `本局分配 ${view.hand.players.reduce((sum, p) => sum + p.won, 0)} chips`;
  const hint = document.createElement("p");
  hint.className = "result-hint";
  hint.textContent = "结算余额为快照；当前余额见席位。";
  results.append(total, hint);
  const list = document.createElement("div");
  list.className = "result-list";
  for (const p of view.hand.players) {
    const row = document.createElement("article");
    row.className = "result-player" + (p.won > 0 ? " winner" : "");
    const top = document.createElement("div");
    top.className = "result-top";
    const name = document.createElement("h3");
    name.textContent = `${p.id === "bot" ? "机器人" : `玩家 ${p.seat + 1}`}${p.id === view.you ? " · 你" : ""}${p.won > 0 ? " · 赢家" : ""}`;
    const gain = document.createElement("strong");
    gain.className = "result-gain";
    gain.textContent = `获得 ${p.won} chips`;
    top.append(name, gain);
    const money = document.createElement("p");
    money.className = "result-money";
    money.textContent = `投入 ${p.invested} · 结算余额 ${p.balance}`;
    const strength = document.createElement("div");
    strength.className = "result-strength";
    const label = document.createElement("span");
    if (p.strength) {
      label.textContent = p.strength.category;
      const best = document.createElement("div");
      best.className = "result-cards best-five";
      best.setAttribute("aria-label", "Best Five 最佳五张");
      appendCards(best, p.strength.cards);
      strength.append(label, best);
    } else {
      label.textContent = p.folded ? "弃牌 · 无领奖资格" : "提前胜出 · 未强制亮牌";
      strength.append(label);
      // 本人允许查看的暗牌仍在席位，不给其他人添隐藏的真实牌值。
      if (p.id === view.you && p.hole?.length) {
        const ownHole = document.createElement("div");
        ownHole.className = "result-cards";
        ownHole.setAttribute("aria-label", "本人的手牌");
        appendCards(ownHole, p.hole);
        strength.append(ownHole);
      }
    }
    row.append(top, money, strength);
    list.append(row);
  }
  results.append(list);
}
function updateCountdown() {
  if (!confirmed || takenOver || leftRoom) { countdown.textContent = ""; return; }
  const now = Date.now() + clockOffset;
  countdown.textContent = confirmed.hand?.deadline ? `当前行动剩余 ${Math.max(0, Math.ceil((confirmed.hand.deadline - now) / 1000))} 秒` : "";
  document.querySelectorAll("[data-grace]").forEach(node => {
    node.textContent = `断线 ${Math.max(0, Math.ceil((Number(node.dataset.grace) - now) / 1000))} 秒` + (node.dataset.labels ? " · " + node.dataset.labels : "");
  });
  voice.tick(); // 十秒提示依据原服务器期限，不逐秒朗读。
}
setInterval(updateCountdown, 250);
function addAction(action, label) {
  const button = document.createElement("button");
  button.type = "button";
  button.textContent = label;
  button.disabled = sending || connecting;
  if (["bet", "call", "start"].includes(action)) button.className = "primary";
  if (action === "allin") button.className = "allin";
  const hints = {check: "无需补筹码，继续参与", fold: "放弃本局获奖资格，已投入不退", call: "按可用筹码补本轮欠额", allin: "投入全部剩余筹码"};
  button.title = hints[action] || "";
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
    render(result, true);
    if (response.ok && action === "leave") {
      leftRoom = true;
      voice.reset(); // 离房后不保留自己的旧行动提醒。
      clearTimeout(reconnectTimer);
      if (socket) { socket.onclose = null; socket.close(); socket = null; }
      statusLine.textContent = "已退出房间 · 已投入筹码不退";
      retryButton.textContent = "重新入房";
    }
    if (result.error) await state();
  } catch { statusLine.textContent = "操作结果待确认，请重新连接查看服务器状态。"; voice.fault("connection"); }
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
  voice.reset(); // 重连只提醒恢复后的当前机会。
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
    voice.fault("connection");
  } finally { connecting = false; retryButton.disabled = takenOver; }
}
function openSocket() {
  const query = new URLSearchParams({pageID, control: String(confirmed.control)});
  const connection = new WebSocket(`${location.protocol === "https:" ? "wss:" : "ws:"}//${location.host}/api/ws?${query}`);
  socket = connection;
  connection.onopen = () => { if (socket !== connection) return; retryDelay = 1000; statusLine.textContent = "已入房 · 筹码仅保存在本次服务内存中"; };
  connection.onmessage = event => {
    if (socket !== connection) return;
    try { render(JSON.parse(event.data), true); } catch { statusLine.textContent = "状态读取失败，请重新连接。"; voice.fault("connection"); }
  };
  connection.onclose = () => {
    if (socket !== connection || leftRoom || takenOver) return;
    statusLine.textContent = "连接或唤醒中…";
    voice.reset();
    voice.fault("connection");
    reconnectTimer = setTimeout(connect, retryDelay);
    retryDelay = Math.min(retryDelay * 2, 15000);
  };
}
retryButton.addEventListener("click", connect);
leaveButton.addEventListener("click", () => sendAction("leave")); // 退出仍经原公开命令处理。
connect();
