package boltstore

import (
	"log"
	"time"

	"context"

	"github.com/sdvcrx/echo-cache/store"
	"github.com/vmihailenco/msgpack/v5"
	bolt "go.etcd.io/bbolt"
)

type BoltStore struct {
	ctx    context.Context
	db     *bolt.DB
	ticker *time.Ticker
	bucket []byte
}

var _ store.Store = (*BoltStore)(nil)

type expirableMessage struct {
	Value     []byte
	ExpiredAt time.Time
}

func (r *expirableMessage) Expired() bool {
	return time.Since(r.ExpiredAt) > 0
}

func New(ctx context.Context, path string) *BoltStore {
	db, err := bolt.Open(path, 0644, nil)
	if err != nil {
		log.Fatalln(err)
	}

	bucket := []byte("cache")
	err = db.Update(func(t *bolt.Tx) error {
		_, err := t.CreateBucketIfNotExists(bucket)
		return err
	})
	if err != nil {
		log.Fatalln("Failed to create bolt bucket", err)
	}

	ba := &BoltStore{
		ctx:    ctx,
		bucket: bucket,
		db:     db,
	}
	ba.startCleanupTicker()
	return ba
}

func (ba *BoltStore) startCleanupTicker() {
	ba.ticker = time.NewTicker(1 * time.Minute)

	go func() {
		for {
			select {
			// when ctx Done, ensure db close first
			case <-ba.ctx.Done():
				ba.db.Close()
				ba.ticker.Stop()
				return
			case <-ba.ticker.C:
				if err := ba.cleanupExpired(); err != nil {
					log.Println("Failed to cleanup bolt expired messages", err)
				}
			}
		}
	}()
}

func (ba *BoltStore) cleanupExpired() error {
	tx, err := ba.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	b := tx.Bucket(ba.bucket)
	err = b.ForEach(func(k, v []byte) error {
		var msg expirableMessage
		err := msgpack.Unmarshal(v, &msg)
		if err != nil {
			return err
		}
		if msg.Expired() {
			return b.Delete(k)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (ba *BoltStore) Get(key string) ([]byte, error) {
	var msg expirableMessage

	err := ba.db.View(func(t *bolt.Tx) error {
		b := t.Bucket(ba.bucket)
		val := b.Get([]byte(key))
		if val == nil {
			return nil
		}

		err := msgpack.Unmarshal(val, &msg)
		if err != nil {
			return err
		}

		if msg.Expired() {
			msg.Value = nil
		}

		return nil
	})

	return msg.Value, err
}

func (ba *BoltStore) Set(key string, val []byte, ttl time.Duration) error {
	msg := expirableMessage{
		Value:     val,
		ExpiredAt: time.Now().Add(ttl),
	}
	return ba.db.Batch(func(t *bolt.Tx) error {
		b := t.Bucket(ba.bucket)
		msgb, err := msgpack.Marshal(msg)
		if err != nil {
			return err
		}
		return b.Put([]byte(key), msgb)
	})
}
