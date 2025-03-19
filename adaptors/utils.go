package adaptors

import (
	"fmt"
	"junoplugin/models"
	"math/big"

	"github.com/NethermindEth/juno/core"
	"golang.org/x/crypto/sha3"
)

func CombineFeltToBigInt(highFelt, lowFelt [32]byte) models.BigInt {
	combinedBytes := make([]byte, 64) // 32 bytes for highFelt and 32 bytes for lowFelt

	// Copy highFelt into the first 32 bytes
	copy(combinedBytes[0:32], highFelt[:])

	// Copy lowFelt into the next 32 bytes
	copy(combinedBytes[32:64], lowFelt[:])

	// Convert the combined bytes to a big.Int
	combinedInt := models.BigInt{Int: new(big.Int).SetBytes(combinedBytes)}

	return combinedInt
}

func FeltToBigInt(felt [32]byte) models.BigInt {

	byteData := make([]byte, 32)
	copy(byteData[:], felt[:])
	return models.BigInt{Int: new(big.Int).SetBytes(byteData)}
}
func FeltToHexString(felt [32]byte) string {

	combinedInt := models.BigInt{Int: new(big.Int).SetBytes(felt[:])}
	// Assuming `f.Value` holds the *big.Int representation of the felt
	return "0x" + combinedInt.Text(16)
}

func BigIntToHexString(f big.Int) string {

	// Assuming `f.Value` holds the *big.Int representation of the felt
	return "0x" + f.Text(16)
}

func DecimalStringToHexString(decimalString string) (string, error) {
	// Create a new big.Int and set its value from the decimal string
	num := new(big.Int)
	_, success := num.SetString(decimalString, 10)
	if !success {
		return "", fmt.Errorf("invalid decimal string")
	}

	// Convert the big.Int to a hexadecimal string
	hexString := num.Text(16)

	// Add "0x" prefix for clarity
	return "0x" + hexString, nil
}

var vaultEventNames = []string{
	"Deposit",
	"Withdrawal",
	"WithdrawalQueued",
	"StashWithdrawn",
	"OptionRoundDeployed",
	"L1RequestFulfilled",
	"PricingDataSet",
	"AuctionStarted",
	"AuctionEnded",
	"OptionRoundSettled",
	"BidPlaced",
	"BidUpdated",
	"UnusedBidsRefunded",
	"OptionsMinted",
	"OptionsExercised",
}

// var roundEventNames = []string{

// 	"AuctionStarted",
// 	"BidPlaced",
// 	"BidUpdated",
// 	"AuctionEnded",
// 	"OptionRoundSettled",
// 	"OptionsExercised",
// 	"OptionsMinted",
// 	"UnusedBidsRefunded",
// }

// keccak256 function to hash the event name
func Keccak256(eventName string) string {
	hasher := sha3.NewLegacyKeccak256()

	// Write the event name as bytes to the hasher
	hasher.Write([]byte(eventName))

	// Compute the full 256-bit hash
	hashBytes := hasher.Sum(nil)

	// Convert the hash to a big integer
	hashInt := new(big.Int).SetBytes(hashBytes)

	// Apply a 250-bit mask to fit StarkNet's felt requirements
	mask := new(big.Int).Lsh(big.NewInt(1), 250)
	mask.Sub(mask, big.NewInt(1))
	hashInt.And(hashInt, mask)

	// Convert the masked hash to a hexadecimal string with "0x" prefix
	return "0x" + hashInt.Text(16)
}

// DecodeEventName decodes the event name from the keys of a StarkNet event
// func DecodeEventNameRound(eventKey string) (string, error) {
// 	for _, name := range roundEventNames {
// 		if Keccak256(name) == eventKey {
// 			return name, nil
// 		}
// 	}
// 	return "", fmt.Errorf("event name not found for key: %s", eventKey)
// }

func DecodeEventNameVault(eventKey string) (string, error) {
	for _, name := range vaultEventNames {
		if Keccak256(name) == eventKey {
			return name, nil
		}
	}
	return "", fmt.Errorf("event name not found for key: %s", eventKey)
}

// EventToStringArrays converts a Juno core.Event keys and data to string arrays
func EventToStringArrays(event core.Event) ([]string, []string) {
	keys := make([]string, len(event.Keys))
	data := make([]string, len(event.Data))

	// Convert keys to strings
	for i, key := range event.Keys {
		keys[i] = key.String()
	}

	// Convert data to strings
	for i, dataItem := range event.Data {
		data[i] = dataItem.String()
	}

	return keys, data
}
