package events

// EventSender interface for sending driver events
type EventSender interface {
	SendVaultCatchupEvent(vaultAddress string, startBlock, endBlock uint64)
}

// VaultCatchupEventSender implements EventSender for vault catchup events
type VaultCatchupEventSender struct {
	sendFunc func(string, uint64, uint64)
}

// NewVaultCatchupEventSender creates a new event sender
func NewVaultCatchupEventSender(sendFunc func(string, uint64, uint64)) *VaultCatchupEventSender {
	return &VaultCatchupEventSender{
		sendFunc: sendFunc,
	}
}

// SendVaultCatchupEvent sends a vault catchup event
func (v *VaultCatchupEventSender) SendVaultCatchupEvent(vaultAddress string, startBlock, endBlock uint64) {
	v.sendFunc(vaultAddress, startBlock, endBlock)
}
