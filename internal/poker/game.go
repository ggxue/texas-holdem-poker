package poker

import (
	"crypto/rand"
	"math"
	"math/big"
)

type participant struct {
	ID       string    `json:"id"`
	Seat     int       `json:"seat"`
	Hole     []Card    `json:"hole,omitempty"`
	Invested int64     `json:"invested"`
	Street   int64     `json:"street"`
	Folded   bool      `json:"folded"`
	AllIn    bool      `json:"allIn"`
	Won      int64     `json:"won"`
	Balance  int64     `json:"balance"`
	Strength *Strength `json:"strength,omitempty"`
	acted    bool
}
type hand struct {
	ID       int64
	Turn     int64
	Stage    string
	Pot      int64
	Board    []Card
	Players  []participant
	Actor    int
	Target   int64
	deck     []Card
	cursor   int
	showdown bool
}
type handView struct {
	ID      int64         `json:"id"`
	Turn    int64         `json:"turn"`
	Stage   string        `json:"stage"`
	Pot     int64         `json:"pot"`
	Board   []Card        `json:"board"`
	Players []participant `json:"players"`
	Actor   string        `json:"actor"`
	Target  int64         `json:"target"`
	Legal   []string      `json:"legal"`
}

func validAction(action string) bool {
	return action == "join" || action == "start" || action == "check" // 限定本票公开命令范围。
}

func shuffledDeck() ([]Card, error) {
	cards := make([]Card, 0, 52)        // 建立一副完整牌，不允许客户端指定牌序。
	for rank := 2; rank <= 14; rank++ { // 遍历全部点数。
		for suit := 0; suit < 4; suit++ { // 遍历四种花色。
			cards = append(cards, Card{rank, suit}) // 加入一张唯一的牌。
		} // 每个点数有四种花色。
	}
	for i := len(cards) - 1; i > 0; i-- { // 使用无偏的密码学随机数洗牌。
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1))) // 抽取当前区间内的交换位置。
		if err != nil {                                         // 检查随机源是否成功。
			return nil, err // 向调用者返回随机源故障。
		} // 随机源失败时不开始牌局。
		j := int(n.Int64())                     // 转为切片下标。
		cards[i], cards[j] = cards[j], cards[i] // 交换两张牌。
	}
	return cards, nil // 返回服务端私有牌序。
}

func (h *hand) visibleTo(id string) *handView {
	v := &handView{ID: h.ID, Turn: h.Turn, Stage: h.Stage, Pot: h.Pot, Board: append([]Card{}, h.Board...), Target: h.Target, Players: make([]participant, len(h.Players))} // 构造独立的公开快照。
	for i, p := range h.Players {                                                                                                                                           // 按参赛身份裁剪暗牌，座位重用不会转移手牌。
		v.Players[i] = p                             // 复制该参赛者的公开字段。
		v.Players[i].Hole = nil                      // 默认隐藏手牌。
		v.Players[i].Strength = nil                  // 默认隐藏牌力。
		if p.ID == id || (h.showdown && !p.Folded) { // 只允许本人或有效摊牌者的手牌公开。
			v.Players[i].Hole = append([]Card{}, p.Hole...) // 复制允许该身份查看的手牌。
		} // 本人或摊牌有效玩家可见手牌。
		if h.showdown && !p.Folded && p.Strength != nil { // 摊牌时公开未弃牌者的最佳五张。
			strength := *p.Strength           // 复制牌力，避免快照共享可变引用。
			v.Players[i].Strength = &strength // 把公开牌力写入快照。
		} // 摊牌公开最佳五张。
	}
	if h.Stage != "finished" && h.Actor >= 0 { // 只有未结束的牌局存在行动者。
		v.Actor = h.Players[h.Actor].ID // 公布当前行动身份。
		if v.Actor == id {              // 仅当前行动者收到合法按钮。
			v.Legal = []string{"check"} // 本票只允许过牌。
		} // 本票只交付合法过牌路径。
	}
	return v // 不发送剩余牌序。
}

func (s *room) balance(id string) int64 {
	if id == "bot" { // 区分机器人钱包与真人钱包。
		return s.Bot.Chips // 读取机器人余额。
	} // 机器人使用独立钱包。
	return s.Accounts[id].Chips // 真人钱包绑定身份。
}
func (s *room) setBalance(id string, n int64) {
	if id == "bot" { // 区分机器人钱包与真人钱包。
		s.Bot.Chips = n // 更新机器人余额。
		return          // 机器人钱包更新后结束。
	} // 更新机器人余额。
	p := s.Accounts[id] // 读取真人钱包。
	p.Chips = n         // 设置权威余额。
	s.Accounts[id] = p  // 保存该身份的钱包。
}

func (a *App) gameCommand(id string, cmd command) string {
	s := &a.state              // 全部牌局变更在App锁内处理。
	if cmd.Action == "start" { // 房主手动开局。
		if s.Host != id { // 核对请求身份是否为房主。
			return "host_only" // 拒绝非房主开局。
		} // 非房主不能开局。
		if s.Hand != nil && s.Hand.Stage != "finished" { // 检查已有牌局是否结束。
			return "hand_active" // 拒绝覆盖未结束牌局。
		} // 禁止覆盖进行中的牌局。
		deck, err := a.deck()               // 从服务端随机源取得牌序。
		if err != nil || !validDeck(deck) { // 检查随机源及整副牌的有效性。
			return "unavailable" // 故障时拒绝开局。
		} // 发牌前检查完整牌序。
		h := &hand{ID: s.Version + 1, Stage: "preflop", Actor: 0, deck: append([]Card(nil), deck...)} // 建立新的牌局。
		for seat, occupant := range s.Seats {                                                         // 固定本局真人名单。
			if occupant != "" { // 只把已占座的真人列入本局。
				h.Players = append(h.Players, participant{ID: occupant, Seat: seat}) // 将真人身份固定在本局名单。
			} // 空位不参加。
		}
		h.Players = append(h.Players, participant{ID: "bot", Seat: 2}) // 每局固定一个机器人。
		for i := range h.Players {                                     // 每名参赛者只收一次底注并发两张牌。
			p := &h.Players[i]       // 取得当前参赛者。
			if s.balance(p.ID) < 1 { // 检查是否能支付底注。
				return "insufficient_chips" // 余额不足时拒绝开局。
			} // 本票不处理归零补给。
		}
		for i := range h.Players { // 所有检查成功后统一扣款发牌。
			p := &h.Players[i]                                        // 取得参赛者。
			s.setBalance(p.ID, s.balance(p.ID)-1)                     // 支付一枚底注。
			p.Invested = 1                                            // 底注计入总投入，不计入本轮跟注。
			h.Pot++                                                   // 实际扣款进入唯一底池。
			p.Hole = append([]Card{}, h.deck[h.cursor:h.cursor+2]...) // 发两张唯一的私人手牌。
			h.cursor += 2                                             // 移动服务端发牌游标。
		}
		s.Hand = h // 提交新牌局。
		h.Turn++   // 首名真人得到新的行动机会。
		return ""  // 开局成功。
	}
	h := s.Hand                            // 读取本局。
	if h == nil || h.Stage == "finished" { // 检查是否存在未结束的牌局。
		return "no_active_hand" // 拒绝对已结束牌局行动。
	} // 已结束牌局不能继续行动。
	if cmd.HandID != h.ID || cmd.TurnID != h.Turn { // 核对本局和行动机会标识。
		return "stale_turn" // 拒绝旧局或旧行动请求。
	} // 旧局或旧机会不得用于当前动作。
	if h.Actor < 0 || h.Players[h.Actor].ID != id { // 核对当前行动身份。
		return "not_your_turn" // 拒绝越过行动顺序。
	} // 只允许当前行动身份操作。
	if cmd.Action != "check" { // 仅接受本票交付的过牌动作。
		return "invalid_action" // 拒绝不合法动作。
	} // 本票仅支持过牌。
	h.Players[h.Actor].acted = true // 记录当前玩家已过牌。
	return s.advance()              // 自动执行机器人并推进阶段。
}

func validDeck(deck []Card) bool {
	if len(deck) != 52 { // 完整牌序必须为52张。
		return false // 发现无效牌序立即拒绝。
	} // 牌序必须包含完整52张牌。
	seen := map[Card]bool{}  // 防止重复发牌。
	for _, c := range deck { // 检查每张牌的范围及唯一性。
		if c.Rank < 2 || c.Rank > 14 || c.Suit < 0 || c.Suit > 3 || seen[c] { // 检查点数、花色及重复牌。
			return false // 发现无效牌序立即拒绝。
		} // 拒绝非法牌序。
		seen[c] = true // 记录已检查的牌。
	}
	return true // 完整且唯一。
}

func (s *room) advance() string {
	h := s.Hand // 取得当前牌局。
	for {       // 连续处理机器人和无需真人输入的阶段。
		next := -1                    // 查找固定顺序下尚未行动的玩家。
		for i, p := range h.Players { // 从真人1到机器人扫描。
			if !p.Folded && !p.AllIn && !p.acted { // 寻找仍需行动的有效玩家。
				next = i // 保存下一名行动者下标。
				break    // 找到第一名后停止扫描。
			} // 跳过弃牌、全押和已完成动作的人。
		}
		if next >= 0 { // 本轮仍有行动机会。
			h.Actor = next                   // 设置当前行动者。
			h.Turn++                         // 创建新的行动标识。
			if h.Players[next].ID != "bot" { // 真人行动需要等待客户端请求。
				return "" // 本次推进成功，停止自动处理。
			} // 等待真人请求。
			h.Players[next].acted = true // 机器人无需跟注时立即过牌。
			continue                     // 机器人完成后继续推进。
		}
		switch h.Stage { // 所有人完成本轮后进入下一阶段。
		case "preflop": // 翻牌前轮结束后进入翻牌。
			h.Stage = "flop" // 切换为翻牌阶段。
			h.deal(3)        // 翻牌一次发三张。
		case "flop": // 翻牌轮结束后进入转牌。
			h.Stage = "turn" // 切换为转牌阶段。
			h.deal(1)        // 转牌发一张。
		case "turn": // 转牌轮结束后进入河牌。
			h.Stage = "river" // 切换为河牌阶段。
			h.deal(1)         // 河牌发一张。
		case "river": // 河牌轮结束后结算。
			return s.settle(true) // 河牌轮结束摊牌。
		}
		h.Target = 0               // 新轮没有下注目标。
		for i := range h.Players { // 依次更新各参赛者。
			h.Players[i].acted = false // 新轮重新允许玩家行动。
			h.Players[i].Street = 0    // 清空本轮投入，不改变总投入。
		} // 重置轮内动作和投入。
	}
}
func (h *hand) deal(n int) {
	h.Board = append(h.Board, h.deck[h.cursor:h.cursor+n]...) // 只从私有牌序追加公共牌。
	h.cursor += n                                             // 移动牌序游标。
}

func (s *room) settle(showdown bool) string {
	h := s.Hand                // 结算只针对当前局。
	if h.Stage == "finished" { // 检查是否已经结算。
		return "" // 本次推进成功，停止自动处理。
	} // 重复推进不得重复派奖。
	winners := []int{}            // 保存并列最强玩家的固定座位顺序。
	var best Strength             // 记录当前最强牌力。
	for i, p := range h.Players { // 仅未弃牌者有资格争夺整个池。
		if p.Folded { // 弃牌玩家没有领奖资格。
			continue // 跳过无资格玩家。
		} // 弃牌者不参与分配。
		strength := Strength{} // 提前胜出不需要计算或公开牌型。
		if showdown {          // 只有摊牌才计算并公开牌力。
			strength, _ = Evaluate(append(append([]Card{}, p.Hole...), h.Board...)) // 从本人手牌和公共牌选择最佳五张。
			h.Players[i].Strength = &strength                                       // 保存摊牌牌型及最佳五张。
		} // 摊牌选择最佳五张。
		comparison := strength.Compare(best)     // 比较完整点数后比较花色。
		if len(winners) == 0 || comparison > 0 { // 首名有效玩家或更强牌成为当前赢家。
			best = strength    // 保存新的最强牌力。
			winners = []int{i} // 更强牌替换所有旧赢家。
		} else if comparison == 0 { // 完全平局时保留多个赢家。
			winners = append(winners, i) // 按固定座位顺序增加平局赢家。
		} // 只保留最强玩家。
	}
	if len(winners) == 0 { // 结算必须存在有效赢家。
		return "no_winner" // 无赢家时报告状态故障。
	} // 无有效玩家时不得凭空分配。
	share, remainder := h.Pot/int64(len(winners)), h.Pot%int64(len(winners)) // 整数平分并保留零头。
	for j, i := range winners {                                              // 先检查所有入账是否会溢出。
		gain := share             // 每名赢家先取得平分份额。
		if int64(j) < remainder { // 只向排在零头范围内的赢家多分一枚。
			gain++ // 增加一枚零头筹码。
		} // 按固定顺序向赢家各分一枚零头。
		if s.balance(h.Players[i].ID) > math.MaxInt64-gain { // 检查派奖后余额是否会溢出。
			return "chips_overflow" // 拒绝整数溢出的结算。
		} // 不允许余额溢出。
	}
	for j, i := range winners { // 检查成功后一起派奖。
		gain := share             // 取得平分份额。
		if int64(j) < remainder { // 只向排在零头范围内的赢家多分一枚。
			gain++ // 增加一枚零头筹码。
		} // 分配剩余零头。
		p := &h.Players[i]                       // 取得赢家身份。
		p.Won = gain                             // 保存本局奖项供结果页面展示。
		s.setBalance(p.ID, s.balance(p.ID)+gain) // 将底池份额转入余额。
	}
	for i := range h.Players { // 依次更新各参赛者。
		h.Players[i].Balance = s.balance(h.Players[i].ID) // 保存该玩家结算时的余额。
	} // 保存结算时余额快照。
	h.Pot = 0             // 清空已分配底池。
	h.Stage = "finished"  // 标记本局只结算一次。
	h.Actor = -1          // 已结束不再接受行动。
	h.showdown = showdown // 仅摊牌才公开有效玩家暗牌。
	return ""             // 结果保留到下一次开局。
}
