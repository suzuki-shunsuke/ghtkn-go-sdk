package api

import (
	"context"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
)

func (tm *TokenManager) SetApp(ctx context.Context, logger *slog.Logger, input *InputGet) error {
	serviceKey := input.KeyringService
	if serviceKey == "" {
		serviceKey = keyring.DefaultServiceKey
	}
	return tm.input.AppStore.Set()
}
