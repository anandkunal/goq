package main

import (
	"encoding/binary"
)

type QueuedItem struct {
	ID   int64
	Data []byte
}

func (qi *QueuedItem) Size() int {
	return (binary.Size(qi.ID) + binary.Size(qi.Data))
}
