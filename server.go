// TODO: Persistence and non-persistence MODE

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"runtime"
	"strings"
	"time"
)

var (
	address        string
	port           int
	maxMemoryBytes int
	syncWrites     bool
	dbPath         string

	db               *QDB
	memoryItems      []*QueuedItem
	memoryItemsBytes int
)

func connections(listener net.Listener) chan net.Conn {
	ch := make(chan net.Conn)
	go func() {
		for {
			client, err := listener.Accept()
			if err != nil {
				log.Printf("Failed to connect to %s\n", err)
				continue
			}
			log.Printf("Connected to %v\n", client.RemoteAddr())
			ch <- client
		}
	}()
	return ch
}

func handle(client net.Conn) {
	b := bufio.NewReader(client)
	for {
		line, err := b.ReadBytes('\n')
		if err != nil {
			break
		}

		chunks := strings.Split(strings.Replace(string(line), "\n", "", -1), " ")
		if len(chunks) >= 1 {
			switch chunks[0] {
			case "enq":
				qi := new(QueuedItem)
				qi.id = time.Now().UnixNano()
				qi.data = []byte(strings.Replace(strings.Replace(string(line), "enq ", "", 1), "\n", "", -1))
				if memoryItemsBytes+qi.Size() < maxMemoryBytes {
					// Write to memory if we have the headroom
					memoryItems = append(memoryItems, qi)
					memoryItemsBytes += qi.Size()
				}
				// Always write to the database
				db.Put(qi)

			case "deq":
				if len(memoryItems) > 0 {
					qi := memoryItems[0]
					memoryItems = memoryItems[1:]
					memoryItemsBytes = memoryItemsBytes - qi.Size()
					db.Remove(qi.id)
					client.Write(qi.data)
					client.Write([]byte("\n"))
				} else {
					// Get more from disk
					// Bug: need this to be recursive
					memoryItems, memoryItemsBytes = db.CacheFetch(maxMemoryBytes)
					client.Write([]byte("NIL\n"))
				}

			case "stats":
				stats := fmt.Sprintf("{\"memory_count\":%d,\"memory_bytes\":%d}\n", len(memoryItems), memoryItemsBytes)
				client.Write([]byte(stats))

			case "version":
				client.Write([]byte("2.0\n"))

			case "quit":
				log.Printf("Disconnected %v\n", client.RemoteAddr())
				client.Close()
			}
		}
	}
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.StringVar(&address, "address", "", "Address to listen on. Default is all.")
	flag.IntVar(&port, "port", 11311, "Port to listen on. Default is 11311.")
	flag.IntVar(&maxMemoryBytes, "memory", 67108864, "Maximum amount of bytes to store in memory and journals. Default is 64MB.")
	flag.BoolVar(&syncWrites, "sync", true, "Synchronize database writes")
	flag.StringVar(&dbPath, "path", "db", "Database path. Default is db in current directory.")

	flag.Parse()
}

func main() {
	log.Printf("Max memory bytes: %d\n", maxMemoryBytes)
	log.Printf("Listening on %s:%d\n", address, port)
	log.Printf("DB Path: %s\n", dbPath)

	db = NewQDB(dbPath, syncWrites)
	memoryItems, memoryItemsBytes = db.CacheFetch(maxMemoryBytes)

	server, err := net.Listen("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		log.Panic(err)
	}
	c := connections(server)

	log.Println("Ready...")
	for {
		go handle(<-c)
	}
}
