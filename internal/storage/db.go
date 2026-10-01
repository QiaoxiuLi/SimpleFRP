package storage

import (
	"encoding/json"
	"time"

	"github.com/simplefrp/simplefrp/internal/sysutil"
	"go.etcd.io/bbolt"
)

var buckets = [][]byte{[]byte("traffic"), []byte("connections"), []byte("tunnels"), []byte("runtime_v2")}

type Store struct {
	db *bbolt.DB
}

func (s *Store) ReadRuntime(out any) error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("runtime_v2"))
		if b == nil || b.Get([]byte("state")) == nil {
			return nil
		}
		return json.Unmarshal(b.Get([]byte("state")), out)
	})
}
func (s *Store) SaveRuntime(value any) error {
	if s == nil || s.db == nil {
		return nil
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bbolt.Tx) error { return tx.Bucket([]byte("runtime_v2")).Put([]byte("state"), b) })
}

func OpenDefault() (*Store, error) {
	if err := sysutil.EnsureBaseDirs(); err != nil {
		return nil, err
	}
	return Open(sysutil.DatabasePath())
}

func Open(path string) (*Store, error) {
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	_ = sysutil.ChownToServiceUser(path)
	s := &Store{db: db}
	err = db.Update(func(tx *bbolt.Tx) error {
		for _, name := range buckets {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return err
			}
		}
		return nil
	})
	return s, err
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
