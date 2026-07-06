package storage

import (
	"time"

	"github.com/simplefrp/simplefrp/internal/sysutil"
	"go.etcd.io/bbolt"
)

var buckets = [][]byte{[]byte("traffic"), []byte("connections"), []byte("tunnels")}

type Store struct {
	db *bbolt.DB
}

func OpenDefault() (*Store, error) {
	if err := sysutil.EnsureBaseDirs(); err != nil {
		return nil, err
	}
	db, err := bbolt.Open(sysutil.DatabasePath(), 0600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	_ = sysutil.ChownToServiceUser(sysutil.DatabasePath())
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
