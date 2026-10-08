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
	ID    string `json:"id"`
	Chips int64  `json:"chips"`
}

type roomView struct {
	Version int64          `json:"version"`
	You     string         `json:"you"`
	Host    string         `json:"host"`
	Seats   [2]*playerView `json:"seats"`
	Bot     playerView     `json:"bot"`
	Error   string         `json:"error"`
	Control int64          `json:"control"`
}

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	app, err := New()
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
	return response.StatusCode, view
}

func pageID(c *http.Client) string { return fmt.Sprintf("page-%p", c) }

func TestAC07TwoHumansEnterAndThirdHumanIsRejected(t *testing.T) {
	server := testServer(t)
	first, second, third := browser(t), browser(t), browser(t)
	initial := getRoom(t, first, server.URL)
	status, one := join(t, first, server.URL, "join-one", initial.Version)
	if status != http.StatusOK {
		t.Fatalf("first join: %d %+v", status, one)
	}
	secondInitial := getRoom(t, second, server.URL)
	status, two := join(t, second, server.URL, "join-two", secondInitial.Version)
	if status != http.StatusOK {
		t.Fatalf("second join: %d %+v", status, two)
	}
	if two.Seats[0] == nil || two.Seats[0].ID != initial.You || two.Seats[1] == nil || two.Seats[1].ID != secondInitial.You || two.Host != initial.You {
		t.Fatalf("seat/host assignment: %+v", two)
	}
	thirdInitial := getRoom(t, third, server.URL)
	status, full := join(t, third, server.URL, "join-three", thirdInitial.Version)
	if status != http.StatusConflict || full.Error != "room_full" {
		t.Fatalf("third join: %d %+v", status, full)
	}
	unchanged := getRoom(t, first, server.URL)
	if unchanged.Seats[0] == nil || unchanged.Seats[1] == nil || unchanged.Seats[0].ID != initial.You || unchanged.Seats[1].ID != secondInitial.You || unchanged.Host != initial.You {
		t.Fatalf("third identity displaced occupants: %+v", unchanged)
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
