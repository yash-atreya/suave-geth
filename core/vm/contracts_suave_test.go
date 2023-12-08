package vm

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/ethereum/go-ethereum/suave/cstore"
	"github.com/stretchr/testify/require"
)

type mockSuaveBackend struct {
}

func (m *mockSuaveBackend) Start() error { return nil }
func (m *mockSuaveBackend) Stop() error  { return nil }

func (m *mockSuaveBackend) InitializeBid(bid suave.Bid) error {
	return nil
}

func (m *mockSuaveBackend) Store(bid suave.Bid, caller common.Address, key string, value []byte) (suave.Bid, error) {
	return suave.Bid{}, nil
}

func (m *mockSuaveBackend) Retrieve(bid suave.Bid, caller common.Address, key string) ([]byte, error) {
	return nil, nil
}

func (m *mockSuaveBackend) SubmitBid(types.Bid) error {
	return nil
}

func (m *mockSuaveBackend) FetchEngineBidById(suave.BidId) (suave.Bid, error) {
	return suave.Bid{}, nil
}

func (m *mockSuaveBackend) FetchBidById(suave.BidId) (suave.Bid, error) {
	return suave.Bid{}, nil
}

func (m *mockSuaveBackend) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.Bid {
	return nil
}

func (m *mockSuaveBackend) BuildEthBlock(ctx context.Context, args *suave.BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error) {
	return nil, nil
}

func (m *mockSuaveBackend) BuildEthBlockFromBundles(ctx context.Context, args *suave.BuildBlockArgs, bundles []types.SBundle) (*engine.ExecutionPayloadEnvelope, error) {
	return nil, nil
}

func (m *mockSuaveBackend) Call(ctx context.Context, contractAddr common.Address, input []byte) ([]byte, error) {
	return nil, nil
}

func (m *mockSuaveBackend) Subscribe() (<-chan cstore.DAMessage, context.CancelFunc) {
	return nil, func() {}
}

func (m *mockSuaveBackend) Publish(cstore.DAMessage) {}

func newTestBackend(t *testing.T) *suaveRuntime {
	confStore := cstore.NewLocalConfidentialStore()
	confEngine := cstore.NewConfidentialStoreEngine(confStore, &cstore.MockTransport{}, cstore.MockSigner{}, cstore.MockChainSigner{})

	require.NoError(t, confEngine.Start())
	t.Cleanup(func() { confEngine.Stop() })

	reqTx := types.NewTx(&types.ConfidentialComputeRequest{
		ConfidentialComputeRecord: types.ConfidentialComputeRecord{
			KettleAddress: common.Address{},
		},
	})

	b := &suaveRuntime{
		suaveContext: &SuaveContext{
			Backend: &SuaveExecutionBackend{
				ConfidentialStore:      confEngine.NewTransactionalStore(reqTx),
				ConfidentialEthBackend: &mockSuaveBackend{},
			},
			ConfidentialComputeRequestTx: reqTx,
		},
	}
	return b
}

func TestSuave_BidWorkflow(t *testing.T) {
	b := newTestBackend(t)

	bid5, err := b.newBid(5, []common.Address{{0x1}}, nil, "a")
	require.NoError(t, err)

	bid10, err := b.newBid(10, []common.Address{{0x1}}, nil, "a")
	require.NoError(t, err)

	bid10b, err := b.newBid(10, []common.Address{{0x1}}, nil, "a")
	require.NoError(t, err)

	cases := []struct {
		cond      uint64
		namespace string
		bids      []types.Bid
	}{
		{0, "a", []types.Bid{}},
		{5, "a", []types.Bid{bid5}},
		{10, "a", []types.Bid{bid10, bid10b}},
		{11, "a", []types.Bid{}},
	}

	for _, c := range cases {
		bids, err := b.fetchBids(c.cond, c.namespace)
		require.NoError(t, err)

		require.ElementsMatch(t, c.bids, bids)
	}
}

func TestSuave_ConfStoreWorkflow(t *testing.T) {
	b := newTestBackend(t)

	callerAddr := common.Address{0x1}
	data := []byte{0x1}

	// cannot store a value for a bid that does not exist
	err := b.confidentialStore(types.BidId{}, "key", data)
	require.Error(t, err)

	bid, err := b.newBid(5, []common.Address{callerAddr}, nil, "a")
	require.NoError(t, err)

	// cannot store the bid if the caller is not allowed to
	err = b.confidentialStore(bid.Id, "key", data)
	require.Error(t, err)

	// now, the caller is allowed to store the bid
	b.suaveContext.CallerStack = append(b.suaveContext.CallerStack, &callerAddr)
	err = b.confidentialStore(bid.Id, "key", data)
	require.NoError(t, err)

	val, err := b.confidentialRetrieve(bid.Id, "key")
	require.NoError(t, err)
	require.Equal(t, data, val)

	// cannot retrieve the value if the caller is not allowed to
	b.suaveContext.CallerStack = []*common.Address{}
	_, err = b.confidentialRetrieve(bid.Id, "key")
	require.Error(t, err)
}

func TestSuave_secp256k1Methods(t *testing.T) {
	b := newTestBackend(t)

	// private key to sign message
	privateKey, err := hex.DecodeString("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	require.NoError(t, err)

	// message to sign
	rawMsg, err := hex.DecodeString("5c783139457468657265756d205369676e6564204d6573736167653a74686520726176656e20686173206c616e6465642e")
	require.NoError(t, err)
	msgHash := crypto.Keccak256Hash(rawMsg)

	// sign the message hash
	sig, err := b.secp256k1Sign(msgHash.Bytes(), privateKey)
	require.NoError(t, err)

	// test signature
	expectedSig, err := hex.DecodeString("a8ffb6c2309a59937caf7ceb5b2c2f8e37c9095099f1128e6e130f2ee762c4d5157febcd0ed6a73fa02dce2330b9d69703675ce667c1d00bf0578864193fdee200")
	require.NoError(t, err)
	require.Equal(t, sig, expectedSig)

	// recover public key from signature
	pubkey, err := b.secp256k1RecoverPubkey(msgHash.Bytes(), sig)
	require.NoError(t, err)

	// convert pubkey to address
	parsedPubkey, err := crypto.UnmarshalPubkey(pubkey)
	require.NoError(t, err)
	address := crypto.PubkeyToAddress(*parsedPubkey)

	// test address
	expectedAddr := common.HexToAddress("f39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	require.Equal(t, expectedAddr, address)

	// recovery param (v) must be removed from the signature
	sigR := new(big.Int).SetBytes(sig[:32])
	sigS := new(big.Int).SetBytes(sig[32:64])
	sig = append(sigR.Bytes(), sigS.Bytes()...)

	// verify that `sig` is signature of the message hash signed by pubkey
	valid, err := b.secp256k1VerifySignature(pubkey, msgHash.Bytes(), sig)
	require.NoError(t, err)
	require.True(t, valid)
}

func TestSuave_secp256k1Artifacts(t *testing.T) {
	// test that the secp256k1 method artifacts are available
	_, ok := artifacts.SuaveMethods["secp256k1Sign"]
	require.True(t, ok)

	_, ok = artifacts.SuaveMethods["secp256k1RecoverPubkey"]
	require.True(t, ok)

	_, ok = artifacts.SuaveMethods["secp256k1VerifySignature"]
	require.True(t, ok)

	// test that the secp256k1 methods are available in the abi
	_, ok = artifacts.SuaveAbi.Methods["secp256k1Sign"]
	require.True(t, ok)

	_, ok = artifacts.SuaveAbi.Methods["secp256k1RecoverPubkey"]
	require.True(t, ok)

	_, ok = artifacts.SuaveAbi.Methods["secp256k1VerifySignature"]
	require.True(t, ok)
}
