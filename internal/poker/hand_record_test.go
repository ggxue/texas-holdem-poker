package poker

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

// Tests decode the public contract, not the record's implementation type.
type recordFact struct {
	ID            string `json:"id"`
	HandID        int64  `json:"handID"`
	Seq           int    `json:"seq"`
	At            int64  `json:"at"`
	Stage         string `json:"stage"`
	Kind          string `json:"kind"`
	ParticipantID string `json:"participantID"`
	Seat          int    `json:"seat"`
	Action        string `json:"action"`
	Amount        string `json:"amount"`
	Pot           string `json:"pot"`
	Balance       string `json:"balance"`
	AllIn         bool   `json:"allIn"`
	Reason        string `json:"reason"`
	Board         []Card `json:"board"`
}

func TestHandRecordActualAllInRetryAndRollback(t *testing.T) {
	app, server, clients, _ := bettingRoom(t, [3]int64{3, 100, 100})
	before := gameState(t, clients[0], server.URL)
	status, bet := gameCommand(t, clients[0], server.URL, "record-short", "bet", before)
	e := bet.Hand.Record[len(bet.Hand.Record)-1]
	if status != http.StatusOK || e.Amount != "2" || e.Balance != "0" || e.Pot != "5" || !e.AllIn {
		t.Fatalf("HRC03 actual short all-in: %d %+v", status, e)
	}
	_, retry := gameCommand(t, clients[0], server.URL, "record-short", "bet", before)
	if !reflect.DeepEqual(retry.Hand.Record, bet.Hand.Record) || !reflect.DeepEqual(gameState(t, clients[0], server.URL).Hand.Record, bet.Hand.Record) {
		t.Fatal("HRC05 retry/query appended or rewrote a record")
	}
	status, rejected := gameCommand(t, clients[0], server.URL, "record-wrong", "bet", gameState(t, clients[0], server.URL))
	if status != http.StatusConflict || !reflect.DeepEqual(rejected.Hand.Record, bet.Hand.Record) {
		t.Fatal("HRC05 rejected action changed records")
	}
	app.mu.Lock() // Only inject the existing offline random-source failure.
	app.botThinkSeconds = thinkSeconds(9)
	app.mu.Unlock()
	before = gameState(t, clients[1], server.URL)
	status, rejected = gameCommand(t, clients[1], server.URL, "record-rollback", "call", before)
	if status != http.StatusConflict || !reflect.DeepEqual(rejected.Hand.Record, before.Hand.Record) || rejected.Hand.Pot != before.Hand.Pot || rejected.Hand.Deadline != before.Hand.Deadline {
		t.Fatal("HRC05 transaction rollback left a successful record or modified the old opportunity")
	}
}

func TestHandRecordRunoutAndEveryAward(t *testing.T) {
	prefix := []Card{{2, 0}, {3, 0}, {2, 1}, {3, 1}, {2, 3}, {3, 3}, {10, 2}, {11, 2}, {12, 2}, {13, 2}, {14, 2}}
	app, server := fixtureServer(t, prefix)
	clients := []*http.Client{browser(t), browser(t)}
	for i, c := range clients {
		v := getRoom(t, c, server.URL)
		join(t, c, server.URL, string(rune('a'+i)), v.Version)
	}
	app.mu.Lock() // Existing offline balance fixture; observations use HTTP only.
	for _, id := range app.state.Seats {
		if id != "" {
			app.state.setBalance(id, 1)
		}
	}
	app.state.Bot.Chips = 1
	app.mu.Unlock()
	v := do(t, clients[0], server.URL, "record-runout", "start")
	kinds := []string{}
	for _, e := range v.Hand.Record {
		kinds = append(kinds, e.Kind)
	}
	want := []string{"start", "ante", "ante", "ante", "deal", "flop", "turn", "river", "showdown", "settlement", "award", "award", "award"}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("HRC04 complete all-in runout: %+v", kinds)
	}
	for i, e := range v.Hand.Record[10:] {
		if e.Amount != "1" || e.Balance != "1" || e.Pot != []string{"2", "1", "0"}[i] || e.Stage != "finished" {
			t.Fatalf("HRC04 post-award snapshot: %+v", e)
		}
	}
	for i, n := range []int{3, 4, 5} {
		if len(v.Hand.Record[5+i].Board) != n {
			t.Fatal("HRC04 stage leaked future community cards")
		}
	}
}

func TestHandRecordStartAndActualBet(t *testing.T) {
	s, clients, clock, _, started := clockRoom(t, 2)
	records := started.Hand.Record
	if len(records) != 5 {
		t.Fatalf("HRC01 expected start, three antes and deal; got %+v", records)
	}
	if records[0].Kind != "start" || records[0].Pot != "0" || records[4].Kind != "deal" || len(records[4].Board) != 0 {
		t.Fatalf("HRC01 start/deal facts: %+v", records)
	}
	for i, seat := range []int{0, 1, 5} {
		e := records[i+1]
		if e.Kind != "ante" || e.Seat != seat || e.Amount != "1" || e.Balance != "99" || e.Pot != []string{"1", "2", "3"}[i] || e.ParticipantID == "" {
			t.Fatalf("HRC01 actual ante %d: %+v", i, e)
		}
	}
	for i, e := range records {
		if e.ID == "" || e.Seq != i+1 || e.HandID != started.Hand.ID || e.At != clock.Now().UnixMilli() {
			t.Fatalf("HRC01 committed identity/time: %+v", e)
		}
	}
	bet := do(t, clients[0], s.URL, "record-bet", "bet")
	e := bet.Hand.Record[len(bet.Hand.Record)-1]
	if e.Kind != "action" || e.Action != "bet" || e.Amount != "10" || e.Pot != "13" || e.Balance != "89" || e.ParticipantID != bet.You || e.Reason != "manual" {
		t.Fatalf("HRC02 exact post-bet fact: %+v", e)
	}
	if !reflect.DeepEqual(bet.Hand.Record[:5], records) {
		t.Fatal("HRC02 earlier facts were rewritten")
	}
	do(t, clients[1], s.URL, "record-call", "call")
	clock.advance(time.Second)
	v := gameState(t, clients[0], s.URL)
	last := v.Hand.Record[len(v.Hand.Record)-1]
	if last.Kind != "flop" || last.Pot != "33" || len(last.Board) != 3 || !reflect.DeepEqual(last.Board, v.Hand.Board) {
		t.Fatalf("HRC02 public flop: %+v", last)
	}
}

func TestHandRecordReturnAndSeatReuseKeepOriginalSnapshots(t *testing.T) {
	s, clients, _, _, _ := clockRoom(t, 2)
	bet := do(t, clients[0], s.URL, "record-lifecycle-bet", "bet")
	oldID := bet.You
	oldRecord := append([]recordFact(nil), bet.Hand.Record...)
	returned := do(t, clients[0], s.URL, "record-return", "join")
	last := returned.Hand.Record[len(returned.Hand.Record)-1]
	if last.Kind != "reset" || last.ParticipantID != oldID || last.Balance != "100" || last.Reason != "reconnect" {
		t.Fatalf("HRC06 reconnect wallet explanation is missing: %+v", last)
	}
	if !reflect.DeepEqual(returned.Hand.Record[:len(oldRecord)], oldRecord) {
		t.Fatal("HRC06 reconnect rewrote the old 89-chip balance")
	}
	left := do(t, clients[0], s.URL, "record-depart", "leave")
	foldFound := false
	for _, e := range left.Hand.Record[len(oldRecord):] {
		if e.Kind == "action" && e.Action == "fold" && e.ParticipantID == oldID && e.Reason == "departure" && e.Amount == "0" {
			foldFound = true
		}
	}
	if !foldFound {
		t.Fatal("HRC07 confirmed leave must explain the old participant's fold")
	}
	newcomer := browser(t)
	v := do(t, newcomer, s.URL, "record-newcomer", "join")
	joined := v.Hand.Record[len(v.Hand.Record)-1]
	if v.Seats[0].ID != v.You || joined.Kind != "join" || joined.ParticipantID != v.You || joined.ParticipantID == oldID || joined.Seat != 0 {
		t.Fatalf("HRC07 same-seat newcomer got old record identity: %+v", joined)
	}
	if !reflect.DeepEqual(v.Hand.Record[:len(oldRecord)], oldRecord) || len(v.Hand.Players[0].Hole) != 0 {
		t.Fatal("HRC07 newcomer inherited history edits or old private cards")
	}
}

func TestHandRecordTimeoutDisconnectAndDirectReturn(t *testing.T) {
	s, clients, clock, ws, started := clockRoom(t, 2)
	clock.advance(30 * time.Second)
	v := gameState(t, clients[1], s.URL)
	last := v.Hand.Record[len(v.Hand.Record)-1]
	if last.Kind != "action" || last.Action != "check" || last.Amount != "0" || last.Reason != "timeout" || last.ParticipantID != started.You {
		t.Fatalf("HRC08 timeout explanation: %+v", last)
	}
	ws[0].Close()
	lost := observedDisconnect(t, clients[1], s.URL, 0)
	last = lost.Hand.Record[len(lost.Hand.Record)-1]
	if last.Kind != "disconnect" || last.ParticipantID != started.You {
		t.Fatalf("HRC08 disconnect record missing: %+v", last)
	}
	gameSocket(t, clients[0], s.URL)
	restored := gameState(t, clients[0], s.URL)
	last = restored.Hand.Record[len(restored.Hand.Record)-1]
	if last.Kind != "return" || last.Balance != "99" || len(restored.Hand.Record) != len(lost.Hand.Record)+1 || restored.Hand.Deadline != lost.Hand.Deadline {
		t.Fatalf("HRC08 direct socket restore reset wallet, deadline or history: %+v", last)
	}
}

func TestHandRecordNewHandRefillAndFailedStartRetention(t *testing.T) {
	app, s, clients, _ := bettingRoom(t, [3]int64{100, 100, 100})
	finished := finishChecks(t, clients, s.URL)
	app.mu.Lock() // Existing offline failure seam, no online administration.
	app.actionStart = selectedStart(9)
	app.mu.Unlock()
	status, failed := gameCommand(t, clients[0], s.URL, "record-bad-next", "start", finished)
	if status != http.StatusConflict || !reflect.DeepEqual(failed.Hand.Record, finished.Hand.Record) {
		t.Fatal("HRC09 failed new hand replaced the completed record")
	}
	app.mu.Lock() // Existing controlled post-settlement wallet input.
	app.actionStart = firstActionStart
	app.state.setBalance(finished.You, 0)
	app.mu.Unlock()
	next := do(t, clients[0], s.URL, "record-good-next", "start")
	if next.Hand.ID == finished.Hand.ID || len(next.Hand.Record) != 6 || next.Hand.Record[0].Kind != "start" || next.Hand.Record[1].Kind != "refill" || next.Hand.Record[1].Balance != "100" || next.Hand.Record[1].Pot != "0" || next.Hand.Record[2].Kind != "ante" || next.Hand.Record[2].Balance != "99" {
		t.Fatalf("HRC09 new hand must replace old facts and refill before ante: %+v", next.Hand.Record)
	}
	for _, e := range next.Hand.Record {
		if e.HandID != next.Hand.ID {
			t.Fatal("HRC09 new record retained old hand entries")
		}
	}
}

func TestHandRecordHTTPAndSocketRecoveryShareExactRecords(t *testing.T) {
	s, clients, _, ws, started := clockRoom(t, 2)
	bet := do(t, clients[0], s.URL, "record-transport", "bet")
	push := socketView(t, ws[1], bet.Version)
	query := gameState(t, clients[1], s.URL)
	if !reflect.DeepEqual(push.Hand.Record, bet.Hand.Record) || !reflect.DeepEqual(query.Hand.Record, bet.Hand.Record) {
		t.Fatal("HRC06 HTTP and WS have different event identity or historical snapshots")
	}
	ws[0].Close()
	observedDisconnect(t, clients[1], s.URL, 0)
	reconnected := gameSocket(t, clients[0], s.URL)
	query = gameState(t, clients[0], s.URL)
	push = socketView(t, reconnected, query.Version)
	if len(push.Hand.Record) <= len(started.Hand.Record) || !reflect.DeepEqual(push.Hand.Record, query.Hand.Record) || len(push.Announcements) != 0 {
		t.Fatal("HRC06 recovered socket must include public history without old speech announcements")
	}
	newServer := testServer(t)
	restarted := gameState(t, clients[0], newServer.URL)
	if restarted.Hand != nil || restarted.You == started.You {
		t.Fatal("HRC09 a fresh process retained the previous hand or identity")
	}
}

func TestHandRecordSingleWinnerAndOddSplit(t *testing.T) {
	t.Run("single winner without showdown", func(t *testing.T) {
		_, s, clients, _ := bettingRoom(t, [3]int64{100, 100, 100})
		do(t, clients[0], s.URL, "record-fold-one", "fold")
		v := do(t, clients[1], s.URL, "record-fold-two", "fold")
		awards := []recordFact{}
		for _, e := range v.Hand.Record {
			if e.Kind == "showdown" {
				t.Fatal("early winner was incorrectly recorded as a showdown")
			}
			if e.Kind == "award" {
				awards = append(awards, e)
			}
		}
		if len(awards) != 1 || awards[0].ParticipantID != "bot" || awards[0].Amount != "3" || awards[0].Balance != "102" || awards[0].Pot != "0" {
			t.Fatalf("HRC04 early single winner must receive the entire pool: %+v", awards)
		}
	})
	t.Run("odd split from actual unequal all-ins", func(t *testing.T) {
		prefix := []Card{{2, 0}, {3, 0}, {2, 1}, {3, 1}, {2, 3}, {3, 3}, {10, 2}, {11, 2}, {12, 2}, {13, 2}, {14, 2}}
		app, s := fixtureServer(t, prefix)
		clients := []*http.Client{browser(t), browser(t)}
		for i, c := range clients {
			v := getRoom(t, c, s.URL)
			join(t, c, s.URL, string(rune('a'+i)), v.Version)
		}
		app.mu.Lock() // Existing offline balance input; no artificial settlement or online admin.
		app.state.setBalance(app.state.Seats[0], 2)
		app.state.setBalance(app.state.Seats[1], 3)
		app.state.Bot.Chips = 2
		app.mu.Unlock()
		do(t, clients[0], s.URL, "record-odd-start", "start")
		do(t, clients[0], s.URL, "record-odd-first", "allin")
		v := do(t, clients[1], s.URL, "record-odd-second", "allin")
		awards := []recordFact{}
		for _, e := range v.Hand.Record {
			if e.Kind == "award" {
				awards = append(awards, e)
			}
		}
		if len(awards) != 3 {
			t.Fatalf("CMA02 expected three winners from a seven-chip single pool: %+v", awards)
		}
		for i, e := range awards {
			if e.Amount != []string{"3", "2", "2"}[i] || e.Balance != e.Amount || e.Pot != []string{"4", "2", "0"}[i] {
				t.Fatalf("CMA02 exact odd award and post-transfer snapshots: %+v", awards)
			}
		}
	})
}
