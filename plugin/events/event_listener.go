package events

import (
	"junoplugin/models"
	"log"
)

// EventListener provides an interface for listening to driver notifications
// This is EXAMPLE CODE showing how the event-processor should integrate with the logger
type EventListener struct {
	driverNotificationChan   <-chan models.DriverEvent
	vaultCatchupNotificationChan  <-chan models.VaultCatchupEvent
	log               *log.Logger
}

// NewEventListener creates a new event listener
func NewEventListener(
	driverNotificationChan <-chan models.DriverEvent,
	vaultCatchupNotificationChan <-chan models.VaultCatchupEvent,
) *EventListener {
	return &EventListener{
		driverNotificationChan:  driverNotificationChan,
		vaultCatchupNotificationChan: vaultCatchupNotificationChan,
		log:              log.Default(),
	}
}

// StartListening starts listening for notifications and calls the provided handlers
func (el *EventListener) StartListening(
	driverEventHandler func(models.DriverEvent),
	vaultCatchupHandler func(models.VaultCatchupEvent),
) {
	go func() {
		for {
			select {
			case event := <-el.driverNotificationChan:
				el.log.Printf("Received driver notification: %s for block %d", event.Type, event.BlockNumber)
				if driverEventHandler != nil {
					driverEventHandler(event)
				}
			case event := <-el.vaultCatchupNotificationChan:
				el.log.Printf("Received vault catchup notification for vault %s, blocks %d-%d", 
					event.VaultAddress, event.StartBlock, event.EndBlock)
				if vaultCatchupHandler != nil {
					vaultCatchupHandler(event)
				}
			}
		}
	}()
}

// Example usage for event-processor:
/*
func main() {
	// Get notification channels from your block processor
	driverNotificationChan := blockProcessor.GetDriverNotificationChannel()
	vaultNotificationChan := blockProcessor.GetVaultCatchupNotificationChannel()
	
	// Create event listener
	listener := events.NewEventListener(driverNotificationChan, vaultNotificationChan)
	
	// Define your handlers
	driverHandler := func(event models.DriverEvent) {
		switch event.Type {
		case "StartBlock":
			// Handle new block processed
			fmt.Printf("Block %d processed successfully\n", event.BlockNumber)
		case "RevertBlock":
			// Handle block reverted
			fmt.Printf("Block %d was reverted\n", event.BlockNumber)
		case "CatchupBlock":
			// Handle catchup block processed
			fmt.Printf("Catchup block %d processed successfully\n", event.BlockNumber)
		}
	}
	
	vaultHandler := func(event models.VaultCatchupEvent) {
		// Handle vault catchup
		fmt.Printf("Catchup vault %s for blocks %d-%d\n", 
			event.VaultAddress, event.StartBlock, event.EndBlock)
	}
	
	// Start listening for notifications
	listener.StartListening(driverHandler, vaultHandler)
	
	// Keep the program running
	select {}
}
*/
