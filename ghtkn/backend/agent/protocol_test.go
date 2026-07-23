package agent_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// TestSecretBytes_wireFormat verifies SecretBytes marshals to and decodes from a plain
// JSON string (not base64), so the passphrase wire format is unchanged and a
// pre-versioning client that sends a plain string stays compatible.
func TestSecretBytes_wireFormat(t *testing.T) {
	t.Parallel()
	b, err := json.Marshal(&agent.Request{Command: agent.CommandUnlock, Passphrase: agent.SecretBytes("s3cret")})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"passphrase":"s3cret"`) {
		t.Fatalf("passphrase must be a plain JSON string, got %s", b)
	}

	var req agent.Request
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
	b, err := json.Marshal(&agent.Request{Command: agent.CommandStatus})
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
	s := agent.SecretBytes("abc")
	s.Zero()
	for i, b := range s {
		if b != 0 {
			t.Fatalf("byte %d was not zeroed: %d", i, b)
		}
	}
}

// TestSecretBytes_redacted verifies the passphrase never appears when a value is
// formatted, so a stray %v/%+v/%s/%#v (e.g. a future debug log) cannot leak it. The
// underlying []byte would otherwise print as its raw byte values.
func TestSecretBytes_redacted(t *testing.T) {
	t.Parallel()
	req := &agent.Request{Command: agent.CommandUnlock, Passphrase: agent.SecretBytes("s3cret")}
	for _, verb := range []string{"%v", "%+v", "%s", "%#v"} {
		got := fmt.Sprintf(verb, req.Passphrase)
		if got != "REDACTED" {
			t.Errorf("SecretBytes formatted with %s = %q, want REDACTED", verb, got)
		}
		if line := fmt.Sprintf(verb, req); strings.Contains(line, "s3cret") {
			t.Errorf("formatting a Request with %s leaked the passphrase: %s", verb, line)
		}
	}
}
