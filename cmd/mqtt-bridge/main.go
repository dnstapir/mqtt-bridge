package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pelletier/go-toml/v2"

	"github.com/dnstapir/mqtt-bridge/setup"
)

const c_ENVVAR_OVERRIDE_MQTT_URL = "DNSTAPIR_BRIDGE_MQTT_URL"
const c_ENVVAR_OVERRIDE_NATS_URL = "DNSTAPIR_BRIDGE_NATS_URL"

func main() {
	var configFile string
	var appConf setup.AppConf

	flag.StringVar(&configFile,
		"config-file",
		"config.toml",
		"Bridge config file",
	)

	flag.Parse()

	file, err := os.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	err = toml.Unmarshal(file, &appConf)
	if err != nil {
		panic(err)
	}

	envMqttUrl, overrideMqttUrl := os.LookupEnv(c_ENVVAR_OVERRIDE_MQTT_URL)
	if overrideMqttUrl {
		appConf.MqttUrl = envMqttUrl
	}

	envNatsUrl, overrideNatsUrl := os.LookupEnv(c_ENVVAR_OVERRIDE_NATS_URL)
	if overrideNatsUrl {
		appConf.NatsUrl = envNatsUrl
	}

	application, err := setup.BuildApp(appConf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building application: '%s', exiting...\n", err)
		os.Exit(-1)
	}

	sigChan := make(chan os.Signal, 1)
	defer close(sigChan)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	err = application.Initialize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing application: '%s', exiting...\n", err)
		os.Exit(-1)
	}

	done := application.Run()

	select {
	case s := <-sigChan:
		fmt.Fprintf(os.Stderr, "Got signal '%s', exiting...\n", s)
	case err := <-done:
		if err != nil {
			fmt.Fprintf(os.Stderr, "App exited with error: '%s'\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Done!\n")
		}
	}

	err = application.Stop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error stopping app: '%s'\n", err)
		os.Exit(-1)
	}

	os.Exit(0)
}
