package events

import (
	"junoplugin/models"
	"log"
)

// EventListener provides an interface for listening to driver events
// This is EXAMPLE CODE showing how the event-processor should integrate with the logger
type EventListener struct {
	driverEventChan   <-chan models.DriverEvent
	vaultCatchupChan  <-chan models.VaultCatchupEvent
	log               *log.Logger
}

// NewEventListener creates a new event listener
func NewEventListener(
	driverEventChan <-chan models.DriverEvent,
	vaultCatchupChan <-chan models.VaultCatchupEvent,
) *EventListener {
	return &EventListener{
		driverEventChan:  driverEventChan,
		vaultCatchupChan: vaultCatchupChan,
		log:              log.Default(),
	}
}

// StartListening starts listening for events and calls the provided handlers
func (el *EventListener) StartListening(
	driverEventHandler func(models.DriverEvent),
	vaultCatchupHandler func(models.VaultCatchupEvent),
) {
	go func() {
		for {
			select {
			case event := <-el.driverEventChan:
				el.log.Printf("Received driver event: %s for block %d", event.Type, event.BlockNumber)
				if driverEventHandler != nil {
					driverEventHandler(event)
				}
			case event := <-el.vaultCatchupChan:
				el.log.Printf("Received vault catchup event for vault %s, blocks %d-%d", 
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
	// Get channels from your block processor
	driverChan := blockProcessor.GetDriverEventChannel()
	vaultChan := blockProcessor.GetVaultCatchupEventChannel()
	
	// Create event listener
	listener := events.NewEventListener(driverChan, vaultChan)
	
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
	
	// Start listening
	listener.StartListening(driverHandler, vaultHandler)
	
	// Keep the program running
	select {}
}
*/
