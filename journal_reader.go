package main

import (
	"bufio"
	"github.com/ugorji/go-msgpack"
	"log"
	"os"
	"sort"
	"strconv"
)

type JournalReader struct {
	file   *os.File
	reader *bufio.Reader
}

func NewJournalReader(path string) *JournalReader {
	log.Printf("Spawning journal reader: %s", path)
	var err error
	jr := new(JournalReader)
	jr.file, err = os.OpenFile(path, os.O_RDONLY, 0777)
	if err != nil {
		log.Panicf("Could not open file: %s", err)
	}
	jr.reader = bufio.NewReader(jr.file)
	return jr
}

func (jr *JournalReader) Verify() []*QueuedItem {
	active_items := make(map[int64][]byte)

	for {
		line, err := jr.reader.ReadBytes('\n')
		if err != nil {
			break
		}

		// Check the bounds
		if len(line) < 21 {
			log.Panicf("Invalid log line: %s", line)
		}

		// Parse the identifier
		id, id_err := strconv.ParseInt(string(line[:19]), 10, 64)
		if id_err != nil {
			log.Panicln(id_err)
		}

		if len(line) > 21 {
			// If the length is greater than 21, then this could be an enqueue
			var data []byte
			msgpack_err := msgpack.Unmarshal(line[19:], &data, nil)
			if msgpack_err != nil {
				log.Panicln(msgpack_err)
			}
			active_items[id] = data
		} else {
			delete(active_items, id)
		}
	}

	queued_items := make([]*QueuedItem, 0)
	for key, value := range active_items {
		qi := new(QueuedItem)
		qi.id = key
		qi.data = value
		queued_items = append(queued_items, qi)
	}
	sort.Sort(ByQueuedItemId(queued_items))

	return queued_items
}

func (jr *JournalReader) Close() bool {
	jr.file.Close()
	return true
}

func (jr *JournalReader) Delete() bool {
	jr.Close()
	os.Remove(jr.file.Name())
	return true
}
