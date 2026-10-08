package poker

import "testing"

func TestAC10MidHandJoinWaitsAndGetsNoPrivateCards(t *testing.T) {
	server := testServer(t)
	first, second := browser(t), browser(t)
	initial := getRoom(t, first, server.URL)
	join(t, first, server.URL, "one", initial.Version)
	do(t, first, server.URL, "start", "start")
	current := getRoom(t, second, server.URL)
	join(t, second, server.URL, "two", current.Version)
	v := gameState(t, second, server.URL)
	if len(v.Hand.Players) != 2 {
		t.Fatal("mid-hand entrant became participant")
	}
	for _, p := range v.Hand.Players {
		if len(p.Hole) != 0 || p.ID == v.You {
			t.Fatal("waiting player obtained private cards or qualification")
		}
	}
	status, rejected := settledCommand(t, second, server.URL, "attempt", "check", v)
	if status != 409 || rejected.Error != "not_your_turn" || rejected.Version != v.Version {
		t.Fatal("waiting player acted")
	}
	ws := gameSocket(t, second, server.URL)
	pushed := socketView(t, ws, v.Version)
	for _, p := range pushed.Hand.Players {
		if len(p.Hole) != 0 {
			t.Fatal("waiting socket saw holes")
		}
	}
	for i := 0; i < 4; i++ {
		do(t, first, server.URL, string(rune('a'+i)), "check")
	}
	do(t, first, server.URL, "next", "start")
	next := gameState(t, second, server.URL)
	if len(next.Hand.Players) != 3 || next.Hand.Players[1].ID != next.You || len(next.Hand.Players[1].Hole) != 2 || next.Seats[1].Chips != 99 {
		t.Fatalf("next participation: %+v", next)
	}
}
func TestAC11And12DepartureTransfersHostWithoutMovingSeats(t *testing.T) {
	_, s, c, initial := bettingRoom(t, [3]int64{100, 100, 100})
	v := do(t, c[0], s.URL, "leave", "leave")
	if v.Seats[0] != nil || v.Host != v.Seats[1].ID || v.Hand.Pot != 3 || !v.Hand.Players[0].Folded || v.Hand.Actor != v.Seats[1].ID {
		t.Fatalf("host departure: %+v", v)
	}
	newcomer := browser(t)
	before := getRoom(t, newcomer, s.URL)
	status, _ := join(t, newcomer, s.URL, "join", before.Version)
	if status != 200 {
		t.Fatal(status)
	}
	n := gameState(t, newcomer, s.URL)
	if n.Host != initial.Seats[1].ID || n.Seats[1].ID != initial.Seats[1].ID || n.Hand.Players[0].ID == n.You {
		t.Fatal("seat reuse transferred host or hand")
	}
	for _, p := range n.Hand.Players {
		if len(p.Hole) != 0 {
			t.Fatal("new seat inherited old cards")
		}
	}
	do(t, c[1], s.URL, "leave", "leave")
	n = gameState(t, newcomer, s.URL)
	if n.Host != n.You || n.Hand.Stage != "finished" || n.Hand.Players[2].Won != 3 || n.Hand.Players[0].Won != 0 {
		t.Fatalf("remaining host or winner: %+v", n)
	}
	do(t, newcomer, s.URL, "leave", "leave")
	empty := gameState(t, newcomer, s.URL)
	if empty.Host != "" || empty.Seats[0] != nil || empty.Seats[1] != nil || empty.Hand.ID != initial.Hand.ID {
		t.Fatal("bot started game or room not empty")
	}
}
func TestAC29ActiveAllInDepartureForfeitsPrizeWithoutRefund(t *testing.T) {
	_, s, c, _ := bettingRoom(t, [3]int64{1, 100, 100})
	before := gameState(t, c[1], s.URL)
	do(t, c[0], s.URL, "leave", "leave")
	after := gameState(t, c[1], s.URL)
	if after.Hand.Pot != 3 || !after.Hand.Players[0].Folded || after.Hand.Turn != before.Hand.Turn {
		t.Fatal("allin exit refunded or reset other action")
	}
	finished := finishChecks(t, c, s.URL)
	if finished.Hand.Players[0].Won != 0 || finished.Hand.Players[0].Invested != 1 {
		t.Fatal("allin leaver retained eligibility")
	}
	do(t, c[1], s.URL, "leave", "leave")
	saved := gameState(t, c[1], s.URL)
	if saved.Hand.Players[1].Won != finished.Hand.Players[1].Won || saved.Hand.Players[2].Won != finished.Hand.Players[2].Won {
		t.Fatal("post-settlement exit undid awards")
	}
}
