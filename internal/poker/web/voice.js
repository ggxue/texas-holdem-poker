/* 本地固定女声；服务器事件只描述公开事实，媒体从不控制牌局时钟。 */
function createPokerVoice({current, now, ready, refresh}) {
  const button = document.getElementById("sound-toggle");
  const slider = document.getElementById("volume");
  const status = document.getElementById("voice-status");
  let enabled = false, volume = .6;
  try {
    const saved = JSON.parse(localStorage.getItem("poker.voice") || "null");
    if (saved) { enabled = saved.enabled === true; if (Number.isFinite(saved.volume)) volume = Math.max(0, Math.min(1, saved.volume)); }
  } catch { /* 损坏或受限的偏好不阻止游戏。 */ }
  let context, gain, source, manifest, active;
  let blocked = enabled, epoch = 0, playing = false, recovery = true;
  let restore = 0;
  let queue = [], opportunity = "", reminded = false, urgent = false, lastFault = "";
  let healthyVersion = -1;
  let identity = "";
  const seen = new Set(), buffers = new Map();
  button.disabled = false;
  slider.disabled = false;
  slider.value = String(Math.round(volume * 100));
  status.title = "后台或锁屏可能冻结页面，声音提醒不能保证必达。";
  function paint(text) {
    button.textContent = !enabled ? "启用声音" : blocked ? "恢复声音" : "静音";
    button.setAttribute("aria-pressed", String(enabled && !blocked));
    status.dataset.voiceState = !enabled ? "muted" : blocked ? "blocked" : "ready";
    if (text) status.textContent = text;
  }
  function save() {
    try { localStorage.setItem("poker.voice", JSON.stringify({enabled, volume})); } catch { /* 偏好不可写时当前页仍可播放。 */ }
  }
  function clear() {
    epoch++;
    queue = [];
    playing = false;
    if (source) { source.onended = null; source.stop(); source = null; }
    active?.cancel?.();
    active = null;
  }
  function reset() {
    restore++;
    clear();
    recovery = true;
    opportunity = "";
    reminded = urgent = false;
  }
  function own() {
    const v = current(), h = v?.hand;
    return ready() && h?.actor === v.you && h.legal?.length && h.deadline > now() ? `${h.id}:${h.turn}` : "";
  }
  function seconds() { return Math.max(0, Math.ceil((current()?.hand?.deadline - now()) / 1000)); }
  function valid(item) { return !item.opportunity || own() === item.opportunity; }
  async function loadManifest() {
    if (!manifest) {
      const response = await fetch("/audio/manifest.json", {cache:"force-cache"});
      if (!response.ok) throw new Error("voice manifest unavailable");
      manifest = (await response.json()).phrases;
    }
    return manifest;
  }
  async function buffer(key) {
    if (!buffers.has(key)) {
      const promise = (async () => {
        const phrases = await loadManifest();
        if (!Object.hasOwn(phrases, key)) throw new Error("unknown voice phrase");
        const response = await fetch(`/audio/${key}.wav`, {cache:"force-cache"});
        if (!response.ok) throw new Error("voice asset unavailable");
        return context.decodeAudioData(await response.arrayBuffer());
      })();
      buffers.set(key, promise);
      promise.catch(() => buffers.delete(key));
    }
    return buffers.get(key);
  }
  async function pump() {
    if (playing || !enabled || blocked || !queue.length) return;
    playing = true;
    const ticket = epoch, item = queue.shift();
    active = item;
    try {
      if (!valid(item)) return;
      const clips = await Promise.all(item.keys.map(buffer));
      if (ticket !== epoch || !enabled || blocked || !valid(item)) return;
      if (context.state !== "running") throw new Error("voice playback needs interaction");
      const phrases = await loadManifest();
      if (ticket !== epoch || !valid(item)) return;
      paint("中文播报 · " + item.keys.map(key => phrases[key]).join(""));
      for (const clip of clips) {
        if (ticket !== epoch || !valid(item)) break;
        await new Promise(resolve => {
          const node = context.createBufferSource();
          node.buffer = clip;
          node.connect(gain);
          source = node;
          // 取消时仍结束异步等待，旧播放不能阻塞新的行动提醒。
          const cancel = () => { node.onended = null; if (source === node) source = null; resolve(); };
          node.onended = cancel;
          node.start();
          item.cancel = cancel;
        });
      }
    } catch {
      if (ticket === epoch) {
        clear();
        blocked = true;
        paint("声音未能播放，点击“恢复声音”；游戏可继续。后台提醒不能保证必达。");
      }
    } finally {
      if (ticket === epoch) { playing = false; source = null; active = null; void pump(); }
    }
  }
  function say(keys, options = {}) {
    if (!enabled || blocked) return;
    if (options.priority) clear();
    queue.push({keys, opportunity:options.opportunity || ""});
    void pump();
  }
  function remind(restored) {
    const token = own();
    if (!token || !enabled || blocked) return;
    opportunity = token;
    reminded = true;
    const remaining = Math.min(30, seconds());
    urgent = restored && remaining <= 10;
    say([restored ? `remaining-${remaining}` : "reminder"], {priority:true, opportunity:token});
  }
  function tick() {
    const token = own();
    if (active?.opportunity && active.opportunity !== token) clear();
    if (token !== opportunity) {
      opportunity = token;
      reminded = urgent = false;
    }
    if (token && reminded && !urgent && seconds() <= 10) {
      urgent = true;
      say(["urgent"], {priority:true, opportunity:token});
    }
  }
  function actionReminder(restored) { remind(restored); }
  function receive(v, live) {
    if (v.you !== identity) { // 服务重启会签发新身份；版本序号可从头开始。
      identity = v.you;
      seen.clear();
      healthyVersion = -1;
      lastFault = "";
      reset(); // 不把新进程事件误当旧事件，也不回放旧声音。
    }
    if (live && v.version > healthyVersion) { lastFault = ""; healthyVersion = v.version; }
    if (live) {
      for (const e of v.announcements || []) {
        if (seen.has(e.id)) continue;
        seen.add(e.id);
        if (seen.size > 1024) seen.delete(seen.values().next().value);
        if (ready()) {
          const keys = eventPhrases(e);
          if (keys.length) say(keys);
        }
      }
    }
    const token = own();
    if (token !== opportunity) { opportunity = token; reminded = urgent = false; }
    if (live && ready()) {
      if (recovery) { recovery = false; if (token) actionReminder(true); }
      else if (token && !reminded) actionReminder(false);
    }
    tick();
  }
  function fault(code) {
    if (code === lastFault || code === "stale_state" || code === "stale_turn") return;
    lastFault = code;
    say([code === "taken_over" ? "takeover" : code === "room_full" ? "roomfull" : code === "connection" ? "connection" : "error"], {priority:true});
  }
  async function enable() {
    enabled = true;
    save();
    clear();
    try {
      if (!context) {
        context = new AudioContext();
        gain = context.createGain();
        gain.gain.value = volume;
        gain.connect(context.destination);
      }
      await context.resume();
      if (context.state !== "running") throw new Error("interaction required");
      blocked = false;
      paint("中文声音已启用 · 后台提醒不能保证必达。");
      // 首次或恢复声音不追播历史，只提醒此刻仍有效的本人机会。
      if (own()) actionReminder(true); else say(["enabled"]);
    } catch {
      blocked = true;
      paint("声音需要交互，点击“恢复声音”；游戏可继续。");
    }
  }
  button.addEventListener("click", () => {
    if (enabled && !blocked) {
      enabled = false;
      clear();
      save();
      paint("中文声音已静音（包括行动提醒） · 后台提醒不能保证必达。");
    } else void enable();
  });
  slider.addEventListener("input", () => {
    volume = Number(slider.value) / 100;
    if (gain) gain.gain.value = volume;
    save();
  });
  document.addEventListener("visibilitychange", async () => {
    if (!document.hidden) {
      reset();
      const request = restore;
      try {
        await refresh();
        if (request !== restore) return; // 重连或更新的恢复流程已替换这次查询。
        recovery = false;
        if (!reminded || opportunity !== own()) actionReminder(true); // WS已提醒同一机会时不再补一遍。
      } catch { if (request === restore) fault("connection"); }
    }
  });
  paint(enabled ? "点击“恢复声音”启用中文播报 · 后台提醒不能保证必达。" : "中文声音待启用 · 后台提醒不能保证必达。");
  return {receive, tick, reset, fault};
}

function eventPhrases(e) {
  const seat = `seat-${e.seat}`;
  if (e.kind === "action") {
    const keys = [seat];
    if (e.reason === "timeout") keys.push("timeout");
    keys.push(e.action === "allin" ? "allin-action" : e.action);
    if (["bet", "call", "allin"].includes(e.action)) keys.push(...chipNumberPhrases(e.amount), "chips");
    if (e.allIn && e.action !== "allin") keys.push("allin");
    return keys;
  }
  if (e.kind === "winner") return [seat, "winner", ...chipNumberPhrases(e.amount), "chips"];
  if (["join", "leave", "disconnect", "return", "host"].includes(e.kind)) return [seat, e.kind];
  if (["start", "deal", "flop", "turn", "river", "showdown", "settlement", "thinking"].includes(e.kind)) return [e.kind];
  return [];
}

// 按四位组组合服务端int64十进制金额，完全不经过JS浮点数。
function chipNumberPhrases(amount) {
  if (!/^\d{1,19}$/.test(amount)) throw new Error("invalid confirmed chip amount");
  const n = BigInt(amount);
  if (n === 0n) return ["n-0"];
  const groups = [];
  for (let rest=n; rest>0n; rest/=10000n) groups.push(Number(rest%10000n));
  const result = [], units = ["", "n-wan", "n-yi", "n-zhao", "n-jing"];
  let missing = false;
  for (let g=groups.length-1; g>=0; g--) {
    const group = groups[g];
    if (!group) { missing = true; continue; }
    if (result.length && (missing || group<1000)) result.push("n-0");
    missing = false;
    let gap = false;
    for (let position=3; position>=0; position--) {
      const digit = Math.floor(group/(10**position))%10;
      if (!digit) { if (group%(10**position)) gap = true; continue; }
      if (gap && result.length && result.at(-1)!=="n-0") result.push("n-0");
      gap = false;
      if (!(digit===1 && position===1 && !result.length)) result.push(`n-${digit}`);
      if (position) result.push(["", "n-ten", "n-hundred", "n-thousand"][position]);
    }
    if (g) result.push(units[g]);
  }
  return result;
}
