// Code generated by suave/gen. DO NOT EDIT.
// Hash: efd6a12efee9ddaebe46473d62e19e2306f1e598b2362b46cba199580d9bef7c
package types

import "github.com/ethereum/go-ethereum/common"

type BidId [16]byte

// Structs

type Bid struct {
	Id                  BidId
	Salt                BidId
	DecryptionCondition uint64
	AllowedPeekers      []common.Address
	AllowedStores       []common.Address
	Version             string
}

type BuildBlockArgs struct {
	Slot           uint64
	ProposerPubkey []byte
	Parent         common.Hash
	Timestamp      uint64
	FeeRecipient   common.Address
	GasLimit       uint64
	Random         common.Hash
	Withdrawals    []*Withdrawal
	Extra          []byte
}

type Bundle struct {
	Transactions [][]byte
	BlockNumber  uint64
}

type MevShareBundle struct {
	Transactions   [][]byte
	InclusionBlock uint64
	RefundPercents []uint8
}

type STransaction struct {
	Nonce    uint64
	GasPrice uint64
	GasLimit uint64
	To       common.Address
	Value    uint64
	Data     []byte
	V        []byte
	R        []byte
	S        []byte
}

type Withdrawal struct {
	Index     uint64
	Validator uint64
	Address   common.Address
	Amount    uint64
}
