package btcman

import (
	"fmt"
	"testing"

	"github.com/0xPolygonHermez/zkevm-node/btcman/mocks"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testContext struct {
	inscriptor *BtcInscriptor
	mockClient *mocks.MockBtcRpcClient
}

func setupTest(_ testing.TB) *testContext {
	mockClient := new(mocks.MockBtcRpcClient)
	const btcAddress string = "bcrt1qfulf03tc5g9z8r20usrrv644w2a2gw0dzpyel5"

	btcInscriptor := BtcInscriptor{
		client:  mockClient,
		net:     &chaincfg.RegressionNetParams,
		address: btcAddress,
	}

	return &testContext{
		inscriptor: &btcInscriptor,
		mockClient: mockClient,
	}
}

func TestGetUtxoAboveThreshold(t *testing.T) {
	ctx := setupTest(t)

	tests := []struct {
		name        string
		utxos       []btcjson.ListUnspentResult
		threshold   float64
		expectedIdx int
	}{
		{
			name: "Above threshold",
			utxos: []btcjson.ListUnspentResult{
				{Amount: 10},
				{Amount: 5},
			},
			threshold:   3,
			expectedIdx: 0,
		},
		{
			name: "All below threshold",
			utxos: []btcjson.ListUnspentResult{
				{Amount: 0.00000010},
				{Amount: 0.00000005},
			},
			threshold:   100,
			expectedIdx: -1,
		},
		{
			name: "Threshold exactly",
			utxos: []btcjson.ListUnspentResult{
				{Amount: 0.00000007},
			},
			threshold:   7,
			expectedIdx: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.inscriptor.getIndexOfUtxoAboveThreshold(tt.threshold, tt.utxos)
			assert.Equal(t, tt.expectedIdx, result)
		})
	}
}
func TestGetTransaction(t *testing.T) {
	ctx := setupTest(t)

	tests := []struct {
		name        string
		txHash      string
		mockResp    *btcjson.GetTransactionResult
		mockErr     error
		expected    *btcjson.GetTransactionResult
		expectedErr error
	}{
		{
			name:        "Invalid transaction",
			txHash:      "f7c24193b3f933fa87e9040436890d4b968bc8c4fc5e4a2df26a085fe41d6f89999",
			mockResp:    nil,
			mockErr:     nil,
			expected:    nil,
			expectedErr: chainhash.ErrHashStrSize,
		},
		{
			name:   "Correct method flow",
			txHash: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			mockResp: &btcjson.GetTransactionResult{
				TxID: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			},
			mockErr:     nil,
			expected:    &btcjson.GetTransactionResult{TxID: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockResp != nil || tt.mockErr != nil {
				hash, _ := chainhash.NewHashFromStr(tt.txHash)
				ctx.mockClient.On("GetTransaction", hash).Return(tt.mockResp, tt.mockErr)
			}

			result, err := ctx.inscriptor.getTransaction(tt.txHash)

			assert.Equal(t, tt.expected, result)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			ctx.mockClient.AssertExpectations(t)
		})

	}
}

func TestListUnspentByAddress(t *testing.T) {
	ctx := setupTest(t)
	btcAddress, _ := btcutil.DecodeAddress(ctx.inscriptor.address, ctx.inscriptor.net)

	tests := []struct {
		name        string
		address     *btcutil.Address
		mockResp    []btcjson.ListUnspentResult
		mockErr     error
		expected    []btcjson.ListUnspentResult
		expectedErr error
	}{
		{
			name:        "Error from ListUnspentMinMaxAddresses",
			address:     &btcAddress,
			mockResp:    nil,
			mockErr:     fmt.Errorf("no utxos spendable with address %s", btcAddress.EncodeAddress()),
			expected:    nil,
			expectedErr: fmt.Errorf("no utxos spendable with address %s", btcAddress.EncodeAddress()),
		},
		{
			name:    "Successful retrieval of UTXOs",
			address: &btcAddress,
			mockResp: []btcjson.ListUnspentResult{
				{
					TxID:          "txid1",
					Vout:          0,
					Address:       "address1",
					Account:       "account1",
					ScriptPubKey:  "script1",
					RedeemScript:  "redeem1",
					Amount:        0.1,
					Confirmations: 1,
					Spendable:     true,
				},
				{
					TxID:          "txid2",
					Vout:          1,
					Address:       "address2",
					Account:       "account2",
					ScriptPubKey:  "script2",
					RedeemScript:  "redeem2",
					Amount:        0.2,
					Confirmations: 1,
					Spendable:     true,
				},
			},
			mockErr: nil,
			expected: []btcjson.ListUnspentResult{
				{
					TxID:          "txid1",
					Amount:        0.1,
					Account:       "account1",
					ScriptPubKey:  "script1",
					RedeemScript:  "redeem1",
					Confirmations: 1,
					Spendable:     true,
					Address:       "address1",
				},
				{
					TxID:          "txid2",
					Amount:        0.2,
					Account:       "account2",
					Vout:          0x1,
					ScriptPubKey:  "script2",
					RedeemScript:  "redeem2",
					Confirmations: 1,
					Spendable:     true,
					Address:       "address2",
				},
			},
			expectedErr: nil,
		},
		{
			name:        "No UTXOs available",
			address:     &btcAddress,
			mockResp:    nil,
			mockErr:     nil,
			expected:    nil,
			expectedErr: fmt.Errorf("no utxos spendable with address %s", btcAddress.EncodeAddress()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockClient := new(mocks.MockBtcRpcClient)
			inscriptor := &BtcInscriptor{
				client:  mockClient,
				net:     &chaincfg.RegressionNetParams,
				address: btcAddress.String(),
			}

			mockClient.On("ListUnspentMinMaxAddresses", 0, 99999, []btcutil.Address{*tt.address}).Return(tt.mockResp, tt.mockErr)

			result, err := inscriptor.listUnspentByAddress(tt.address)

			assert.Equal(t, tt.expected, result)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestCreateInscriptionRequest(t *testing.T) {
	ctx := setupTest(t)
	btcAddress, _ := btcutil.DecodeAddress(ctx.inscriptor.address, ctx.inscriptor.net)
	hash, err := chainhash.NewHashFromStr("572d859a88a26af3ca7c7715f3e9565ec5f53a040ca6ec8a208933075ac43421")
	if err != nil {
		fmt.Println(err)
	}
	tests := []struct {
		name             string
		message          string
		utxoThreshold    float64
		consolidateTxFee float64
		mockUTXO         *btcjson.ListUnspentResult
		mockErr          error
		expected         *InscriptionRequest
		expectedErr      error
	}{
		{
			name:             "Successful request creation",
			message:          "Hello, world!",
			utxoThreshold:    0.01,
			consolidateTxFee: 0.0001,
			mockUTXO: &btcjson.ListUnspentResult{
				TxID:   "572d859a88a26af3ca7c7715f3e9565ec5f53a040ca6ec8a208933075ac43421",
				Vout:   0,
				Amount: 0.01,
			},
			mockErr: nil,
			expected: &InscriptionRequest{
				CommitTxOutPointList: []*wire.OutPoint{
					wire.NewOutPoint(hash, 0),
				},
				CommitFeeRate: 3,
				FeeRate:       2,
				DataList: []InscriptionData{
					{
						ContentType: "text/plain;charset=utf-8",
						Body:        []byte("Hello, world!"),
						Destination: btcAddress.EncodeAddress(),
					},
				},
				SingleRevealTxOnly: true,
			},
			expectedErr: nil,
		},
		{
			name:             "UTXO retrieval error",
			message:          "This should fail",
			utxoThreshold:    0.01,
			consolidateTxFee: 0.0001,
			mockUTXO: &btcjson.ListUnspentResult{
				TxID:   "572d859a88a26af3ca7c7715f3e9565ec5f53a040ca6ec8a208933075ac43421999",
				Vout:   0,
				Amount: 0.01,
			},
			mockErr:     fmt.Errorf("failed to retrieve UTXO"),
			expected:    nil,
			expectedErr: fmt.Errorf("failed to retrieve UTXO"),
		},
		{
			name:             "Invalid UTXO TxID",
			message:          "Message",
			utxoThreshold:    0.01,
			consolidateTxFee: 0.0001,
			mockUTXO: &btcjson.ListUnspentResult{
				TxID:   "572d859a88a26af3ca7c7715f3e9565ec5f53a040ca6ec8a208933075ac43421999",
				Vout:   0,
				Amount: 0.01,
			},
			mockErr:     nil,
			expected:    nil,
			expectedErr: chainhash.ErrHashStrSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(mocks.MockBtcRpcClient)
			ctx.inscriptor.client = mockClient
			if tt.mockUTXO != nil {
				mockClient.On("ListUnspentMinMaxAddresses", 0, 99999, []btcutil.Address{btcAddress}).
					Return([]btcjson.ListUnspentResult{*tt.mockUTXO}, tt.mockErr)
			} else {
				mockClient.On("ListUnspentMinMaxAddresses", 0, 99999, []btcutil.Address{btcAddress}).
					Return(nil, tt.mockErr)
			}

			result, err := ctx.inscriptor.createInscriptionRequest(tt.message, tt.utxoThreshold, tt.consolidateTxFee)

			assert.Equal(t, tt.expected, result)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}

			ctx.mockClient.AssertExpectations(t)
		})
	}
}

func TestConsolidateUTXOS(t *testing.T) {
	ctx := setupTest(t)
	address, _ := btcutil.DecodeAddress(ctx.inscriptor.address, ctx.inscriptor.net)
	// Test case 1: Successful consolidation
	utxos := []btcjson.ListUnspentResult{
		{TxID: "txid1", Vout: 0, Amount: 0.00001},
		{TxID: "txid2", Vout: 1, Amount: 0.00002},
		{TxID: "txid3", Vout: 2, Amount: 0.000005},
		{TxID: "txid1", Vout: 3, Amount: 0.00001},
		{TxID: "txid2", Vout: 5, Amount: 0.00002},
		{TxID: "txid3", Vout: 6, Amount: 0.000005},
		{TxID: "txid2", Vout: 7, Amount: 0.00002},
		{TxID: "txid3", Vout: 8, Amount: 0.000005},
		{TxID: "txid3", Vout: 9, Amount: 0.000005},
		{TxID: "txid2", Vout: 10, Amount: 0.00002},
		{TxID: "txid3", Vout: 11, Amount: 0.000005},
	}
	threshold := float64(10000)
	consolidationTxFee := 0.00001

	ctx.mockClient.On("CreateRawTransaction", mock.Anything, mock.Anything, mock.Anything).Return(&wire.MsgTx{}, nil)

	ctx.mockClient.On("SignRawTransactionWithWallet", mock.Anything).Return(&wire.MsgTx{}, true, nil)

	ctx.mockClient.On("SendRawTransaction", mock.Anything, false).Return(&chainhash.Hash{}, nil)

	txHash, err := ctx.inscriptor.consolidateUTXOS(utxos, threshold, consolidationTxFee, &address)
	assert.NoError(t, err)
	assert.NotNil(t, txHash)

	// Test case 2: Not enough UTXOs to consolidate
	utxos = []btcjson.ListUnspentResult{
		{TxID: "txid1", Vout: 0, Amount: 0.000001},
	}

	txHash, err = ctx.inscriptor.consolidateUTXOS(utxos, threshold, consolidationTxFee, &address)
	assert.NoError(t, err)
	assert.Nil(t, txHash)
}

func TestGetUTXO(t *testing.T) {
	ctx := setupTest(t)

	utxoThreshold := 0.00015
	consolidationTxFee := 0.00001

	utxos := []btcjson.ListUnspentResult{
		{TxID: "txid1", Vout: 0, Amount: 0.00011},
		{TxID: "txid2", Vout: 1, Amount: 0.00021},
	}

	address, _ := btcutil.DecodeAddress(ctx.inscriptor.address, ctx.inscriptor.net)

	ctx.mockClient.On("ListUnspentMinMaxAddresses", 0, 99999, []btcutil.Address{address}).Return(utxos, nil)
	utxo, err := ctx.inscriptor.getUTXO(utxoThreshold, consolidationTxFee)
	assert.NoError(t, err)
	assert.NotNil(t, utxo)
	assert.Equal(t, "txid1", utxo.TxID)
}
