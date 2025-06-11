package nodeman

import (
    "crypto/tls"
    "errors"
    "net/url"
	"net/http"

	"github.com/dnstapir/mqtt-bridge/shared"
)

type Conf struct {
	Log           shared.ILogger
	NodemanApiUrl string
}

type nodemanclient struct {
	url    *url.URL
    client http.Client
}

func Create(conf Conf) (*nodemanclient, error) {
    newClient := new(nodemanclient)

	nodemanUrl, err := url.Parse(conf.NodemanApiUrl)
	if err != nil {
		return nil, errors.New("invalid nodeman api url")
	}
    newClient.url = nodemanUrl

	tlsCfg := tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	tr := &http.Transport{
		MaxIdleConns:       10,
		DisableCompression: true,
		TLSClientConfig:    &tlsCfg,
	}

	client := http.Client{
		Transport: tr,
	}

    newClient.client = client

    return newClient, nil
}

func (nc *nodemanclient) Subscribe(subject string) (<-chan []byte, error) {
    return nil, errors.New("not implemented")
}

func (nc *nodemanclient) StartPublishing(subject string) (chan<- []byte, error) {
    return nil, errors.New("not implemented")
}
