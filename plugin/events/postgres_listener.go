package events

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// PostgreSQLListener listens to PostgreSQL NOTIFY events
// This is EXAMPLE CODE showing how the event-processor should integrate with the logger
type PostgreSQLListener struct {
	db     *sql.DB
	logger *log.Logger
}

// DriverEvent represents a driver notification event from PostgreSQL
type DriverEvent struct {
	Type        string    `json:"type"`
	BlockNumber uint64    `json:"block_number"`
	BlockHash   string    `json:"block_hash"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewPostgreSQLListener creates a new PostgreSQL listener
func NewPostgreSQLListener(connectionString string) (*PostgreSQLListener, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgreSQLListener{
		db:     db,
		logger: log.Default(),
	}, nil
}

// StartListening starts listening for PostgreSQL NOTIFY events
func (pl *PostgreSQLListener) StartListening(ctx context.Context) error {
	// Listen to both notification channels
	_, err := pl.db.ExecContext(ctx, "LISTEN driver_events")
	if err != nil {
		return fmt.Errorf("failed to listen to driver_events: %w", err)
	}

	_, err = pl.db.ExecContext(ctx, "LISTEN vault_catchup_events")
	if err != nil {
		return fmt.Errorf("failed to listen to vault_catchup_events: %w", err)
	}

	pl.logger.Println("Started listening for PostgreSQL NOTIFY events on 'driver_events' and 'vault_catchup_events' channels")

	// Start notification handler
	go pl.handleNotifications(ctx)

	return nil
}

// handleNotifications processes incoming PostgreSQL notifications
func (pl *PostgreSQLListener) handleNotifications(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			pl.logger.Println("Stopping PostgreSQL notification listener")
			return
		default:
			// Wait for notification
			notification, err := pl.db.Conn(ctx)
			if err != nil {
				pl.logger.Printf("Error getting connection: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Listen for notifications
			notification.ExecContext(ctx, "LISTEN driver_events")

			// This is a simplified example - in production you'd use a proper notification listener
			// like github.com/lib/pq's notification system
			pl.logger.Println("Waiting for notifications...")
			time.Sleep(1 * time.Second)
		}
	}
}
