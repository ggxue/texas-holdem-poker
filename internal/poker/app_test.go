package poker_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"texas-poker/internal/poker"
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
}

func testDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL required: run scripts/start-test-postgres.ps1 first")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close(context.Background()) })
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatal(err)
	}
	name := "poker_test_" + hex.EncodeToString(suffix[:])
	quoted := pgx.Identifier{name}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := admin.Exec(ctx, "DROP DATABASE "+quoted+" WITH (FORCE)"); err != nil {
			t.Error(err)
		}
	})
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.Database = name
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	if err := db.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	return db
}

func testServer(t *testing.T, db *pgxpool.Pool) *httptest.Server {
	t.Helper()
	app, err := poker.New(context.Background(), db)
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
	data, err := json.Marshal(map[string]any{"requestID": requestID, "version": version, "action": "join"})
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

func TestAC07TwoHumansEnterAndThirdHumanIsRejected(t *testing.T) {
	server := testServer(t, testDatabase(t))
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

func TestAC25CookiePreservesThirtySevenChipsAndMissingCookieCreatesNewIdentity(t *testing.T) {
	for _, kind := range []string{"different-browser", "cleared-cookie"} {
		t.Run(kind, func(t *testing.T) {
			db := testDatabase(t)
			server := testServer(t, db)
			client := browser(t)
			initial := getRoom(t, client, server.URL)
			status, joined := join(t, client, server.URL, "initial", initial.Version)
			if status != http.StatusOK || joined.Seats[0] == nil || joined.Seats[0].Chips != 100 {
				t.Fatalf("initial wallet: %d %+v", status, joined)
			}
			// Offline fixture permitted by AC25; observe the balance through HTTP.
			if _, err := db.Exec(context.Background(), `UPDATE poker_room SET state = jsonb_set(state, ARRAY['accounts', $1, 'chips'], '37'::jsonb) WHERE id = 1`, initial.You); err != nil {
				t.Fatal(err)
			}
			client.CloseIdleConnections()
			reopened := browser(t)
			reopened.Jar = client.Jar // Closing the page preserves its browser Cookie.
			returned := getRoom(t, reopened, server.URL)
			status, returned = join(t, reopened, server.URL, "reopen", returned.Version)
			if status != http.StatusOK || returned.You != initial.You || returned.Seats[0] == nil || returned.Seats[0].Chips != 37 {
				t.Fatalf("same Cookie did not retain 37: %d %+v", status, returned)
			}
			other := browser(t)
			if kind == "cleared-cookie" {
				other = reopened
				jar, err := cookiejar.New(nil)
				if err != nil {
					t.Fatal(err)
				}
				other.Jar = jar
			}
			newIdentity := getRoom(t, other, server.URL)
			status, both := join(t, other, server.URL, "new-identity", newIdentity.Version)
			if status != http.StatusOK || both.Seats[0] == nil || both.Seats[1] == nil || both.Seats[0].Chips != 37 || both.Seats[1].Chips != 100 || both.Seats[1].ID == initial.You {
				t.Fatalf("new identity changed old funds: %d %+v", status, both)
			}
		})
	}
}
