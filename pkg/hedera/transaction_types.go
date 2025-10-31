package hedera

// TransactionType represents the different types of Hedera transactions
type TransactionType string

const (
	// HBAR transfers between accounts
	TransactionTypeCryptoTransfer TransactionType = "CryptoTransfer"

	// Token transfers (fungible tokens)
	TransactionTypeTokenTransfer TransactionType = "TokenTransfer"

	// Smart contract creation
	TransactionTypeContractCreate TransactionType = "ContractCreate"

	// Smart contract execution/call
	TransactionTypeContractCall TransactionType = "ContractCall"

	// Consensus service message submission
	TransactionTypeConsensusSubmitMessage TransactionType = "ConsensusSubmitMessage"

	// File operations
	TransactionTypeFileOperation TransactionType = "FileOperation"

	// Unknown or unsupported transaction type
	TransactionTypeUnknown TransactionType = "Unknown"
)

// String returns the string representation of the transaction type
func (t TransactionType) String() string {
	return string(t)
}

// IsValid checks if a transaction type is a known type
func (t TransactionType) IsValid() bool {
	switch t {
	case TransactionTypeCryptoTransfer,
		TransactionTypeTokenTransfer,
		TransactionTypeContractCreate,
		TransactionTypeContractCall,
		TransactionTypeConsensusSubmitMessage,
		TransactionTypeFileOperation,
		TransactionTypeUnknown:
		return true
	default:
		return false
	}
}
