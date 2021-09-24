package tron

import (
	"encoding/json"
	"fmt"

	"github.com/boltdb/bolt"
)

var (
	playerBucket = []byte("players")
)

//store is a storage mechanism for
//various game structs. disk or memory.
type Database struct {
	*bolt.DB
}

func NewDatabase(loc string, reset bool) (*Database, error) {
	b, err := bolt.Open(loc, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("Database error (%s)", err)
	}
	db := &Database{
		DB: b,
	}
	if reset {
		db.Update(func(tx *bolt.Tx) error {
			return tx.DeleteBucket(playerBucket)
		})
	}
	return db, nil
}

func (db *Database) save(p *Player) error {
	err := db.Update(func(tx *bolt.Tx) error {
		ps, err := tx.CreateBucketIfNotExists(playerBucket)
		if err != nil {
			return err
		}
		val, err := json.Marshal(p)
		if err != nil {
			return err
		}
		if err := ps.Put([]byte(p.hash), val); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		// log.Printf("failed to save player scores: %s", p.dbkey)
		return err
	}
	return nil
}

func (db *Database) load(p *Player) error {
	err := db.View(func(tx *bolt.Tx) error {
		ps := tx.Bucket(playerBucket)
		if ps == nil {
			return nil
		}
		val := ps.Get([]byte(p.hash))
		if val == nil {
			return nil
		}
		tmp := Player{}
		if err := json.Unmarshal(val, &tmp); err != nil {
			return err
		}
		//only load KDs
		p.Kills = tmp.Kills
		p.Deaths = tmp.Deaths
		return nil
	})
	if err != nil {
		// log.Printf("failed to load player scores: %s", p.dbkey)
		return err
	}
	return nil
}

func (db *Database) loadAll() ([]*Player, error) {
	players := []*Player{}
	err := db.View(func(tx *bolt.Tx) error {
		ps := tx.Bucket(playerBucket)
		if ps == nil {
			return nil
		}
		return ps.ForEach(func(key []byte, val []byte) error {
			p := &Player{}
			if err := json.Unmarshal(val, p); err != nil {
				return err
			}
			p.hash = string(key)
			players = append(players, p)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return players, nil
}
