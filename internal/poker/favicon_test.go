package poker

import (
	"encoding/xml"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAAC18LocalSpadeFaviconResourceAndPageDeclaration(t *testing.T) {
	s := testServer(t)
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		req, err := http.NewRequest(method, s.URL+"/favicon.svg", nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != 200 || !strings.HasPrefix(response.Header.Get("Content-Type"), "image/svg+xml") || response.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("%s local favicon must be available as SVG: status=%d headers=%v", method, response.StatusCode, response.Header)
		}
		if method == http.MethodHead {
			if len(body) != 0 {
				t.Fatal("HEAD returned a body")
			}
			continue
		}
		var svg struct {
			XMLName xml.Name
			ViewBox string `xml:"viewBox,attr"`
		}
		if err := xml.Unmarshal(body, &svg); err != nil || svg.XMLName.Local != "svg" || svg.ViewBox == "" {
			t.Fatalf("favicon must be a valid self-contained SVG: %s (%v)", body, err)
		}
	}
	response, err := http.Get(s.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	page, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), `<link rel="icon" type="image/svg+xml" href="/favicon.svg">`) {
		t.Fatal("page must declare its local SVG favicon")
	}
	for _, input := range []struct {
		method, path string
		status       int
	}{{"POST", "/favicon.svg", 405}, {"GET", "/favicon.svg/other", 404}, {"GET", "/web/favicon.svg", 404}} {
		req, err := http.NewRequest(input.method, s.URL+input.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		rejected, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		rejected.Body.Close()
		if rejected.StatusCode != input.status {
			t.Fatalf("%s %s: %d want %d", input.method, input.path, rejected.StatusCode, input.status)
		}
	}
}
