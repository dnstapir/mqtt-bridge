package keys

import (
	"crypto/ed25519"

	"path/filepath"
	"testing"

	"github.com/dnstapir/mqtt-bridge/inject/fake"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
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
	keyGenerated, err := GenerateSignKey(keyfile, "tmp-key-utest-keys")
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

	alg, found := keyGenerated.Algorithm()
	if !found {
		t.Fatalf("Algorithm not found")
	}

	if alg.String() != jwa.Ed25519().String() {
		t.Fatalf("Unexpected algorithm %s for keytype %s", alg.String(), keyGenerated.KeyType().String())
	}
}

func TestAlgEd25519NoHeader(t *testing.T) {
	setup()

	_, keyRaw, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("%s", err)
	}

	keyJWK, err := jwk.Import[jwk.Key](keyRaw)
	if err != nil {
		t.Fatalf("%s", err)
	}

	data := []byte("hello")

	_, err = Sign(data, keyJWK)
	if err == nil {
		t.Fatalf("Data was signed even though key had no \"alg\" header")
	}
}
