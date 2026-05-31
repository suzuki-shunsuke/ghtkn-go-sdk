package ghtkn

import (
	"context"
	"log/slog"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
)

// deviceCodeUIAdapter adapts a public DeviceCodeUI to the internal deviceflow.DeviceCodeUI
// interface, converting the internal device code response to its public representation
// before delegating to the user-provided UI.
type deviceCodeUIAdapter struct {
	ui DeviceCodeUI
}

func (a *deviceCodeUIAdapter) Show(ctx context.Context, logger *slog.Logger, deviceCode *deviceflow.DeviceCodeResponse, expirationDate time.Time) error {
	return a.ui.Show(ctx, logger, fromDeviceCodeResponse(deviceCode), expirationDate) //nolint:wrapcheck
}
