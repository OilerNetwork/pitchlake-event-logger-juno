package db

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
	tx   pgx.Tx
	ctx  context.Context
	url  string
}

func Init(dbUrl string) (*DB, error) {
	config, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// m, err := migrate.New(
	// 	"file://db/migrations",
	// 	dbUrl)
	// if err != nil {
	// 	log.Printf("FAIlED HERE 1")
	// 	return nil, err
	// }
	// if err := m.Up(); err != nil {
	// 	if err != migrate.ErrNoChange {
	// 		return nil, err
	// 	}

	// }
	// m.Close()

	return &DB{
		Pool: pool,
		ctx:  context.Background(),
		url:  dbUrl, //Unsafe possibly, need to consolidate config better
	}, nil

}

func (db *DB) Shutdown() {
	db.Pool.Close()
}

func (db *DB) BeginTx() {
	tx, err := db.Pool.Begin(context.TODO())
	if err != nil {
		log.Printf("WTHELLY TX WAALA")
		log.Fatal(err)
	}
	db.tx = tx
}

func (db *DB) CommitTx() {
	db.tx.Commit(db.ctx)
	db.tx = nil
}

func (db *DB) RollbackTx() {
	db.tx.Rollback(db.ctx)
	db.tx = nil
}
