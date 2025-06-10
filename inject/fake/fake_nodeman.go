package fake

import (
    "errors"
)

type FakeNodemanClient struct {
}

func (nc *FakeNodemanClient) Subscribe(subject string) (<-chan []byte, error) {
    return nil, errors.New("not implemented")
}

func (nc *FakeNodemanClient) StartPublishing(subject string) (chan<- []byte, error) {
    return nil, errors.New("not implemented")
}
