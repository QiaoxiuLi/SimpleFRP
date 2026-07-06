package storage

import (
	"path/filepath"
	"testing"
	"time"

	"go.etcd.io/bbolt"
)

func TestStorePersistsDashboardData(t *testing.T) {
	db, err := bbolt.Open(filepath.Join(t.TempDir(), "simplefrp.db"), 0600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	store := &Store{db: db}
	defer store.Close()
	if err := db.Update(func(tx *bbolt.Tx) error {
		for _, name := range buckets {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("create buckets: %v", err)
	}

	if err := store.SetTunnel(TunnelRecord{ID: 1, LocalPort: 25001, PublicPort: 35001, Status: "success"}); err != nil {
		t.Fatalf("set tunnel: %v", err)
	}
	if err := store.AddTraffic(100, 50); err != nil {
		t.Fatalf("add traffic: %v", err)
	}
	if err := store.AddConnection(ConnectionLog{ID: "conn-1", ConnectedAt: time.Now(), TunnelID: 1, Status: "success"}); err != nil {
		t.Fatalf("add connection: %v", err)
	}

	if got := store.Tunnels(); len(got) != 1 || got[0].PublicPort != 35001 {
		t.Fatalf("unexpected tunnels: %+v", got)
	}
	if got := store.Traffic("30d"); got.UploadedBytes != 100 || got.DownloadedBytes != 50 || got.TotalBytes != 150 {
		t.Fatalf("unexpected traffic: %+v", got)
	}
	if got := store.Connections(10); len(got) != 1 || got[0].ID != "conn-1" {
		t.Fatalf("unexpected connections: %+v", got)
	}
}
