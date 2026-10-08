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

type manualClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*manualTimer
}
type manualTimer struct {
	clock   *manualClock
	at      time.Time
	f       func()
	stopped bool
}

func newClock() *manualClock          { return &manualClock{now: time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)} }
func (c *manualClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *manualClock) AfterFunc(d time.Duration, f func()) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	timer := &manualTimer{clock: c, at: c.now.Add(d), f: f}
	c.timers = append(c.timers, timer)
	return timer
}
func (t *manualTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	active := !t.stopped
	t.stopped = true
	return active
}
func (c *manualClock) advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	callbacks := []func(){}
	for _, timer := range c.timers {
		if !timer.stopped && !timer.at.After(c.now) {
			timer.stopped = true
			callbacks = append(callbacks, timer.f)
		}
	}
	c.mu.Unlock()
	for _, f := range callbacks {
		f()
	}
}
func clockRoom(t *testing.T, humans int, balances ...[3]int64) (*httptest.Server, [2]*http.Client, *manualClock, [2]*websocket.Conn, gameView) {
	t.Helper()
	clock := newClock()
	app, e := NewWithOptions(Options{Deck: fixedDeck(nil), Clock: clock, ActionStart: firstActionStart})
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	t.Cleanup(func() { app.Close(); server.Close() })
	clients := [2]*http.Client{browser(t), browser(t)}
	sockets := [2]*websocket.Conn{}
	for i := 0; i < humans; i++ {
		v := getRoom(t, clients[i], server.URL)
		status, _ := join(t, clients[i], server.URL, string(rune('a'+i)), v.Version)
		if status != 200 {
			t.Fatal(status)
		}
		sockets[i] = gameSocket(t, clients[i], server.URL)
	}
	if len(balances) > 0 {
		app.mu.Lock()
		for i, id := range app.state.Seats {
			if id != "" {
				app.state.setBalance(id, balances[0][i])
			}
		}
		app.state.Bot.Chips = balances[0][2]
		app.mu.Unlock()
	}
	started := do(t, clients[0], server.URL, "start", "start")
	return server, clients, clock, sockets, started
}
func observedDisconnect(t *testing.T, c *http.Client, url string, seat int) gameView {
	t.Helper()
	for i := 0; i < 100; i++ {
		v := gameState(t, c, url)
		if v.Seats[seat] != nil && v.Seats[seat].DisconnectedUntil != 0 {
			return v
		}
	}
	t.Fatal("socket loss was not observed")
	return gameView{}
}
func TestAC27ReconnectPreservesOriginalDeadline(t *testing.T) {
	s, c, clock, ws, v := clockRoom(t, 2)
	want := clock.Now().Add(30 * time.Second).UnixMilli()
	if v.Hand.Deadline != want {
		t.Fatalf("new turn deadline %d want %d", v.Hand.Deadline, want)
	}
	ws[0].Close()
	lost := observedDisconnect(t, c[1], s.URL, 0)
	if lost.Seats[0].DisconnectedUntil != want {
		t.Fatal("grace must start when control socket is lost")
	}
	clock.advance(10 * time.Second)
	returning := browser(t)
	returning.Jar = c[0].Jar
	v = do(t, returning, s.URL, "return", "join")
	if v.Hand.Deadline != want || v.Seats[0].DisconnectedUntil != 0 || v.Seats[0].Chips != 100 || v.Host != v.You {
		t.Fatalf("return extended deadline: %+v", v)
	}
	gameSocket(t, returning, s.URL)
	clock.advance(20 * time.Second)
	v = gameState(t, returning, s.URL)
	if v.Hand.Actor != v.Seats[1].ID || v.Seats[0] == nil || v.Hand.Deadline != clock.Now().Add(30*time.Second).UnixMilli() {
		t.Fatalf("old timeout or stale grace: %+v", v)
	}
}
func TestAC28LateDisconnectTimesOutBeforeReturn(t *testing.T) {
	s, c, clock, ws, v := clockRoom(t, 2)
	deadline := v.Hand.Deadline
	clock.advance(25 * time.Second)
	ws[0].Close()
	lost := observedDisconnect(t, c[1], s.URL, 0)
	if lost.Hand.Deadline != deadline {
		t.Fatal("disconnect reset deadline")
	}
	clock.advance(5 * time.Second)
	timed := gameState(t, c[1], s.URL)
	if timed.Hand.Actor != timed.You || timed.Hand.Turn == v.Hand.Turn || timed.Hand.Players[0].Folded {
		t.Fatal("timeout did not check automatically")
	}
	returning := browser(t)
	returning.Jar = c[0].Jar
	restored := do(t, returning, s.URL, "return", "join")
	if restored.Hand.Actor != timed.Hand.Actor || restored.Hand.Turn != timed.Hand.Turn || restored.Hand.Deadline != timed.Hand.Deadline {
		t.Fatal("return undid timeout or extended new actor deadline")
	}
}
func TestAC29OwingTimeoutFoldsButOnlineSeatRemains(t *testing.T) {
	s, c, clock, _, _ := clockRoom(t, 2)
	do(t, c[0], s.URL, "check", "check")
	clock.advance(5 * time.Second)
	do(t, c[1], s.URL, "bet", "bet")
	before := gameState(t, c[0], s.URL)
	clock.advance(30 * time.Second)
	after := gameState(t, c[0], s.URL)
	if !after.Hand.Players[0].Folded || after.Seats[0] == nil || after.Host != after.You || after.Hand.Stage != "flop" || after.Hand.Pot != before.Hand.Pot {
		t.Fatalf("owed timeout: %+v", after)
	}
}
func TestAC30GraceBlocksNewHandUntilDeparture(t *testing.T) {
	s, c, clock, ws, _ := clockRoom(t, 2)
	finishChecks(t, c, s.URL)
	ws[1].Close()
	lost := observedDisconnect(t, c[0], s.URL, 1)
	status, rejected := gameCommand(t, c[0], s.URL, "blocked", "start", lost)
	if status != 409 || rejected.Error != "connection_grace" || rejected.Version != lost.Version {
		t.Fatalf("grace start %d %+v", status, rejected)
	}
	clock.advance(30 * time.Second)
	after := gameState(t, c[0], s.URL)
	if after.Seats[1] != nil || after.Host != after.You {
		t.Fatal("expired grace did not release seat")
	}
	next := do(t, c[0], s.URL, "next", "start")
	if len(next.Hand.Players) != 2 || next.Hand.Pot != 2 {
		t.Fatal("cannot start after grace expiry")
	}
}
func TestAC38SameDeadlineDeparturePrecedesAction(t *testing.T) {
	s, c, clock, ws, _ := clockRoom(t, 1)
	ws[0].Close()
	observedDisconnect(t, c[0], s.URL, 0)
	clock.advance(30 * time.Second)
	v := gameState(t, c[0], s.URL)
	if v.Seats[0] != nil || v.Host != "" || v.Hand.Stage != "finished" || len(v.Hand.Board) != 0 || v.Hand.Players[1].Won != 2 {
		t.Fatalf("departure must precede auto check: %+v", v)
	}
}
func TestAC26TakeoverKeepsDeadlineAndOldCloseStartsNoGrace(t *testing.T) {
	s, c, clock, ws, v := clockRoom(t, 1)
	original := v.Hand.Deadline
	clock.advance(10 * time.Second)
	newPage := browser(t)
	newPage.Jar = c[0].Jar
	after := do(t, newPage, s.URL, "takeover", "join")
	gameSocket(t, newPage, s.URL)
	ws[0].Close()
	clock.advance(20 * time.Second)
	current := gameState(t, newPage, s.URL)
	if after.Hand.Deadline != original || current.Seats[0] == nil || current.Seats[0].DisconnectedUntil != 0 || current.Hand.Stage != "flop" {
		t.Fatalf("takeover deadline or old close: %+v", current)
	}
}

func TestAC29ExpiredAllInPlayerLosesUnsettledEligibility(t *testing.T) {
	s, c, clock, ws, started := clockRoom(t, 2, [3]int64{1, 100, 100})
	if !started.Hand.Players[0].AllIn {
		t.Fatal("fixture must start all-in")
	}
	ws[0].Close()
	observedDisconnect(t, c[1], s.URL, 0)
	clock.advance(30 * time.Second)
	v := gameState(t, c[1], s.URL)
	if v.Seats[0] != nil || !v.Hand.Players[0].Folded || v.Hand.Pot != 3 {
		t.Fatalf("expired allin: %+v", v)
	}
	finished := finishChecks(t, c, s.URL)
	if finished.Hand.Players[0].Won != 0 || finished.Hand.Players[0].Invested != 1 {
		t.Fatal("expired allin kept prize or refunded ante")
	}
}

func TestAC38ConcurrentLateRequestsCannotBeatExpiredDeparture(t *testing.T) {
	s, c, clock, ws, _ := clockRoom(t, 1)
	ws[0].Close()
	before := observedDisconnect(t, c[0], s.URL, 0)
	// 先把公开时钟输入推进到期限，让回调和HTTP读取竞争同一已过期状态。
	clock.mu.Lock()
	clock.now = clock.now.Add(30 * time.Second)
	clock.mu.Unlock()
	var wg sync.WaitGroup
	statuses := make(chan int, 8)
	wg.Add(1)
	go func() { defer wg.Done(); clock.advance(0) }()
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			status, _ := gameCommand(t, c[0], s.URL, fmt.Sprintf("late-%d", i), "check", before)
			statuses <- status
		}(i)
	}
	wg.Wait()
	close(statuses)
	for status := range statuses {
		if status != 409 {
			t.Fatalf("expired command accepted: %d", status)
		}
	}
	after := gameState(t, c[0], s.URL)
	if after.Hand.Stage != "finished" || len(after.Hand.Board) != 0 || after.Hand.Players[1].Won != 2 || after.Seats[0] != nil {
		t.Fatalf("concurrent expiration: %+v", after)
	}
}

func TestAC30EntryWithoutControlHandshakeExpires(t *testing.T) {
	clock := newClock()
	app, e := NewWithOptions(Options{Deck: fixedDeck(nil), Clock: clock})
	if e != nil {
		t.Fatal(e)
	}
	s := httptest.NewServer(app)
	t.Cleanup(func() { app.Close(); s.Close() })
	c := browser(t)
	initial := gameState(t, c, s.URL)
	status, joined := gameCommand(t, c, s.URL, "join", "join", initial)
	if status != 200 || joined.Seats[0] == nil {
		t.Fatalf("join: %d %+v", status, joined)
	}
	status, rejected := gameCommand(t, c, s.URL, "start", "start", joined)
	if status != 409 || rejected.Error != "connection_grace" {
		t.Fatal("page with no control socket could start")
	}
	clock.advance(10 * time.Second)
	status, retry := gameCommand(t, c, s.URL, "retry-join", "join", gameState(t, c, s.URL))
	if status != 200 || retry.Seats[0].ConnectingUntil != joined.Seats[0].ConnectingUntil {
		t.Fatal("failed handshake retry extended original connection deadline")
	}
	clock.advance(20 * time.Second)
	expired := gameState(t, c, s.URL)
	if expired.Seats[0] != nil || expired.Host != "" || expired.Hand != nil {
		t.Fatal("failed handshake retained seat forever")
	}
}
