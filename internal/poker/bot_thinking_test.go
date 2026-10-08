package poker

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

var fixtureClocks sync.Map

func registerClock(t *testing.T, url string, clock *manualClock) {
	t.Helper()
	fixtureClocks.Store(url, clock)
	t.Cleanup(func() { fixtureClocks.Delete(url) })
}

// Finish only observable bot opportunities in legacy scenarios that test human actions.
// Timing acceptance uses rawAction/gameCommand instead, preserving the actual receipt.
func finishBot(t *testing.T, c *http.Client, url string, v gameView) gameView {
	t.Helper()
	for v.Hand != nil && v.Hand.Actor == "bot" {
		value, ok := fixtureClocks.Load(url)
		if !ok {
			t.Fatal("fixture has no controllable clock")
		}
		value.(*manualClock).advance(time.Second)
		v = gameState(t, c, url)
	}
	return v
}

func settledCommand(t *testing.T, c *http.Client, url, id, action string, v gameView) (int, gameView) {
	t.Helper()
	status, result := gameCommand(t, c, url, id, action, v)
	if status == http.StatusOK {
		result = finishBot(t, c, url, result)
	}
	return status, result
}

func thinkSeconds(seconds int) func() (int, error) {
	return func() (int, error) { return seconds, nil }
}
func rawAction(t *testing.T, c *http.Client, url, id, action string) gameView {
	t.Helper()
	status, v := gameCommand(t, c, url, id, action, gameState(t, c, url))
	if status != http.StatusOK {
		t.Fatalf("%s: %d %+v", action, status, v)
	}
	return v
}
func TestUAC13BotWaitsOneSecondBeforeCheck(t *testing.T) {
	s, clients, _, clock := humanRoom(t, 1, Options{ActionStart: selectedStart(5), BotThinkSeconds: thinkSeconds(1)})
	v := rawAction(t, clients[0], s.URL, "start", "start")
	if v.Hand.Actor != "bot" || v.Hand.Deadline != clock.Now().Add(30*time.Second).UnixMilli() {
		t.Fatalf("bot must wait with 30-second display: %+v", v.Hand)
	}
	clock.advance(999 * time.Millisecond)
	if gameState(t, clients[0], s.URL).Hand.Actor != "bot" {
		t.Fatal("bot checked early")
	}
	clock.advance(time.Millisecond)
	current := gameState(t, clients[0], s.URL)
	if current.Hand.Actor != current.You || current.Hand.Turn == v.Hand.Turn || current.Hand.Deadline != clock.Now().Add(30*time.Second).UnixMilli() {
		t.Fatalf("bot did not check once then give human 30 seconds: %+v", current.Hand)
	}
	version := current.Version
	if gameState(t, clients[0], s.URL).Version != version {
		t.Fatal("state query repeated bot action")
	}
}

func TestUAC14BotWaitsEightSecondsThenCallsActualBalance(t *testing.T) {
	for _, balance := range []int64{100, 5} {
		t.Run(fmt.Sprint(balance), func(t *testing.T) {
			clock := newClock()
			app, s := fixtureServer(t, nil, Options{Clock: clock, BotThinkSeconds: thinkSeconds(8)})
			c := browser(t)
			do(t, c, s.URL, "join", "join")
			app.mu.Lock()
			app.state.Bot.Chips = balance // 离线余额输入，断言仅检查公开状态。
			app.mu.Unlock()
			rawAction(t, c, s.URL, "start", "start")
			waiting := rawAction(t, c, s.URL, "bet", "bet")
			clock.advance(7999 * time.Millisecond)
			before := gameState(t, c, s.URL)
			if before.Hand.Actor != "bot" || before.Bot.Chips != balance-1 || before.Hand.Pot != waiting.Hand.Pot || !before.Hand.BotThinking {
				t.Fatal("call occurred before eight seconds")
			}
			clock.advance(time.Millisecond)
			after := gameState(t, c, s.URL)
			amount := min(int64(10), balance-1)
			if after.Hand.Players[1].Invested != 1+amount || after.Hand.Players[1].Folded || after.Hand.Players[1].AllIn != (balance == 5) {
				t.Fatalf("wrong robot policy or deduction: %+v", after.Hand)
			}
			if balance == 5 && (after.Hand.Stage != "finished" || len(after.Hand.Board) != 5) {
				t.Fatal("short call must run out immediately")
			}
			if balance == 100 && (after.Hand.Stage != "flop" || after.Hand.Actor != after.You) {
				t.Fatal("full call must advance to human")
			}
		})
	}
}

func TestUAC15And17NewCallOpportunityDrawsAgainAndHumanGetsThirtySeconds(t *testing.T) {
	choices := []int{1, 3, 8}
	s, c, ws, clock := humanRoom(t, 2, Options{ActionStart: selectedStart(5), BotThinkSeconds: func() (int, error) {
		n := choices[0]
		if len(choices) > 1 {
			choices = choices[1:]
		}
		return n, nil
	}})
	start := rawAction(t, c[0], s.URL, "start", "start")
	clock.advance(time.Second)
	rawAction(t, c[0], s.URL, "check", "check")
	bet := rawAction(t, c[1], s.URL, "bet", "bet")
	if bet.Hand.Turn == start.Hand.Turn || !bet.Hand.BotThinking {
		t.Fatal("new call opportunity missing")
	}
	clock.advance(2 * time.Second)
	before := gameState(t, c[0], s.URL)
	if before.Hand.Actor != "bot" || before.Hand.Deadline-beforeServerTime(clock) != 28000 {
		t.Fatal("second draw or display basis was reused")
	}
	clock.advance(time.Second)
	after := gameState(t, c[0], s.URL)
	if after.Hand.Actor != after.You || after.Hand.Deadline != clock.Now().Add(30*time.Second).UnixMilli() || after.Hand.Players[2].Invested != 11 {
		t.Fatalf("human deadline or call: %+v", after.Hand)
	}
	push := socketView(t, ws[0], after.Version)
	if push.Hand.Turn != after.Hand.Turn || push.Hand.BotThinking || len(push.Hand.Players[1].Hole) != 0 {
		t.Fatal("push must preserve privacy and completed action")
	}
}

func beforeServerTime(clock *manualClock) int64 { return clock.Now().UnixMilli() }

func TestUAC16And18QueriesTakeoverAndOldClosePreserveThinking(t *testing.T) {
	s, c, ws, clock := humanRoom(t, 5, Options{ActionStart: selectedStart(5), BotThinkSeconds: thinkSeconds(8)})
	start := rawAction(t, c[0], s.URL, "start", "start")
	clock.advance(3 * time.Second)
	for i := 0; i < 5; i++ {
		current := gameState(t, c[i], s.URL)
		if current.Hand.Deadline != start.Hand.Deadline || current.Hand.Actor != "bot" {
			t.Fatal("query reset thinking")
		}
		for _, p := range current.Hand.Players {
			if p.ID != current.You && len(p.Hole) > 0 {
				t.Fatal("waiting query leaked hole")
			}
		}
	}
	ws[0].Close()
	observedDisconnect(t, c[1], s.URL, 0)
	page := browser(t)
	page.Jar = c[0].Jar
	joined := rawAction(t, page, s.URL, "takeover", "join")
	gameSocket(t, page, s.URL)
	if joined.Seats[0].Chips != 100 || joined.Hand.Turn != start.Hand.Turn || joined.Hand.Deadline != start.Hand.Deadline {
		t.Fatal("takeover changed original opportunity")
	}
	clock.advance(4999 * time.Millisecond)
	if gameState(t, page, s.URL).Hand.Actor != "bot" {
		t.Fatal("actual delay shortened")
	}
	clock.advance(time.Millisecond)
	current := gameState(t, page, s.URL)
	if current.Hand.Actor != current.You || current.Seats[0].DisconnectedUntil != 0 || current.Hand.Deadline != clock.Now().Add(30*time.Second).UnixMilli() {
		t.Fatal("actual delay extended or old close harmed new page")
	}
}

func TestUAC19And20DepartureSettlesImmediatelyAndOldCallbacksCannotAffectNextHand(t *testing.T) {
	s, c, _, clock := humanRoom(t, 1, Options{ActionStart: selectedStart(5), BotThinkSeconds: thinkSeconds(8)})
	rawAction(t, c[0], s.URL, "start", "start")
	clock.mu.Lock()
	callbacks := append([]*manualTimer{}, clock.timers...)
	clock.mu.Unlock()
	end := rawAction(t, c[0], s.URL, "leave", "leave")
	if end.Hand.Stage != "finished" || end.Hand.Players[1].Won != 2 || len(end.Hand.Board) != 0 || end.Hand.BotThinking {
		t.Fatal("last human leave waited for bot")
	}
	for _, timer := range callbacks {
		timer.f()
	} // 模拟已经取消却迟到的底层唤醒。
	if gameState(t, c[0], s.URL).Version != end.Version {
		t.Fatal("late callback repeated settlement")
	}
	rawAction(t, c[0], s.URL, "rejoin", "join")
	gameSocket(t, c[0], s.URL)
	next := rawAction(t, c[0], s.URL, "next", "start")
	for _, timer := range callbacks {
		timer.f()
	}
	current := gameState(t, c[0], s.URL)
	if current.Version != next.Version || current.Hand.ID != next.Hand.ID || current.Hand.Turn != next.Hand.Turn || current.Hand.Pot != 2 {
		t.Fatal("old callback changed new hand")
	}
}

func TestUAC20GraceDeparturePrecedesThinkingAtSameInstant(t *testing.T) {
	s, c, ws, clock := humanRoom(t, 1, Options{BotThinkSeconds: thinkSeconds(8)})
	rawAction(t, c[0], s.URL, "start", "start")
	ws[0].Close()
	observedDisconnect(t, c[0], s.URL, 0)
	clock.advance(22 * time.Second)
	bet := rawAction(t, c[0], s.URL, "bet", "bet")
	clock.advance(8 * time.Second)
	end := gameState(t, c[0], s.URL)
	if end.Seats[0] != nil || end.Hand.Stage != "finished" || end.Hand.Players[1].Invested != 1 || end.Hand.Players[1].Won != bet.Hand.Pot || len(end.Hand.Board) != 0 {
		t.Fatalf("bot called before same-time departure: %+v", end.Hand)
	}
}

func TestUAC21AllInRobotAndRunoutDoNotCreateThinking(t *testing.T) {
	clock := newClock()
	app, s := fixtureServer(t, nil, Options{Clock: clock, ActionStart: selectedStart(5), BotThinkSeconds: func() (int, error) { return 0, fmt.Errorf("must not create a bot opportunity") }})
	c := browser(t)
	do(t, c, s.URL, "join", "join")
	app.mu.Lock()
	app.state.Bot.Chips = 1
	app.mu.Unlock()
	end := rawAction(t, c, s.URL, "start", "start")
	if end.Hand.Stage != "finished" || len(end.Hand.Board) != 5 || !end.Hand.Players[1].AllIn || end.Hand.BotThinking || end.Hand.Deadline != 0 {
		t.Fatal("all-in runout acquired a delay")
	}
}

func TestInvalidThinkingSamplerRollsBackCommandAndReportsAutomaticFault(t *testing.T) {
	for _, seconds := range []int{0, 9} {
		s, c, _, _ := humanRoom(t, 1, Options{ActionStart: selectedStart(5), BotThinkSeconds: thinkSeconds(seconds)})
		before := gameState(t, c[0], s.URL)
		status, after := gameCommand(t, c[0], s.URL, "start", "start", before)
		if status != 409 || after.Error != "unavailable" || after.Version != before.Version || after.Hand != nil || after.Seats[0].Chips != 100 || after.Bot.Chips != 100 {
			t.Fatal("bad delay left a partial start")
		}
	}
	clock := newClock()
	app, e := NewWithOptions(Options{Clock: clock, ActionStart: firstActionStart, BotThinkSeconds: func() (int, error) { return 0, fmt.Errorf("entropy failed") }})
	if e != nil {
		t.Fatal(e)
	}
	s := httptest.NewServer(app)
	t.Cleanup(func() { app.Close(); s.Close() })
	c := browser(t)
	rawAction(t, c, s.URL, "join", "join")
	gameSocket(t, c, s.URL)
	start := rawAction(t, c, s.URL, "start", "start")
	clock.advance(30 * time.Second)
	fault := gameState(t, c, s.URL)
	if fault.Error != "unavailable" || fault.Version != start.Version || fault.Hand.Actor != start.Hand.Actor || fault.Hand.Pot != start.Hand.Pot {
		t.Fatal("automatic sampler fault did not roll back")
	}
}
