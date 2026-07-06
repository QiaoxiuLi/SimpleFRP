package storage

import (
	"encoding/binary"

	"go.etcd.io/bbolt"
)

type TrafficSummary struct {
	UploadedBytes   int64 `json:"uploaded_bytes"`
	DownloadedBytes int64 `json:"downloaded_bytes"`
	TotalBytes      int64 `json:"total_bytes"`
}

func (s *Store) Traffic(_ string) TrafficSummary {
	var summary TrafficSummary
	if s == nil || s.db == nil {
		return summary
	}
	_ = s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("traffic"))
		if b == nil {
			return nil
		}
		summary.UploadedBytes = readInt64(b.Get([]byte("uploaded")))
		summary.DownloadedBytes = readInt64(b.Get([]byte("downloaded")))
		summary.TotalBytes = summary.UploadedBytes + summary.DownloadedBytes
		return nil
	})
	return summary
}

func (s *Store) AddTraffic(uploaded, downloaded int64) error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("traffic"))
		if err := putInt64(b, []byte("uploaded"), readInt64(b.Get([]byte("uploaded")))+uploaded); err != nil {
			return err
		}
		return putInt64(b, []byte("downloaded"), readInt64(b.Get([]byte("downloaded")))+downloaded)
	})
}

func readInt64(value []byte) int64 {
	if len(value) != 8 {
		return 0
	}
	return int64(binary.BigEndian.Uint64(value))
}

func putInt64(b *bbolt.Bucket, key []byte, value int64) error {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(value))
	return b.Put(key, buf)
}
