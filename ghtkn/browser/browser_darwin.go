package browser

import "context"

// cmds returns the list of commands to try for opening a browser on macOS.
// On macOS, the standard command is "open".
func cmds() []string {
	return []string{"open"}
}

// openB opens a browser on macOS using the appropriate system command.
// It delegates to runCmd with the platform-specific commands.
func (b *Browser) openB(ctx context.Context, url string) error {
	return b.runCmd(ctx, url)
}

// availableB reports whether a browser command is available on macOS.
func (b *Browser) availableB() bool {
	return b.hasCmd()
}
