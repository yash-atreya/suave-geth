package vm

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

var (
	confStorePrecompileStoreMeter    = metrics.NewRegisteredMeter("suave/confstore/store", nil)
	confStorePrecompileRetrieveMeter = metrics.NewRegisteredMeter("suave/confstore/retrieve", nil)
)

var (
	isConfidentialAddress               = common.HexToAddress("0x42010000")
	errIsConfidentialInvalidInputLength = errors.New("invalid input length")

	confidentialInputsAddress = common.HexToAddress("0x42010001")

	confStoreStoreAddress    = common.HexToAddress("0x42020000")
	confStoreRetrieveAddress = common.HexToAddress("0x42020001")

	newBidAddress    = common.HexToAddress("0x42030000")
	fetchBidsAddress = common.HexToAddress("0x42030001")
)

/* General utility precompiles */

type isConfidentialPrecompile struct{}

func (c *isConfidentialPrecompile) RequiredGas(input []byte) uint64 {
	return 0 // incurs only the call cost (100)
}

func (c *isConfidentialPrecompile) Name() string {
	return "isConfidential"
}

func (c *isConfidentialPrecompile) Address() common.Address {
	return isConfidentialAddress
}

func (c *isConfidentialPrecompile) Do(suaveContext *SuaveContext) ([]byte, error) {
	fmt.Println("? DO ?")
	return []byte{0x1}, nil
}

func (c *isConfidentialPrecompile) Run(input []byte) ([]byte, error) {
	fmt.Println("? RUN ?")
	if len(input) == 1 {
		// The precompile was called *directly* confidentially, and the result was cached - return 1
		if input[0] == 0x01 {
			return []byte{0x01}, nil
		} else {
			return nil, errors.New("incorrect value passed in")
		}
	}

	if len(input) > 1 {
		return nil, errIsConfidentialInvalidInputLength
	}

	return []byte{0x00}, nil
}

func (c *isConfidentialPrecompile) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	if len(input) != 0 {
		return nil, errIsConfidentialInvalidInputLength
	}
	return []byte{0x01}, nil
}

type confidentialInputsPrecompile struct{}

func (c *confidentialInputsPrecompile) RequiredGas(input []byte) uint64 {
	return 0 // incurs only the call cost (100)
}

func (c *confidentialInputsPrecompile) Run(input []byte) ([]byte, error) {
	return nil, errors.New("not available in this suaveContext")
}

func (c *confidentialInputsPrecompile) Address() common.Address {
	return confidentialInputsAddress
}

func (c *confidentialInputsPrecompile) Name() string {
	return "confidentialInputs"
}

func (c *confidentialInputsPrecompile) Do(suaveContext *SuaveContext) ([]byte, error) {
	return suaveContext.ConfidentialInputs, nil
}

func (c *confidentialInputsPrecompile) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	return suaveContext.ConfidentialInputs, nil
}

/* Confidential store precompiles */

type confStoreStore struct {
	inoutAbi abi.Method
}

func newConfStoreStore() *confStoreStore {
	inoutAbi := mustParseMethodAbi(`[{"inputs":[{"type":"bytes16"}, {"type":"bytes16"}, {"type":"string"}, {"type":"bytes"}],"name":"store","outputs":[],"stateMutability":"nonpayable","type":"function"}]`, "store")

	return &confStoreStore{inoutAbi}
}

func (c *confStoreStore) RequiredGas(input []byte) uint64 {
	return uint64(100 * len(input))
}

func (c *confStoreStore) Run(input []byte) ([]byte, error) {
	return nil, errors.New("not available in this suaveContext")
}

func (c *confStoreStore) Name() string {
	return "confidentialStoreStore"
}

func (c *confStoreStore) Address() common.Address {
	return confStoreStoreAddress
}

func (c *confStoreStore) Do(suaveContext *SuaveContext, bidId suave.BidId, key string, data []byte) error {
	return c.runImpl(suaveContext, bidId, key, data)
}

func (c *confStoreStore) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	if len(suaveContext.CallerStack) == 0 {
		return []byte("not allowed"), errors.New("not allowed in this suaveContext")
	}

	unpacked, err := c.inoutAbi.Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}

	bidId := unpacked[0].(types.BidId)
	key := unpacked[1].(string)
	data := unpacked[2].([]byte)

	if err := c.runImpl(suaveContext, bidId, key, data); err != nil {
		return []byte(err.Error()), err
	}
	return nil, nil
}

func (c *confStoreStore) runImpl(suaveContext *SuaveContext, bidId suave.BidId, key string, data []byte) error {
	if len(suaveContext.CallerStack) == 0 {
		return errors.New("not allowed in this suaveContext")
	}

	// Can be zeroes in some fringe cases!
	var caller common.Address
	for i := len(suaveContext.CallerStack) - 1; i >= 0; i-- {
		// Most recent non-nil non-this caller
		if _c := suaveContext.CallerStack[i]; _c != nil && *_c != confStoreStoreAddress {
			caller = *_c
			break
		}
	}

	if metrics.Enabled {
		confStorePrecompileStoreMeter.Mark(int64(len(data)))
	}

	_, err := suaveContext.Backend.ConfidentialStore.Store(bidId, caller, key, data)
	if err != nil {
		return err
	}

	return nil
}

type confStoreRetrieve struct {
	inoutAbi abi.Method
}

func newConfStoreRetrieve() *confStoreRetrieve {
	inoutAbi := mustParseMethodAbi(`[{"inputs":[{"type":"bytes16"}, {"type":"bytes16"}, {"type":"string"}],"name":"retrieve","outputs":[{"type":"bytes"}],"stateMutability":"nonpayable","type":"function"}]`, "retrieve")

	return &confStoreRetrieve{inoutAbi}
}

func (c *confStoreRetrieve) RequiredGas(input []byte) uint64 {
	return 100
}

func (c *confStoreRetrieve) Run(input []byte) ([]byte, error) {
	return nil, errors.New("not available in this suaveContext")
}

func (c *confStoreRetrieve) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	if len(suaveContext.CallerStack) == 0 {
		return []byte("not allowed"), errors.New("not allowed in this suaveContext")
	}

	unpacked, err := c.inoutAbi.Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}

	bidId := unpacked[0].(suave.BidId)
	key := unpacked[1].(string)

	return c.runImpl(suaveContext, bidId, key)
}

func (c *confStoreRetrieve) Address() common.Address {
	return confStoreRetrieveAddress
}

func (c *confStoreRetrieve) Do(suaveContext *SuaveContext, bidId suave.BidId, key string) ([]byte, error) {
	return c.runImpl(suaveContext, bidId, key)
}

func (c *confStoreRetrieve) Name() string {
	return "confidentialStoreRetrieve"
}

func (c *confStoreRetrieve) runImpl(suaveContext *SuaveContext, bidId suave.BidId, key string) ([]byte, error) {
	if len(suaveContext.CallerStack) == 0 {
		return nil, errors.New("not allowed in this suaveContext")
	}

	log.Info("confStoreRetrieve", "bidId", bidId, "key", key)

	// Can be zeroes in some fringe cases!
	var caller common.Address
	for i := len(suaveContext.CallerStack) - 1; i >= 0; i-- {
		// Most recent non-nil non-this caller
		if _c := suaveContext.CallerStack[i]; _c != nil && *_c != confStoreRetrieveAddress {
			caller = *_c
			break
		}
	}

	data, err := suaveContext.Backend.ConfidentialStore.Retrieve(bidId, caller, key)
	if err != nil {
		return []byte(err.Error()), err
	}

	if metrics.Enabled {
		confStorePrecompileRetrieveMeter.Mark(int64(len(data)))
	}

	return data, nil
}

/* Bid precompiles */

type newBid struct {
	inoutAbi abi.Method
}

func newNewBid() *newBid {
	inoutAbi := mustParseMethodAbi(`[{ "inputs": [ { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" }, { "internalType": "string", "name": "BidType", "type": "string" } ], "name": "newBid", "outputs": [ { "components": [ { "internalType": "Suave.BidId", "name": "id", "type": "bytes16" }, { "internalType": "Suave.BidId", "name": "salt", "type": "bytes16" }, { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" } ], "internalType": "struct Suave.Bid", "name": "", "type": "tuple" } ], "stateMutability": "view", "type": "function" }]`, "newBid")

	return &newBid{inoutAbi}
}

func (c *newBid) RequiredGas(input []byte) uint64 {
	return 1000
}

func (c *newBid) Run(input []byte) ([]byte, error) {
	return input, nil
}

func (c *newBid) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	unpacked, err := c.inoutAbi.Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}
	version := unpacked[2].(string)

	decryptionCondition := unpacked[0].(uint64)
	allowedPeekers := unpacked[1].([]common.Address)

	bid, err := c.runImpl(suaveContext, version, decryptionCondition, allowedPeekers, []common.Address{})
	if err != nil {
		return []byte(err.Error()), err
	}

	return c.inoutAbi.Outputs.Pack(bid)
}

func (c *newBid) Address() common.Address {
	return newBidAddress
}

func (c *newBid) Do(suaveContext *SuaveContext, decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address, version string) (*types.Bid, error) {
	return c.runImpl(suaveContext, version, decryptionCondition, allowedPeekers, allowedStores)
}

func (c *newBid) runImpl(suaveContext *SuaveContext, version string, decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address) (*types.Bid, error) {
	if suaveContext.ConfidentialComputeRequestTx == nil {
		panic("newBid: source transaction not present")
	}

	bid, err := suaveContext.Backend.ConfidentialStore.InitializeBid(types.Bid{
		Salt:                suave.RandomBidId(),
		DecryptionCondition: decryptionCondition,
		AllowedPeekers:      allowedPeekers,
		AllowedStores:       allowedStores,
		Version:             version, // TODO : make generic
	})
	if err != nil {
		return nil, err
	}

	return &bid, nil
}

type fetchBids struct {
	inoutAbi abi.Method
}

func newFetchBids() *fetchBids {
	inoutAbi := mustParseMethodAbi(`[ { "inputs": [ { "internalType": "uint64", "name": "cond", "type": "uint64" }, { "internalType": "string", "name": "namespace", "type": "string" } ], "name": "fetchBids", "outputs": [ { "components": [ { "internalType": "Suave.BidId", "name": "id", "type": "bytes16" }, { "internalType": "Suave.BidId", "name": "salt", "type": "bytes16" }, { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" }, { "internalType": "address[]", "name": "allowedStores", "type": "address[]" }, { "internalType": "string", "name": "version", "type": "string" } ], "internalType": "struct Suave.Bid[]", "name": "", "type": "tuple[]" } ], "stateMutability": "view", "type": "function" } ]`, "fetchBids")

	return &fetchBids{inoutAbi}
}

func (c *fetchBids) RequiredGas(input []byte) uint64 {
	return 1000
}

func (c *fetchBids) Run(input []byte) ([]byte, error) {
	return input, nil
}

func (c *fetchBids) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	unpacked, err := c.inoutAbi.Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}

	targetBlock := unpacked[0].(uint64)
	namespace := unpacked[1].(string)

	bids, err := c.runImpl(suaveContext, targetBlock, namespace)
	if err != nil {
		return []byte(err.Error()), err
	}

	return c.inoutAbi.Outputs.Pack(bids)
}

func (c *fetchBids) Address() common.Address {
	return fetchBidsAddress
}

func (c *fetchBids) Do(suaveContext *SuaveContext, targetBlock uint64, namespace string) ([]types.Bid, error) {
	return c.runImpl(suaveContext, targetBlock, namespace)
}

func (c *fetchBids) runImpl(suaveContext *SuaveContext, targetBlock uint64, namespace string) ([]types.Bid, error) {
	bids1 := suaveContext.Backend.ConfidentialStore.FetchBidsByProtocolAndBlock(targetBlock, namespace)

	fmt.Println("-- bids --", targetBlock, namespace, bids1)

	bids := make([]types.Bid, 0, len(bids1))
	for _, bid := range bids1 {
		bids = append(bids, bid.ToInnerBid())
	}

	return bids, nil
}

func mustParseAbi(data string) abi.ABI {
	inoutAbi, err := abi.JSON(strings.NewReader(data))
	if err != nil {
		panic(err.Error())
	}

	return inoutAbi
}

func mustParseMethodAbi(data string, method string) abi.Method {
	inoutAbi := mustParseAbi(data)
	return inoutAbi.Methods[method]
}

func formatPeekerError(format string, args ...any) ([]byte, error) {
	err := fmt.Errorf(format, args...)
	return []byte(err.Error()), err
}

type suaveRuntime struct {
	suaveContext *SuaveContext
}

var _ SuaveRuntime = &suaveRuntime{}

func (b *suaveRuntime) ethcall(contractAddr common.Address, input []byte) ([]byte, error) {
	return (&ethCallPrecompile{}).runImpl(b.suaveContext, contractAddr, input)
}

func (b *suaveRuntime) buildEthBlock(blockArgs types.BuildBlockArgs, bid types.BidId, namespace string) ([]byte, []byte, error) {
	return (&buildEthBlock{}).runImpl(b.suaveContext, blockArgs, bid, namespace)
}

func (b *suaveRuntime) confidentialInputs() ([]byte, error) {
	return nil, nil
}

func (b *suaveRuntime) confidentialStoreRetrieve(bidId types.BidId, key string) ([]byte, error) {
	return (&confStoreRetrieve{}).runImpl(b.suaveContext, bidId, key)
}

func (b *suaveRuntime) confidentialStoreStore(bidId types.BidId, key string, data []byte) error {
	return (&confStoreStore{}).runImpl(b.suaveContext, bidId, key, data)
}

func (b *suaveRuntime) extractHint(bundleData []byte) ([]byte, error) {
	return (&extractHint{}).runImpl(b.suaveContext, bundleData)
}

func (b *suaveRuntime) fetchBids(cond uint64, namespace string) ([]types.Bid, error) {
	bids, err := (&fetchBids{}).runImpl(b.suaveContext, cond, namespace)
	if err != nil {
		return nil, err
	}
	return bids, nil
}

func (b *suaveRuntime) newBid(decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address, BidType string) (types.Bid, error) {
	bid, err := (&newBid{}).runImpl(b.suaveContext, BidType, decryptionCondition, allowedPeekers, allowedStores)
	if err != nil {
		return types.Bid{}, err
	}
	return *bid, nil
}

func (b *suaveRuntime) simulateBundle(bundleData []byte) (uint64, error) {
	num, err := (&simulateBundle{}).runImpl(b.suaveContext, bundleData)
	if err != nil {
		return 0, err
	}
	return num, nil
}

func (b *suaveRuntime) submitEthBlockBidToRelay(relayUrl string, builderBid []byte) ([]byte, error) {
	return (&submitEthBlockBidToRelay{}).runImpl(b.suaveContext, relayUrl, builderBid)
}
