package poker

import "testing"

func TestAC24OnlyZeroBalanceRefillsBeforeNextHand(t *testing.T) {
	for _, balances := range [][3]int64{{0, 37, 180}, {180, 37, 0}, {1, 37, 180}} {
		t.Run(string(rune('a'+balances[0]%26)), func(t *testing.T) {
			app, s, c, _ := bettingRoom(t, [3]int64{100, 100, 100})
			finished := finishChecks(t, c, s.URL)
			oldID := finished.Hand.ID
			// 已确认的结算余额fixture；仅通过下一局公开命令断言。
			app.mu.Lock()
			for i, id := range app.state.Seats {
				if id != "" {
					app.state.setBalance(id, balances[i])
				}
			}
			app.state.Bot.Chips = balances[2]
			app.mu.Unlock()
			before := gameState(t, c[0], s.URL)
			status, v := gameCommand(t, c[0], s.URL, "next", "start", before)
			if status != 200 || v.Hand.ID == oldID {
				t.Fatalf("next start: %d %+v", status, v)
			}
			got := [3]int64{v.Seats[0].Chips, v.Seats[1].Chips, v.Bot.Chips}
			for i, n := range balances {
				want := n - 1
				if n == 0 {
					want = 99
				}
				if got[i] != want {
					t.Fatalf("wallet%d: %d want%d", i, got[i], want)
				}
			}
			if balances[0] == 1 && (!v.Hand.Players[0].AllIn || v.Hand.Actor == v.Seats[0].ID) {
				t.Fatal("positive one must ante allin without refill")
			}
			status, retry := gameCommand(t, c[0], s.URL, "next", "start", before)
			if status != 200 || retry.Version != v.Version || retry.Seats[0].Chips != v.Seats[0].Chips || retry.Bot.Chips != v.Bot.Chips {
				t.Fatal("retry refilled or charged twice")
			}
		})
	}
}
