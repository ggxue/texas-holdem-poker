"use strict";
const statusLine = document.getElementById("status");
const seats = document.getElementById("seats");
const retryButton = document.getElementById("retry");
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
    }
    seats.append(card);
  });
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
    if (socket !== connection) return;
    statusLine.textContent = "连接或唤醒中…";
    reconnectTimer = setTimeout(connect, retryDelay);
    retryDelay = Math.min(retryDelay * 2, 15000);
  };
}
retryButton.addEventListener("click", connect);
connect();
