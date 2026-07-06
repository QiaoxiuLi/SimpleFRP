package storage

import (
	"encoding/binary"
	"encoding/json"

	"go.etcd.io/bbolt"
)

type TunnelRecord struct {
	ID         int    `json:"id"`
	LocalPort  int    `json:"local_port"`
	PublicPort int    `json:"public_port"`
	Status     string `json:"status"`
}

func (s *Store) Tunnels() []TunnelRecord {
	records := []TunnelRecord{}
	if s == nil || s.db == nil {
		return records
	}
	_ = s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("tunnels"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, value []byte) error {
			var record TunnelRecord
			if err := json.Unmarshal(value, &record); err == nil {
				records = append(records, record)
			}
			return nil
		})
	})
	return records
}

func (s *Store) SetTunnel(record TunnelRecord) error {
	if s == nil || s.db == nil {
		return nil
	}
	value, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte("tunnels")).Put(intKey(record.PublicPort), value)
	})
}

func (s *Store) DeleteTunnel(publicPort int) error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte("tunnels")).Delete(intKey(publicPort))
	})
}

func intKey(v int) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, uint64(v))
	return key
}
