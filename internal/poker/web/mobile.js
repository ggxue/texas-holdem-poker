"use strict";

// Presentation only: retain the live controls/results rather than duplicate their state or listeners.
function createMobileTable() {
  const phone = matchMedia("(max-width:700px)");
  const sheet = document.getElementById("mobile-sheet");
  const heading = document.getElementById("mobile-sheet-title");
  const body = sheet.querySelector(".mobile-sheet-body");
  const closeButton = document.getElementById("mobile-sheet-close");
  const tools = [...document.querySelectorAll(".mobile-tools button")];
  const controls = document.querySelector(".header-controls");
  const reference = document.querySelector(".hand-reference");
  const footer = document.querySelector(".app-footer");
  const results = document.getElementById("results");
  const locations = new Map();
  for (const node of [controls, reference, footer, results]) {
    const marker = document.createComment("mobile detail location");
    node.before(marker);
    locations.set(node, marker);
  }
  let panel = null;
  let opener = null;
  function restore() {
    for (const [node, marker] of locations) marker.after(node);
  }
  function close(returnFocus=true) {
    sheet.hidden = true;
    panel = null;
    restore();
    body.replaceChildren();
    tools.forEach(button => button.setAttribute("aria-expanded", "false"));
    if (returnFocus && phone.matches) opener?.focus();
  }
  function players() {
    const roster = document.createElement("div");
    roster.className = "mobile-roster";
    // These seat contents already contain only the recipient's permitted cards and amounts.
    for (const seat of document.querySelectorAll("#seats .seat")) roster.append(seat.cloneNode(true));
    body.replaceChildren(roster);
  }
  function open(button) {
    if (!phone.matches) return;
    if (panel === button.dataset.panel) { close(); return; }
    restore();
    body.replaceChildren();
    panel = button.dataset.panel;
    opener = button;
    heading.textContent = {players:"玩家详情", results:"本局", more:"设置与说明"}[panel];
    if (panel === "players") players();
    else if (panel === "results") body.append(results);
    else body.append(controls, reference, footer);
    tools.forEach(tool => tool.setAttribute("aria-expanded", String(tool === button)));
    sheet.hidden = false;
    body.scrollTop = 0;
    closeButton.focus();
  }
  tools.forEach(button => button.addEventListener("click", () => open(button)));
  closeButton.addEventListener("click", () => close());
  document.addEventListener("keydown", event => {
    if (event.key === "Escape" && !sheet.hidden) { event.preventDefault(); close(); }
  });
  phone.addEventListener("change", () => {
    close(false);
  });
  const measure = () => {
    const root = document.documentElement;
    root.style.setProperty("--action-height", `${document.querySelector(".action-panel").getBoundingClientRect().height}px`);
    root.style.setProperty("--mobile-header-bottom", `${document.querySelector(".app-header").getBoundingClientRect().bottom}px`);
  };
  const observer = new ResizeObserver(measure);
  observer.observe(document.querySelector(".action-panel"));
  observer.observe(document.querySelector(".app-header"));
  return {refresh() { if (panel === "players") players(); }};
}
