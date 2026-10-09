package poker

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestAAC01ActiveAllInOpensAtRemainingBalance(t *testing.T) {
	_, server, clients, before := bettingRoom(t, [3]int64{100, 100, 100})
	if !slices.Contains(before.Hand.Legal, "allin") || before.Hand.AllInAmount != "99" {
		t.Fatalf("all-in offer must show remaining 99: %+v", before.Hand)
	}
	status, after := settledCommand(t, clients[0], server.URL, "push", "allin", before)
	if status != 200 || after.Hand.Target != 99 || after.Hand.Pot != 102 || after.Seats[0].Chips != 0 || after.Hand.Players[0].Invested != 100 || !after.Hand.Players[0].AllIn {
		t.Fatalf("all-in must transfer 99 to the single pot: status=%d %+v", status, after)
	}
	next := gameState(t, clients[1], server.URL)
	if next.Hand.Actor != next.You || next.Hand.CallAmount != "99" || next.Hand.CallRequired != "99" || !slices.Contains(next.Hand.Legal, "call") || slices.Contains(next.Hand.Legal, "check") {
		t.Fatalf("next player must be offered response to 99: %+v", next.Hand)
	}
}

func TestAAC02RaiseReopensEarlierBettorForDifference(t *testing.T) {
	_, s, c, _ := bettingRoom(t, [3]int64{100, 100, 100})
	do(t, c[0], s.URL, "bet-ten", "bet")
	before := gameState(t, c[1], s.URL)
	v := do(t, c[1], s.URL, "raise", "allin")
	current := gameState(t, c[0], s.URL)
	if v.Hand.Actor != current.You || current.Hand.CallAmount != "89" || current.Hand.CallRequired != "89" || current.Hand.AllInAmount != "89" || v.Hand.Pot != 211 || v.Hand.Turn <= before.Hand.Turn || v.Hand.Deadline <= before.Hand.Deadline {
		t.Fatalf("earlier bettor must respond to 99 by paying only 89: %+v", current.Hand)
	}
	v = do(t, c[0], s.URL, "difference", "call")
	if v.Hand.Stage != "finished" || len(v.Hand.Board) != 5 || v.Hand.Players[0].Won != 300 || v.Hand.Players[0].Invested != 100 || v.Hand.Players[2].Invested != 100 {
		t.Fatalf("three full wallets must settle once as 300: %+v", v.Hand)
	}
}

func TestAAC03SmallestActiveAllInWinsWholePool(t *testing.T) {
	_, s, c, _ := bettingRoom(t, [3]int64{6, 21, 21})
	do(t, c[0], s.URL, "five", "allin")
	v := do(t, c[1], s.URL, "twenty", "allin")
	if v.Hand.Stage != "finished" || v.Hand.Target != 20 || v.Hand.Players[0].Won != 48 || v.Seats[0].Chips != 48 || v.Hand.Players[0].Invested != 6 || v.Hand.Players[1].Won != 0 {
		t.Fatalf("five plus two twenties and three antes all belong to strongest small stack: %+v", v.Hand)
	}
}

// 复用已确认的离线开局钱包及牌序输入；所有结果经公开HTTP/WS观察。
func multiAllInRoom(t *testing.T, prefix []Card, balances []int64) (*App, *httptest.Server, []*http.Client) {
	t.Helper()
	app, s := fixtureServer(t, prefix)
	c := make([]*http.Client, len(balances)-1)
	for i := range c {
		c[i] = browser(t)
		do(t, c[i], s.URL, "join", "join")
	}
	app.mu.Lock()
	for i, id := range app.state.Seats {
		if id != "" {
			app.state.setBalance(id, balances[i])
		}
	}
	app.state.Bot.Chips = balances[len(balances)-1]
	app.mu.Unlock()
	do(t, c[0], s.URL, "start", "start")
	return app, s, c
}

func TestAAC04ContinuousRaisesRetainEarlierAllIns(t *testing.T) {
	prefix := []Card{{14, 3}, {14, 2}, {13, 3}, {13, 2}, {12, 3}, {12, 2}, {10, 3}, {10, 2}, {2, 0}, {4, 1}, {7, 2}, {9, 0}, {11, 1}}
	_, s, c := multiAllInRoom(t, prefix, []int64{21, 51, 81, 101})
	for i, target := range []int64{20, 50, 80} {
		v := do(t, c[i], s.URL, "push", "allin")
		if v.Hand.Target != target || v.Hand.Players[0].Invested != 21 {
			t.Fatalf("raise %d must retain earlier all-in without supplement: %+v", target, v.Hand)
		}
	}
	v := gameState(t, c[0], s.URL)
	if v.Hand.Stage != "finished" || v.Hand.Players[0].Won != 234 || v.Bot.Chips != 20 || v.Hand.Players[1].Invested != 51 || v.Hand.Players[2].Invested != 81 {
		t.Fatalf("20+50+80+80+four antes must settle 234: %+v", v)
	}
}

func TestRaiseReopensEarlierCallerWithoutChargingOriginalCallAgain(t *testing.T) {
	prefix := []Card{{14, 3}, {14, 2}, {13, 3}, {13, 2}, {12, 3}, {12, 2}, {10, 3}, {10, 2}, {2, 0}, {4, 1}, {7, 2}, {9, 0}, {11, 1}}
	_, s, c := multiAllInRoom(t, prefix, []int64{50, 51, 81, 101})
	do(t, c[0], s.URL, "ten", "bet")
	do(t, c[1], s.URL, "first-call", "call")
	do(t, c[2], s.URL, "eighty", "allin")
	do(t, c[0], s.URL, "short-response", "call")
	before := gameState(t, c[1], s.URL)
	if before.Hand.Actor != before.You || before.Hand.CallRequired != "70" || before.Hand.CallAmount != "40" || before.Hand.Players[1].Street != 10 || before.Hand.Players[1].Invested != 11 {
		t.Fatalf("earlier caller must respond to higher target using only remaining difference: %+v", before.Hand)
	}
	end := do(t, c[1], s.URL, "second-call", "call")
	if end.Hand.Stage != "finished" || end.Hand.Target != 80 || end.Hand.Players[1].Invested != 51 || end.Hand.Players[0].Won != 263 || end.Bot.Chips != 20 {
		t.Fatalf("49+50+80+80 plus four antes must settle once as 263: %+v", end.Hand)
	}
}

func TestAAC05And06ShortAndLargeResponses(t *testing.T) {
	for _, third := range []int64{30, 300} {
		for _, action := range []string{"call", "allin", "fold"} {
			t.Run(fmt.Sprintf("%d-%s", third, action), func(t *testing.T) {
				_, s, c := multiAllInRoom(t, nil, []int64{100, 51, third + 1, 501})
				do(t, c[0], s.URL, "first", "allin")
				short := gameState(t, c[1], s.URL)
				if short.Hand.CallAmount != "50" || short.Hand.CallRequired != "99" || !slices.Contains(short.Hand.Legal, "allin") {
					t.Fatalf("short responder offers 50 against target 99: %+v", short.Hand)
				}
				v := do(t, c[1], s.URL, "second", "allin")
				if v.Hand.Target != 99 || !v.Hand.Players[1].AllIn || v.Hand.Players[1].Invested != 51 {
					t.Fatalf("short all-in must not lower 99: %+v", v.Hand)
				}
				v = rawAction(t, c[2], s.URL, "third", action)
				expectedBalance, expectedTarget := int64(0), int64(99)
				if action == "fold" {
					expectedBalance = third
				} else if third == 300 && action == "call" {
					expectedBalance = 201
				} else if third == 300 && action == "allin" {
					expectedTarget = 300
				}
				// raw响应保存动作时的钱包，避免已自动结算的奖项混入扣款断言。
				found := false
				for _, e := range v.Announcements {
					if e.Kind == "action" && e.Seat == 2 && e.Action == action {
						found = true
						amount := "30"
						if third == 300 {
							amount = "99"
						}
						if action == "allin" {
							amount = fmt.Sprint(third)
						}
						if action == "fold" {
							amount = "0"
						}
						if e.Amount != amount {
							t.Fatalf("actual response amount: %+v", e)
						}
					}
				}
				if !found || v.Hand.Target != expectedTarget || v.Seats[2].Chips != expectedBalance || (action == "fold" && !v.Hand.Players[2].Folded) {
					t.Fatalf("response %s from %d: %+v", action, third, v)
				}
			})
		}
	}
}

func TestAAC07LoneLargeResponderCannotAddUnanswerableExcess(t *testing.T) {
	_, s, c, _ := bettingRoom(t, [3]int64{11, 100, 1})
	do(t, c[0], s.URL, "push", "allin")
	before := gameState(t, c[1], s.URL)
	if !reflect.DeepEqual(before.Hand.Legal, []string{"call", "fold"}) || before.Hand.CallAmount != "10" {
		t.Fatalf("lone large responder must only call or fold: %+v", before.Hand)
	}
	status, rejected := gameCommand(t, c[1], s.URL, "fake", "allin", before)
	if status != 409 || rejected.Error != "invalid_action" || rejected.Version != before.Version || rejected.Hand.Pot != 13 || rejected.Seats[1].Chips != 99 || rejected.Hand.Deadline != before.Hand.Deadline || len(rejected.Announcements) != 0 {
		t.Fatalf("illegal excess changed state: %d %+v", status, rejected)
	}
	v := do(t, c[1], s.URL, "call-ten", "call")
	if v.Hand.Stage != "finished" || v.Hand.Players[0].Won != 23 || v.Hand.Players[1].Balance != 89 {
		t.Fatalf("only response then runout: %+v", v.Hand)
	}
}

func TestAAC09ReconnectAllInIncludesPreviousStreetInvestment(t *testing.T) {
	_, s, c, _ := bettingRoom(t, [3]int64{100, 50, 300})
	do(t, c[0], s.URL, "ten", "bet")
	do(t, c[1], s.URL, "raise-49", "allin")
	before := gameState(t, c[0], s.URL)
	returning := browser(t)
	returning.Jar = c[0].Jar
	v := do(t, returning, s.URL, "return", "join")
	if v.Hand.Turn != before.Hand.Turn || v.Hand.Deadline != before.Hand.Deadline || v.Seats[0].Chips != 100 || v.Hand.Players[0].Street != 10 {
		t.Fatalf("return must retain previous 10 and opportunity: %+v", v)
	}
	v = rawAction(t, returning, s.URL, "new-push", "allin")
	if v.Hand.Target != 110 || v.Hand.Players[0].Street != 110 || v.Hand.Pot != 211 || v.Seats[0].Chips != 0 || v.Hand.Players[0].Invested != 111 {
		t.Fatalf("new 100 plus prior 10 must target 110: %+v", v)
	}
}

func TestAAC10AllInRemainsStickyAfterResetAndTakeover(t *testing.T) {
	_, s, c, _ := bettingRoom(t, [3]int64{100, 100, 100})
	v := rawAction(t, c[0], s.URL, "push", "allin")
	old := c[0]
	returning := browser(t)
	returning.Jar = old.Jar
	returned := do(t, returning, s.URL, "return", "join")
	if returned.Seats[0].Chips != 100 || !returned.Hand.Players[0].AllIn || returned.Hand.Players[0].Invested != 100 || returned.Hand.Pot != 102 || returned.Hand.Turn != v.Hand.Turn || returned.Hand.Deadline != v.Hand.Deadline || len(returned.Hand.Legal) != 0 {
		t.Fatalf("wallet reset must not cancel all-in or original response opportunity: %+v", returned)
	}
	status, denied := gameCommand(t, old, s.URL, "old-control", "allin", v)
	if status != 409 || denied.Error != "taken_over" {
		t.Fatalf("old control: %d %+v", status, denied)
	}
	status, denied = gameCommand(t, returning, s.URL, "again", "allin", returned)
	if status != 409 || denied.Hand.Pot != 102 || len(denied.Announcements) != 0 {
		t.Fatalf("sticky all-in paid again: %d %+v", status, denied)
	}
}

func TestAAC11And14ConcurrentRetryPublishesOneExactEvent(t *testing.T) {
	s, c, _, ws, before := clockRoom(t, 2)
	var results [2]gameView
	var statuses [2]int
	var group sync.WaitGroup
	begin := make(chan struct{})
	for i := range results {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			<-begin
			statuses[i], results[i] = gameCommand(t, c[0], s.URL, "same-push", "allin", before)
		}(i)
	}
	close(begin)
	group.Wait()
	if statuses != [2]int{200, 200} || !reflect.DeepEqual(results[0], results[1]) || results[0].Hand.Pot != 102 || results[0].Seats[0].Chips != 0 {
		t.Fatalf("concurrent same request paid twice: %v %+v", statuses, results)
	}
	events := heard(t, ws[1], results[0].Version)
	if len(events) != 1 || events[0].Action != "allin" || events[0].Amount != "99" || !events[0].AllIn || events[0].Seat != 0 || events[0].HandID != before.Hand.ID || events[0].TurnID != before.Hand.Turn {
		t.Fatalf("one exact public confirmation: %+v", events)
	}
	status, rejected := gameCommand(t, c[0], s.URL, "same-push", "fold", before)
	if status != 409 || rejected.Error != "request_conflict" || len(rejected.Announcements) != 0 {
		t.Fatalf("changed retry contents: %d %+v", status, rejected)
	}
	status, rejected = gameCommand(t, c[0], s.URL, "stale", "allin", before)
	if status != 409 || rejected.Version != results[0].Version || len(rejected.Announcements) != 0 {
		t.Fatalf("stale request: %d %+v", status, rejected)
	}
	visible := gameState(t, c[1], s.URL)
	for _, p := range visible.Hand.Players {
		if p.ID != visible.You && len(p.Hole) != 0 {
			t.Fatal("all-in leaked another player's hole cards")
		}
	}
	if len(gameState(t, c[0], s.URL).Announcements) != 0 {
		t.Fatal("state query replayed all-in")
	}
	if len(heard(t, gameSocket(t, c[1], s.URL), visible.Version)) != 0 {
		t.Fatal("new socket replayed all-in")
	}
}

func TestAAC12TimeoutAnswersRaiseWithFoldAndBotWaits(t *testing.T) {
	s, c, clock, _, _ := clockRoom(t, 2)
	rawAction(t, c[0], s.URL, "ten", "bet")
	clock.advance(5 * time.Second)
	waiting := rawAction(t, c[1], s.URL, "push", "allin")
	if waiting.Hand.Actor != "bot" || waiting.Hand.Players[2].Invested != 1 {
		t.Fatal("bot did not wait")
	}
	clock.advance(time.Second)
	reopened := gameState(t, c[0], s.URL)
	if reopened.Hand.CallAmount != "89" || reopened.Hand.Deadline != clock.Now().Add(30*time.Second).UnixMilli() || reopened.Hand.Players[2].Invested != 100 {
		t.Fatalf("new response deadline or bot call: %+v", reopened.Hand)
	}
	clock.advance(29 * time.Second)
	if gameState(t, c[0], s.URL).Hand.Actor != reopened.You {
		t.Fatal("reopened turn expired early")
	}
	clock.advance(time.Second)
	end := gameState(t, c[0], s.URL)
	if end.Hand.Stage != "finished" || !end.Hand.Players[0].Folded || end.Hand.Players[0].Invested != 11 || end.Seats[0].Chips != 89 {
		t.Fatalf("timeout must fold rather than push 89: %+v", end.Hand)
	}
}

func TestAAC13OverflowAndFailedBotOpportunityRollBackAllIn(t *testing.T) {
	t.Run("overflow", func(t *testing.T) {
		_, s, c, before := bettingRoom(t, [3]int64{math.MaxInt64, 100, 100})
		if before.Hand.AllInAmount != "9223372036854775806" {
			t.Fatalf("large offer lost precision: %+v", before.Hand)
		}
		status, rejected := gameCommand(t, c[0], s.URL, "overflow", "allin", before)
		if status != 409 || rejected.Error != "chips_overflow" || rejected.Hand.Pot != 3 || rejected.Hand.Target != 0 || rejected.Seats[0].Chips != math.MaxInt64-1 || rejected.Hand.Deadline != before.Hand.Deadline || rejected.Version != before.Version || len(rejected.Announcements) != 0 {
			t.Fatalf("overflow left partial mutation: %d %+v", status, rejected)
		}
	})
	t.Run("failed-opportunity", func(t *testing.T) {
		app, s, c, _ := bettingRoom(t, [3]int64{11, 100, 100})
		rawAction(t, c[0], s.URL, "push", "allin")
		app.mu.Lock() // 已有离线故障源先例，不在HTTP暴露控制。
		app.botThinkSeconds = thinkSeconds(9)
		app.mu.Unlock()
		before := gameState(t, c[1], s.URL)
		status, rejected := gameCommand(t, c[1], s.URL, "rollback", "allin", before)
		if status != 409 || rejected.Error != "unavailable" || rejected.Version != before.Version || rejected.Hand.Pot != 13 || rejected.Hand.Target != 10 || rejected.Seats[1].Chips != 99 || rejected.Hand.Players[1].AllIn || rejected.Hand.Deadline != before.Hand.Deadline || len(rejected.Announcements) != 0 {
			t.Fatalf("failed next opportunity left partial all-in: %d %+v", status, rejected)
		}
	})
}

func TestLargeAllInOffersAndEventsStayExactBeyondJSIntegerRange(t *testing.T) {
	_, s, c, before := bettingRoom(t, [3]int64{9007199254740994, 100, 100})
	if before.Hand.AllInAmount != "9007199254740993" {
		t.Fatalf("exact action amount: %+v", before.Hand)
	}
	v := rawAction(t, c[0], s.URL, "large", "allin")
	if v.Hand.Pot != 9007199254740996 || v.Announcements[0].Amount != "9007199254740993" {
		t.Fatalf("exact transfer and voice event: %+v", v)
	}
	next := gameState(t, c[1], s.URL)
	if next.Hand.CallRequired != "9007199254740993" || next.Hand.CallAmount != "99" {
		t.Fatalf("exact owed versus actual call: %+v", next.Hand)
	}
}

func TestAAC19SixWayTieIncludesSmallStackAndOddChips(t *testing.T) {
	prefix := []Card{{2, 0}, {3, 0}, {2, 1}, {3, 1}, {2, 2}, {3, 2}, {2, 3}, {3, 3}, {4, 0}, {4, 1}, {4, 2}, {4, 3}, {10, 2}, {11, 2}, {12, 2}, {13, 2}, {14, 2}}
	_, s, c := multiAllInRoom(t, prefix, []int64{6, 21, 21, 21, 21, 21})
	do(t, c[0], s.URL, "small", "allin")
	do(t, c[1], s.URL, "raise", "allin")
	for i := 2; i < len(c); i++ {
		do(t, c[i], s.URL, "call", "call")
	}
	v := gameState(t, c[0], s.URL)
	if v.Hand.Stage != "finished" || v.Hand.Pot != 0 {
		t.Fatalf("tie did not settle: %+v", v.Hand)
	}
	for i, expected := range []int64{19, 19, 19, 18, 18, 18} {
		if v.Hand.Players[i].Won != expected || v.Hand.Players[i].Balance != expected {
			t.Fatalf("111 split at seat %d: %+v", i, v.Hand.Players)
		}
	}
}

func TestAAC08OnlyResponderMayAllInBelowAmountOwed(t *testing.T) {
	_, server, clients, _ := bettingRoom(t, [3]int64{11, 6, 1})
	do(t, clients[0], server.URL, "push-ten", "allin")
	before := gameState(t, clients[1], server.URL)
	if !slices.Contains(before.Hand.Legal, "allin") || before.Hand.CallRequired != "10" || before.Hand.CallAmount != "5" {
		t.Fatalf("only responder must be able to push remaining five: %+v", before.Hand)
	}
	after := do(t, clients[1], server.URL, "push-five", "allin")
	if after.Hand.Stage != "finished" || len(after.Hand.Board) != 5 || after.Hand.Target != 10 || !after.Hand.Players[1].AllIn || after.Hand.Players[1].Invested != 6 || after.Hand.Players[0].Won != 18 {
		t.Fatalf("short response must retain target and run out single pot 18: %+v", after.Hand)
	}
}
