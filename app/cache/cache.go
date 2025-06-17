package cache


import (
	"github.com/dnstapir/mqtt-bridge/app/keys"

	lru "github.com/hashicorp/golang-lru/v2"
)

const cCACHE_SIZE = 1000

type LruCache struct {
	valKeyCache *lru.Cache[string, keys.ValKey]
}

type Conf struct {
}

func Create(conf Conf) (*LruCache, error) {
    newCache := new(LruCache)

    newLru, err := lru.New[string, keys.ValKey](cCACHE_SIZE)
    if err != nil {
        return nil, err
    }

    newCache.valKeyCache = newLru

    return newCache, nil
}

func (l *LruCache) GetValkeyFromCache(keyID string) keys.ValKey {
	key, ok := l.valKeyCache.Get(keyID)

    if !ok {
        return nil
    }

    return key
}

func (l *LruCache) StoreValkeyInCache(key keys.ValKey) error {
    l.valKeyCache.Add(key.KeyID(), key)
    return nil
}
