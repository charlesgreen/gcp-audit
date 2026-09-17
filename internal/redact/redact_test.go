package redact

import (
	"strings"
	"testing"
)

func TestPrivateKeyData(t *testing.T) {
	in := []byte(`{"keys":[{"privateKeyData":"-----BEGIN PRIVATE KEY-----\nabc\n"}]}`)
	out, err := JSON(in)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "BEGIN PRIVATE") {
		t.Fatal("private key leaked")
	}
}

func TestVPNSharedSecret(t *testing.T) {
	in := []byte(`{"vpnTunnels":[{"sharedSecret":"supersecret"}]}`)
	out, err := JSON(in)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "supersecret") {
		t.Fatal("PSK leaked")
	}
}
