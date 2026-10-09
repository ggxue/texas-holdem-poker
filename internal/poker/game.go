package poker

import (
	"crypto/rand"
	"math"
	"math/big"
	"strconv"
	"time"
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
	ID              int64
	Turn            int64
	Stage           string
	Pot             int64
	Board           []Card
	Players         []participant
	Actor           int
	Start           int
	Target          int64
	deck            []Card
	cursor          int
	showdown        bool
	Deadline        time.Time
	ThinkingStarted time.Time
	deadlineTurn    int64
}
type handView struct {
	ID           int64         `json:"id"`
	Turn         int64         `json:"turn"`
	Stage        string        `json:"stage"`
	Pot          int64         `json:"pot"`
	Board        []Card        `json:"board"`
	Players      []participant `json:"players"`
	Actor        string        `json:"actor"`
	Target       int64         `json:"target"`
	Legal        []string      `json:"legal"`
	CallAmount   string        `json:"callAmount"`
	CallRequired string        `json:"callRequired"`
	AllInAmount  string        `json:"allInAmount"`
	BetAmount    int64         `json:"betAmount"`
	Deadline     int64         `json:"deadline"`
	StartSeat    int           `json:"actionStartSeat"`
	BotThinking  bool          `json:"botThinking"`
}

func randomBotThinkSeconds() (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(8)) // 八个整数秒各有相同概率。
	if err != nil {                                // 随机源故障不能冒充有效机会。
		return 0, err // 由调用方原子拒绝或报告自动事件故障。
	}
	return int(n.Int64()) + 1, nil // 返回一到八秒，不包含零秒。
}

func randomActionStart(seats []int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(seats)))) // 从全部参赛候选中无偏等概率抽一个下标。
	if err != nil {                                                // 随机源故障必须返回给开局事务。
		return 0, err // 不以固定座位替代失败抽样。
	} // 随机源失败时不开局。
	return seats[int(n.Int64())], nil // 返回座位编号，空位不在候选内。
}

func validAction(action string) bool {
	return action == "join" || action == "start" || action == "check" || action == "bet" || action == "call" || action == "fold" || action == "allin" || action == "leave" // 全押金额仅由服务端余额决定。
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
	v := &handView{ID: h.ID, Turn: h.Turn, Stage: h.Stage, Pot: h.Pot, Board: append([]Card{}, h.Board...), Target: h.Target, Players: make([]participant, len(h.Players)), CallAmount: "0", CallRequired: "0", AllInAmount: "0"} // 构造独立快照，非行动者没有扣款提示。
	v.StartSeat = h.Players[h.Start].Seat                                                                                                                                                                                         // 全局保留同一公开行动起点，包含已全押或离房者。
	if !h.Deadline.IsZero() {                                                                                                                                                                                                     // 有有效行动机会时才公开期限。
		v.Deadline = h.Deadline.UnixMilli() // 使用绝对毫秒时间，重连不会重新计时。
		if !h.ThinkingStarted.IsZero() {    // 机器人实际期限与页面三十秒显示分别保存。
			v.BotThinking = true                                             // 显示思考状态，不暴露未来实际动作时刻。
			v.Deadline = h.ThinkingStarted.Add(30 * time.Second).UnixMilli() // 刷新沿用原始显示依据。
		}
	}
	for i, p := range h.Players { // 按参赛身份裁剪暗牌，座位重用不会转移手牌。
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
		if len(s.Disconnected) > 0 || len(s.Connecting) > 0 { // 有断线或尚未建立控制连接者时不能开新局。
			return "connection_grace" // 避免对离线玩家补给或扣底注。
		}
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
		h.Players = append(h.Players, participant{ID: "bot", Seat: humanSeatCount}) // 机器人固定在五个真人座位之后。
		seats := make([]int, len(h.Players))                                        // 固定名单中的所有参赛者均参与起点抽选。
		for i, p := range h.Players {                                               // 收集所有本局参赛者，而非只收集可行动者。
			seats[i] = p.Seat // 包含底注后全押者和机器人，排除空位及待局者。
		} // 包含底注后全押者和机器人，排除空位及待局者。
		startSeat, err := a.actionStart(seats) // 成功开局仅抽一次，不在轮转换或重连时重抽。
		if err != nil {                        // 随机源故障不能收底注或替换已有结果。
			return "unavailable" // 拒绝本次开局，由公开命令事务回滚。
		} // 随机源错误时原子拒绝开局，不收底注。
		h.Start = -1                  // 配置也必须返回本局候选中的有效座位。
		for i, p := range h.Players { // 将候选座位定位到固定本局名单。
			if p.Seat == startSeat { // 起点必须属于本局参赛者。
				h.Start = i // 保存每轮和零头共用的唯一循环起点。
			}
		} // 保存固定名单中的起点下标。
		if h.Start < 0 { // 离线配置返回非候选座位也不能开局。
			return "unavailable" // 保持原余额、名单和底池不变。
		} // 不接受非参赛座位作为起点。
		for i := range h.Players { // 每名参赛者只收一次底注并发两张牌。
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
		s.Hand = h                                        // 提交新牌局。
		s.announce(announcement{Kind: "start", Seat: -1}) // 开局只确认一次。
		s.announce(announcement{Kind: "deal", Seat: -1})  // 只说发手牌，不公开任何牌值。
		return s.advance()                                // 自动选择首名可行动者，必要时补齐牌。
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
	if rejection := s.act(cmd.Action, "manual"); rejection != "" { // 通过统一动作规则校验并执行。
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
				for len(h.Board) < 5 { // 全押跨过的每个阶段仍按顺序确认。
					s.revealNextStreet() // 不延迟牌局、不等待声音。
				}
				return s.settle(true) // 完成摊牌和单池派奖。
			} // 无人欠注时补齐五张公共牌。
		}
		if h.Actor >= 0 && !h.Players[h.Actor].Folded && !h.Players[h.Actor].AllIn && !h.Players[h.Actor].acted { // 他人退出不重置当前合法机会。
			return "" // 保留原行动者及行动标识。
		}
		next := -1       // 查找固定顺序下尚未行动的玩家。
		after := h.Actor // 已有轮内行动时从上一行动者之后继续。
		if after < 0 {   // 新轮尚无上一行动者，必须沿用本局起点。
			after = h.Start - 1 // 扫描偏移从起点之前开始，使首候选为起点。
		} // 每轮首次扫描从同一本局起点开始。
		for offset := 1; offset <= len(h.Players); offset++ { // 固定顺序继续，轮末再回应此前过牌者。
			i := (after + offset) % len(h.Players) // 按固定座位循环跳过不具行动资格者。
			p := h.Players[i]                      // 读取该位置的本轮状态。
			if !p.Folded && !p.AllIn && !p.acted { // 寻找仍需行动的有效玩家。
				next = i // 保存下一名行动者下标。
				break    // 找到第一名后停止扫描。
			} // 跳过弃牌、全押和已完成动作的人。
		}
		if next >= 0 { // 本轮仍有行动机会。
			h.Actor = next // 设置当前行动者。
			h.Turn++       // 创建新的行动标识。
			return ""      // 真人等待命令，机器人等待独立的实际思考期限。
		}
		switch h.Stage { // 所有人完成本轮后进入下一阶段。
		case "preflop", "flop", "turn": // 正常推进与全押补牌采用同一阶段事实。
			s.revealNextStreet() // 分别发三、一、一张，不念牌值。
		case "river": // 河牌轮结束后结算。
			return s.settle(true) // 河牌轮结束摊牌。
		}
		h.Target = 0               // 新轮没有下注目标。
		h.Actor = -1               // 新轮沿本局已确定起点扫描，空位跳过。
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

func (s *room) revealNextStreet() {
	h := s.Hand           // 只根据已有公开牌数量推进。
	n := 1                // 转牌与河牌各一张。
	switch len(h.Board) { // 无人下注时也完整记录各阶段。
	case 0: // 第一批公共牌。
		h.Stage, n = "flop", 3 // 翻牌三张。
	case 3: // 已有翻牌。
		h.Stage = "turn" // 转牌阶段。
	case 4: // 已有转牌。
		h.Stage = "river" // 河牌阶段。
	}
	h.deal(n)                                         // 发牌仍只在服务端。
	s.announce(announcement{Kind: h.Stage, Seat: -1}) // 音频不会携带卡牌信息。
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
	ordered := make([]int, 0, len(winners))                                  // 零头只给赢家，顺序沿本局起点循环。
	for offset := 0; offset < len(h.Players); offset++ {                     // 起点即使已离房仍保留，不重新抽选。
		i := (h.Start + offset) % len(h.Players) // 与每轮行动使用同一个座位循环。
		for _, winner := range winners {         // 过滤非赢家和空位，不按下注额度分层。
			if winner == i { // 该循环位置确为并列最强赢家才可分零头。
				ordered = append(ordered, i) // 每人一次，零头不会发给非赢家。
			} // 每名赢家仅出现一次，最多多得一枚。
		}
	}
	winners = ordered           // 后续溢出检查和入账共用同一奖项顺序。
	for j, i := range winners { // 先检查所有入账是否会溢出。
		gain := share             // 每名赢家先取得平分份额。
		if int64(j) < remainder { // 只向排在零头范围内的赢家多分一枚。
			gain++ // 增加一枚零头筹码。
		} // 按固定顺序向赢家各分一枚零头。
		if s.balance(h.Players[i].ID) > math.MaxInt64-gain { // 检查派奖后余额是否会溢出。
			return "chips_overflow" // 拒绝整数溢出的结算。
		} // 不允许余额溢出。
	}
	if showdown { // 全部金额检查通过才记录摊牌事实。
		s.announce(announcement{Kind: "showdown", Seat: -1}) // 不读手牌、牌型或最佳五张。
	}
	s.announce(announcement{Kind: "settlement", Seat: -1}) // 先结算，再按固定顺序读赢家金额。
	for j, i := range winners {                            // 检查成功后一起派奖。
		gain := share             // 取得平分份额。
		if int64(j) < remainder { // 只向排在零头范围内的赢家多分一枚。
			gain++ // 增加一枚零头筹码。
		} // 分配剩余零头。
		p := &h.Players[i]                                                                          // 取得赢家身份。
		p.Won = gain                                                                                // 保存本局奖项供结果页面展示。
		s.announce(announcement{Kind: "winner", Seat: p.Seat, Amount: strconv.FormatInt(gain, 10)}) // 精确奖项用字符串跨浏览器边界。
		s.setBalance(p.ID, s.balance(p.ID)+gain)                                                    // 将底池份额转入余额。
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

func (h *hand) legal(balance int64) []string {
	p := h.Players[h.Actor]  // 读取当前玩家的轮内投入。
	if p.Folded || p.AllIn { // 已弃牌或全押者不能行动。
		return nil // 不返回任何合法按钮。
	} // 已弃牌或全押不能行动。
	actions := []string{"check", "fold"} // 不欠注时允许过牌或弃牌。
	if p.Street < h.Target {             // 欠注时不能用过牌跳过付款。
		actions = []string{"call", "fold"} // 跟注金额由实际余额限制。
	} // 全押是否可用另按仍能回应的对手判断。
	capable := 0                      // 统计仍能下注的人数。
	for _, other := range h.Players { // 统计仍能下注的对手。
		if !other.Folded && !other.AllIn { // 跳过弃牌和全押的参赛者。
			capable++ // 增加能下注的人数。
		}
	} // 跳过弃牌和全押。
	if h.Target == 0 && capable >= 2 { // 本轮尚无下注且至少两人有行动能力。
		actions = append(actions, "bet") // 增加一次固定下注的合法按钮。
	} // 每轮仅一次下注，禁止独自追加。
	if balance > 0 && (capable >= 2 || (p.Street < h.Target && balance <= h.Target-p.Street)) { // 唯一欠注者仅能用不超过欠额的余款全押。
		actions = append(actions, "allin") // 不增加任意金额或普通加注命令。
	} // 已全押者不会重新获得动作。
	return actions // 服务器统一决定合法按钮。
}
func (s *room) act(action, reason string) string {
	h := s.Hand                                                         // 动作只作用于当前牌局。
	valid := false                                                      // 默认动作无效。
	for _, allowed := range h.legal(s.balance(h.Players[h.Actor].ID)) { // 与公开按钮共用服务端合法性判断。
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
	if action == "allin" { // 主动全押不接受客户端金额。
		amount = s.balance(p.ID) // 本次投入全部剩余筹码。
	} // 之前本轮投入仍另外累计到目标。
	if amount > s.balance(p.ID) { // 余额不足时不能透支。
		amount = s.balance(p.ID) // 将实际扣款限制为本人全部余额。
	} // 不足时扣实际余额全押。
	if amount > math.MaxInt64-h.Pot || amount > math.MaxInt64-p.Invested { // 投入及底池累加必须能用整数表示。
		return "chips_overflow" // 拒绝会溢出的扣款。
	} // 扣款前检查整数累加。
	if action == "fold" { // 弃牌会失去本局领奖资格。
		p.Folded = true // 标记玩家已经弃牌。
	} // 弃牌立即失去领奖资格。
	if p.Street+amount > h.Target { // 更高的本轮累计投入要求此前玩家重新回应。
		h.Target = p.Street + amount // 全押提高目标；短额跟注/全押不降低目标。
		for i := range h.Players {   // 按固定名单更新本轮应答状态。
			h.Players[i].acted = false // 让玩家回应新下注。
		} // 其余未弃牌且非全押玩家重新应答。
	}
	s.setBalance(p.ID, s.balance(p.ID)-amount)                                                                                                    // 从本人余额扣除实际金额。
	p.Street += amount                                                                                                                            // 累加本轮投入。
	p.Invested += amount                                                                                                                          // 累加本局投入。
	h.Pot += amount                                                                                                                               // 全部有效投入进单池，不退款。
	p.AllIn = p.AllIn || s.balance(p.ID) == 0                                                                                                     // 全押资格不因重连补给而重置。
	p.acted = true                                                                                                                                // 记录当前机会已经完成。
	s.announce(announcement{Kind: "action", Seat: p.Seat, Action: action, Amount: strconv.FormatInt(amount, 10), AllIn: p.AllIn, Reason: reason}) // 记录实际扣款与确认原因，不从快照猜测动作。
	return ""                                                                                                                                     // 同一内存变更统一生效。
}

func (s *room) release(id string) string {
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
	delete(s.Disconnected, id)                          // 清除已经确认离房的宽限期限。
	delete(s.Connecting, id)                            // 清除未建立控制连接的期限。
	s.announce(announcement{Kind: "leave", Seat: seat}) // 已确认离房，不读钱包余额。
	s.Seats[seat] = ""                                  // 立即释放座位，剩余余额仍绑定原身份。
	if s.Host == id {                                   // 房主离开后自动交接。
		s.Host = ""                        // 默认无真人时没有房主。
		for _, occupant := range s.Seats { // 检查全部留房真人，保持其原座位。
			if occupant != "" && (s.Host == "" || s.Accounts[occupant].joinedVersion < s.Accounts[s.Host].joinedVersion) { // 比较本次占座先后，不使用座位编号。
				s.Host = occupant // 房主交给最早入房且仍在房的真人。
			}
		}
		for newSeat, occupant := range s.Seats { // 只宣布实际接任的在房玩家。
			if occupant != "" && occupant == s.Host { // 空房间没有伪房主。
				s.announce(announcement{Kind: "host", Seat: newSeat}) // 不改变席位或牌局。
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
	return "" // 资格与座位已经更新，统一推进时所有投入不退。
}

func (s *room) depart(id string) string {
	if rejection := s.release(id); rejection != "" { // 先释放席位与未结算资格。
		return rejection // 无座位时报告无效退出。
	}
	if s.Hand != nil && s.Hand.Stage != "finished" { // 未结束牌局才继续推进。
		return s.advance() // 跳过离房玩家，必要时结算。
	}
	return "" // 已结算奖项保持不变。
}
