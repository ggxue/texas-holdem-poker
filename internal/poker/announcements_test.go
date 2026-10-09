package poker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// 只解码HTTP/WebSocket公开事件，测试不依赖实现中的事件类型。
type heardEvent struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Seat   int    `json:"seat"`
	Action string `json:"action"`
	Amount string `json:"amount"`
	AllIn  bool   `json:"allIn"`
	Reason string `json:"reason"`
	At     int64  `json:"at"`
	HandID int64  `json:"handID"`
	TurnID int64  `json:"turnID"`
}

func TestVoiceAllInRunoutKeepsEveryStageAndWinner(t *testing.T) {
	prefix := []Card{{2, 0}, {3, 0}, {2, 1}, {3, 1}, {2, 3}, {3, 3}, {10, 2}, {11, 2}, {12, 2}, {13, 2}, {14, 2}}
	app, s := fixtureServer(t, prefix)
	c := [2]*http.Client{browser(t), browser(t)}
	for i := range c {
		v := getRoom(t, c[i], s.URL)
		join(t, c[i], s.URL, fmt.Sprint(i), v.Version)
	}
	app.mu.Lock() // 仅离线fixture预置余额，断言全部通过公开命令。
	for _, id := range app.state.Seats {
		if id != "" {
			app.state.setBalance(id, 1)
		}
	}
	app.state.Bot.Chips = 1
	app.mu.Unlock()
	v := do(t, c[0], s.URL, "all-in-start", "start")
	kinds := []string{}
	for _, e := range v.Announcements {
		kinds = append(kinds, e.Kind)
	}
	if !reflect.DeepEqual(kinds, []string{"start", "deal", "flop", "turn", "river", "showdown", "settlement", "winner", "winner", "winner"}) {
		t.Fatalf("complete runout: %+v", v.Announcements)
	}
	for i, e := range v.Announcements[7:] {
		if e.Seat != []int{0, 1, 5}[i] || e.Amount != "1" {
			t.Fatalf("tied exact awards: %+v", v.Announcements)
		}
	}
}

func heard(t *testing.T, ws *websocket.Conn, version int64) []heardEvent {
	t.Helper()
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		var v struct {
			Version int64        `json:"version"`
			Events  []heardEvent `json:"announcements"`
		}
		if err = json.Unmarshal(data, &v); err != nil {
			t.Fatal(err)
		}
		if v.Version >= version {
			return v.Events
		}
	}
}

func TestVoiceConfirmedBetHasActualAmountAndStableIdentity(t *testing.T) {
	s, c, clock, ws, started := clockRoom(t, 2)
	bet := do(t, c[0], s.URL, "bet-voice", "bet")
	events := heard(t, ws[1], bet.Version)
	if len(events) != 1 || events[0].Kind != "action" || events[0].Seat != 0 || events[0].Action != "bet" || events[0].Amount != "10" || events[0].AllIn || events[0].Reason != "manual" || events[0].ID == "" || events[0].At != clock.Now().UnixMilli() || events[0].HandID != started.Hand.ID || events[0].TurnID != started.Hand.Turn {
		t.Fatalf("confirmed exact bet announcement: %+v", events)
	}
}

func TestVoiceTimeoutExplainsCheckAndFold(t *testing.T) {
	for _, owing := range []bool{false, true} {
		t.Run(map[bool]string{false: "check", true: "fold"}[owing], func(t *testing.T) {
			s, c, clock, ws, _ := clockRoom(t, 2)
			wantSeat, wantAction := 0, "check"
			if owing {
				do(t, c[0], s.URL, "bet", "bet")
				wantSeat, wantAction = 1, "fold"
			}
			clock.advance(30 * time.Second)
			v := gameState(t, c[0], s.URL)
			events := heard(t, ws[0], v.Version)
			if len(events) == 0 || events[0].Kind != "action" || events[0].Seat != wantSeat || events[0].Action != wantAction || events[0].Reason != "timeout" || events[0].Amount != "0" {
				t.Fatalf("timeout fact: %+v", events)
			}
		})
	}
}

func TestVoiceRoomChangesAndRobotThinking(t *testing.T) {
	s, c, clock, ws, _ := clockRoom(t, 2)
	do(t, c[0], s.URL, "check-one", "check")
	v := rawAction(t, c[1], s.URL, "check-two", "check")
	events := heard(t, ws[0], v.Version)
	if len(events) != 2 || events[1].Kind != "thinking" || events[1].Seat != 5 {
		t.Fatalf("robot thinking: %+v", events)
	}
	clock.advance(time.Second)
	v = gameState(t, c[0], s.URL)
	events = heard(t, ws[1], v.Version)
	if len(events) < 2 || events[0].Seat != 5 || events[0].Action != "check" || events[0].Reason != "robot" || events[1].Kind != "flop" {
		t.Fatalf("robot and stage: %+v", events)
	}
	ws[0].Close()
	v = observedDisconnect(t, c[1], s.URL, 0)
	events = heard(t, ws[1], v.Version)
	if len(events) != 1 || events[0].Kind != "disconnect" || events[0].Seat != 0 {
		t.Fatalf("disconnect: %+v", events)
	}
	v = rawAction(t, c[0], s.URL, "return", "join")
	if len(v.Announcements) != 1 || v.Announcements[0].Kind != "return" {
		t.Fatalf("return: %+v", v.Announcements)
	}
	gameSocket(t, c[0], s.URL)
	v = do(t, c[0], s.URL, "leave", "leave")
	if len(v.Announcements) < 2 || v.Announcements[0].Kind != "leave" || v.Announcements[1].Kind != "host" || v.Announcements[1].Seat != 1 {
		t.Fatalf("host transfer: %+v", v.Announcements)
	}
}

func TestVoiceRetryRejectAndReconnectDoNotReplay(t *testing.T) {
	s, c, _, ws, _ := clockRoom(t, 2)
	before := gameState(t, c[0], s.URL)
	_, bet := gameCommand(t, c[0], s.URL, "stable-bet", "bet", before)
	first := heard(t, ws[1], bet.Version)
	_, retry := gameCommand(t, c[0], s.URL, "stable-bet", "bet", before)
	if !reflect.DeepEqual(first, retry.Announcements) {
		t.Fatalf("retry identity changed: %+v / %+v", first, retry.Announcements)
	}
	status, rejected := gameCommand(t, c[0], s.URL, "wrong-turn", "bet", gameState(t, c[0], s.URL))
	if status != 409 || len(rejected.Announcements) != 0 {
		t.Fatalf("rejected speech: %+v", rejected)
	}
	query := gameState(t, c[0], s.URL)
	if len(query.Announcements) != 0 {
		t.Fatalf("state replay: %+v", query.Announcements)
	}
	newSocket := gameSocket(t, c[1], s.URL)
	if events := heard(t, newSocket, query.Version); len(events) != 0 {
		t.Fatalf("new socket replay: %+v", events)
	}
}

func TestVoiceRollbackAndShortAllInAreTruthful(t *testing.T) {
	app, s, c, _ := bettingRoom(t, [3]int64{3, 100, 100})
	v := rawAction(t, c[0], s.URL, "short-bet", "bet")
	if len(v.Announcements) != 1 || v.Announcements[0].Amount != "2" || !v.Announcements[0].AllIn {
		t.Fatalf("actual all-in amount: %+v", v.Announcements)
	}
	app.mu.Lock() // 离线故障源使下个机器人机会抽样失败。
	app.botThinkSeconds = thinkSeconds(9)
	app.mu.Unlock()
	before := gameState(t, c[1], s.URL)
	status, rejected := gameCommand(t, c[1], s.URL, "rollback-call", "call", before)
	if status != 409 || len(rejected.Announcements) != 0 || rejected.Hand.Pot != before.Hand.Pot {
		t.Fatalf("rolled-back event: %+v", rejected)
	}
}

func TestVoiceDirectSocketReturnIsAnnouncedWithoutReplay(t *testing.T) {
	s, c, _, ws, _ := clockRoom(t, 2)
	ws[0].Close()
	lost := observedDisconnect(t, c[1], s.URL, 0)
	heard(t, ws[1], lost.Version)
	restored := gameSocket(t, c[0], s.URL)
	v := gameState(t, c[1], s.URL)
	if events := heard(t, restored, v.Version); len(events) != 0 {
		t.Fatalf("returning socket received history: %+v", events)
	}
	events := heard(t, ws[1], v.Version)
	if len(events) != 1 || events[0].Kind != "return" || events[0].Seat != 0 {
		t.Fatalf("direct control connection return: %+v", events)
	}
}
