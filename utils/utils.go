package utils

import (
	"fmt"
	"junoplugin/models"
	"math/big"
	"strings"

	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/felt"
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

// HexStringToFelt properly converts a hex string to bytes
func HexStringToFelt(hexStr string) ([]byte, error) {
	// Remove 0x prefix if present
	if len(hexStr) >= 2 && hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}

	// Create a new big.Int from the hex string
	bigInt := new(big.Int)
	_, success := bigInt.SetString(hexStr, 16)
	if !success {
		return nil, fmt.Errorf("invalid hex string: %s", hexStr)
	}

	// Return the bytes representation directly
	return bigInt.Bytes(), nil
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

func FeltArrayToStringArrays(feltArray []*felt.Felt) []string {
	strArray := make([]string, len(feltArray))

	for i, felt := range feltArray {
		strArray[i] = felt.String()
	}

	return strArray
}

// Func to remove 0 padding after 0x,
//
//	Address: 0x50aa16a833664c92d4163b14fed470786fa4411ffd3b3addbb97a70ae56efbd
//	Vault address: 0x050aa16a833664c92d4163b14fed470786fa4411ffd3b3addbb97a70ae56efbd
//	These 2 addresses should match

// NormalizeHexAddress removes leading zeros after 0x prefix
// Example:
// Input:  "0x050aa16a833664c92d4163b14fed470786fa4411ffd3b3addbb97a70ae56efbd"
// Output: "0x50aa16a833664c92d4163b14fed470786fa4411ffd3b3addbb97a70ae56efbd"
func NormalizeHexAddress(hexStr string) (string, error) {
	if !strings.HasPrefix(hexStr, "0x") {
		return "", fmt.Errorf("hex string must start with 0x prefix")
	}

	// Remove 0x and leading zeros, but keep at least one digit
	trimmed := strings.TrimLeft(hexStr[2:], "0")
	if trimmed == "" {
		return "0x0", nil
	}

	return "0x" + trimmed, nil
}
