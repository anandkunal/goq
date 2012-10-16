package main

import (
	"encoding/binary"
)

type QueuedItem struct {
	id   int64
	data []byte
}

func (qi *QueuedItem) Size() int {
	return (binary.Size(qi.id) + binary.Size(qi.data))
}
