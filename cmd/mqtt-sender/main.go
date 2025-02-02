package main


import (
    "flag"
    "tapir-core-mqtt-sender/app"
)

var (
    fDebug bool
)

func init() {
    flag.BoolVar(&fDebug, "debug", false, "Enable debug logs")
}

func main() {
    flag.Parse()

    app, err := app.Create(
        app.Debug(fDebug),
    )

    if err != nil {
        panic(err)
    }

    app.Run()
}
