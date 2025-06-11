package cache


import (
    "errors"

	"github.com/dnstapir/mqtt-bridge/shared"

	"github.com/dnstapir/mqtt-bridge/app/keys"
)

type cache struct {
    log           shared.ILogger
}

type Conf struct {
    Log           shared.ILogger
}

func Create(conf Conf) (*cache, error) {
    return nil, errors.New("not implemented")
}

func (c *cache) GetValkeyFromCache(keyID string) *keys.ValKey {
    return nil
}

func (c *cache) StoreValkeyInCache(key keys.ValKey) error {
    return errors.New("not implemented")
}
