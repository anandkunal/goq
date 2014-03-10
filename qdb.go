package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type QDB struct {
	path  string
	wo    opt.WriteOptions
	db    *leveldb.DB
	mutex sync.Mutex
}

func NewQDB(path string, syncWrites bool) *QDB {
	// Open
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		panic(fmt.Sprintf("goq: Unable to open db: %v", err))
	}

	log.Println("goq: Starting health check")

	// Health check each record
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		_, err := strconv.Atoi(string(iter.Key()))
		if err != nil {
			panic(fmt.Sprintf("goq: Health check failure (key not int): %s, %s, %v", string(iter.Key()), string(iter.Value()), err))
		}
	}

	// General
	if iter.Error() != nil {
		panic(fmt.Sprintf("goq: Error loading db: %v", err))
	}

	log.Println("goq: Health check successful")

	return &QDB{
		path: path,
		wo:   opt.WriteOptions{Sync: syncWrites},
		db:   db,
	}
}

func (self *QDB) Get(id int64) (*QueuedItem, error) {
	// Grab from level
	value, err := self.db.Get(QDBKey(id), nil)

	// Error retrieving key
	if err != nil {
		return nil, err
	}

	// Nil value, should never happen
	if value == nil {
		return nil, nil
	}

	return &QueuedItem{id, value}, nil
}

func (self *QDB) Put(qi *QueuedItem) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	// Put into level
	return self.db.Put(QDBKey(qi.id), qi.data, &self.wo)
}

func (self *QDB) Remove(id int64) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	// Delete from level
	return self.db.Delete(QDBKey(id), &self.wo)
}

func (self *QDB) Close() {
	self.db.Close()
}

func (self *QDB) Drop() {
	self.Close()
	err := os.RemoveAll(self.path)
	if err != nil {
		panic(fmt.Sprintf("goq: Error removing db from disk: %v", err))
	}
}

// Abstractions

func (self *QDB) Next(remove bool) *QueuedItem {
	iter := self.db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		id, err := strconv.Atoi(string(iter.Key()))
		if err != nil {
			panic(fmt.Sprintf("goq: Key not int: %s, %s, %v", string(iter.Key()), string(iter.Value()), err))
		}
		if remove {
			self.Remove(int64(id))
		}
		return &QueuedItem{int64(id), iter.Value()}
	}

	return nil
}

func (self *QDB) CacheFetch(maxBytes int) ([]*QueuedItem, int) {
	iter := self.db.NewIterator(nil, nil)
	defer iter.Release()

	// Collect
	items := make([]*QueuedItem, 0)
	totalSize := 0
	for iter.Next() {
		id, err := strconv.Atoi(string(iter.Key()))
		if err != nil {
			panic(fmt.Sprintf("goq: Key not int: %s, %s, %v", string(iter.Key()), string(iter.Value()), err))
		}
		qi := QueuedItem{int64(id), iter.Value()}
		if totalSize+qi.Size() < maxBytes {
			items = append(items, &qi)
		}
	}

	return items, totalSize
}
