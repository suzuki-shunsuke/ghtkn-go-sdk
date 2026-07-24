package deviceflow

import (
	"io"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow/ui"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
)

func newMockInput() *Input {
	return &Input{
		Stderr:        io.Discard,
		Logger:        log.NewLogger(),
		OnetimeCodeUI: ui.New(nil),
	}
}
