"use strict";
const cardSuits = ["♦", "♣", "♥", "♠"]; // 与服务端花色编号一致。
const cardSuitLabels = ["方块", "梅花", "红桃", "黑桃"]; // 读屏不用依赖花色字体。
const cardRanks = {11: "J", 12: "Q", 13: "K", 14: "A"}; // 点数只用于呈现，不在前端比牌。
function createCard(card, kind = "face") { // 手、公、示例及结果共用简洁卡片。
  const node = document.createElement("span"); // 不加载图片或保留无权查看的牌数据。
  node.className = `playing-card card-${kind}`; // 牌面、牌背及未发轮廓分别呈现。
  node.setAttribute("role", "img"); // 给卡片完整可读名称。
  if (kind === "face" && card) { // 只处理服务端允许公开或固定示例的牌面。
    const rank = cardRanks[card.rank] || String(card.rank); // 点数保持可读。
    node.setAttribute("aria-label", `${cardSuitLabels[card.suit]}${rank}`); // 辅助技术可辨点数与花色。
    if (card.suit === 0 || card.suit === 2) node.classList.add("red"); // 红黑花色同时保留符号。
    const value = document.createElement("span"); // 角落显示点数。
    value.className = "card-rank"; // 各区域采用同一牌面结构。
    value.textContent = rank; // 不把文字当成HTML。
    const suit = document.createElement("span"); // 下方显示大花色。
    suit.className = "card-suit"; // 放大便于手机阅读。
    suit.textContent = cardSuits[card.suit]; // 与点数一起完整标识该牌。
    node.append(value, suit); // 不新增牌型判断。
  } else { // 没有可见牌面时不附带真实点数或花色。
    node.setAttribute("aria-label", kind === "back" ? "暗牌" : "未发公共牌"); // 牌背和未发牌位可区分。
  }
  return node; // 返回无外部依赖的呈现元素。
}
function appendCards(container, cards, count = cards.length, hidden = false) { // 数量来自明确区域，不伪造参赛资格。
  for (let index = 0; index < count; index++) { // 固定公共牌槽与两张手牌共用排列。
    container.append(createCard(cards[index], hidden ? "back" : cards[index] ? "face" : "empty")); // 暗牌调用不传入私人牌值。
  }
}
const referenceHands = [ // 独立固定牌例，顺序沿用项目既有十类术语。
  ["同花大顺", "Royal Flush", [[14,3],[13,3],[12,3],[11,3],[10,3]]],
  ["同花顺", "Straight Flush", [[9,1],[8,1],[7,1],[6,1],[5,1]]],
  ["四条", "Four of a Kind", [[9,3],[9,2],[9,1],[9,0],[13,3]]],
  ["葫芦", "Full House", [[10,3],[10,2],[10,1],[6,3],[6,2]]],
  ["同花", "Flush", [[14,2],[11,2],[9,2],[6,2],[2,2]]],
  ["顺子", "Straight", [[10,3],[9,2],[8,1],[7,0],[6,3]]],
  ["三条", "Three of a Kind", [[7,3],[7,2],[7,1],[14,0],[2,3]]],
  ["两对", "Two Pair", [[13,3],[13,2],[8,1],[8,0],[14,3]]],
  ["一对", "One Pair", [[12,3],[12,2],[14,1],[9,0],[3,3]]],
  ["高牌", "High Card", [[14,3],[13,2],[11,1],[9,0],[7,3]]],
];
const referenceList = document.getElementById("reference-list"); // 静态参考只构造一次。
for (const [name, english, examples] of referenceHands) { // 不根据局面改变参考内容。
  const row = document.createElement("div"); // 每类五张示例牌与中英文名称。
  row.className = "reference-row"; // 桌面与手机使用相同内容。
  const label = document.createElement("div");
  label.className = "reference-label"; // 中英文名称共享标题。
  const title = document.createElement("strong"); // 中文主名称。
  title.textContent = name; // 使用项目既有术语。
  const translation = document.createElement("small"); // 英文用于对照。
  translation.textContent = english; // 不用英文替换领域名称。
  label.append(title, translation); // 名称与牌例分别排列。
  const cards = document.createElement("div"); // 简洁五张参考牌。
  cards.className = "reference-cards"; // 小卡片不挤压牌桌。
  appendCards(cards, examples.map(([rank, suit]) => ({rank, suit}))); // 明确牌例转换只用于呈现。
  row.append(label, cards); // 保持十行一致。
  referenceList.append(row); // 完整强弱顺序常驻桌面。
}
const referencePanel = document.querySelector(".hand-reference"); // 一个面板服务两个视口。
const referenceMobile = matchMedia("(max-width: 700px)"); // 与布局断点一致，窄窗口保留可用卡片尺寸。
function adaptReference() { // 换视口时恢复对应入口状态。
  referencePanel.open = !referenceMobile.matches; // 手机默认收起，桌面始终展开。
}
adaptReference(); // 首次加载使用当前视口。
referenceMobile.addEventListener("change", adaptReference); // 调整宽度不会留下隐藏操作入口。
referencePanel.addEventListener("toggle", () => { // 桌面参考常驻，不因键盘折叠消失。
  if (!referenceMobile.matches && !referencePanel.open) referencePanel.open = true; // 手机允许自由展开收起。
});
