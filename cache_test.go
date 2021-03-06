package freecache

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestFreeCache(t *testing.T) {
	cache := NewCache(1024)
	if cache.HitRate() != 0 {
		t.Error("initial hit rate should be zero")
	}
	if cache.AverageAccessTime() != 0 {
		t.Error("initial average access time should be zero")
	}
	key := []byte("abcd")
	val := []byte("efghijkl")
	err := cache.Set(key, val, 0)
	if err != nil {
		t.Error("err should be nil")
	}
	value, err := cache.Get(key)
	if err != nil || !bytes.Equal(value, val) {
		t.Error("value not equal")
	}
	affected := cache.Del(key)
	if !affected {
		t.Error("del should return affected true")
	}
	value, err = cache.Get(key)
	if err != ErrNotFound {
		t.Error("error should be ErrNotFound after being deleted")
	}
	affected = cache.Del(key)
	if affected {
		t.Error("del should not return affected true")
	}

	cache.Clear()
	n := 5000
	for i := 0; i < n; i++ {
		keyStr := fmt.Sprintf("key%v", i)
		valStr := strings.Repeat(keyStr, 10)
		err = cache.Set([]byte(keyStr), []byte(valStr), 0)
		if err != nil {
			t.Error(err)
		}
	}
	time.Sleep(time.Second)
	for i := 1; i < n; i += 2 {
		keyStr := fmt.Sprintf("key%v", i)
		cache.Get([]byte(keyStr))
	}

	for i := 1; i < n; i += 8 {
		keyStr := fmt.Sprintf("key%v", i)
		cache.Del([]byte(keyStr))
	}

	for i := 0; i < n; i += 2 {
		keyStr := fmt.Sprintf("key%v", i)
		valStr := strings.Repeat(keyStr, 10)
		err = cache.Set([]byte(keyStr), []byte(valStr), 0)
		if err != nil {
			t.Error(err)
		}
	}
	for i := 1; i < n; i += 2 {
		keyStr := fmt.Sprintf("key%v", i)
		expectedValStr := strings.Repeat(keyStr, 10)
		value, err = cache.Get([]byte(keyStr))
		if err == nil {
			if string(value) != expectedValStr {
				t.Errorf("value is %v, expected %v", string(value), expectedValStr)
			}
		}
	}

	t.Logf("hit rate is %v, evacuates %v, entries %v, average time %v\n",
		cache.HitRate(), cache.EvacuateCount(), cache.EntryCount(), cache.AverageAccessTime())
}

func TestOverwrite(t *testing.T) {
	cache := NewCache(1024)
	key := []byte("abcd")
	val := []byte("efgh")
	cache.Set(key, val, 0)
	val = append(val, 'i')
	cache.Set(key, val, 0)
	if count := cache.OverwriteCount(); count != 0 {
		t.Error("overwrite count is", count, "expected ", 0)
	}
	res, _ := cache.Get(key)
	if string(res) != string(val) {
		t.Error(string(res))
	}
	val = append(val, 'j')
	cache.Set(key, val, 0)
	res, _ = cache.Get(key)
	if string(res) != string(val) {
		t.Error(string(res), "aaa")
	}
	val = append(val, 'k')
	cache.Set(key, val, 0)
	res, _ = cache.Get(key)
	if string(res) != "efghijk" {
		t.Error(string(res))
	}
	val = append(val, 'l')
	cache.Set(key, val, 0)
	res, _ = cache.Get(key)
	if string(res) != "efghijkl" {
		t.Error(string(res))
	}
	val = append(val, 'm')
	cache.Set(key, val, 0)
	if count := cache.OverwriteCount(); count != 3 {
		t.Error("overwrite count is", count, "expected ", 3)
	}

}

func TestExpire(t *testing.T) {
	cache := NewCache(1024)
	key := []byte("abcd")
	val := []byte("efgh")
	err := cache.Set(key, val, 1)
	if err != nil {
		t.Error("err should be nil")
	}
	time.Sleep(time.Second)
	val, err = cache.Get(key)
	if err == nil {
		t.Fatal("key should be expired", string(val))
	}
}

func TestLargeEntry(t *testing.T) {
	cacheSize := 512 * 1024
	cache := NewCache(cacheSize)
	key := make([]byte, 65536)
	val := []byte("efgh")
	err := cache.Set(key, val, 0)
	if err != ErrLargeKey {
		t.Error("large key should return ErrLargeKey")
	}
	val, err = cache.Get(key)
	if val != nil {
		t.Error("value should be nil when get a big key")
	}
	key = []byte("abcd")
	maxValLen := cacheSize/1024 - ENTRY_HDR_SIZE - len(key)
	val = make([]byte, maxValLen+1)
	err = cache.Set(key, val, 0)
	if err != ErrLargeEntry {
		t.Error("err should be ErrLargeEntry", err)
	}
	val = make([]byte, maxValLen-2)
	err = cache.Set(key, val, 0)
	if err != nil {
		t.Error(err)
	}
	val = append(val, 0)
	err = cache.Set(key, val, 0)
	if err != nil {
		t.Error(err)
	}
	val = append(val, 0)
	err = cache.Set(key, val, 0)
	if err != nil {
		t.Error(err)
	}
	if cache.OverwriteCount() != 1 {
		t.Error("over write count should be one.")
	}
	val = append(val, 0)
	err = cache.Set(key, val, 0)
	if err != ErrLargeEntry {
		t.Error("err should be ErrLargeEntry", err)
	}
}

func BenchmarkCacheSet(b *testing.B) {
	cache := NewCache(256 * 1024 * 1024)
	var key [8]byte
	for i := 0; i < b.N; i++ {
		binary.LittleEndian.PutUint64(key[:], uint64(i))
		cache.Set(key[:], make([]byte, 8), 0)
	}
}

func BenchmarkMapSet(b *testing.B) {
	m := make(map[string][]byte)
	var key [8]byte
	for i := 0; i < b.N; i++ {
		binary.LittleEndian.PutUint64(key[:], uint64(i))
		m[string(key[:])] = make([]byte, 8)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	b.StopTimer()
	cache := NewCache(256 * 1024 * 1024)
	var key [8]byte
	for i := 0; i < b.N; i++ {
		binary.LittleEndian.PutUint64(key[:], uint64(i))
		cache.Set(key[:], make([]byte, 8), 0)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		binary.LittleEndian.PutUint64(key[:], uint64(i))
		cache.Get(key[:])
	}
}

func BenchmarkMapGet(b *testing.B) {
	b.StopTimer()
	m := make(map[string][]byte)
	var key [8]byte
	for i := 0; i < b.N; i++ {
		binary.LittleEndian.PutUint64(key[:], uint64(i))
		m[string(key[:])] = make([]byte, 8)
	}
	b.StartTimer()
	var hitCount int64
	for i := 0; i < b.N; i++ {
		binary.LittleEndian.PutUint64(key[:], uint64(i))
		if m[string(key[:])] != nil {
			hitCount++
		}
	}
}
