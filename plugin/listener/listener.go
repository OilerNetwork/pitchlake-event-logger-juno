package listener

import (
	"junoplugin/db"
	"junoplugin/models"
	"junoplugin/plugin/vault"
	"log"
)

// Service handles listening for new vault registrations
type Service struct {
	db           *db.DB
	vaultManager *vault.Manager
	channel      chan models.VaultRegistry
	log          *log.Logger
}

// NewService creates a new listener service
func NewService(db *db.DB, vaultManager *vault.Manager) *Service {
	return &Service{
		db:           db,
		vaultManager: vaultManager,
		channel:      make(chan models.VaultRegistry),
		log:          log.Default(),
	}
}

// Start starts the listener service
func (ls *Service) Start() error {
	ls.log.Println("Starting vault registry listener")

	// Start listening for new vaults in a goroutine
	go ls.listen()

	return nil
}

// listen listens for new vault registrations
func (ls *Service) listen() {
	ls.db.ListenerNewVault(ls.channel)

	for {
		select {
		case vault := <-ls.channel:
			ls.log.Printf("Received new vault registration: %s", vault.Address)
			if err := ls.vaultManager.InitializeVault(vault); err != nil {
				ls.log.Printf("Error initializing vault %s: %v", vault.Address, err)
			}
		}
	}
}

// Stop stops the listener service
func (ls *Service) Stop() {
	ls.log.Println("Stopping vault registry listener")
	close(ls.channel)
}
