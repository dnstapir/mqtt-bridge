package shared

type NatsIF interface {
	Connect() error
	Subscribe(string, string) (<-chan []byte, error)
	StartPublishing(string, string) (chan<- []byte, error)
    Stop()
}
