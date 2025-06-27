package kvstore

import (
	"fmt"
	"sync"
)

type KvStore struct {
	data map[string]string
	mu   sync.RWMutex
}

var KV *KvStore

func NewKvStore() *KvStore {
	KV = &KvStore{
		data: make(map[string]string),
		mu:   sync.RWMutex{},
	}
	data, err := LoadFileToMap()
	if err != nil {
		panic(err)
	}
	KV.data = *data
	fmt.Println("KVStore initialized", KV.data)
	return KV
}
