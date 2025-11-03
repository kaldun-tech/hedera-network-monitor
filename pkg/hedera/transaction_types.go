package hedera

import hiero "github.com/hiero-ledger/hiero-sdk-go/v2/sdk"

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

// GetTransactionType determines the transaction type from a Hedera TransactionRecord
func GetTransactionType(rec *hiero.TransactionRecord) TransactionType {
	if rec.CallResult != nil {
		if rec.CallResultIsCreate {
			return TransactionTypeContractCreate
		}
		return TransactionTypeContractCall
	}
	if len(rec.TokenTransfers) > 0 {
		return TransactionTypeTokenTransfer
	}
	if len(rec.Transfers) > 0 {
		return TransactionTypeCryptoTransfer
	}
	if rec.Receipt.TopicID != nil {
		return TransactionTypeConsensusSubmitMessage
	}
	if rec.Receipt.FileID != nil {
		return TransactionTypeFileOperation
	}
	return TransactionTypeUnknown
}
