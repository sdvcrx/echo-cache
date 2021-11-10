package cache

import (
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"time"

	"context"

	bolt "go.etcd.io/bbolt"
)

type BoltAdapter struct {
	ctx    context.Context
	db     *bolt.DB
	ticker *time.Ticker
	bucket []byte
}

var _ CacheAdapter = &BoltAdapter{}

type ExpirableMessage struct {
	Response  *Response `json:"response"`
	ExpiredAt time.Time `json:"expired_at"`
}

func (r *ExpirableMessage) Expired() bool {
	return time.Since(r.ExpiredAt) > 0
}

func NewBoltAdapter(ctx context.Context, path string) *BoltAdapter {
	db, err := bolt.Open(path, fs.FileMode(os.O_RDWR|os.O_CREATE), nil)
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

	ba := &BoltAdapter{
		ctx:    ctx,
		bucket: bucket,
		db:     db,
	}
	ba.startCleanupTicker()
	return ba
}

func (ba *BoltAdapter) startCleanupTicker() {
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

func (ba *BoltAdapter) cleanupExpired() error {
	tx, err := ba.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	b := tx.Bucket(ba.bucket)
	err = b.ForEach(func(k, v []byte) error {
		var msg ExpirableMessage
		err := json.Unmarshal(v, &msg)
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

func (ba *BoltAdapter) Get(key string) (*Response, error) {
	var msg ExpirableMessage

	err := ba.db.View(func(t *bolt.Tx) error {
		b := t.Bucket(ba.bucket)
		val := b.Get([]byte(key))
		if val == nil {
			return nil
		}

		err := json.Unmarshal(val, &msg)
		if err != nil {
			return err
		}

		if msg.Expired() {
			msg.Response = nil
		}

		return nil
	})

	return msg.Response, err
}

func (ba *BoltAdapter) Set(key string, response *Response, ttl time.Duration) error {
	msg := ExpirableMessage{
		Response:  response,
		ExpiredAt: time.Now().Add(ttl),
	}
	return ba.db.Batch(func(t *bolt.Tx) error {
		b := t.Bucket(ba.bucket)
		msgb, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		return b.Put([]byte(key), msgb)
	})
}
