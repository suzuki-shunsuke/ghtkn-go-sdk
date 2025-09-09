package browser

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/suzuki-shunsuke/go-exec/goexec"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// ErrNoCommandFound is returned when no suitable command is found to open a browser.
var ErrNoCommandFound = errors.New("no command found to open the browser")

type Browser struct{}

// Open opens the specified URL in the system's default browser.
// It is platform-specific and delegates to the appropriate implementation.
func (b *Browser) Open(ctx context.Context, url string) error {
	return openB(ctx, url)
}

// runCmd attempts to open a URL using available browser commands.
// It tries each command in order until one succeeds or all fail.
// Returns errNoCommandFound if no suitable command is available.
func runCmd(ctx context.Context, url string) error {
	for _, cmd := range cmds() {
		if _, err := exec.LookPath(cmd); err != nil {
			continue
		}
		if err := goexec.Command(ctx, cmd, url).Run(); err != nil {
			return fmt.Errorf("open the browser: %w", slogerr.With(err, "command_to_open_browser", cmd))
		}
		return nil
	}
	return ErrNoCommandFound
}
