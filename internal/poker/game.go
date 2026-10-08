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
	ID         int64         `json:"id"`
	Turn       int64         `json:"turn"`
	Stage      string        `json:"stage"`
	Pot        int64         `json:"pot"`
	Board      []Card        `json:"board"`
	Players    []participant `json:"players"`
	Actor      string        `json:"actor"`
	Target     int64         `json:"target"`
	Legal      []string      `json:"legal"`
	CallAmount int64         `json:"callAmount"`
	BetAmount  int64         `json:"betAmount"`
}

func validAction(action string) bool {
	return action == "join" || action == "start" || action == "check" || action == "bet" || action == "call" || action == "fold" || action == "leave" // 限定本票公开命令范围。
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
			v.Legal = h.legal() // 返回服务器计算的合法动作。
		} // 只向本人返回当前合法的四动作子集。
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
	if cmd.Action == "leave" { // 主动退出不需要等到本人回合。
		return s.depart(id) // 立即释放座位并失去未结算资格。
	}
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
		h := &hand{ID: s.Version + 1, Stage: "preflop", Actor: -1, deck: append([]Card(nil), deck...)} // 建立新的牌局。
		for seat, occupant := range s.Seats {                                                          // 固定本局真人名单。
			if occupant != "" { // 只把已占座的真人列入本局。
				h.Players = append(h.Players, participant{ID: occupant, Seat: seat}) // 将真人身份固定在本局名单。
			} // 空位不参加。
		}
		h.Players = append(h.Players, participant{ID: "bot", Seat: 2}) // 每局固定一个机器人。
		for i := range h.Players {                                     // 每名参赛者只收一次底注并发两张牌。
			p := &h.Players[i]        // 取得当前参赛者。
			if s.balance(p.ID) == 0 { // 只有本次参赛者余额归零才免费补给。
				s.setBalance(p.ID, 100) // 下一局底注前补到100，不给正余额增加筹码。
			}
		}
		for i := range h.Players { // 所有检查成功后统一扣款发牌。
			p := &h.Players[i]                                        // 取得参赛者。
			s.setBalance(p.ID, s.balance(p.ID)-1)                     // 支付一枚底注。
			p.Invested = 1                                            // 底注计入总投入，不计入本轮跟注。
			p.AllIn = s.balance(p.ID) == 0                            // 底注扣完为零即全押。
			h.Pot++                                                   // 实际扣款进入唯一底池。
			p.Hole = append([]Card{}, h.deck[h.cursor:h.cursor+2]...) // 发两张唯一的私人手牌。
			h.cursor += 2                                             // 移动服务端发牌游标。
		}
		s.Hand = h         // 提交新牌局。
		return s.advance() // 自动选择首名可行动者，必要时补齐牌。
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
	if rejection := s.act(cmd.Action); rejection != "" { // 通过统一动作规则校验并执行。
		return rejection // 动作失败时向调用者报告错误。
	} // 校验动作后才改变状态。
	return s.advance() // 自动执行机器人并推进阶段。
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
		alive, capable := 0, 0        // 统计领奖资格及下注能力。
		for _, p := range h.Players { // 统计所有固定参赛者的状态。
			if !p.Folded { // 未弃牌者仍有领奖资格。
				alive++       // 增加仍能领奖的人数。
				if !p.AllIn { // 未全押者仍可能下注。
					capable++ // 增加能下注的人数。
				}
			}
		} // 全押仍保留领奖资格。
		if alive == 1 { // 只剩一名未弃牌者时提前结束。
			return s.settle(false) // 直接分配全池，不强制亮牌。
		} // 唯一未弃牌者立即赢全池，不强制亮牌。
		if capable < 2 { // 无足够对手时禁止单独加钱。
			owing := false                // 判断是否还需回应当前下注。
			for i, p := range h.Players { // 检查每名玩家的本轮状态。
				if !p.Folded && !p.AllIn && p.Street < h.Target { // 找到唯一仍欠注的可行动者。
					h.Players[i].acted = false // 让玩家回应新下注。
					owing = true               // 记录尚需跟注或弃牌。
				}
			} // 唯一欠注者先得到跟注或弃牌机会。
			if !owing { // 无人欠注时不再提供独自下注。
				h.deal(5 - len(h.Board)) // 补足五张公共牌，不重复发河牌。
				return s.settle(true)    // 完成摊牌和单池派奖。
			} // 无人欠注时补齐五张公共牌。
		}
		if h.Actor >= 0 && !h.Players[h.Actor].Folded && !h.Players[h.Actor].AllIn && !h.Players[h.Actor].acted { // 他人退出不重置当前合法机会。
			return "" // 保留原行动者及行动标识。
		}
		next := -1                                            // 查找固定顺序下尚未行动的玩家。
		for offset := 1; offset <= len(h.Players); offset++ { // 固定顺序继续，轮末再回应此前过牌者。
			i := (h.Actor + offset) % len(h.Players) // 从上一行动者之后开始查找。
			p := h.Players[i]                        // 读取该位置的本轮状态。
			if !p.Folded && !p.AllIn && !p.acted {   // 寻找仍需行动的有效玩家。
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
			action := "check"                      // 机器人默认过牌。
			if h.Players[next].Street < h.Target { // 机器人欠当前目标时需要跟注。
				action = "call" // 机器人选择跟注，金额由服务器计算。
			} // 欠注时只跟注，不分析牌力或随机选择。
			if rejection := s.act(action); rejection != "" { // 用相同规则立即执行机器人动作。
				return rejection // 动作失败时向调用者报告错误。
			} // 立即执行并按余额扣款，不等待动画。
			continue // 机器人完成后继续推进。
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
		h.Actor = -1               // 新轮从真人1重新扫描，空位跳过。
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

func (h *hand) legal() []string {
	p := h.Players[h.Actor]  // 读取当前玩家的轮内投入。
	if p.Folded || p.AllIn { // 已弃牌或全押者不能行动。
		return nil // 不返回任何合法按钮。
	} // 已弃牌或全押不能行动。
	if p.Street < h.Target { // 判断当前玩家是否欠注。
		return []string{"call", "fold"} // 欠注时只允许跟注或弃牌。
	} // 欠注时只能跟注或弃牌。
	actions := []string{"check", "fold"} // 不欠注时允许过牌或弃牌。
	capable := 0                         // 统计仍能下注的人数。
	for _, other := range h.Players {    // 统计仍能下注的对手。
		if !other.Folded && !other.AllIn { // 跳过弃牌和全押的参赛者。
			capable++ // 增加能下注的人数。
		}
	} // 跳过弃牌和全押。
	if h.Target == 0 && capable >= 2 { // 本轮尚无下注且至少两人有行动能力。
		actions = append(actions, "bet") // 增加一次固定下注的合法按钮。
	} // 每轮仅一次下注，禁止独自追加。
	return actions // 服务器统一决定合法按钮。
}
func (s *room) act(action string) string {
	h := s.Hand                         // 动作只作用于当前牌局。
	valid := false                      // 默认动作无效。
	for _, allowed := range h.legal() { // 遍历服务器计算的合法动作。
		if allowed == action { // 请求动作必须在合法列表内。
			valid = true // 确认该动作合法。
		}
	} // 校验合法动作。
	if !valid { // 拒绝非法动作后保持原状态。
		return "invalid_action" // 向页面返回非法动作提示。
	} // 非法动作不改变状态。
	p := &h.Players[h.Actor] // 取得当前行动者。
	amount := int64(0)       // 过牌和弃牌不扣款。
	if action == "bet" {     // 固定下注需要扣十枚或全部余额。
		amount = 10 // 设置标准下注金额。
	} // 首次主动下注固定十枚。
	if action == "call" { // 跟注只补本轮欠款。
		amount = h.Target - p.Street // 计算当前欠注，不抵底注。
	} // 跟注只补本轮差额，底注不抵扣。
	if amount > s.balance(p.ID) { // 余额不足时不能透支。
		amount = s.balance(p.ID) // 将实际扣款限制为本人全部余额。
	} // 不足时扣实际余额全押。
	if amount > math.MaxInt64-h.Pot || amount > math.MaxInt64-p.Invested { // 投入及底池累加必须能用整数表示。
		return "chips_overflow" // 拒绝会溢出的扣款。
	} // 扣款前检查整数累加。
	if action == "fold" { // 弃牌会失去本局领奖资格。
		p.Folded = true // 标记玩家已经弃牌。
	} // 弃牌立即失去领奖资格。
	if action == "bet" { // 新下注要求此前过牌者重新回应。
		h.Target = amount          // 短额下注的实际金额成为目标。
		for i := range h.Players { // 按固定名单更新本轮应答状态。
			h.Players[i].acted = false // 让玩家回应新下注。
		} // 其余未弃牌且非全押玩家重新应答。
	}
	s.setBalance(p.ID, s.balance(p.ID)-amount) // 从本人余额扣除实际金额。
	p.Street += amount                         // 累加本轮投入。
	p.Invested += amount                       // 累加本局投入。
	h.Pot += amount                            // 全部有效投入进单池，不退款。
	p.AllIn = p.AllIn || s.balance(p.ID) == 0  // 全押资格不因重连补给而重置。
	p.acted = true                             // 记录当前机会已经完成。
	return ""                                  // 同一内存变更统一生效。
}

func (s *room) depart(id string) string {
	seat := -1                         // 查找该身份当前占用的真人座位。
	for i, occupant := range s.Seats { // 只释放本人座位。
		if occupant == id { // 确认身份与座位一致。
			seat = i // 保存需要释放的位置。
			break    // 一名真人最多占一个位置。
		}
	}
	if seat < 0 { // 未入房身份没有可退出的座位。
		return "not_in_room" // 退出请求不影响任何其他玩家。
	}
	s.Seats[seat] = "" // 立即释放座位，剩余余额仍绑定原身份。
	if s.Host == id {  // 房主离开后自动交接。
		s.Host = ""                        // 默认无真人时没有房主。
		for _, occupant := range s.Seats { // 最多只剩一名真人，保持其原座位。
			if occupant != "" { // 找到仍在房间的人。
				s.Host = occupant // 把房主交给留房真人。
				break             // 完成交接后停止查找。
			}
		}
	}
	h := s.Hand                            // 检查尚未结算的本局资格。
	if h == nil || h.Stage == "finished" { // 已结算奖项不能因退出撤销。
		return "" // 无须继续改变本局。
	}
	for i := range h.Players { // 资格绑定开局身份，不绑定可重用座位。
		if h.Players[i].ID == id { // 仅影响该身份的参赛资格。
			h.Players[i].Folded = true // 即使全押也失去未结算领奖资格。
			h.Players[i].acted = true  // 不再等待该玩家应答。
		}
	}
	return s.advance() // 必要时跳过空位或提前结算，所有投入不退。
}
