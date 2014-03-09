package main

import (
	"fmt"
)

func QDBKey(id int64) []byte {
	return []byte(fmt.Sprintf("%d", id))
}
