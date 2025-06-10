package fake

import (
    "errors"
)

type FakeMqttClient struct {
}


func (nc *FakeMqttClient) Subscribe(topic string) (<-chan []byte, error) {
    return nil, errors.New("not implemented")
}

func (nc *FakeMqttClient) StartPublishing(topic string) (chan<- []byte, error) {
    return nil, errors.New("not implemented")
}
