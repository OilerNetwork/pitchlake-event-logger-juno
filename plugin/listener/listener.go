package listener

import (
	"context"
	"encoding/json"
	"fmt"
	"junoplugin/models"
	"junoplugin/plugin/vault"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

// Service handles listening for new vault registrations
type Service struct {
	conn         *pgx.Conn
	vaultManager *vault.Manager
	channel      chan models.VaultRegistry
	log          *log.Logger
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewService creates a new listener service
func NewListenerService(vaultManager *vault.Manager) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	dbUrl := os.Getenv("DB_URL")
	conn, err := pgx.Connect(ctx, dbUrl)
	if err != nil {
		fmt.Errorf("unable to connect to database: %w", err)
	}
	return &Service{
		vaultManager: vaultManager,
		channel:      make(chan models.VaultRegistry),
		log:          log.Default(),
		ctx:          ctx,
		cancel:       cancel,
		conn:         conn,
	}
}

// Start starts the listener service
func (ls *Service) Start() error {
	ls.log.Println("Starting vault registry listener")
	if ls.conn == nil {
		return nil
	}
	// Start listening for new vaults in a goroutine
	go ls.listen()

	return nil
}

// listen listens for new vault registrations
func (ls *Service) listen() {
	ls.log.Println("Starting to listen for vault notifications...")

	// Start the database listener in a goroutine
	go ls.ListenerNewVault(ls.channel)

	var err error
	for {
		select {
		case <-ls.ctx.Done():
			ls.log.Println("Listener context cancelled, shutting down")
			return
		case vault := <-ls.channel:
			ls.log.Printf("Received new vault registration: %s", vault.Address)
			if err = ls.vaultManager.InitializeVault(&vault); err != nil {
				ls.log.Printf("Error initializing vault %s: %v", vault.Address, err)
				return
			} else {
				ls.log.Printf("Successfully initialized vault: %s", vault.Address)
			}
		}
	}
}

// Stop stops the listener service
func (ls *Service) Stop() {
	ls.log.Println("Stopping vault registry listener")
	ls.cancel() // This will signal the context to cancel
	close(ls.channel)
}

func (ls *Service) ListenerNewVault(channel chan<- models.VaultRegistry) {
	_, err := ls.conn.Exec(context.Background(), "LISTEN vault_insert")
	if err != nil {
		log.Printf("Failed to start listening: %v", err)
		return
	}

	for {
		notification, err := ls.conn.WaitForNotification(context.Background())
		if err != nil {
			log.Printf("Error waiting for notification: %v", err)
			continue
		}

		var vault models.VaultRegistry
		if err := json.Unmarshal([]byte(notification.Payload), &vault); err != nil {
			log.Printf("Error unmarshaling vault data: %v", err)
			continue
		}

		log.Printf("Received vault notification: %s", vault.Address)
		channel <- vault
	}
}
