package poker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type gameView struct {
	Version int64         `json:"version"`
	You     string        `json:"you"`
	Host    string        `json:"host"`
	Error   string        `json:"error"`
	Control int64         `json:"control"`
	Seats   []*playerView `json:"seats"`
	Bot     playerView    `json:"bot"`
	Hand    *struct {
		ID       int64    `json:"id"`
		Turn     int64    `json:"turn"`
		Actor    string   `json:"actor"`
		Stage    string   `json:"stage"`
		Pot      int64    `json:"pot"`
		Target   int64    `json:"target"`
		Legal    []string `json:"legal"`
		Deadline int64    `json:"deadline"`
		Board    []Card   `json:"board"`
		Players  []struct {
			ID       string    `json:"id"`
			Seat     int       `json:"seat"`
			Hole     []Card    `json:"hole"`
			Invested int64     `json:"invested"`
			Street   int64     `json:"street"`
			Folded   bool      `json:"folded"`
			AllIn    bool      `json:"allIn"`
			Won      int64     `json:"won"`
			Balance  int64     `json:"balance"`
			Strength *Strength `json:"strength"`
		} `json:"players"`
	} `json:"hand"`
}

func fixedDeck(prefix []Card) func() ([]Card, error) {
	return func() ([]Card, error) {
		cards := append([]Card{}, prefix...)
		used := map[Card]bool{}
		for _, c := range prefix {
			used[c] = true
		}
		for rank := 2; rank <= 14; rank++ {
			for suit := 0; suit < 4; suit++ {
				c := Card{rank, suit}
				if !used[c] {
					cards = append(cards, c)
				}
			}
		}
		return cards, nil
	}
}
func fixtureServer(t *testing.T, prefix []Card) (*App, *httptest.Server) {
	t.Helper()
	app, e := NewWithOptions(Options{Deck: fixedDeck(prefix)})
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	t.Cleanup(func() { app.Close(); server.Close() })
	return app, server
}
func TestAC09InvalidStartsPreserveState(t *testing.T) {
	server := testServer(t)
	one, two := browser(t), browser(t)
	v := getRoom(t, one, server.URL)
	join(t, one, server.URL, "one", v.Version)
	v = getRoom(t, two, server.URL)
	join(t, two, server.URL, "two", v.Version)
	before := gameState(t, two, server.URL)
	status, rejected := gameCommand(t, two, server.URL, "not-host", "start", before)
	if status != 409 || rejected.Error != "host_only" || rejected.Version != before.Version {
		t.Fatalf("non-host: %d %+v", status, rejected)
	}
	status, started := gameCommand(t, one, server.URL, "host", "start", gameState(t, one, server.URL))
	if status != 200 {
		t.Fatal(status)
	}
	status, rejected = gameCommand(t, one, server.URL, "again", "start", started)
	if status != 409 || rejected.Error != "hand_active" || rejected.Version != started.Version || rejected.Hand.Pot != started.Hand.Pot || rejected.Hand.ID != started.Hand.ID {
		t.Fatalf("active restart: %+v", rejected)
	}
}
func TestAC22OnlyTiedWinnersShareOddPot(t *testing.T) {
	// 两名赢家共用红桃同花顺；真人1的额外黑桃高牌不能超越顺子同花。
	prefix := []Card{{2, 0}, {3, 0}, {2, 1}, {3, 1}, {2, 3}, {3, 3}, {10, 2}, {11, 2}, {12, 2}, {13, 2}, {14, 2}}
	app, server := fixtureServer(t, prefix)
	one, two := browser(t), browser(t)
	v := getRoom(t, one, server.URL)
	join(t, one, server.URL, "one", v.Version)
	v = getRoom(t, two, server.URL)
	join(t, two, server.URL, "two", v.Version)
	_, started := gameCommand(t, one, server.URL, "start", "start", gameState(t, one, server.URL))
	// 非公开fixture只预置已确认的结算输入，所有断言使用HTTP视图。
	app.mu.Lock()
	app.state.Hand.Pot = 21
	app.state.Hand.Players[0].Folded = true
	app.state.Hand.Players[0].acted = true
	app.state.Hand.Actor = 1
	app.mu.Unlock()
	current := gameState(t, two, server.URL)
	for i := 0; i < 4; i++ {
		status, next := gameCommand(t, two, server.URL, string(rune('a'+i)), "check", current)
		if status != 200 {
			t.Fatalf("check %d %+v", status, next)
		}
		current = next
	}
	if current.Hand.Players[0].Won != 0 || current.Hand.Players[1].Won != 11 || current.Hand.Players[2].Won != 10 || current.Hand.Pot != 0 {
		t.Fatalf("odd split: %+v", current.Hand)
	}
	if current.Hand.Players[0].Strength != nil || len(current.Hand.Players[0].Hole) != 0 {
		t.Fatal("folded hole cards leaked")
	}
	status, retry := gameCommand(t, one, server.URL, "start", "start", gameView{Version: started.Version - 1, Control: started.Control})
	if status != 200 || retry.Hand.ID != started.Hand.ID {
		t.Fatalf("cached start: %d %+v", status, retry)
	}
	if gameState(t, two, server.URL).Seats[1].Chips != 110 {
		t.Fatal("award repeated")
	}
}

func gameState(t *testing.T, c *http.Client, url string) gameView {
	t.Helper()
	r, e := c.Get(url + "/api/state")
	if e != nil {
		t.Fatal(e)
	}
	defer r.Body.Close()
	var v gameView
	if e = json.NewDecoder(r.Body).Decode(&v); e != nil {
		t.Fatal(e)
	}
	return v
}
func gameCommand(t *testing.T, c *http.Client, url, id, action string, v gameView) (int, gameView) {
	t.Helper()
	body := map[string]any{"requestID": id, "version": v.Version, "action": action, "pageID": pageID(c), "control": v.Control}
	if v.Hand != nil {
		body["handID"] = v.Hand.ID
		body["turnID"] = v.Hand.Turn
	}
	data, _ := json.Marshal(body)
	r, e := c.Post(url+"/api/command", "application/json", strings.NewReader(string(data)))
	if e != nil {
		t.Fatal(e)
	}
	defer r.Body.Close()
	var result gameView
	if e = json.NewDecoder(r.Body).Decode(&result); e != nil {
		t.Fatal(e)
	}
	return r.StatusCode, result
}
func TestAC08And13HostStartsAndChecksToShowdown(t *testing.T) {
	for _, humans := range []int{1, 2} {
		t.Run(string(rune('0'+humans)), func(t *testing.T) {
			server := testServer(t)
			first := browser(t)
			initial := getRoom(t, first, server.URL)
			status, _ := join(t, first, server.URL, "join", initial.Version)
			if status != 200 {
				t.Fatal(status)
			}
			second := browser(t)
			if humans == 2 {
				v := getRoom(t, second, server.URL)
				status, _ = join(t, second, server.URL, "join", v.Version)
				if status != 200 {
					t.Fatal(status)
				}
			}
			before := gameState(t, first, server.URL)
			status, v := gameCommand(t, first, server.URL, "start", "start", before)
			if status != 200 || v.Hand == nil {
				t.Fatalf("start: %d %+v", status, v)
			}
			if v.Hand.Pot != int64(humans+1) || v.Seats[0].Chips != 99 || v.Bot.Chips != 99 || len(v.Hand.Board) != 0 {
				t.Fatalf("ante: %+v", v)
			}
			if len(v.Hand.Players) != humans+1 || len(v.Hand.Players[0].Hole) != 2 || len(v.Hand.Players[len(v.Hand.Players)-1].Hole) != 0 {
				t.Fatalf("private cards: %+v", v.Hand)
			}
			status, retry := gameCommand(t, first, server.URL, "start", "start", before)
			if status != 200 || retry.Version != v.Version {
				t.Fatalf("start retry: %d %+v", status, retry)
			}
			for street, want := range []int{3, 4, 5, 5} {
				status, v = gameCommand(t, first, server.URL, string(rune('a'+street)), "check", v)
				if status != 200 {
					t.Fatalf("check: %d %+v", status, v)
				}
				if humans == 2 {
					other := gameState(t, second, server.URL)
					status, _ = gameCommand(t, second, server.URL, string(rune('a'+street)), "check", other)
					v = gameState(t, first, server.URL)
					if status != 200 {
						t.Fatal(status)
					}
				}
				if len(v.Hand.Board) != want {
					t.Fatalf("street %d board %d", street, len(v.Hand.Board))
				}
			}
			if v.Hand.Stage != "finished" || v.Hand.Pot != 0 {
				t.Fatalf("not settled: %+v", v.Hand)
			}
			won := int64(0)
			seen := map[Card]bool{}
			for _, card := range v.Hand.Board {
				if seen[card] {
					t.Fatal("AC08 duplicate public card")
				}
				seen[card] = true
			}
			for _, p := range v.Hand.Players {
				won += p.Won
				if len(p.Hole) != 2 || p.Strength == nil {
					t.Fatalf("showdown visibility: %+v", p)
				}
				for _, card := range p.Hole {
					if seen[card] {
						t.Fatal("AC08 duplicate dealt card")
					}
					seen[card] = true
				}
			}
			if won != int64(humans+1) {
				t.Fatalf("awards %d", won)
			}
			saved := gameState(t, first, server.URL)
			if saved.Version != v.Version {
				t.Fatal("settlement repeated")
			}
		})
	}
}
