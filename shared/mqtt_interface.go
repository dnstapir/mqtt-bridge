package shared

type MqttIF interface {
	Connect() error
	Subscribe(string) (<-chan MqttData, error)
	StartPublishing(string, bool) (chan<- []byte, error)
	CheckConnection() bool
	Stop()
}

type MqttData struct {
	Topic   string
	Payload []byte
}
