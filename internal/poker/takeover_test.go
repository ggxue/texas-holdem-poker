package poker

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAC26NewPageTakesControlAndOldEventsAreRejected(t *testing.T) {
	s := testServer(t)
	oldPage := browser(t)
	initial := getRoom(t, oldPage, s.URL)
	status, _ := join(t, oldPage, s.URL, "join", initial.Version)
	if status != 200 {
		t.Fatal(status)
	}
	oldSocket := gameSocket(t, oldPage, s.URL)
	before := do(t, oldPage, s.URL, "start", "start")
	newPage := browser(t)
	newPage.Jar = oldPage.Jar
	current := gameState(t, newPage, s.URL)
	status, after := gameCommand(t, newPage, s.URL, "new-page", "join", current)
	if status != 200 || after.You != before.You || after.Hand.ID != before.Hand.ID || after.Hand.Turn != before.Hand.Turn || after.Seats[0].Chips != 100 || after.Hand.Pot != 2 || after.Host != before.Host {
		t.Fatalf("takeover: %d %+v", status, after)
	}
	oldSocket.SetReadDeadline(time.Now().Add(5 * time.Second))
	found := false
	for i := 0; i < 5; i++ {
		_, data, e := oldSocket.ReadMessage()
		if e != nil {
			t.Fatal(e)
		}
		var v gameView
		if e = json.Unmarshal(data, &v); e != nil {
			t.Fatal(e)
		}
		if v.Error == "taken_over" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("old socket did not receive takeover message")
	}
	status, rejected := gameCommand(t, oldPage, s.URL, "late", "check", before)
	if status != 409 || rejected.Error != "taken_over" || rejected.Version != after.Version {
		t.Fatalf("old action: %d %+v", status, rejected)
	}
	oldCurrent := gameState(t, oldPage, s.URL)
	status, rejected = gameCommand(t, oldPage, s.URL, "old-reconnect", "join", oldCurrent)
	if status != 409 || rejected.Error != "taken_over" {
		t.Fatalf("old reconnect: %d %+v", status, rejected)
	}
	oldSocket.Close()
	newSocket := gameSocket(t, newPage, s.URL)
	pushed := socketView(t, newSocket, after.Version)
	if pushed.Host != before.Host || pushed.Hand.Turn != before.Hand.Turn || len(pushed.Hand.Players[0].Hole) != 2 {
		t.Fatal("old close changed control or new view")
	}
	do(t, newPage, s.URL, "new-check", "check")
}
func TestAC32OldHandTurnAndForgedWalletAreRejected(t *testing.T) {
	_, s, c, _ := bettingRoom(t, [3]int64{100, 100, 100})
	current := gameState(t, c[0], s.URL)
	oldHand := current
	handCopy := *current.Hand
	oldHand.Hand = &handCopy
	oldHand.Hand.ID--
	status, v := gameCommand(t, c[0], s.URL, "old-hand", "check", oldHand)
	if status != 409 || v.Error != "stale_turn" || v.Version != current.Version {
		t.Fatal("old hand accepted")
	}
	oldTurn := current
	turnCopy := *current.Hand
	oldTurn.Hand = &turnCopy
	oldTurn.Hand.Turn--
	status, v = gameCommand(t, c[0], s.URL, "old-turn", "check", oldTurn)
	if status != 409 || v.Error != "stale_turn" || v.Version != current.Version {
		t.Fatal("old turn accepted")
	}
	data, _ := json.Marshal(map[string]any{"requestID": "forge", "version": current.Version, "action": "check", "chips": 10000, "pageID": pageID(c[0]), "control": current.Control, "handID": current.Hand.ID, "turnID": current.Hand.Turn})
	response, e := c[0].Post(s.URL+"/api/command", "application/json", strings.NewReader(string(data)))
	if e != nil {
		t.Fatal(e)
	}
	response.Body.Close()
	if response.StatusCode != 400 || gameState(t, c[0], s.URL).Version != current.Version {
		t.Fatal("forged wallet accepted")
	}
}
