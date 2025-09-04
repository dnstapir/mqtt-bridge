package nodeman

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/dnstapir/mqtt-bridge/shared"
)

type Conf struct {
	Log           shared.LoggerIF
	NodemanApiUrl string
}

type nodemanclient struct {
	log    shared.LoggerIF
	url    *url.URL
	client http.Client
}

const cNODEMAN_NODE_API_FMT = "/node/%s/public_key"

func Create(conf Conf) (*nodemanclient, error) {
	newNodeman := new(nodemanclient)

	if conf.Log == nil {
		return nil, errors.New("nil logger when creating nodeman client")
	}
	newNodeman.log = conf.Log

	nodemanUrl, err := url.Parse(conf.NodemanApiUrl)
	if err != nil {
		return nil, errors.New("invalid nodeman api url")
	}
	newNodeman.url = nodemanUrl

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
		Timeout:   30 * time.Second,
	}

	newNodeman.client = client

	return newNodeman, nil
}

func (n *nodemanclient) GetKey(keyID string) ([]byte, error) {
	req, err := http.NewRequest("GET",
		n.url.JoinPath(fmt.Sprintf(cNODEMAN_NODE_API_FMT, keyID)).String(),
		nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("accept", "application/json")

	rsp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		n.log.Error("nodeman API returned status %d", rsp.StatusCode)
		return nil, errors.New("bad reponse error code")
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
