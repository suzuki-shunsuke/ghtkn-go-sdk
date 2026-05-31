package ghtkn

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
)

// fakeDeviceCodeUI records the device code response it receives.
type fakeDeviceCodeUI struct {
	got *DeviceCodeResponse
	err error
}

func (f *fakeDeviceCodeUI) Show(_ context.Context, _ *slog.Logger, deviceCode *DeviceCodeResponse, _ time.Time) error {
	f.got = deviceCode
	return f.err
}

func TestDeviceCodeUIAdapter_Show(t *testing.T) {
	t.Parallel()

	fake := &fakeDeviceCodeUI{}
	adapter := &deviceCodeUIAdapter{ui: fake}

	internal := &deviceflow.DeviceCodeResponse{
		DeviceCode:      "dc",
		UserCode:        "uc",
		VerificationURI: "https://github.com/login/device",
		ExpiresIn:       900,
		Interval:        5,
	}

	if err := adapter.Show(t.Context(), nil, internal, time.Time{}); err != nil {
		t.Fatal(err)
	}

	want := &DeviceCodeResponse{
		DeviceCode:      "dc",
		UserCode:        "uc",
		VerificationURI: "https://github.com/login/device",
		ExpiresIn:       900,
		Interval:        5,
	}
	if diff := cmp.Diff(want, fake.got); diff != "" {
		t.Error(diff)
	}
}
