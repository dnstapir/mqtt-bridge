package fake

import (
)

type nodeman struct {
}

func Nodeman() *nodeman {
    nodeman := new(nodeman)
    return nodeman
}
