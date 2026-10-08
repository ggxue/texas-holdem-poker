package poker

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// The fixture controls only cards and time; assertions cross HTTP and WebSocket.
func humanRoom(t *testing.T, humans int, options ...Options) (*httptest.Server, []*http.Client, []*websocket.Conn, *manualClock) {
	t.Helper()
	clock := newClock()
	config := Options{Deck: fixedDeck(nil), Clock: clock, ActionStart: firstActionStart}
	if len(options) > 0 {
		config = options[0]
		config.Clock = clock
		if config.Deck == nil {
			config.Deck = fixedDeck(nil)
		}
		if config.ActionStart == nil {
			config.ActionStart = firstActionStart
		}
	}
	if config.BotThinkSeconds == nil {
		config.BotThinkSeconds = thinkSeconds(1)
	}
	app, err := NewWithOptions(config)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(app)
	registerClock(t, server.URL, clock)
	t.Cleanup(func() { app.Close(); server.Close() })
	clients := make([]*http.Client, humans)
	sockets := make([]*websocket.Conn, humans)
	for i := range clients {
		clients[i] = browser(t)
		before := gameState(t, clients[i], server.URL)
		status, joined := settledCommand(t, clients[i], server.URL, "enter", "join", before)
		if status != http.StatusOK {
			t.Fatalf("human %d entry: %d %+v", i+1, status, joined)
		}
		sockets[i] = gameSocket(t, clients[i], server.URL)
	}
	return server, clients, sockets, clock
}

func TestUAC05HostPassesToEarliestRemainingHumanAfterSeatReuse(t *testing.T) {
	s, clients, _, _ := humanRoom(t, 3)
	original := gameState(t, clients[0], s.URL)
	do(t, clients[0], s.URL, "leave-first", "leave")
	newcomer := browser(t)
	do(t, newcomer, s.URL, "fill-empty-seat", "join")
	gameSocket(t, newcomer, s.URL)
	after := do(t, clients[1], s.URL, "leave-second", "leave")
	if after.Host != original.Seats[2].ID || after.Seats[2].ID != original.Seats[2].ID || after.Seats[0].ID == original.Seats[0].ID {
		t.Fatalf("later occupant of seat 1 must not overtake original human 3: %+v", after)
	}
}

func TestUAC02OneToFiveHumansReceivePrivateCardsAndPayOneAnte(t *testing.T) {
	for _, tc := range []struct {
		humans int
		pot    int64
	}{{1, 2}, {2, 3}, {3, 4}, {4, 5}, {5, 6}} {
		t.Run(fmt.Sprint(tc.humans), func(t *testing.T) {
			s, clients, sockets, _ := humanRoom(t, tc.humans)
			before := gameState(t, clients[0], s.URL)
			status, started := settledCommand(t, clients[0], s.URL, "start", "start", before)
			if status != http.StatusOK || started.Hand == nil || started.Hand.Pot != tc.pot || len(started.Hand.Players) != tc.humans+1 || started.Bot.Chips != 99 {
				t.Fatalf("start/ante: %d %+v", status, started)
			}
			bot := started.Hand.Players[len(started.Hand.Players)-1]
			if bot.ID != "bot" || bot.Seat != 5 || bot.Invested != 1 {
				t.Fatalf("fixed bot seat/ante: %+v", bot)
			}
			seen := map[Card]bool{}
			for i, client := range clients {
				v := gameState(t, client, s.URL)
				if v.Seats[i].Chips != 99 || v.Hand.Players[i].Seat != i || v.Hand.Players[i].ID != v.You || v.Hand.Players[i].Invested != 1 {
					t.Fatalf("human %d ante/participation: %+v", i+1, v)
				}
				for _, visible := range []gameView{v, socketView(t, sockets[i], started.Version)} {
					for _, p := range visible.Hand.Players {
						want := 0
						if p.ID == visible.You {
							want = 2
						}
						if len(p.Hole) != want || p.Strength != nil {
							t.Fatalf("human %d HTTP/WS private view: %+v", i+1, p)
						}
					}
				}
				for _, card := range v.Hand.Players[i].Hole {
					if seen[card] {
						t.Fatalf("duplicate human hole card: %+v", card)
					}
					seen[card] = true
				}
			}
			status, retry := settledCommand(t, clients[0], s.URL, "start", "start", before)
			if status != http.StatusOK || retry.Version != started.Version || gameState(t, clients[0], s.URL).Hand.Pot != tc.pot {
				t.Fatal("start retry repeated ante or changed hand")
			}
			finished := finishHumanChecks(t, clients, s.URL)
			seen = map[Card]bool{}
			for _, card := range finished.Hand.Board {
				seen[card] = true
			}
			if len(seen) != 5 {
				t.Fatal("showdown requires five distinct public cards")
			}
			won := int64(0)
			for _, p := range finished.Hand.Players {
				won += p.Won
				if len(p.Hole) != 2 || p.Strength == nil {
					t.Fatalf("showdown card visibility: %+v", p)
				}
				for _, card := range p.Hole {
					if seen[card] {
						t.Fatalf("duplicate dealt card at showdown: %+v", card)
					}
					seen[card] = true
				}
			}
			if won != tc.pot || finished.Hand.Pot != 0 {
				t.Fatal("ante-only hand did not conserve chips at settlement")
			}
		})
	}
}

func finishHumanChecks(t *testing.T, clients []*http.Client, address string) gameView {
	t.Helper()
	for attempts := 0; attempts <= 20; attempts++ {
		v := gameState(t, clients[0], address)
		if v.Hand.Stage == "finished" {
			return v
		}
		found := false
		for _, client := range clients {
			current := gameState(t, client, address)
			if current.You == v.Hand.Actor {
				do(t, client, address, fmt.Sprintf("check-%d", current.Hand.Turn), "check")
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no human controls current action: %+v", v.Hand)
		}
	}
	t.Fatal("four check streets did not settle")
	return gameView{}
}

func TestUAC03FullRoomTakeoverPreservesTurnAndOldCloseIsHarmless(t *testing.T) {
	s, clients, sockets, clock := humanRoom(t, 5)
	do(t, clients[0], s.URL, "start", "start")
	for i := 0; i < 4; i++ {
		do(t, clients[i], s.URL, "pass", "check")
	}
	before := gameState(t, clients[4], s.URL)
	clock.advance(10 * time.Second)
	newPage := browser(t)
	newPage.Jar = clients[4].Jar
	after := do(t, newPage, s.URL, "takeover", "join")
	gameSocket(t, newPage, s.URL)
	if after.You != before.You || after.Seats[4].ID != before.You || after.Seats[4].Chips != 100 || after.Hand.ID != before.Hand.ID || after.Hand.Turn != before.Hand.Turn || after.Hand.Deadline != before.Hand.Deadline || after.Hand.Pot != 6 || len(after.Hand.Players) != 6 {
		t.Fatalf("takeover changed seat, hand, ante or existing deadline: %+v", after)
	}
	status, denied := settledCommand(t, clients[4], s.URL, "old-action", "check", before)
	if status != http.StatusConflict || denied.Error != "taken_over" {
		t.Fatalf("old page action: %d %+v", status, denied)
	}
	sockets[4].SetReadDeadline(time.Now().Add(5 * time.Second))
	found := false
	for messages := 0; messages < 10; messages++ {
		var oldView gameView
		if err := sockets[4].ReadJSON(&oldView); err != nil {
			t.Fatal(err)
		}
		if oldView.Error == "taken_over" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("old socket takeover notice missing")
	}
	sockets[4].Close()
	clock.advance(20 * time.Second)
	current := finishBot(t, newPage, s.URL, gameState(t, newPage, s.URL))
	if current.Seats[4] == nil || current.Seats[4].DisconnectedUntil != 0 || current.Seats[4].ConnectingUntil != 0 || current.Hand.Stage != "flop" || current.Hand.Actor != current.Seats[0].ID || current.Hand.Deadline != clock.Now().Add(30*time.Second).UnixMilli() || current.Hand.Players[4].Folded {
		t.Fatalf("old close departed new page or original timeout failed: %+v", current)
	}
}

func TestUAC04MidHandEntrantCannotInheritParticipationOrPrize(t *testing.T) {
	for _, reuse := range []bool{false, true} {
		t.Run(fmt.Sprintf("reuse=%t", reuse), func(t *testing.T) {
			s, clients, _, _ := humanRoom(t, 3)
			started := do(t, clients[0], s.URL, "start", "start")
			if reuse {
				do(t, clients[2], s.URL, "leave", "leave")
			}
			newcomer := browser(t)
			entered := do(t, newcomer, s.URL, "enter", "join")
			gameSocket(t, newcomer, s.URL)
			seat := 3
			if reuse {
				seat = 2
			}
			if entered.Seats[seat] == nil || entered.Seats[seat].ID != entered.You || entered.Seats[seat].Chips != 100 || entered.Hand.ID != started.Hand.ID || entered.Hand.Pot != 4 || len(entered.Hand.Players) != 4 || len(entered.Hand.Legal) != 0 {
				t.Fatalf("mid-hand entry changed hand or inherited actions: %+v", entered)
			}
			for _, visible := range []gameView{entered, socketView(t, gameSocket(t, newcomer, s.URL), entered.Version)} {
				for _, p := range visible.Hand.Players {
					if p.ID == entered.You || len(p.Hole) != 0 || p.Strength != nil {
						t.Fatalf("new seat inherited participation/private cards: %+v", p)
					}
				}
			}
			before := gameState(t, newcomer, s.URL)
			status, denied := settledCommand(t, newcomer, s.URL, "cannot-act", "check", before)
			if status != http.StatusConflict || denied.Error != "not_your_turn" || denied.Version != before.Version || denied.Hand.Pot != before.Hand.Pot {
				t.Fatalf("waiting entrant acted: %d %+v", status, denied)
			}
			finished := finishHumanChecks(t, clients, s.URL)
			won := int64(0)
			for _, p := range finished.Hand.Players {
				won += p.Won
				if p.ID == entered.You || (reuse && p.ID == started.Seats[2].ID && (!p.Folded || p.Won != 0 || p.Invested != 1)) {
					t.Fatalf("seat reuse transferred award eligibility or refunded ante: %+v", p)
				}
			}
			if won != 4 || gameState(t, newcomer, s.URL).Seats[seat].Chips != 100 {
				t.Fatal("waiting entrant received prize or ante was refunded")
			}
			do(t, clients[0], s.URL, "next", "start")
			next := gameState(t, newcomer, s.URL)
			participating := false
			for _, p := range next.Hand.Players {
				if p.ID == next.You && p.Seat == seat && len(p.Hole) == 2 && p.Invested == 1 {
					participating = true
				}
			}
			if !participating || next.Seats[seat].Chips != 99 {
				t.Fatal("waiting entrant did not participate in next hand")
			}
		})
	}
}

func TestUAC05TakeoverPreservesEntryOrderAndDepartureResetsIt(t *testing.T) {
	for _, returnFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("returnFirst=%t", returnFirst), func(t *testing.T) {
			s, clients, _, _ := humanRoom(t, 3)
			original := gameState(t, clients[0], s.URL)
			do(t, clients[0], s.URL, "leave", "leave")
			newcomer := browser(t)
			if returnFirst {
				newcomer.Jar = clients[0].Jar
			}
			do(t, newcomer, s.URL, "return-or-enter", "join")
			gameSocket(t, newcomer, s.URL)
			newPage := browser(t)
			newPage.Jar = clients[2].Jar
			do(t, newPage, s.URL, "takeover", "join")
			gameSocket(t, newPage, s.URL)
			after := do(t, clients[1], s.URL, "leave", "leave")
			if after.Host != original.Seats[2].ID {
				t.Fatalf("takeover changed entry order or departed member retained old priority: %+v", after)
			}
		})
	}
}

func TestUAC01ConcurrentEntriesCannotOverfillOrDuplicateSeats(t *testing.T) {
	s, _, _, _ := humanRoom(t, 0)
	clients := make([]*http.Client, 8)
	initial := make([]gameView, 8)
	for i := range clients {
		clients[i] = browser(t)
		initial[i] = gameState(t, clients[i], s.URL)
	}
	type entryResult struct {
		client int
		status int
		view   gameView
	}
	results := make(chan entryResult, 8)
	var wg sync.WaitGroup
	for i, client := range clients {
		wg.Add(1)
		go func(i int, client *http.Client) {
			defer wg.Done()
			current := initial[i]
			for attempt := 0; attempt < 8; attempt++ {
				status, v := settledCommand(t, client, s.URL, fmt.Sprintf("enter-%d", attempt), "join", current)
				if v.Error != "stale_state" {
					results <- entryResult{i, status, v}
					return
				}
				current = gameState(t, client, s.URL)
			}
			results <- entryResult{client: i}
		}(i, client)
	}
	wg.Wait()
	close(results)
	joined := map[string]bool{}
	rejected := 0
	for result := range results {
		if result.status == http.StatusOK {
			joined[result.view.You] = true
			gameSocket(t, clients[result.client], s.URL)
		} else if result.status == http.StatusConflict && result.view.Error == "room_full" {
			rejected++
		} else {
			t.Fatalf("entry did not converge to entered/full: %+v", result)
		}
	}
	if len(joined) != 5 || rejected != 3 {
		t.Fatalf("concurrent capacity: joined %d, rejected %d", len(joined), rejected)
	}
	final := gameState(t, clients[0], s.URL)
	seen := map[string]bool{}
	for _, seat := range final.Seats {
		if seat == nil || !joined[seat.ID] || seen[seat.ID] || seat.Chips != 100 {
			t.Fatalf("concurrent entry lost/duplicated an identity: %+v", final)
		}
		seen[seat.ID] = true
	}
	if !joined[final.Host] {
		t.Fatal("host is not one of the confirmed room members")
	}
}
