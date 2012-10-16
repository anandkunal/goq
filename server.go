package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type State int

const (
	READ_CURRENT State = 1
	READ_BEHIND  State = 2
)

var (
	max_memory_bytes   int
	port               int
	journal_directory  string
	journal_mutex      *sync.Mutex    = new(sync.Mutex)
	total_count        int            = 0
	current_state      State          = READ_CURRENT
	current_journal    *JournalWriter = nil
	future_journal     *JournalWriter = nil
	journals           []string       = make([]string, 0)
	queued_items       []*QueuedItem  = make([]*QueuedItem, 0)
	queued_items_bytes int            = 0
)

func generate_journal_path() string {
	return fmt.Sprintf("%s/%d.log", journal_directory, time.Now().Unix())
}

func journals_dir_exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func healthcheck_journals() {
	journals_list, _ := filepath.Glob(fmt.Sprintf("%s/*.log", journal_directory))
	sort.Strings(journals_list)
	for _, j := range journals_list {
		log.Printf("Verifying journal: %s\n", j)
		reader := NewJournalReader(j)
		items := reader.Verify()
		total_count = total_count + len(items)
		if len(items) == 0 {
			log.Printf("Deleting empty journal: %s\n", j)
			reader.Delete()
		} else {
			reader.Close()
			journals = append(journals, j)
		}
	}
}

func replay_next_journal() {
	// Load the new writer
	future := journals[0]
	journals = journals[1:]
	log.Printf("Loading next journal: %s", future)
	if len(journals) == 0 {
		if future_journal != nil {
			future_journal.Close()
			future_journal = nil
		}
		current_state = READ_CURRENT
	}
	current_journal = NewJournalWriter(future)

	// Replay items
	temp_reader := NewJournalReader(current_journal.file.Name())
	defer temp_reader.Close()
	log.Printf("Replaying journal: %s", temp_reader.file.Name())
	items := temp_reader.Verify()
	for _, value := range items {
		queued_items = append(queued_items, value)
		queued_items_bytes = queued_items_bytes + value.Size()
	}
}

func server_connections(listener net.Listener) chan net.Conn {
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
				journal_mutex.Lock()
				qi := new(QueuedItem)
				qi.id = time.Now().UnixNano()
				qi.data = []byte(strings.Replace(strings.Replace(string(line), "enq ", "", 1), "\n", "", -1))
				// If we have exceeded memory, then drop to read behind
				if queued_items_bytes >= max_memory_bytes {
					// If we have already gotten behind, check if we need to rotate the future journal
					if current_state == READ_BEHIND {
						if future_journal != nil && future_journal.written_byte_count >= max_memory_bytes {
							future_journal.Close()
							future_path := generate_journal_path()
							journals = append(journals, future_path)
							future_journal = NewJournalWriter(future_path)
						}
					} else {
						future_path := generate_journal_path()
						journals = append(journals, future_path)
						future_journal = NewJournalWriter(future_path)
						current_state = READ_BEHIND
					}
				}
				// If we are in current, we push to memory
				if current_state == READ_CURRENT {
					queued_items = append(queued_items, qi)
					queued_items_bytes = queued_items_bytes + qi.Size()
					current_journal.WriteEnqueue(qi)
				} else {
					future_journal.WriteEnqueue(qi)
				}
				total_count++
				journal_mutex.Unlock()

			case "deq":
				journal_mutex.Lock()
			DQ:
				if len(queued_items) > 0 {
					qi := queued_items[0]
					queued_items = queued_items[1:]
					queued_items_bytes = queued_items_bytes - qi.Size()
					total_count--
					current_journal.WriteDequeue(qi)
					client.Write(qi.data)
					client.Write([]byte("\n"))
				} else {
					if len(journals) > 0 {
						log.Println("Everything dequeued. Checking disk for more.")
						current_journal.Delete()
						replay_next_journal()
						goto DQ
					} else {
						// Keep the log pruned
						if current_state == READ_CURRENT {
							temp_path := current_journal.file.Name()
							current_journal.Delete()
							current_journal = NewJournalWriter(temp_path)
						}
						client.Write([]byte("NIL\n"))
						current_state = READ_CURRENT
					}
				}
				journal_mutex.Unlock()

			case "stats":
				stats := fmt.Sprintf("{\"memory_count\":%d,\"total_count\":%d,\"memory_bytes\":%d\"current_state\":%d}\n", len(queued_items), total_count, queued_items_bytes, current_state)
				client.Write([]byte(stats))

			case "version":
				client.Write([]byte("1.0\n"))

			case "quit":
				log.Printf("Disconnected %v\n", client.RemoteAddr())
				client.Close()
			}
		}
	}
}

func main() {
	runtime.GOMAXPROCS(2)

	memory_usage := "Maximum amount of bytes to store in memory and journals. Default is 64MB."
	flag.IntVar(&max_memory_bytes, "memory", 67108864, memory_usage)
	flag.IntVar(&max_memory_bytes, "m", 67108864, memory_usage+" (shorthand)")
	port_usage := "Port to listen on. Default is 11311."
	flag.IntVar(&port, "port", 11311, port_usage)
	flag.IntVar(&port, "p", 11311, port_usage+" (shorthand)")
	journal_usage := "Journals directory. Default is journals (current directory)."
	flag.StringVar(&journal_directory, "journals", "journals", journal_usage)
	flag.StringVar(&journal_directory, "j", "journals", journal_usage+" (shorthand)")
	flag.Parse()
	log.Printf("Memory bytes: %d\n", max_memory_bytes)
	log.Printf("Listening on port: %d\n", port)
	log.Printf("Journals directory: %s\n", journal_directory)

	if exists, _ := journals_dir_exists(journal_directory); !exists {
		log.Panicf("Journals directory: %s does not exist\n", journal_directory)
	}

	server, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Panic(err)
	}
	connections := server_connections(server)

	healthcheck_journals()
	if len(journals) > 0 {
		replay_next_journal()
	}

	if current_journal == nil {
		current_journal = NewJournalWriter(generate_journal_path())
	}

	log.Println("Ready...")
	for {
		go handle(<-connections)
	}
}
