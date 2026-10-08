// Browser-only offline fixture: fixed cards/time randomness via the public constructor.
// Run locally with: go run ./scripts/browser-fixture (never part of the production executable).
package main

import (
	"fmt"
	"net/http"
	"os"

	"texas-poker/internal/poker"
)

func main() {
	app, err := poker.NewWithOptions(poker.Options{ // 复用公开离线seam，不提供在线选牌参数。
		Deck: tieDeck, // 六名参赛者共享红桃同花大顺，独立预期每人得一枚底注。
		ActionStart: func(seats []int) (int, error) { // 始终从机器人起点验证其实际等待。
			return seats[len(seats)-1], nil // 候选最后一席是固定机器人。
		},
		BotThinkSeconds: func() (int, error) { // 验收等待仍遵守一到八秒范围。
			return 3, nil // 固定三秒供页面正常30秒递减的组合验收。
		},
	})
	if err != nil { // 构造失败必须退出，不能冒充验收服务。
		fmt.Fprintln(os.Stderr, err) // 返回可定位的失败信息。
		os.Exit(1)                   // 明确以失败状态结束。
	}
	defer app.Close()                                                  // 生命周期由独立验收进程管理。
	if err = http.ListenAndServe("127.0.0.1:18082", app); err != nil { // 仅绑定回环地址，无云部署入口。
		fmt.Fprintln(os.Stderr, err) // 端口冲突也必须报告失败。
		os.Exit(1)                   // 不静默略过浏览器验收。
	}
}

func tieDeck() ([]poker.Card, error) {
	deck := []poker.Card{{Rank: 2, Suit: 0}, {Rank: 3, Suit: 0}, {Rank: 2, Suit: 1}, {Rank: 3, Suit: 1}, {Rank: 2, Suit: 2}, {Rank: 3, Suit: 2}, {Rank: 2, Suit: 3}, {Rank: 3, Suit: 3}, {Rank: 4, Suit: 0}, {Rank: 4, Suit: 1}, {Rank: 4, Suit: 2}, {Rank: 4, Suit: 3}, {Rank: 10, Suit: 2}, {Rank: 11, Suit: 2}, {Rank: 12, Suit: 2}, {Rank: 13, Suit: 2}, {Rank: 14, Suit: 2}} // 前十二张是六份暗牌，随后五张是共同最佳牌。
	used := map[poker.Card]bool{}                                                                                                                                                                                                                                                                                                                                                 // 完整牌序不能有重复牌。
	for _, card := range deck {                                                                                                                                                                                                                                                                                                                                                   // 记录固定前缀中的全部牌。
		used[card] = true // 补牌只选择尚未出现者。
	}
	for rank := 2; rank <= 14; rank++ { // 补齐一副标准五十二张牌。
		for suit := 0; suit < 4; suit++ { // 点数与花色构成唯一标识。
			card := poker.Card{Rank: rank, Suit: suit} // 构造当前候选牌。
			if !used[card] {                           // 不重复加入前缀中的牌。
				deck = append(deck, card) // 保持合法且完整的服务端牌序。
			}
		}
	}
	return deck, nil // 通过原构造边界校验发牌，不绕过游戏规则。
}
