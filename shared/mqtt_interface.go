package shared

type MqttIF interface {
	Connect() error
	Subscribe(string) (<-chan MqttData, error)
	StartPublishing(string) (chan<- []byte, error)
	CheckConnection() bool
	Stop()
}

type MqttData struct {
    Topic string
    Payload []byte
}
