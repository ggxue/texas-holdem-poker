package poker

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
)

func firstActionStart(seats []int) (int, error) { return seats[0], nil }
func selectedStart(seat int) func([]int) (int, error) {
	return func(seats []int) (int, error) {
		for _, candidate := range seats {
			if candidate == seat {
				return seat, nil
			}
		}
		return 0, fmt.Errorf("seat %d is not participating", seat)
	}
}

func TestUAC11OddChipGoesToWinnerFirstInThisHandsCycle(t *testing.T) {
	prefix := []Card{{2, 0}, {3, 0}, {2, 1}, {3, 1}, {2, 2}, {3, 2}, {2, 3}, {3, 3}, {4, 0}, {4, 1}, {4, 2}, {4, 3}, {10, 2}, {11, 2}, {12, 2}, {13, 2}, {14, 2}}
	app, s := fixtureServer(t, prefix, Options{ActionStart: selectedStart(3)})
	clients := make([]*http.Client, 5)
	for i := range clients {
		clients[i] = browser(t)
		do(t, clients[i], s.URL, "join", "join")
	}
	do(t, clients[0], s.URL, "start", "start")
	// 已确认的离线结算输入：底池21、仅真人1与真人4未弃牌，结果只从公开视图读取。
	app.mu.Lock()
	app.state.Hand.Pot = 21
	for i := range app.state.Hand.Players {
		if i != 0 && i != 3 {
			app.state.Hand.Players[i].Folded = true
			app.state.Hand.Players[i].acted = true
		}
	}
	app.mu.Unlock()
	finished := finishHumanChecks(t, clients, s.URL)
	if finished.Hand.Players[3].Won != 11 || finished.Hand.Players[0].Won != 10 || finished.Hand.Pot != 0 {
		t.Fatalf("expected human4=11 human1=10: %+v", finished.Hand.Players)
	}
}

func TestUAC06EachStreetUsesTheSameSelectedActionStart(t *testing.T) {
	s, clients, _, _ := humanRoom(t, 5, Options{ActionStart: selectedStart(3)})
	do(t, clients[0], s.URL, "start", "start")
	for street := 0; street < 4; street++ {
		for _, seat := range []int{3, 4, 0, 1, 2} {
			v := gameState(t, clients[seat], s.URL)
			if v.Hand.Actor != v.You {
				t.Fatalf("street %d expected human %d action, got %s", street, seat+1, v.Hand.Actor)
			}
			do(t, clients[seat], s.URL, fmt.Sprintf("check-%d", street), "check")
		}
	}
	if gameState(t, clients[0], s.URL).Hand.Stage != "finished" {
		t.Fatal("four streets must settle")
	}
}

func TestUAC07AllParticipantsIncludingBotCanBeSelected(t *testing.T) {
	for _, seat := range []int{0, 1, 2, 3, 4, 5} {
		t.Run(fmt.Sprint(seat), func(t *testing.T) {
			s, clients, _, _ := humanRoom(t, 5, Options{ActionStart: selectedStart(seat)})
			v := do(t, clients[0], s.URL, "start", "start")
			want := seat
			if want == 5 {
				want = 0
			}
			if v.Hand.StartSeat != seat || v.Hand.Actor != v.Seats[want].ID {
				t.Fatalf("selected start/actor: %+v", v.Hand)
			}
		})
	}
	s, clients, _, _ := humanRoom(t, 2, Options{ActionStart: selectedStart(5)})
	do(t, clients[0], s.URL, "leave", "leave")
	v := do(t, clients[1], s.URL, "start", "start")
	if v.Hand.StartSeat != 5 || v.Hand.Actor != v.You || len(v.Hand.Players) != 2 {
		t.Fatal("empty seat was not skipped")
	}
	newcomer := browser(t)
	entered := do(t, newcomer, s.URL, "enter", "join")
	if entered.Hand.StartSeat != 5 || len(entered.Hand.Players) != 2 {
		t.Fatal("waiting entrant changed candidate set or start")
	}
}

func TestUAC08AllInAndDepartedStartRemainTheHandsStart(t *testing.T) {
	for _, allIn := range []bool{false, true} {
		t.Run(fmt.Sprint(allIn), func(t *testing.T) {
			app, s := fixtureServer(t, nil, Options{ActionStart: selectedStart(3)})
			clients := make([]*http.Client, 5)
			for i := range clients {
				clients[i] = browser(t)
				do(t, clients[i], s.URL, "join", "join")
			}
			if allIn {
				id := gameState(t, clients[3], s.URL).You
				app.mu.Lock()
				app.state.setBalance(id, 1)
				app.mu.Unlock()
			}
			do(t, clients[0], s.URL, "start", "start")
			if !allIn {
				do(t, clients[3], s.URL, "leave", "leave")
			}
			v := gameState(t, clients[4], s.URL)
			if v.Hand.StartSeat != 3 || v.Hand.Actor != v.You || (allIn && !v.Hand.Players[3].AllIn) {
				t.Fatalf("invalid start skip: %+v", v.Hand)
			}
			finished := finishHumanChecks(t, clients, s.URL)
			if finished.Hand.StartSeat != 3 || (!allIn && finished.Hand.Players[3].Won != 0) {
				t.Fatal("start changed or departed player won")
			}
		})
	}
}

func TestUAC09EarlierCheckRespondsToBetInSelectedCycle(t *testing.T) {
	s, clients, _, _ := humanRoom(t, 5, Options{ActionStart: selectedStart(3)})
	do(t, clients[0], s.URL, "start", "start")
	do(t, clients[3], s.URL, "check", "check")
	do(t, clients[4], s.URL, "bet", "bet")
	for _, seat := range []int{0, 1, 2} {
		do(t, clients[seat], s.URL, "call", "call")
	}
	v := gameState(t, clients[3], s.URL)
	if v.Hand.Actor != v.You || !slices.Equal(v.Hand.Legal, []string{"call", "fold"}) {
		t.Fatalf("earlier checker was skipped: %+v", v.Hand)
	}
	after := do(t, clients[3], s.URL, "call", "call")
	if after.Hand.Stage != "flop" || after.Hand.Actor != after.You || after.Hand.Pot != 66 {
		t.Fatalf("matched players repeated or next street changed start: %+v", after.Hand)
	}
}

func TestUAC10TakeoverKeepsStartAndNewHandDrawsAgain(t *testing.T) {
	choices := []int{3, 2}
	s, clients, _, _ := humanRoom(t, 5, Options{ActionStart: func([]int) (int, error) {
		seat := choices[0]
		if len(choices) > 1 {
			choices = choices[1:]
		}
		return seat, nil
	}})
	started := do(t, clients[0], s.URL, "start", "start")
	newPage := browser(t)
	newPage.Jar = clients[3].Jar
	v := do(t, newPage, s.URL, "takeover", "join")
	if v.Hand.StartSeat != 3 || v.Hand.Deadline != started.Hand.Deadline {
		t.Fatal("takeover changed start or deadline")
	}
	clients[3] = newPage
	if finishHumanChecks(t, clients, s.URL).Hand.StartSeat != 3 {
		t.Fatal("new street redrew start")
	}
	if do(t, clients[0], s.URL, "next", "start").Hand.StartSeat != 2 {
		t.Fatal("new hand did not use new controlled choice")
	}
}

func TestUAC12OddChipsSkipNonWinnerAndDepartedStart(t *testing.T) {
	for _, leave := range []bool{false, true} {
		t.Run(fmt.Sprint(leave), func(t *testing.T) {
			prefix := []Card{{2, 0}, {3, 0}, {2, 1}, {3, 1}, {2, 2}, {3, 2}, {2, 3}, {3, 3}, {4, 0}, {4, 1}, {4, 2}, {4, 3}, {10, 2}, {11, 2}, {12, 2}, {13, 2}, {14, 2}}
			app, s := fixtureServer(t, prefix, Options{ActionStart: selectedStart(2)})
			clients := make([]*http.Client, 5)
			for i := range clients {
				clients[i] = browser(t)
				do(t, clients[i], s.URL, "join", "join")
			}
			do(t, clients[0], s.URL, "start", "start")
			if leave {
				do(t, clients[2], s.URL, "leave", "leave")
			}
			app.mu.Lock()
			app.state.Hand.Pot = 23
			app.state.Hand.Actor = 3
			for i := range app.state.Hand.Players {
				if i != 0 && i != 3 && i != 4 {
					app.state.Hand.Players[i].Folded = true
					app.state.Hand.Players[i].acted = true
				}
			}
			app.mu.Unlock()
			v := finishHumanChecks(t, clients, s.URL)
			if v.Hand.StartSeat != 2 || v.Hand.Players[3].Won != 8 || v.Hand.Players[4].Won != 8 || v.Hand.Players[0].Won != 7 {
				t.Fatalf("only winners should receive at most one odd chip: %+v", v.Hand.Players)
			}
		})
	}
}
