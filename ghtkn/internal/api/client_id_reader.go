package api

import (
	"fmt"
	"io"
	"syscall"

	"github.com/charmbracelet/x/term"
)

type passwordReader struct {
	stdout  io.Writer
	message string
}

func NewPasswordReader(stdout io.Writer, message string) *passwordReader {
	return &passwordReader{
		stdout:  stdout,
		message: message,
	}
}

func (p *passwordReader) Read() ([]byte, error) {
	fmt.Fprint(p.stdout, p.message)
	b, err := term.ReadPassword(uintptr(syscall.Stdin))
	fmt.Fprintln(p.stdout, "")
	if err != nil {
		return nil, fmt.Errorf("read a secret from terminal: %w", err)
	}
	return b, nil
}
