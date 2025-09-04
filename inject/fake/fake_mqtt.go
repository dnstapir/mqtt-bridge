package fake

type mqtt struct {
	subCh chan []byte
	pubCh chan []byte
}

func Mqtt() *mqtt {
	mqtt := new(mqtt)
	mqtt.subCh = make(chan []byte, 1)
	mqtt.pubCh = make(chan []byte)

	return mqtt
}

func (m *mqtt) Connect() error {
	return nil
}

func (m *mqtt) Subscribe(topic string) (<-chan []byte, error) {
	return m.subCh, nil
}

func (m *mqtt) Stop() {
}

func (m *mqtt) Inject(data []byte) {
	m.subCh <- data
}

func (m *mqtt) StartPublishing(subject string) (chan<- []byte, error) {
	return m.pubCh, nil
}

func (m *mqtt) CheckConnection() bool {
	return true
}

func (m *mqtt) Eavesdrop() []byte {
	data := <-m.pubCh
	return data
}
