package deviceflow

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
)

type mockBrowser struct {
	err error
}

func newMockBrowser(err error) pubdeviceflow.Browser {
	return &mockBrowser{err: err}
}

func (b *mockBrowser) Open(_ context.Context, _ *slog.Logger, _ string) error {
	return b.err
}

func newMockInput() *Input {
	return &Input{
		Now:           time.Now,
		Stderr:        io.Discard,
		Browser:       newMockBrowser(nil),
		Logger:        log.NewLogger(),
		OnetimeCodeUI: newOnetimeCodeUI(strings.NewReader("\n"), io.Discard, &mockWaiter{}),
	}
}

// newTestDeviceFlow builds a real library-backed DeviceFlow whose HTTP requests are
// redirected to the given test server and whose polling ticks near-instantly, so the
// device flow can be driven end-to-end in tests without hitting github.com or waiting
// for the real polling interval.
func newTestDeviceFlow(server *httptest.Server, now func() time.Time) DeviceFlow {
	return newLibDeviceFlow(
		&http.Client{Transport: &testTransport{server: server, base: http.DefaultTransport}},
		now,
		func(_ time.Duration) *time.Ticker { return time.NewTicker(time.Millisecond) },
	)
}

type mockWaiter struct {
	err error
}

func (w *mockWaiter) Wait(ctx context.Context, duration time.Duration) error {
	return w.err
}

// testTransport is a custom transport that redirects GitHub API requests to our test server
type testTransport struct {
	server *httptest.Server
	base   http.RoundTripper
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect GitHub API requests to our test server
	if strings.Contains(req.URL.Host, "github.com") {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(t.server.URL, "http://")
	}
	return t.base.RoundTrip(req) //nolint:wrapcheck
}
