package poker

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestUAC28FiveHumansShortAllInTakeoverSeatReuseAndNextHand(t *testing.T) {
	// 独立牌例：真人1的AA胜过其余对子，板面J9742不组成顺子或同花。
	prefix := []Card{{14, 3}, {14, 2}, {13, 3}, {13, 2}, {12, 3}, {12, 2}, {10, 3}, {10, 2}, {8, 3}, {8, 2}, {6, 3}, {6, 2}, {2, 0}, {4, 1}, {7, 2}, {9, 0}, {11, 1}}
	clock := newClock()
	app, s := fixtureServer(t, prefix, Options{Clock: clock, ActionStart: selectedStart(3), BotThinkSeconds: thinkSeconds(3)})
	clients := make([]*http.Client, 5)
	for i := range clients {
		clients[i] = browser(t)
		do(t, clients[i], s.URL, "join", "join")
	}
	oldSocket := gameSocket(t, clients[3], s.URL)
	oldID := gameState(t, clients[2], s.URL).You
	firstID := gameState(t, clients[0], s.URL).You
	// 离线开局钱包输入，所有行为与结果断言仍穿过HTTP／WebSocket。
	app.mu.Lock()
	app.state.setBalance(firstID, 5)
	app.mu.Unlock()
	started := rawAction(t, clients[0], s.URL, "start", "start")
	if started.Hand.StartSeat != 3 || started.Hand.Pot != 6 || len(started.Hand.Players) != 6 {
		t.Fatal("five humans and controlled start not established")
	}
	rawAction(t, clients[3], s.URL, "bet", "bet")
	waiting := rawAction(t, clients[4], s.URL, "call", "call")
	if waiting.Hand.Actor != "bot" || !waiting.Hand.BotThinking {
		t.Fatal("bot call must wait")
	}
	rawAction(t, clients[2], s.URL, "depart", "leave")
	newcomer := browser(t)
	rawAction(t, newcomer, s.URL, "fill", "join")
	gameSocket(t, newcomer, s.URL)
	view := gameState(t, newcomer, s.URL)
	if view.Seats[2].ID != view.You || view.Hand.Players[2].ID != oldID || !view.Hand.Players[2].Folded || view.Hand.StartSeat != 3 || view.Hand.Turn != waiting.Hand.Turn || view.Hand.Deadline != waiting.Hand.Deadline {
		t.Fatal("seat reuse changed fixed participants or waiting opportunity")
	}
	for _, p := range view.Hand.Players {
		if len(p.Hole) != 0 {
			t.Fatal("waiting entrant inherited private cards")
		}
	}
	oldPage := clients[3]
	newPage := browser(t)
	newPage.Jar = oldPage.Jar
	takeover := rawAction(t, newPage, s.URL, "takeover", "join")
	gameSocket(t, newPage, s.URL)
	oldSocket.Close()
	clients[3] = newPage
	if takeover.Seats[3].Chips != 100 || takeover.Hand.Players[3].Invested != 11 || takeover.Hand.Pot != 26 || takeover.Hand.Turn != waiting.Hand.Turn || takeover.Hand.Deadline != waiting.Hand.Deadline {
		t.Fatal("takeover altered investment or bot timing")
	}
	status, denied := gameCommand(t, oldPage, s.URL, "old", "check", waiting)
	if status != 409 || denied.Error != "taken_over" {
		t.Fatal("old controller affected waiting hand")
	}
	clock.advance(2999 * time.Millisecond)
	before := gameState(t, clients[0], s.URL)
	if before.Hand.Actor != "bot" || before.Hand.Players[5].Invested != 1 {
		t.Fatal("robot call occurred early")
	}
	// 让五个身份查询与真实到期竞争，仍只断言各公开响应及最终扣款。
	begin := make(chan struct{})
	var group sync.WaitGroup
	for _, client := range clients {
		group.Add(1)
		go func(c *http.Client) {
			defer group.Done()
			<-begin
			visible := gameState(t, c, s.URL)
			for _, p := range visible.Hand.Players {
				if p.ID != visible.You && len(p.Hole) > 0 {
					t.Error("concurrent query leaked hole")
				}
			}
		}(client)
	}
	group.Add(1)
	go func() { defer group.Done(); <-begin; clock.advance(time.Millisecond) }()
	close(begin)
	group.Wait()
	called := gameState(t, clients[0], s.URL)
	if called.Hand.Players[5].Invested != 11 || called.Hand.Pot != 36 || called.Hand.Actor != called.You || called.Hand.Deadline != clock.Now().Add(30*time.Second).UnixMilli() {
		t.Fatal("concurrent arrival repeated bot action or stole human time")
	}
	short := rawAction(t, clients[0], s.URL, "short", "call")
	if short.Hand.Players[0].Invested != 5 || !short.Hand.Players[0].AllIn || short.Hand.Pot != 40 || short.Seats[0].Chips != 0 {
		t.Fatal("short all-in must pay four chips")
	}
	returning := browser(t)
	returning.Jar = clients[0].Jar
	reset := rawAction(t, returning, s.URL, "return", "join")
	gameSocket(t, returning, s.URL)
	clients[0] = returning
	if reset.Seats[0].Chips != 100 || !reset.Hand.Players[0].AllIn || reset.Hand.Players[0].Invested != 5 || reset.Hand.Deadline != short.Hand.Deadline || reset.Host != reset.You {
		t.Fatal("all-in takeover reset qualification or next human deadline")
	}
	rawAction(t, clients[1], s.URL, "call", "call")
	for step := 0; step < 30; step++ {
		current := gameState(t, clients[0], s.URL)
		if current.Hand.Stage == "finished" {
			break
		}
		if current.Hand.Actor == "bot" {
			clock.advance(3 * time.Second)
			continue
		}
		found := false
		for _, client := range clients {
			v := gameState(t, client, s.URL)
			if v.You == current.Hand.Actor {
				rawAction(t, client, s.URL, fmt.Sprintf("check-%d", step), "check")
				found = true
				break
			}
		}
		if !found {
			t.Fatal("waiting entrant or departed identity gained an action")
		}
	}
	finished := gameState(t, clients[0], s.URL)
	// 手算：6底注＋真人4/5/2和bot各10＋真人1短Call4＝50，真人1重连后100再得50。
	if finished.Hand.Stage != "finished" || finished.Hand.Players[0].Won != 50 || finished.Hand.Players[0].Balance != 150 || finished.Hand.Pot != 0 || finished.Hand.StartSeat != 3 || finished.Hand.Players[2].Won != 0 || finished.Hand.Players[2].Invested != 1 {
		t.Fatalf("single pot, reset100 or departed eligibility: %+v", finished.Hand)
	}
	for _, p := range finished.Hand.Players {
		if p.ID == view.You {
			t.Fatal("new identity inherited old prize")
		}
	}
	push := socketView(t, gameSocket(t, newcomer, s.URL), finished.Version)
	if len(push.Hand.Players[2].Hole) != 0 || push.Hand.Players[2].Strength != nil {
		t.Fatal("folded old occupant leaked at showdown")
	}
	clients[2] = newcomer
	next := rawAction(t, clients[0], s.URL, "next", "start")
	if next.Hand.ID == finished.Hand.ID || next.Hand.Pot != 6 || next.Seats[0].Chips != 149 || next.Seats[2].Chips != 99 || next.Hand.Players[2].ID != view.You || next.Host != next.You {
		t.Fatal("next hand lost wallets, entry order or new participation")
	}
}
