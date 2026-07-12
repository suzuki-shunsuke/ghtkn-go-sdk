package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestSecretBytes_wireFormat verifies SecretBytes marshals to and decodes from a plain
// JSON string (not base64), so the passphrase wire format is unchanged and a
// pre-versioning client that sends a plain string stays compatible.
func TestSecretBytes_wireFormat(t *testing.T) {
	t.Parallel()
	b, err := json.Marshal(&Request{Command: CommandUnlock, Passphrase: SecretBytes("s3cret")})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"passphrase":"s3cret"`) {
		t.Fatalf("passphrase must be a plain JSON string, got %s", b)
	}

	var req Request
	if err := json.Unmarshal([]byte(`{"command":"UNLOCK","passphrase":"s3cret"}`), &req); err != nil {
		t.Fatal(err)
	}
	if string(req.Passphrase) != "s3cret" {
		t.Fatalf("passphrase = %q, want s3cret", req.Passphrase)
	}
}

// TestSecretBytes_omitempty verifies an empty passphrase is omitted from the wire.
func TestSecretBytes_omitempty(t *testing.T) {
	t.Parallel()
	b, err := json.Marshal(&Request{Command: CommandStatus})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "passphrase") {
		t.Fatalf("an empty passphrase must be omitted, got %s", b)
	}
}

// TestSecretBytes_zero verifies Zero scrubs the bytes.
func TestSecretBytes_zero(t *testing.T) {
	t.Parallel()
	s := SecretBytes("abc")
	s.Zero()
	for i, b := range s {
		if b != 0 {
			t.Fatalf("byte %d was not zeroed: %d", i, b)
		}
	}
}
