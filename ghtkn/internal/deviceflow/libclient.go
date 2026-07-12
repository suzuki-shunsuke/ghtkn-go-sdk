package deviceflow

import (
	"context"
	"log/slog"
	"net/http"

	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/go-github-device-flow/deviceflow"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// libDeviceFlow adapts the go-github-device-flow library to the DeviceFlow
// interface. It converts between the library's response types and the SDK's own
// contract types so the library stays out of the public API, and enriches errors
// with the HTTP status code and body that the library returns for inspection.
type libDeviceFlow struct {
	client *deviceflow.Client
}

// newLibDeviceFlow builds a libDeviceFlow whose HTTP client, clock, and ticker
// factory are injected, so production and tests share the same seams the SDK used
// before the library extraction.
func newLibDeviceFlow(httpClient *http.Client) *libDeviceFlow {
	return &libDeviceFlow{
		client: deviceflow.New(&deviceflow.Input{
			HTTPClient: httpClient,
		}),
	}
}

func (l *libDeviceFlow) GetDeviceCode(ctx context.Context, clientID string) (*pubdeviceflow.DeviceCodeResponse, error) {
	deviceCode, resp, body, err := l.client.GetDeviceCode(ctx, clientID)
	if err != nil {
		if resp != nil {
			return nil, slogerr.With(err, //nolint:wrapcheck
				"status_code", resp.StatusCode,
				"body", string(body))
		}
		return nil, err //nolint:wrapcheck
	}
	return &pubdeviceflow.DeviceCodeResponse{
		DeviceCode:      deviceCode.DeviceCode,
		UserCode:        deviceCode.UserCode,
		VerificationURI: deviceCode.VerificationURI,
		ExpiresIn:       deviceCode.ExpiresIn,
		Interval:        deviceCode.Interval,
	}, nil
}

func (l *libDeviceFlow) Poll(ctx context.Context, logger *slog.Logger, clientID string, deviceCode *pubdeviceflow.DeviceCodeResponse) (*deviceflow.AccessToken, error) {
	return l.client.Poll(ctx, logger, clientID, &deviceflow.DeviceCodeResponse{ //nolint:wrapcheck
		DeviceCode:      deviceCode.DeviceCode,
		UserCode:        deviceCode.UserCode,
		VerificationURI: deviceCode.VerificationURI,
		ExpiresIn:       deviceCode.ExpiresIn,
		Interval:        deviceCode.Interval,
	}, nil)
}
