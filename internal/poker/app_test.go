package poker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type playerView struct {
	ID                string `json:"id"`
	Chips             int64  `json:"chips"`
	DisconnectedUntil int64  `json:"disconnectedUntil"`
	ConnectingUntil   int64  `json:"connectingUntil"`
}

type roomView struct {
	Version int64         `json:"version"`
	You     string        `json:"you"`
	Host    string        `json:"host"`
	Seats   []*playerView `json:"seats"`
	Bot     playerView    `json:"bot"`
	Error   string        `json:"error"`
	Control int64         `json:"control"`
}

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	app, err := NewWithOptions(Options{ActionStart: firstActionStart})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(app)
	t.Cleanup(func() { app.Close(); server.Close() })
	return server
}

func browser(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar, Timeout: 5 * time.Second}
}

func getRoom(t *testing.T, client *http.Client, address string) roomView {
	t.Helper()
	response, err := client.Get(address + "/api/state")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("state status = %d", response.StatusCode)
	}
	var view roomView
	if err := json.NewDecoder(response.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	return view
}

func join(t *testing.T, client *http.Client, address, requestID string, version int64) (int, roomView) {
	t.Helper()
	current := getRoom(t, client, address)
	data, err := json.Marshal(map[string]any{"requestID": requestID, "version": version, "action": "join", "pageID": pageID(client), "control": current.Control})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Post(address+"/api/command", "application/json", strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var view roomView
	if err := json.NewDecoder(response.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode == http.StatusOK {
		gameSocket(t, client, address)
	}
	return response.StatusCode, view

}

func pageID(c *http.Client) string { return fmt.Sprintf("page-%p", c) }

func TestUAC01FiveHumansEnterAndSixthHumanIsRejected(t *testing.T) {
	server := testServer(t)
	clients := make([]*http.Client, 5)
	identities := make([]string, 5)
	for i := range clients {
		clients[i] = browser(t)
		before := getRoom(t, clients[i], server.URL)
		identities[i] = before.You
		status, entered := join(t, clients[i], server.URL, "enter", before.Version)
		if status != http.StatusOK {
			t.Fatalf("human %d join: %d %+v", i+1, status, entered)
		}
		if entered.Seats[i] == nil || entered.Seats[i].ID != before.You || entered.Seats[i].Chips != 100 || entered.Host != identities[0] {
			t.Fatalf("human %d seat/host assignment: %+v", i+1, entered)
		}
	}
	before := getRoom(t, clients[0], server.URL)
	sixth := browser(t)
	sixthInitial := getRoom(t, sixth, server.URL)
	status, full := join(t, sixth, server.URL, "full", sixthInitial.Version)
	if status != http.StatusConflict || full.Error != "room_full" {
		t.Fatalf("sixth join: %d %+v", status, full)
	}
	unchanged := getRoom(t, clients[0], server.URL)
	if unchanged.Version != before.Version || unchanged.Host != identities[0] {
		t.Fatalf("rejected entry changed room: %+v", unchanged)
	}
	for i, id := range identities {
		if unchanged.Seats[i] == nil || unchanged.Seats[i].ID != id {
			t.Fatalf("sixth identity displaced human %d: %+v", i+1, unchanged)
		}
	}
}
func TestAC25ReconnectResetsChipsAndMissingCookieCreatesNewIdentity(t *testing.T) {
	for _, kind := range []string{"same-cookie", "different-browser", "cleared-cookie"} {
		t.Run(kind, func(t *testing.T) {
			app, err := New()
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(app)
			t.Cleanup(func() { app.Close(); server.Close() })
			client := browser(t)
			initial := getRoom(t, client, server.URL)
			status, joined := join(t, client, server.URL, "initial", initial.Version)
			if status != http.StatusOK || joined.Seats[0] == nil || joined.Seats[0].Chips != 100 {
				t.Fatalf("initial wallet: %d %+v", status, joined)
			}
			// Only fixture setup touches memory; all assertions use the public HTTP view.
			app.mu.Lock()
			p := app.state.Accounts[initial.You]
			p.Chips = 37
			app.state.Accounts[initial.You] = p
			app.mu.Unlock()
			client.CloseIdleConnections()
			returning := browser(t)
			if kind == "same-cookie" {
				returning.Jar = client.Jar
			}
			if kind == "cleared-cookie" {
				returning = client
				returning.Jar, err = cookiejar.New(nil)
				if err != nil {
					t.Fatal(err)
				}
			}
			current := getRoom(t, returning, server.URL)
			status, result := join(t, returning, server.URL, "return", current.Version)
			if status != http.StatusOK {
				t.Fatalf("return: %d %+v", status, result)
			}
			if kind == "same-cookie" {
				if result.You != initial.You || result.Seats[0] == nil || result.Seats[0].Chips != 100 || result.Seats[1] != nil || result.Host != initial.You {
					t.Fatalf("reconnect did not reset to 100 in same seat: %+v", result)
				}
			} else if result.You == initial.You || result.Seats[1] == nil || result.Seats[1].Chips != 100 || result.Seats[0] == nil || result.Seats[0].Chips != 37 {
				t.Fatalf("new identity transferred or reset another wallet: %+v", result)
			}
		})
	}
}
