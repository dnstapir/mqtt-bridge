package shared

type MqttIF interface {
	Connect() error
	Subscribe(string) (<-chan []byte, error)
	StartPublishing(string) (chan<- []byte, error)
	CheckConnection() bool
	Stop()
}
