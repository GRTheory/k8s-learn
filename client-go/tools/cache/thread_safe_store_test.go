package cache

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestThreadSafeStoreDeleteRemovesEmptySetsFromIndex(t *testing.T) {
	testIndexer := "testIndexer"

	indexers := Indexers{
		testIndexer: func(obj interface{}) (strings []string, e error) {
			indexes := []string{obj.(string)}
			return indexes, nil
		},
	}

	indices := Indices{}
	store := NewThreadSafeStore(indexers, indices).(*threadSafeMap)

	testKey := "testKey"

	store.Add(testKey, testKey)

	set := store.index.indices[testIndexer][testKey]

	if len(set) != 1 {
		t.Errorf("Initial assumption of index backing string set having 1 element failed. Actual elements: %d", len(set))
		return
	}

	store.Delete(testKey)
	set, present := store.index.indices[testIndexer][testKey]

	if present {
		t.Errorf("Index backing string set not deletedd from index. Set length: %d", len(set))
	}
}

func TestThreadSafeStoreAddKeepNonEmptySetPostDeleteFromIndex(t *testing.T) {
	testIndexer := "testIndexer"
	testIndex := "testIndex"

	indexers := Indexers{
		testIndexer: func(obj interface{}) (strings []string, e error) {
			indexes := []string{testIndex}
			return indexes, nil
		},
	}

	indices := Indices{}
	store := NewThreadSafeStore(indexers, indices).(*threadSafeMap)

	store.Add("retain", "retain")
	store.Add("delete", "delete")

	set := store.index.indices[testIndexer][testIndex]

	if len(set) != 2 {
		t.Errorf("Initial assumption of index backing string set having 2 elements failed. Actual elements: %d", len(set))
		return
	}

	store.Delete("delete")
	set, present := store.index.indices[testIndexer][testIndex]

	if !present {
		t.Errorf("Index backing string set erroneously deleted from index.")
		return
	}

	if len(set) != 1 {
		t.Errorf("Index backing string set has incorrect length, expect 1. Set length: %d", len(set))
	}
}

func TestThreadSafeStoreIndexingFunctionsWithMultipleValues(t *testing.T) {
	testIndexer := "testIndexer"

	indexers := Indexers{
		testIndexer: func(obj any) ([]string, error) {
			return strings.Split(obj.(string), ","), nil
		},
	}

	indices := Indices{}
	store := NewThreadSafeStore(indexers, indices).(*threadSafeMap)

	store.Add("key1", "foo")
	store.Add("key2", "bar")

	assert := assert.New(t)

	compare := func(key string, expected []string) error {
		values := store.index.indices[testIndexer][key].List()
		if cmp.Equal(values, expected) {
			return nil
		}
		return fmt.Errorf("unexpected index for key %s, diff=%s", key, cmp.Diff(values, expected))
	}

	assert.NoError(compare("foo", []string{"key1"}))
	assert.NoError(compare("bar", []string{"key2"}))

	store.Update("key2", "foo,bar")
	assert.NoError(compare("foo", []string{"key1", "key2"}))
	assert.NoError(compare("bar", []string{"key2"}))

	store.Update("key1", "foo,bar")
	assert.NoError(compare("foo", []string{"key1", "key2"}))
	assert.NoError(compare("bar", []string{"key1", "key2"}))

	store.Add("key3", "foo,bar,baz")
	assert.NoError(compare("foo", []string{"key1", "key2", "key3"}))
	assert.NoError(compare("bar", []string{"key1", "key2", "key3"}))
	assert.NoError(compare("baz", []string{"key3"}))

	store.Update("key1", "foo")

	assert.NoError(compare("foo", []string{"key1", "key2", "key3"}))
	assert.NoError(compare("bar", []string{"key2", "key3"}))
	assert.NoError(compare("baz", []string{"key3"}))

	store.Update("key2", "bar")

	assert.NoError(compare("foo", []string{"key1", "key3"}))
	assert.NoError(compare("bar", []string{"key2", "key3"}))
	assert.NoError(compare("baz", []string{"key3"}))

	store.Delete("key1")

	assert.NoError(compare("foo", []string{"key3"}))
	assert.NoError(compare("bar", []string{"key2", "key3"}))
	assert.NoError(compare("baz", []string{"key3"}))

	store.Delete("key3")

	assert.NoError(compare("foo", []string{}))
	assert.NoError(compare("bar", []string{"key2"}))
	assert.NoError(compare("baz", []string{}))
}

func BenchmarkIndexer(b *testing.B) {
	testIndexer := "testIndexer"

	indexers := Indexers{
		testIndexer: func(obj interface{}) (strings []string, e error) {
			indexes := []string{obj.(string)}
			return indexes, nil
		},
	}

	indices := Indices{}
	store := NewThreadSafeStore(indexers, indices).(*threadSafeMap)

	objectCount := 5000
	objects := make([]string, 0, 5000)
	for i := 0; i < objectCount; i++ {
		objects = append(objects, fmt.Sprintf("object-number-%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Update(objects[i%objectCount], objects[i%objectCount])
	}
}
