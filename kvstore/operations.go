package kvstore

import (
	"errors"
	"fmt"
)

func (kv *KvStore) Put(key string, value string) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv == nil {
		return errors.New("kvstore is nil")
	}
	kv.data[key] = value
	fmt.Println(kv.data, key, value)
	return nil
}

func (kv *KvStore) Get(key string) (string, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	if kv == nil {
		return "", errors.New("kvstore is nil")
	}
	value, ok := kv.data[key]
	if !ok {
		return "", errors.New("key not found")
	}
	fmt.Println("keyfound", key, value)
	return value, nil
}

func (kv *KvStore) Delete(key string) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv == nil {
		return errors.New("kvstore is nil")
	}
	delete(kv.data, key)
	return nil
}

func (kv *KvStore) GetAll() (map[string]string, error) {
	if kv == nil {
		return nil, errors.New("kvstore is nil")
	}
	return kv.data, nil
}

func (kv *KvStore) Close() error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv == nil {
		return errors.New("kvstore is nil")
	}
	kv.data = nil
	return nil
}

func SnapShot(ch chan bool) {
	KV.mu.RLock()
	defer KV.mu.RUnlock()
	if KV == nil {
		ch <- true
		return
	}
	res := KV.data
	saveMapToFile(res)
	fmt.Println(res)
	ch <- true
}
