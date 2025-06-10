package fake

import (
    "errors"
)

type FakeNatsClient struct {
}

func (nc *FakeNatsClient) Subscribe(subject string) (<-chan []byte, error) {
    return nil, errors.New("not implemented")
}

func (nc *FakeNatsClient) StartPublishing(subject string) (chan<- []byte, error) {
    return nil, errors.New("not implemented")
}
