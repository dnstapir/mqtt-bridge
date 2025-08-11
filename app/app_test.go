package app

import (
	"encoding/json"
	"github.com/dnstapir/mqtt-bridge/app/keys"
	"github.com/dnstapir/mqtt-bridge/inject/fake"
	"path/filepath"
	"testing"
)

func TestAppDownBasic(t *testing.T) {
	fakeNats := fake.Nats()
	fakeMqtt := fake.Mqtt()

	application := App{
		Log:     fake.Logger(),
		Nats:    fakeNats,
		Mqtt:    fakeMqtt,
		Nodeman: fake.Nodeman(),
		Bridges: make([]Bridge, 0),
	}

	/* Actual generation of key happens later */
	workdir := t.TempDir()
	keyfile := filepath.Join(workdir, "testkey.json")

	bridge := Bridge{
		Direction:   "down",
		MqttTopic:   "testtopic",
		NatsSubject: "testsubject",
		NatsQueue:   "testqueue",
		Key:         keyfile,
		Schema:      "",
	}

	application.Bridges = append(application.Bridges, bridge)

	err := application.Initialize()
	if err != nil {
		t.Fatalf("Error initializing app: %s", err)
	}

	/*
	 * Generate the key now that the app is initialized (to ensure logger
	 * object for "keys" package has been set, else package won't work)
	 */
	_, err = keys.GenerateSignKey(keyfile, "tmp-key-utest-app")
	if err != nil {
		t.Fatalf("Error generating key: %s", err)
	}

	application.Run()

	in := []byte("{\"foo\": \"bar\"}")
	fakeNats.Inject(in)
	out := fakeMqtt.Eavesdrop()

	err = application.Stop()
	if err != nil {
		t.Fatalf("Error stopping application: %s", err)
	}

	valkey, err := keys.GetValKey(keyfile)
	if err != nil {
		t.Fatalf("Error getting validation key: %s", err)
	}

	checkedIn, err := keys.CheckSignature(out, valkey)
	if err != nil {
		t.Fatalf("Error checking signature: %s", err)
	}

	if len(in) != len(checkedIn) {
		t.Fatalf("Data mismatch, want: '%s', got: '%s'", in, checkedIn)
	}

	for i := range in {
		if in[i] != checkedIn[i] {
			t.Fatalf("Data mismatch [%d], want: '%s', got: '%s'", i, in, checkedIn)
		}
	}
}

func TestAppUpBasic(t *testing.T) {
	fakeNats := fake.Nats()
	fakeMqtt := fake.Mqtt()

	application := App{
		Log:     fake.Logger(),
		Nats:    fakeNats,
		Mqtt:    fakeMqtt,
		Nodeman: fake.Nodeman(),
		Bridges: make([]Bridge, 0),
	}

	/* Actual generation of key happens later */
	workdir := t.TempDir()
	keyfile := filepath.Join(workdir, "testkey.json")

	bridge := Bridge{
		Direction:   "up",
		MqttTopic:   "testtopic",
		NatsSubject: "testsubject",
		NatsQueue:   "testqueue",
		Key:         keyfile,
		Schema:      "",
	}

	application.Bridges = append(application.Bridges, bridge)

	err := application.Initialize()
	if err != nil {
		t.Fatalf("Error initializing app: %s", err)
	}

	/*
	 * Generate the key now that the app is initialized (to ensure logger
	 * object for "keys" package has been set, else package won't work)
	 */
	_, err = keys.GenerateSignKey(keyfile, "tmp-key-utest-app")
	if err != nil {
		t.Fatalf("Error generating key: %s", err)
	}

	application.Run()

	in := []byte("{\"foo\": \"bar\"}")

	signkey, err := keys.GetSignKey(bridge.Key)
	if err != nil {
		t.Fatalf("Error getting signing key: %s", err)
	}

	signedIn, err := keys.Sign(in, signkey)
	if err != nil {
		t.Fatalf("Error signing data: %s", err)
	}

	fakeMqtt.Inject(signedIn)
	out := fakeNats.Eavesdrop()
	if len(in) != len(out) {
		t.Fatalf("Data mismatch, want: '%s', got: '%s'", in, out)
	}

	for i := range in {
		if in[i] != out[i] {
			t.Fatalf("Data mismatch [%d], want: '%s', got: '%s'", i, in, out)
		}
	}
}

func TestAppUpNoKeyInConfig(t *testing.T) {
	fakeNats := fake.Nats()
	fakeMqtt := fake.Mqtt()
	fakeNodeman := fake.Nodeman()

	application := App{
		Log:     fake.Logger(),
		Nats:    fakeNats,
		Mqtt:    fakeMqtt,
		Nodeman: fakeNodeman,
		Bridges: make([]Bridge, 0),
	}

	/* Actual generation of key happens later */
	workdir := t.TempDir()
	keyfile := filepath.Join(workdir, "testkey.json")

	bridge := Bridge{
		Direction:   "up",
		MqttTopic:   "testtopic",
		NatsSubject: "testsubject",
		NatsQueue:   "testqueue",
		Schema:      "",
	}

	application.Bridges = append(application.Bridges, bridge)

	err := application.Initialize()
	if err != nil {
		t.Fatalf("Error initializing app: %s", err)
	}

	/*
	 * Generate the key now that the app is initialized (to ensure logger
	 * object for "keys" package has been set, else package won't work)
	 */
	_, err = keys.GenerateSignKey(keyfile, "tmp-key-utest-app")
	if err != nil {
		t.Fatalf("Error generating key: %s", err)
	}

	/* Get validation part and prepare fake nodeman with it */
	valKey, err := keys.GetValKey(keyfile)
	if err != nil {
		t.Fatalf("Error getting validation key: %s", err)
	}

	valkeyBytes, err := json.Marshal(valKey)
	if err != nil {
		t.Fatalf("Error serializing validation key: %s", err)
	}

	fakeNodeman.PrepareKey(valkeyBytes)

	application.Run()

	in := []byte("{\"foo\": \"bar\"}")

	signkey, err := keys.GetSignKey(keyfile)
	if err != nil {
		t.Fatalf("Error getting signing key: %s", err)
	}

	signedIn, err := keys.Sign(in, signkey)
	if err != nil {
		t.Fatalf("Error signing data: %s", err)
	}

	fakeMqtt.Inject(signedIn)
	out := fakeNats.Eavesdrop()
	if len(in) != len(out) {
		t.Fatalf("Data mismatch, want: '%s', got: '%s'", in, out)
	}

	for i := range in {
		if in[i] != out[i] {
			t.Fatalf("Data mismatch [%d], want: '%s', got: '%s'", i, in, out)
		}
	}
}
