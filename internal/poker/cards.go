// 本文件校验牌面并评估德州扑克牌力；涉及流程：阶段推进与结算。
package poker

import (
	"fmt"
	"sort"
)

// Card 的 Rank 固定为2至14，Suit固定为0至3；一副牌中同一组合只能出现一次。
type Card struct {
	Rank int `json:"rank"`
	Suit int `json:"suit"`
}
type Strength struct {
	Category string  `json:"category"`
	Cards    [5]Card `json:"cards"`
	// key 保存类别和逐级点数比较值，花色只在点数完全相同时决胜。
	key [6]int
}

// 【阶段推进与结算#6/11】校验 cards.go:Evaluate
// 上一步：#5 game.go:settle；下一步：#7 cards.go:evaluateFive
// 职责：选择最佳五张牌。
// 前置条件：输入为待评估的已知牌集合；不改游戏状态；牌数、范围或重复校验失败返回错误。
func Evaluate(cards []Card) (Strength, error) {
	if len(cards) < 5 || len(cards) > 7 {
		return Strength{}, fmt.Errorf("需要五至七张牌")
	}
	seen := map[Card]bool{}
	for _, c := range cards {
		if c.Rank < 2 || c.Rank > 14 || c.Suit < 0 || c.Suit > 3 || seen[c] {
			return Strength{}, fmt.Errorf("非法或重复牌")
		}
		seen[c] = true
	}
	var best Strength
	first := true
	for a := 0; a < len(cards)-4; a++ {
		for b := a + 1; b < len(cards)-3; b++ {
			for c := b + 1; c < len(cards)-2; c++ {
				for d := c + 1; d < len(cards)-1; d++ {
					for e := d + 1; e < len(cards); e++ {
						s := evaluateFive([5]Card{cards[a], cards[b], cards[c], cards[d], cards[e]})
						if first || s.Compare(best) > 0 {
							best, first = s, false
						}
					}
				}
			}
		}
	}
	return best, nil
}

// 【阶段推进与结算#7/11】结算 cards.go:evaluateFive
// 上一步：#6 cards.go:Evaluate；下一步：#8 cards.go:Strength.Compare
// 职责：计算五张组合的比较键。
// 前置条件：输入是已验证的五张组合；只计算牌力；不改变牌局。
func evaluateFive(cards [5]Card) Strength { // haifeng: 计算各玩家最强牌力
	sort.Slice(cards[:], func(i, j int) bool {
		if cards[i].Rank != cards[j].Rank {
			return cards[i].Rank > cards[j].Rank
		}
		return cards[i].Suit > cards[j].Suit
	})
	s := Strength{Cards: cards}
	counts := map[int]int{}
	flush := true
	for _, c := range cards {
		counts[c.Rank]++
		flush = flush && c.Suit == cards[0].Suit
	}
	straight := 0
	if len(counts) == 5 {
		if cards[0].Rank-cards[4].Rank == 4 {
			straight = cards[0].Rank
		}
		if cards[0].Rank == 14 && cards[1].Rank == 5 && cards[4].Rank == 2 {
			straight = 5
			s.Cards = [5]Card{cards[1], cards[2], cards[3], cards[4], cards[0]}
		}
	}
	type group struct{ rank, count int }
	groups := make([]group, 0, len(counts))
	for rank, count := range counts {
		groups = append(groups, group{rank, count})
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].count != groups[j].count {
			return groups[i].count > groups[j].count
		}
		return groups[i].rank > groups[j].rank
	})
	switch {
	case flush && straight == 14:
		s.key[0], s.key[1] = 9, 14
	case flush && straight > 0:
		s.key[0], s.key[1] = 8, straight
	case groups[0].count == 4:
		s.key[0] = 7
	case groups[0].count == 3 && groups[1].count == 2:
		s.key[0] = 6
	case flush:
		s.key[0] = 5
	case straight > 0:
		s.key[0], s.key[1] = 4, straight
	case groups[0].count == 3:
		s.key[0] = 3
	case groups[0].count == 2 && groups[1].count == 2:
		s.key[0] = 2
	case groups[0].count == 2:
		s.key[0] = 1
	}
	if straight == 0 || (!flush && s.key[0] != 4) {
		for i, g := range groups {
			s.key[i+1] = g.rank
		}
	}
	s.Category = [...]string{"高牌", "一对", "两对", "三条", "顺子", "同花", "葫芦", "四条", "同花顺", "同花大顺"}[s.key[0]]
	return s
}

// 【阶段推进与结算#8/11】结算 cards.go:Strength.Compare
// 上一步：#7 cards.go:evaluateFive；下一步：#9 hand_record.go:record
func (s Strength) Compare(other Strength) int { // haifeng: 找出赢家
	for i, n := range s.key {
		if n > other.key[i] {
			return 1
		}
		if n < other.key[i] {
			return -1
		}
	}
	for i, c := range s.Cards {
		if c.Suit > other.Cards[i].Suit {
			return 1
		}
		if c.Suit < other.Cards[i].Suit {
			return -1
		}
	}
	return 0
}
