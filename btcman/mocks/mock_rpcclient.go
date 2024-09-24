package mocks

import (
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/stretchr/testify/mock"
)

// MockBtcRpcClient is a mock implementation of the BtcRpcClienter interface
type MockBtcRpcClient struct {
	mock.Mock
}

// ListUnspentMinMaxAddresses mocks the ListUnspentMinMaxAddresses method
func (m *MockBtcRpcClient) ListUnspentMinMaxAddresses(min, max int, addresses []btcutil.Address) ([]btcjson.ListUnspentResult, error) {
	args := m.Called(min, max, addresses)
	return args.Get(0).([]btcjson.ListUnspentResult), args.Error(1)
}

// CreateRawTransaction mocks the CreateRawTransaction method
func (m *MockBtcRpcClient) CreateRawTransaction(inputs []btcjson.TransactionInput, amounts map[btcutil.Address]btcutil.Amount, lockTime *int64) (*wire.MsgTx, error) {
	args := m.Called(inputs, amounts, lockTime)
	return args.Get(0).(*wire.MsgTx), args.Error(1)
}

// SignRawTransactionWithWallet mocks the SignRawTransactionWithWallet method
func (m *MockBtcRpcClient) SignRawTransactionWithWallet(tx *wire.MsgTx) (*wire.MsgTx, bool, error) {
	args := m.Called(tx)
	return args.Get(0).(*wire.MsgTx), args.Bool(1), args.Error(2)
}

// SendRawTransaction mocks the SendRawTransaction method
func (m *MockBtcRpcClient) SendRawTransaction(tx *wire.MsgTx, allowHighFees bool) (*chainhash.Hash, error) {
	args := m.Called(tx, allowHighFees)
	return args.Get(0).(*chainhash.Hash), args.Error(1)
}

// GetTransaction mocks the GetTransaction method
func (m *MockBtcRpcClient) GetTransaction(hash *chainhash.Hash) (*btcjson.GetTransactionResult, error) {
	args := m.Called(hash)
	return args.Get(0).(*btcjson.GetTransactionResult), args.Error(1)
}

// GetRawTransactionVerbose mocks the GetRawTransactionVerbose method
func (m *MockBtcRpcClient) GetRawTransactionVerbose(hash *chainhash.Hash) (*btcjson.TxRawResult, error) {
	args := m.Called(hash)
	return args.Get(0).(*btcjson.TxRawResult), args.Error(1)
}

// Shutdown mocks the Shutdown method
func (m *MockBtcRpcClient) Shutdown() {
	m.Called()
}
