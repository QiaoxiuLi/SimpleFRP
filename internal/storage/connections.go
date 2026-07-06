package storage

import (
	"encoding/binary"
	"encoding/json"
	"time"

	"go.etcd.io/bbolt"
)

type ConnectionLog struct {
	ID          string    `json:"id"`
	IPv4        string    `json:"ipv4"`
	IPv6        string    `json:"ipv6,omitempty"`
	ConnectedAt time.Time `json:"connected_at"`
	Duration    string    `json:"duration"`
	TunnelID    int       `json:"tunnel_id"`
	LocalPort   int       `json:"local_port"`
	PublicPort  int       `json:"public_port"`
	Status      string    `json:"status"`
	ErrorReason string    `json:"error_reason,omitempty"`
}

func (s *Store) Connections(limit int) []ConnectionLog {
	if limit <= 0 {
		limit = 100
	}
	records := []ConnectionLog{}
	if s == nil || s.db == nil {
		return records
	}
	_ = s.db.View(func(tx *bbolt.Tx) error {
		c := tx.Bucket([]byte("connections")).Cursor()
		for k, v := c.Last(); k != nil && len(records) < limit; k, v = c.Prev() {
			var record ConnectionLog
			if err := json.Unmarshal(v, &record); err == nil {
				records = append(records, record)
			}
		}
		return nil
	})
	return records
}

func (s *Store) AddConnection(record ConnectionLog) error {
	if s == nil || s.db == nil {
		return nil
	}
	value, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("connections"))
		id, err := b.NextSequence()
		if err != nil {
			return err
		}
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, id)
		return b.Put(key, value)
	})
}
