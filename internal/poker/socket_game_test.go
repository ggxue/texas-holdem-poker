package poker

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func gameSocket(t *testing.T, c *http.Client, address string) *websocket.Conn {
	t.Helper()
	u, _ := url.Parse(address)
	header := http.Header{}
	for _, cookie := range c.Jar.Cookies(u) {
		header.Add("Cookie", cookie.String())
	}
	ws, _, e := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(address, "http")+"/api/ws", header)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { ws.Close() })
	return ws
}
func socketView(t *testing.T, ws *websocket.Conn, version int64) gameView {
	t.Helper()
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		_, data, e := ws.ReadMessage()
		if e != nil {
			t.Fatal(e)
		}
		var raw map[string]json.RawMessage
		if e = json.Unmarshal(data, &raw); e != nil {
			t.Fatal(e)
		}
		if handData, ok := raw["hand"]; ok && string(handData) != "null" {
			var fields map[string]json.RawMessage
			if e = json.Unmarshal(handData, &fields); e != nil {
				t.Fatal(e)
			}
			for _, key := range []string{"deck", "cursor"} {
				if _, exists := fields[key]; exists {
					t.Fatalf("private field %s leaked in actual message", key)
				}
			}
		}
		var v gameView
		if e = json.Unmarshal(data, &v); e != nil {
			t.Fatal(e)
		}
		if v.Version >= version {
			return v
		}
	}
}
func TestAC31SocketViewsHideOpponentsAndDeck(t *testing.T) {
	server := testServer(t)
	one, two := browser(t), browser(t)
	v := getRoom(t, one, server.URL)
	join(t, one, server.URL, "one", v.Version)
	v = getRoom(t, two, server.URL)
	join(t, two, server.URL, "two", v.Version)
	first, second := gameSocket(t, one, server.URL), gameSocket(t, two, server.URL)
	status, started := gameCommand(t, one, server.URL, "start", "start", gameState(t, one, server.URL))
	if status != 200 {
		t.Fatal(status)
	}
	for i, ws := range []*websocket.Conn{first, second} {
		state := socketView(t, ws, started.Version)
		for seat, p := range state.Hand.Players {
			want := 0
			if seat == i {
				want = 2
			}
			if len(p.Hole) != want || p.Strength != nil {
				t.Fatalf("socket %d leaked seat %d: %+v", i, seat, p)
			}
		}
	}
}
