package boltdb

import (
	"path"

	"github.com/ben-han-cn/kvzoo"
	"github.com/ben-han-cn/kvzoo/backend/bolt"
)

const dbFileName = "ddi_agent.db"

type BoltDB struct {
	db kvzoo.DB
}

var globalDB *BoltDB

func New(dbPath string) error {
	db, err := bolt.New(path.Join(dbPath, dbFileName))
	if err != nil {
		return err
	}

	globalDB = &BoltDB{db: db}
	return nil
}

func GetDB() *BoltDB {
	return globalDB
}

func (b *BoltDB) Close() error {
	return b.db.Close()
}

func (b *BoltDB) CreateOrGetTable(table string) (kvzoo.Table, error) {
	return b.db.CreateOrGetTable(kvzoo.TableName(table))
}

func (b *BoltDB) DeleteTable(table string) error {
	return b.db.DeleteTable(kvzoo.TableName(table))
}

func (b *BoltDB) GetTableKVs(table string) (map[string][]byte, error) {
	tb, err := b.db.CreateOrGetTable(kvzoo.TableName(table))
	if err != nil {
		return nil, err
	}

	tx, err := tb.Begin()
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()
	return tx.List()
}

func (b *BoltDB) GetTables(table string) ([]string, error) {
	tb, err := b.db.CreateOrGetTable(kvzoo.TableName(table))
	if err != nil {
		return nil, err
	}

	tx, err := tb.Begin()
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()
	return tx.Tables()
}

func (b *BoltDB) AddKVs(tableName string, values map[string][]byte) error {
	tb, err := b.db.CreateOrGetTable(kvzoo.TableName(tableName))
	if err != nil {
		return err
	}

	tx, err := tb.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()
	for k, v := range values {
		if err := tx.Add(k, v); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (b *BoltDB) UpdateKVs(tableName string, values map[string][]byte) error {
	tb, err := b.db.CreateOrGetTable(kvzoo.TableName(tableName))
	if err != nil {
		return err
	}

	tx, err := tb.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()
	for k, v := range values {
		if err := tx.Update(k, v); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (b *BoltDB) DeleteKVs(tableName string, keys []string) error {
	tb, err := b.db.CreateOrGetTable(kvzoo.TableName(tableName))
	if err != nil {
		return err
	}

	tx, err := tb.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()
	for _, key := range keys {
		if err := tx.Delete(key); err != nil {
			return err
		}
	}

	return tx.Commit()
}
