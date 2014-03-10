package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	VERSION = "1.0"
)

var (
	address    string
	port       int
	syncWrites bool
	dbPath     string

	db *QDB

	totalEnqueues int
	totalDequeues int
	totalEmpties  int
)

func Enqueue(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if req.Method != "POST" {
		w.WriteHeader(405)
		fmt.Fprint(w, "{success:false, message:\"post request required\"}")
		return
	}

	data := strings.TrimSpace(req.FormValue("data"))
	if len(data) == 0 {
		w.WriteHeader(400)
		fmt.Fprint(w, "{success:false, message:\"data with length > 0 required\"}")
		return
	}

	db.Put(&QueuedItem{time.Now().UnixNano(), []byte(data)})
	w.WriteHeader(200)
	fmt.Fprint(w, "{success:true, message:\"worked\"}")

	totalEnqueues++
}

func Dequeue(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if req.Method != "GET" {
		w.WriteHeader(405)
		fmt.Fprint(w, "{success:false, message:\"get request required\"}")
		return
	}

	count, err := strconv.Atoi(strings.TrimSpace(req.FormValue("count")))
	if err != nil {
		count = 1
	}

	w.WriteHeader(200)

	items := make([]string, 0)
	for i := 0; i < count; i++ {
		qi := db.Next(true)
		if qi == nil {
			break
		}
		items = append(items, string(qi.Data))
	}

	itemsJson, jsonErr := json.Marshal(items)

	if jsonErr != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, "{success:false, data:[], message:\"internal error\"}")
		return
	}

	fmt.Fprint(w, fmt.Sprintf("{success:true, data:%s, message:\"worked\"}", string(itemsJson)))

	totalDequeues += len(items)
	if len(items) == 0 {
		totalEmpties++
	}
}

func Statistics(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	fmt.Fprint(w, fmt.Sprintf("{\"enqueues\":%d, \"dequeues\":%d, \"empties\":%d}", totalEnqueues, totalDequeues, totalEmpties))
}

func Version(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	fmt.Fprint(w, fmt.Sprintf("{version:\"%s\"}", VERSION))
}

func HealthCheck(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	fmt.Fprint(w, 1)
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.StringVar(&address, "address", "", "Address to listen on. Default is all.")
	flag.IntVar(&port, "port", 11311, "Port to listen on. Default is 11311.")
	flag.BoolVar(&syncWrites, "sync", true, "Synchronize database writes")
	flag.StringVar(&dbPath, "path", "db", "Database path. Default is db in current directory.")
	flag.Parse()
}

func main() {
	log.Printf("Listening on %s:%d\n", address, port)
	log.Printf("DB Path: %s\n", dbPath)

	db = NewQDB(dbPath, syncWrites)

	http.HandleFunc("/enqueue", Enqueue)
	http.HandleFunc("/dequeue", Dequeue)
	http.HandleFunc("/statistics", Statistics)
	http.HandleFunc("/version", Version)
	http.HandleFunc("/", HealthCheck)

	err := http.ListenAndServe(fmt.Sprintf("%s:%d", address, port), nil)
	if err != nil {
		panic(fmt.Sprintf("goq failed to launch: %v", err))
	}
}
