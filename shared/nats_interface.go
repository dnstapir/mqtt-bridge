package shared

const NATSHEADER_DNSTAPIR_MESSAGE_SCHEMA = "DNSTAPIR-Message-Schema"
const NATSHEADER_DNSTAPIR_MQTT_TOPIC = "DNSTAPIR-Mqtt-Topic"
const NATSHEADER_DNSTAPIR_KEY_IDENTIFIER = "DNSTAPIR-Key-Identifier"
const NATSHEADER_DNSTAPIR_KEY_THUMBPRINT = "DNSTAPIR-Key-Thumbprint"

var NATSHEADERS_DNSTAPIR_ALL = []string{
    NATSHEADER_DNSTAPIR_MESSAGE_SCHEMA,
    NATSHEADER_DNSTAPIR_MQTT_TOPIC,
    NATSHEADER_DNSTAPIR_KEY_IDENTIFIER,
    NATSHEADER_DNSTAPIR_KEY_THUMBPRINT,
}

type NatsIF interface {
	Connect() error
	Subscribe(string, string) (<-chan []byte, error)
	StartPublishing(string, string) (chan<- NatsData, error)
	Stop()
}

type NatsData struct {
    Headers map[string]string
    Payload []byte
}
