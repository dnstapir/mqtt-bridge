package fake

type nodeman struct {
    key []byte
}

func Nodeman() *nodeman {
    nodeman := new(nodeman)
    return nodeman
}

func (n* nodeman) GetKey(keyID string) ([]byte, error) {
    return n.key, nil
}

func (n* nodeman) PrepareKey(key []byte) {
    n.key = key
}
