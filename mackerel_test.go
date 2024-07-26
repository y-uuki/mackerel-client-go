package mackerel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-Api-Key") != "dummy-key" {
			t.Error("X-Api-Key header should contains passed key")
		}

		if h := req.Header.Get("User-Agent"); h != defaultUserAgent {
			t.Errorf("User-Agent should be '%s' but %s", defaultUserAgent, h)
		}
	}))
	defer ts.Close()

	client, _ := NewClientWithOptions("dummy-key", ts.URL, false)

	req, _ := http.NewRequest("GET", client.urlFor("/", nil).String(), nil)
	_, err := client.Request(req)
	if err != nil {
		t.Errorf("request is error %v", err)
	}
}

func Test_requestInternal(t *testing.T) {
	tests := []struct {
		method               string
		body                 io.Reader
		hasContentTypeHeader bool
	}{
		{http.MethodGet, nil, false},
		{http.MethodPost, nil, true},
		{http.MethodPut, nil, true},
		{http.MethodDelete, nil, true},
		{http.MethodGet, strings.NewReader("some"), true},
		{http.MethodPost, strings.NewReader("some"), true},
		{http.MethodPut, strings.NewReader("some"), true},
		{http.MethodDelete, strings.NewReader("some"), true},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s with %v body", test.method, test.body), func(tt *testing.T) {
			// Test server that make requests consistent with Mackerel behavior
			ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				if !test.hasContentTypeHeader && req.Header.Get("Content-Type") == "application/json" {
					t.Error("Content-Type header should not have application/json")
				}
				if test.hasContentTypeHeader && req.Header.Get("Content-Type") != "application/json" {
					t.Error("Content-Type header should have application/json")
				}
				res.Write([]byte(`{"success": true}`)) // nolint
			}))
			defer ts.Close()

			client, _ := NewClientWithOptions("dummy-key", ts.URL, false)
			res, _, err := requestInternal[struct {
				Success bool `json:"success"`
			}](client, test.method, "/", url.Values{}, test.body)
			if err != nil {
				t.Errorf("request is error %v", err)
			}
			if !res.Success {
				t.Errorf("response is invalid %v", res)
			}
		})
	}
}

func Test_requestInternalContext_cancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		time.Sleep(1 * time.Second)
		fmt.Fprint(res, "ok")
	}))
	t.Cleanup(ts.Close)

	client, _ := NewClientWithOptions("dummy-key", ts.URL, false)
	ctx, cancel := context.WithCancelCause(context.Background())
	expectedErr := errors.New("expected error")
	cancel(expectedErr)

	_, _, err := requestInternalContext[struct{}](ctx, client, "GET", "/", nil, nil)
	if cause := context.Cause(ctx); err == nil || !errors.Is(cause, expectedErr) {
		t.Errorf("got %v; want %v", cause, expectedErr)
	}
}

func TestUrlFor(t *testing.T) {
	client, _ := NewClientWithOptions("dummy-key", "https://example.com/with/ignored/path", false)
	expected := "https://example.com/some/super/endpoint"
	if url := client.urlFor("/some/super/endpoint", nil).String(); url != expected {
		t.Errorf("urlFor should be %q but %q", expected, url)
	}

	expected += "?test1=value1&test1=value2&test2=value2"
	params := url.Values{}
	params.Add("test1", "value1")
	params.Add("test1", "value2")
	params.Add("test2", "value2")
	if url := client.urlFor("/some/super/endpoint", params).String(); url != expected {
		t.Errorf("urlFor should be %q but %q", expected, url)
	}
}

func TestBuildReq(t *testing.T) {
	cl := NewClient("dummy-key")
	xVer := "1.0.1"
	xRev := "shasha"
	cl.AdditionalHeaders = http.Header{
		"X-Agent-Version": []string{xVer},
		"X-Revision":      []string{xRev},
	}
	cl.UserAgent = "mackerel-agent"
	req, _ := http.NewRequest("GET", cl.urlFor("/", nil).String(), nil)
	req = cl.buildReq(req)

	if req.Header.Get("X-Api-Key") != "dummy-key" {
		t.Error("X-Api-Key header should contains passed key")
	}
	if h := req.Header.Get("User-Agent"); h != cl.UserAgent {
		t.Errorf("User-Agent should be '%s' but %s", cl.UserAgent, h)
	}
	if h := req.Header.Get("X-Agent-Version"); h != xVer {
		t.Errorf("X-Agent-Version should be '%s' but %s", xVer, h)
	}
	if h := req.Header.Get("X-Revision"); h != xRev {
		t.Errorf("X-Revision should be '%s' but %s", xRev, h)
	}
}

func TestLogger(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("OK")) //nolint
	}))
	defer ts.Close()

	client, _ := NewClientWithOptions("dummy-key", ts.URL, true)
	var buf bytes.Buffer
	client.Logger = log.New(&buf, "<api>", 0)
	req, _ := http.NewRequest("GET", client.urlFor("/", nil).String(), nil)
	_, err := client.Request(req)
	if err != nil {
		t.Errorf("request is error %v", err)
	}
	s := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(s, "<api>") || !strings.HasSuffix(s, "OK") {
		t.Errorf("verbose log should match /<api>.*OK/; but %s", s)
	}
}

type fakeLogger struct {
	w io.Writer
}

func (p *fakeLogger) Tracef(format string, v ...interface{}) {
	fmt.Fprintf(p.w, format, v...)
}
func (p *fakeLogger) Debugf(format string, v ...interface{})   {}
func (p *fakeLogger) Infof(format string, v ...interface{})    {}
func (p *fakeLogger) Warningf(format string, v ...interface{}) {}
func (p *fakeLogger) Errorf(format string, v ...interface{})   {}

func TestPrivateTracef(t *testing.T) {
	var (
		stdbuf bytes.Buffer
		logbuf bytes.Buffer
		pbuf   bytes.Buffer
	)
	log.SetOutput(&stdbuf)
	defer log.SetOutput(os.Stderr)
	oflags := log.Flags()
	defer log.SetFlags(oflags)
	log.SetFlags(0)

	msg := "test\n"
	t.Run("Logger+PrioritizedLogger", func(t *testing.T) {
		var c Client
		c.Logger = log.New(&logbuf, "", 0)
		c.PrioritizedLogger = &fakeLogger{w: &pbuf}
		c.tracef(msg)
		if s := stdbuf.String(); s != "" {
			t.Errorf("tracef(%q): log.Printf(%q); want %q", msg, s, "")
		}
		if s := logbuf.String(); s != msg {
			t.Errorf("tracef(%q): Logger.Printf(%q); want %q", msg, s, msg)
		}
		if s := pbuf.String(); s != msg {
			t.Errorf("tracef(%q): PrioritizedLogger.Tracef(%q); want %q", msg, s, msg)
		}
	})

	stdbuf.Reset()
	logbuf.Reset()
	pbuf.Reset()
	t.Run("Logger", func(t *testing.T) {
		var c Client
		c.Logger = log.New(&logbuf, "", 0)
		c.tracef(msg)
		if s := stdbuf.String(); s != "" {
			t.Errorf("tracef(%q): log.Printf(%q); want %q", msg, s, "")
		}
		if s := logbuf.String(); s != msg {
			t.Errorf("tracef(%q): Logger.Printf(%q); want %q", msg, s, msg)
		}
		if s := pbuf.String(); s != "" {
			t.Errorf("tracef(%q): PrioritizedLogger.Tracef(%q); want %q", msg, s, "")
		}
	})

	stdbuf.Reset()
	logbuf.Reset()
	pbuf.Reset()
	t.Run("PrioritizedLogger", func(t *testing.T) {
		var c Client
		c.PrioritizedLogger = &fakeLogger{w: &pbuf}
		c.tracef(msg)
		if s := stdbuf.String(); s != "" {
			t.Errorf("tracef(%q): log.Printf(%q); want %q", msg, s, "")
		}
		if s := logbuf.String(); s != "" {
			t.Errorf("tracef(%q): Logger.Printf(%q); want %q", msg, s, "")
		}
		if s := pbuf.String(); s != msg {
			t.Errorf("tracef(%q): PrioritizedLogger.Tracef(%q); want %q", msg, s, msg)
		}
	})

	stdbuf.Reset()
	logbuf.Reset()
	pbuf.Reset()
	t.Run("default", func(t *testing.T) {
		var c Client
		c.tracef(msg)
		if s := stdbuf.String(); s != msg {
			t.Errorf("tracef(%q): log.Printf(%q); want %q", msg, s, msg)
		}
		if s := logbuf.String(); s != "" {
			t.Errorf("tracef(%q): Logger.Printf(%q); want %q", msg, s, "")
		}
		if s := pbuf.String(); s != "" {
			t.Errorf("tracef(%q): PrioritizedLogger.Tracef(%q); want %q", msg, s, "")
		}
	})
}
