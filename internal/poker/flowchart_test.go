package poker

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFlowchartImageIsAvailableFromGameService(t *testing.T) {
	s := testServer(t)
	viewer, err := http.Get(s.URL + "/doc")
	if err != nil {
		t.Fatal(err)
	}
	viewerBody, err := io.ReadAll(viewer.Body)
	viewer.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if viewer.StatusCode != http.StatusOK || !strings.HasPrefix(viewer.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("flowchart viewer must be available as HTML: status=%d headers=%v", viewer.StatusCode, viewer.Header)
	}
	if !strings.Contains(string(viewerBody), `src="/doc/sequence-diagram-image.png"`) ||
		!strings.Contains(string(viewerBody), `href="/doc/viewer.css"`) ||
		!strings.Contains(string(viewerBody), `src="/doc/viewer.js"`) {
		t.Fatal("flowchart viewer must center the embedded flowchart image")
	}
	style, err := http.Get(s.URL + "/doc/viewer.css")
	if err != nil {
		t.Fatal(err)
	}
	styleBody, err := io.ReadAll(style.Body)
	style.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if style.StatusCode != http.StatusOK || !strings.Contains(string(styleBody), "width: var(--diagram-width, 100vw)") || !strings.Contains(string(styleBody), "height: auto") {
		t.Fatalf("flowchart viewer stylesheet must fit image to viewport width without stretching: status=%d", style.StatusCode)
	}
	script, err := http.Get(s.URL + "/doc/viewer.js")
	if err != nil {
		t.Fatal(err)
	}
	scriptBody, err := io.ReadAll(script.Body)
	script.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if script.StatusCode != http.StatusOK || !strings.Contains(string(scriptBody), "zoom + 25") || !strings.Contains(string(scriptBody), "zoom - 25") {
		t.Fatalf("flowchart viewer must provide zoom controls: status=%d", script.StatusCode)
	}

	redirectClient := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	for _, imageURL := range []string{"/doc/sequence-diagram.png", "/doc/texas-poker-flow.png", "/docs/"} {
		response, err := redirectClient.Get(s.URL + imageURL)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusTemporaryRedirect || response.Header.Get("Location") != "/doc" {
			t.Fatalf("%s must open the centered viewer: status=%d location=%q", imageURL, response.StatusCode, response.Header.Get("Location"))
		}
	}

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		req, err := http.NewRequest(method, s.URL+"/doc/sequence-diagram-image.png", nil)
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
		if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "image/png") {
			t.Fatalf("%s flowchart image must be available as PNG: status=%d headers=%v", method, response.StatusCode, response.Header)
		}
		if method == http.MethodHead {
			if len(body) != 0 {
				t.Fatal("HEAD returned a body")
			}
			continue
		}
		if len(body) < 8 || string(body[:8]) != "\x89PNG\r\n\x1a\n" {
			t.Fatal("flowchart response is not a PNG image")
		}
	}

	for _, input := range []struct {
		method, path string
		status       int
	}{
		{http.MethodPost, "/doc/sequence-diagram.png", http.StatusMethodNotAllowed},
		{http.MethodPost, "/doc/sequence-diagram-image.png", http.StatusMethodNotAllowed},
		{http.MethodGet, "/docs/unknown.png", http.StatusNotFound},
	} {
		req, err := http.NewRequest(input.method, s.URL+input.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != input.status {
			t.Fatalf("%s %s: %d want %d", input.method, input.path, response.StatusCode, input.status)
		}
	}
}
