package browser

import (
	"context"

	"golang.org/x/sys/windows"
)

// cmds returns the list of commands to try for opening a browser on Windows.
// On Windows, ShellExecute is used directly, so no commands are needed.
func cmds() []string {
	return nil
}

// openB opens a browser on Windows using the ShellExecute API.
// It directly calls the Windows API to open the default browser.
func (b *Browser) openB(_ context.Context, url string) error {
	return windows.ShellExecute(0, nil, windows.StringToUTF16Ptr(url), nil, nil, windows.SW_SHOWNORMAL)
}

// availableB reports whether a browser can be opened on Windows.
// ShellExecute is always available, so it returns true.
func (b *Browser) availableB() bool {
	return true
}
