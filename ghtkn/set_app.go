package ghtkn

import (
	"context"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/api"
)

type InputSetApp = api.InputSetApp

func (c *Client) SetApp(ctx context.Context, logger *slog.Logger, input *InputSetApp) error {
	return c.tm.SetApp(ctx, logger, input)
}
