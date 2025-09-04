package main

import (
	"fmt"
	//"time"
	"github.com/dnstapir/mqtt-bridge/inject/logging"
	"github.com/dnstapir/mqtt-bridge/inject/mqtt"
)

func main() {
	log := logging.Create(true, false)

	mqttConf := mqtt.Conf{
		Log:     log,
		MqttUrl: "mqtt://127.0.0.1:1883",
	}

	mqttClient, err := mqtt.Create(mqttConf)
	if err != nil {
		panic(err)
	}

	err = mqttClient.Connect()
	if err != nil {
		panic(err)
	}

	ch, err := mqttClient.Subscribe("subdemo_topic")
	for data := range ch {
		fmt.Println(string(data))
	}

	//ch, err := mqttClient.StartPublishing("subdemo_topic")
	//if err != nil {
	//    panic(err)
	//}

	//for data := range 123456789 {
	//    str := fmt.Sprintf("msg: %d", data)
	//    ch <- []byte(str)
	//    time.Sleep(1*time.Second)
	//}
}
