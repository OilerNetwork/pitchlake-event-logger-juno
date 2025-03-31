package db

import (
	"context"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
	Conn *pgx.Conn
	tx   pgx.Tx
	ctx  context.Context
}

func Init(dbUrl string) (*DB, error) {
	config, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}

	conn, err := pgx.Connect(context.Background(), dbUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	m, err := migrate.New(
		"file://db/migrations",
		dbUrl)
	if err != nil {
		log.Printf("FAIlED HERE 1")
		return nil, err
	}
	if err := m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			return nil, err
		}

	}
	m.Close()

	return &DB{
		Pool: pool,
		Conn: conn,
		ctx:  context.Background(),
	}, nil

}

func (db *DB) Shutdown() {
	db.Pool.Close()
	db.tx.Rollback(context.Background())
	db.tx.Conn().Close(context.Background())
	db.Conn.Close(context.Background())
}

func (db *DB) BeginTx() {
	tx, err := db.Pool.Begin(db.ctx)
	if err != nil {
		log.Fatal(err)
	}
	db.tx = tx
}

func (db *DB) CommitTx() {
	db.tx.Commit(db.ctx)
	db.tx.Conn().Close(db.ctx)
	db.tx = nil
}

func (db *DB) RollbackTx() {
	db.tx.Rollback(db.ctx)
	db.tx.Conn().Close(db.ctx)
	db.tx = nil
}
