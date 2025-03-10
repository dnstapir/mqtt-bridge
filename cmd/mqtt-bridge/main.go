package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pelletier/go-toml/v2"

	"github.com/dnstapir/mqtt-bridge/app"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	var filename string

	var application app.App

	flag.StringVar(&filename,
		"config-file",
		"config.toml",
		"Bridge config file",
	)

	flag.Parse()

	file, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	err = toml.Unmarshal(file, &application)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Read conf %+v\n", application)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	application.Ctx = ctx

	fmt.Println("###### starting mqtt-bridge...")
	go application.Run()

	s := <-c
	fmt.Println(fmt.Sprintf("###### mqtt-bridge got signal '%s', exiting...", s))
}
