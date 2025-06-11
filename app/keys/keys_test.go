package keys

import (
    "path/filepath"
    "testing"

	"github.com/dnstapir/mqtt-bridge/inject/fake"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

func setup() {
    err := SetLogger(fake.Logger())
    if err != nil {
        panic(err)
    }
}

func TestGenerateSignKey(t *testing.T) {
    setup()

    workdir := t.TempDir()
    keyfile := filepath.Join(workdir, "testkey.json")
    keyGenerated, err := GenerateSignKey(keyfile)
    if err != nil {
        panic(err)
    }

    keyRead, err := GetSignKey(keyfile)
    if err != nil {
        panic(err)
    }

    if !jwk.Equal(keyGenerated, keyRead) {
        t.Fatalf("Keys have different thumbprints")
    }
}
