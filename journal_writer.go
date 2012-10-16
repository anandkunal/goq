package main

import (
	"bufio"
	"fmt"
	"github.com/ugorji/go-msgpack"
	"log"
	"os"
)

type JournalWriter struct {
	file               *os.File
	writer             *bufio.Writer
	written_byte_count int
}

func NewJournalWriter(path string) *JournalWriter {
	log.Printf("Spawning journal writer: %s", path)
	var err error
	jw := new(JournalWriter)
	jw.file, err = os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		log.Panicf("Could not open file: %s", err)
	}
	jw.writer = bufio.NewWriter(jw.file)
	jw.written_byte_count = 0
	return jw
}

func (jw *JournalWriter) WriteEnqueue(qi *QueuedItem) bool {
	packed, err := msgpack.Marshal(qi.data)
	if err != nil {
		log.Panicf("Could not pack bytes: %s", qi.data)
	}
	jw.writer.Write([]byte(fmt.Sprintf("%d", qi.id)))
	jw.writer.Write(packed)
	jw.writer.Write([]byte("\r\n"))
	jw.writer.Flush()
	jw.written_byte_count = jw.written_byte_count + qi.Size()
	return true
}

func (jw *JournalWriter) WriteDequeue(qi *QueuedItem) bool {
	jw.writer.Write([]byte(fmt.Sprintf("%d\r\n", qi.id)))
	jw.writer.Flush()
	jw.written_byte_count = jw.written_byte_count + 8 // int64
	return true
}

func (jw *JournalWriter) Close() bool {
	jw.writer.Flush()
	jw.file.Close()
	return true
}

func (jw *JournalWriter) Delete() bool {
	jw.file.Close()
	log.Printf("Deleting journal: %s", jw.file.Name())
	os.Remove(jw.file.Name())
	return true
}
