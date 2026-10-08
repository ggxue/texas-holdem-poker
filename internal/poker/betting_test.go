package poker

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func bettingRoom(t *testing.T, balances [3]int64) (*App, *httptest.Server, [2]*http.Client, gameView) {
	t.Helper()
	prefix := []Card{{14, 3}, {14, 2}, {13, 3}, {13, 2}, {12, 3}, {12, 2}, {2, 0}, {4, 1}, {7, 2}, {9, 0}, {11, 1}}
	app, server := fixtureServer(t, prefix)
	clients := [2]*http.Client{browser(t), browser(t)}
	for i, c := range clients {
		v := getRoom(t, c, server.URL)
		status, _ := join(t, c, server.URL, string(rune('a'+i)), v.Version)
		if status != 200 {
			t.Fatal(status)
		}
	}
	// 已确认的非公开余额fixture；行为断言只经公开命令及视图。
	app.mu.Lock()
	for i, id := range app.state.Seats {
		app.state.setBalance(id, balances[i])
	}
	app.state.Bot.Chips = balances[2]
	app.mu.Unlock()
	status, v := gameCommand(t, clients[0], server.URL, "start", "start", gameState(t, clients[0], server.URL))
	if status != 200 {
		t.Fatalf("start %d %+v", status, v)
	}
	return app, server, clients, v
}
func do(t *testing.T, c *http.Client, url, id, action string) gameView {
	t.Helper()
	status, v := gameCommand(t, c, url, id, action, gameState(t, c, url))
	if status != 200 {
		t.Fatalf("%s: %d %+v", action, status, v)
	}
	if action == "join" {
		gameSocket(t, c, url)
		return gameState(t, c, url)
	}
	return v
}
func finishChecks(t *testing.T, clients [2]*http.Client, url string) gameView {
	t.Helper()
	for i := 0; i < 12; i++ {
		v := gameState(t, clients[0], url)
		if v.Hand.Stage == "finished" {
			return v
		}
		c := clients[0]
		if v.Hand.Actor != v.You {
			c = clients[1]
		}
		do(t, c, url, string(rune('A'+i)), "check")
	}
	t.Fatal("hand did not finish")
	return gameView{}
}
func TestAC14And15OneBetAndEarlierCheckerResponds(t *testing.T) {
	_, s, c, _ := bettingRoom(t, [3]int64{100, 100, 100})
	do(t, c[0], s.URL, "check-first", "check")
	beforeBet := gameState(t, c[1], s.URL)
	status, v := gameCommand(t, c[1], s.URL, "bet", "bet", beforeBet)
	if status != 200 {
		t.Fatalf("bet: %d %+v", status, v)
	}
	status, retry := gameCommand(t, c[1], s.URL, "bet", "bet", beforeBet)
	if status != 200 || retry.Version != v.Version || retry.Hand.Pot != v.Hand.Pot {
		t.Fatal("AC34 lost-response retry repeated bet")
	}
	status, conflict := gameCommand(t, c[1], s.URL, "bet", "call", beforeBet)
	if status != 409 || conflict.Error != "request_conflict" || conflict.Version != v.Version {
		t.Fatal("AC34 same request ID changed contents")
	}
	if v.Hand.Pot != 23 || v.Hand.Target != 10 || v.Hand.Actor != gameState(t, c[0], s.URL).You || v.Bot.Chips != 89 {
		t.Fatalf("bet and bot call: %+v", v.Hand)
	}
	current := gameState(t, c[0], s.URL)
	for i, action := range []string{"bet", "check"} {
		status, rejected := gameCommand(t, c[0], s.URL, string(rune('a'+i)), action, current)
		if status != 409 || rejected.Version != current.Version || rejected.Hand.Pot != 23 {
			t.Fatalf("illegal %s: %d %+v", action, status, rejected)
		}
	}
	v = do(t, c[0], s.URL, "call", "call")
	if v.Hand.Stage != "flop" || v.Hand.Pot != 33 || v.Hand.Players[0].Street != 0 || v.Seats[0].Chips != 89 {
		t.Fatalf("call: %+v", v)
	}
	before := gameState(t, c[0], s.URL)
	status, old := gameCommand(t, c[0], s.URL, "old", "check", current)
	if status != 409 || old.Version != before.Version {
		t.Fatal("stale action changed state")
	}
	data := `{"requestID":"amount","version":` + strconv.FormatInt(v.Version, 10) + `,"action":"bet","amount":6,"pageID":"` + pageID(c[0]) + `","control":` + strconv.FormatInt(v.Control, 10) + `,"handID":` + strconv.FormatInt(v.Hand.ID, 10) + `,"turnID":` + strconv.FormatInt(v.Hand.Turn, 10) + `}`
	response, e := c[0].Post(s.URL+"/api/command", "application/json", strings.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	response.Body.Close()
	if response.StatusCode != 400 {
		t.Fatal("arbitrary amount accepted")
	}
}
func TestAC16FirstShortBetSetsActualTarget(t *testing.T) {
	_, s, c, _ := bettingRoom(t, [3]int64{7, 100, 100})
	v := do(t, c[0], s.URL, "short", "bet")
	if v.Hand.Target != 6 || v.Seats[0].Chips != 0 || !v.Hand.Players[0].AllIn {
		t.Fatalf("short bet: %+v", v)
	}
	v = do(t, c[1], s.URL, "call", "call")
	if v.Bot.Chips != 93 || v.Seats[1].Chips != 93 || v.Hand.Pot != 21 || v.Hand.Stage != "flop" {
		t.Fatalf("target 6 calls: %+v", v)
	}
}
func TestAC20ShortCallerWinsWholeSinglePot(t *testing.T) {
	_, s, c, _ := bettingRoom(t, [3]int64{6, 100, 100})
	do(t, c[0], s.URL, "check", "check")
	do(t, c[1], s.URL, "bet", "bet")
	v := do(t, c[0], s.URL, "short-call", "call")
	if v.Hand.Pot != 28 || v.Seats[0].Chips != 0 || !v.Hand.Players[0].AllIn {
		t.Fatalf("short call: %+v", v)
	}
	v = finishChecks(t, c, s.URL)
	if v.Hand.Players[0].Won != 28 || v.Seats[0].Chips != 28 || v.Hand.Players[1].Won != 0 || v.Hand.Players[2].Won != 0 {
		t.Fatalf("single pot: %+v", v)
	}
}
func TestAC18And21UncalledExcessStaysAndRunout(t *testing.T) {
	app, s, c, _ := bettingRoom(t, [3]int64{11, 6, 6})
	// 仅fixture换赢家私牌，牌序依然合法唯一。
	app.mu.Lock()
	app.state.Hand.Players[0].Hole, app.state.Hand.Players[1].Hole = app.state.Hand.Players[1].Hole, app.state.Hand.Players[0].Hole
	app.mu.Unlock()
	do(t, c[0], s.URL, "bet", "bet")
	v := do(t, c[1], s.URL, "call", "call")
	if v.Hand.Stage != "finished" || len(v.Hand.Board) != 5 || v.Hand.Players[1].Won != 23 || v.Seats[1].Chips != 23 || v.Seats[0].Chips != 0 {
		t.Fatalf("23 runout: %+v", v)
	}
}

func TestAC18LoneCapablePlayerMustAnswerBeforeRunout(t *testing.T) {
	_, s, c, _ := bettingRoom(t, [3]int64{100, 11, 6})
	do(t, c[0], s.URL, "check", "check")
	v := do(t, c[1], s.URL, "bet", "bet")
	if v.Hand.Stage != "preflop" || v.Hand.Actor != v.Seats[0].ID || v.Hand.Target != 10 || v.Hand.Players[2].Invested != 6 {
		t.Fatalf("owed call skipped: %+v", v)
	}
	v = do(t, c[0], s.URL, "call", "call")
	if v.Hand.Stage != "finished" || len(v.Hand.Board) != 5 || v.Hand.Players[0].Invested != 11 {
		t.Fatalf("lone call runout: %+v", v)
	}
}
func TestAC17AnteAllInAndAC23BotShortCall(t *testing.T) {
	_, s, c, v := bettingRoom(t, [3]int64{1, 100, 5})
	if !v.Hand.Players[0].AllIn || v.Hand.Actor != v.Seats[1].ID {
		t.Fatalf("ante allin: %+v", v.Hand)
	}
	v = do(t, c[1], s.URL, "bet", "bet")
	if v.Hand.Stage != "finished" || v.Hand.Players[2].Invested != 5 || v.Hand.Players[1].Invested != 11 || v.Hand.Players[0].Invested != 1 {
		t.Fatalf("bot short call: %+v", v)
	}
}
func TestAC19FoldEndsWithoutExposingCards(t *testing.T) {
	_, s, c, _ := bettingRoom(t, [3]int64{100, 100, 100})
	do(t, c[0], s.URL, "fold-one", "fold")
	v := do(t, c[1], s.URL, "fold-two", "fold")
	if v.Hand.Stage != "finished" || v.Hand.Players[2].Won != 3 || len(v.Hand.Board) != 0 || len(v.Hand.Players[2].Hole) != 0 || len(v.Hand.Players[0].Hole) != 0 {
		t.Fatalf("early win: %+v", v)
	}
}
