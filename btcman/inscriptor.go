package btcman

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"strings"

	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
)

type BtcInscriptor struct {
	client  BtcRpcClienter
	net     *chaincfg.Params
	address string
}

func NewBtcInscriptor(host, user, password, address string, netType string, disableTLS bool) (BtcInscriptorer, error) {
	connCfg := &rpcclient.ConnConfig{
		Host:         host,
		User:         user,
		Pass:         password,
		HTTPPostMode: true,       // Bitcoin core only supports HTTP POST mode
		DisableTLS:   disableTLS, // Bitcoin core does not provide TLS by default
	}

	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, err
	}

	var net *chaincfg.Params
	switch netType {
	case "regtest":
		net = &chaincfg.RegressionNetParams
	case "testnet":
		net = &chaincfg.TestNet3Params
	case "mainnet":
		net = &chaincfg.MainNetParams
	default:
		net = nil
	}

	btcInscriptor := BtcInscriptor{
		client:  client,
		net:     net,
		address: address,
	}
	return &btcInscriptor, nil
}

// Shutdown closes the rpc client
func (bi *BtcInscriptor) Shutdown() {
	bi.client.Shutdown()
}

// getUTXO returns a UTXO spendable by address,consolidates the address utxo set if needed
func (bi *BtcInscriptor) getUTXO(utxoThreshold, consolidateTxFee float64) (*btcjson.ListUnspentResult, error) {
	btcAddress, err := btcutil.DecodeAddress(bi.address, bi.net)
	if err != nil {
		log.Errorf("Can't create %v", err)
		return nil, err
	}

	utxos, err := bi.listUnspentByAddress(&btcAddress)
	if err != nil {
		return nil, err
	}
	utxoIndex := 0
	utxo := utxos[utxoIndex]

	if utxo.Amount*btcutil.SatoshiPerBitcoin <= utxoThreshold {
		consolidateTxHash, err := bi.consolidateUTXOS(utxos, utxoThreshold, consolidateTxFee, &btcAddress)
		if err != nil {
			return nil, err
		}
		if consolidateTxHash != nil {
			log.Infof("UTXOs consolidated successfully: %s", consolidateTxHash.String())

			utxos, err = bi.listUnspentByAddress(&btcAddress)
			if err != nil {
				return nil, err
			}
		}
		utxoIndex = bi.getIndexOfUtxoAboveThreshold(utxoThreshold, utxos)
		if utxoIndex == -1 {
			return nil, fmt.Errorf("can't find utxo to inscribe")
		}

		utxo = utxos[utxoIndex]
	}
	log.Infof("UTXO for address %s was found", bi.address)
	return &utxo, nil
}

// listUnspentByAddress returns a list of unsent utxos filtered by address
func (bi *BtcInscriptor) listUnspentByAddress(btcAddress *btcutil.Address) ([]btcjson.ListUnspentResult, error) {

	utxos, err := bi.client.ListUnspentMinMaxAddresses(0, 99999, []btcutil.Address{*btcAddress})
	if err != nil {
		return nil, err
	}

	if len(utxos) == 0 {
		return nil, fmt.Errorf("no utxos spendable with address %s", bi.address)
	}
	return utxos, nil
}

// consolidateUTXOS combines multiple utxo in one if the utxos are under a specific threshold and over a specific count
func (bi *BtcInscriptor) consolidateUTXOS(utxos []btcjson.ListUnspentResult, threshold, consolidationTxFee float64, address *btcutil.Address) (*chainhash.Hash, error) {
	minUtxoCount := 10
	var inputs []btcjson.TransactionInput
	totalAmount := btcutil.Amount(0)
	for _, utxo := range utxos {
		amount := btcutil.Amount(utxo.Amount * btcutil.SatoshiPerBitcoin)
		thresholdAmount := btcutil.Amount(threshold)
		if amount < thresholdAmount {
			inputs = append(inputs, btcjson.TransactionInput{
				Txid: utxo.TxID,
				Vout: utxo.Vout,
			})
			totalAmount += amount
		}
	}

	if len(inputs) < minUtxoCount {
		log.Infof("Not enough UTXOs under the specified amount to consolidate. [%d/%d utoxs under %f]", len(inputs), minUtxoCount, threshold)
		return nil, nil
	}

	outputs := map[btcutil.Address]btcutil.Amount{
		*address: totalAmount - btcutil.Amount(consolidationTxFee),
	}

	rawTx, err := bi.client.CreateRawTransaction(inputs, outputs, nil)
	if err != nil {
		log.Fatalf("error creating raw transaction: %v", err)
	}

	signedTx, _, err := bi.client.SignRawTransactionWithWallet(rawTx)
	if err != nil {
		return nil, fmt.Errorf("error signing raw transaction: %v", err)
	}

	txHash, err := bi.client.SendRawTransaction(signedTx, false)
	if err != nil {
		return nil, fmt.Errorf("error sending transaction: %v", err)
	}
	return txHash, nil
}

// getUtxoAboveThreshold returns the index of utxo over a specific threshold from a utxo set, if doesn't exist returns -1
func (bi *BtcInscriptor) getIndexOfUtxoAboveThreshold(threshold float64, utxos []btcjson.ListUnspentResult) int {
	for index, utxo := range utxos {
		if utxo.Amount*btcutil.SatoshiPerBitcoin >= float64(threshold) {

			return index
		}
	}
	return -1
}

// createInscriptionRequest cretes the request for the insription with the inscription data
func (bi *BtcInscriptor) createInscriptionRequest(data string, utxoThreshold, consolidateTxFee float64) (*InscriptionRequest, error) {
	utxo, err := bi.getUTXO(utxoThreshold, consolidateTxFee)
	if err != nil {
		log.Errorf("Can't find utxo %s", err)
		return nil, err
	}

	commitTxOutPoint := new(wire.OutPoint)
	inTxid, err := chainhash.NewHashFromStr(utxo.TxID)
	if err != nil {
		log.Error("Failed to create inscription request")
		return nil, err
	}

	commitTxOutPoint = wire.NewOutPoint(inTxid, utxo.Vout)

	dataList := make([]InscriptionData, 0)

	dataList = append(dataList, InscriptionData{
		ContentType: "text/plain;charset=utf-8",
		Body:        []byte(data),
		Destination: bi.address,
	})

	request := InscriptionRequest{
		CommitTxOutPointList: []*wire.OutPoint{commitTxOutPoint},
		CommitFeeRate:        3,
		FeeRate:              2,
		DataList:             dataList,
		SingleRevealTxOnly:   true,
		// RevealOutValue:       500,
	}
	return &request, nil
}

// createInscriptionTool returns a new inscription tool struct
func (bi *BtcInscriptor) createInscriptionTool(message string, utxoThreshold, consolidateTxFee float64) (*InscriptionTool, error) {
	request, err := bi.createInscriptionRequest(message, utxoThreshold, consolidateTxFee)
	if err != nil {
		log.Errorf("Failed to create inscription request: %s", err)
		return nil, err
	}

	tool, err := NewInscriptionTool(bi.net, bi.client, request)
	if err != nil {
		log.Errorf("Failed to create inscription tool: %s", err)
		return nil, err
	}
	return tool, nil
}

// Inscribe writes data to a BTC transaction, using the provided data
func (bi *BtcInscriptor) Inscribe(data string) (string, error) {
	utxoThreshold := float64(5000)
	consolidateTxFee := float64(1000)

	tool, err := bi.createInscriptionTool(data, utxoThreshold, consolidateTxFee)
	if err != nil {
		log.Errorf("Can't create inscription tool: %s", err)
		return "", err
	}

	commitTxHash, revealTxHashList, inscriptions, fees, err := tool.Inscribe()
	if err != nil {
		log.Errorf("send tx errr, %v", err)
		return "", err
	}
	revealTxHash := revealTxHashList[0]
	inscription := inscriptions[0]

	log.Infof("CommitTxHash: %s", commitTxHash.String())
	log.Infof("RevealTxHash: %s", revealTxHash.String())
	log.Infof("Inscription: %s", inscription)
	log.Infof("Fees: %d", fees)

	return revealTxHash.String(), nil
}

// DecodeInscription reads the inscribed message from a BTC by a transaction hash
func (bi *BtcInscriptor) DecodeInscription(txHash, separator string) error {
	tx, err := bi.getTransaction(txHash)
	if err != nil {
		return err
	}
	inscriptionHex, err := bi.getInscriptionHex(tx.Hex)
	if err != nil {
		return err
	}
	message, err := bi.getMessageFromHex(inscriptionHex)
	if err != nil {
		return err
	}
	truncatedMessage, err := bi.removeLengthBytes(message, separator)
	if err != nil {
		return err
	}
	log.Infof("MESSAGE: %s", truncatedMessage)
	return nil
}

// getTransaction returns a transaction from BTC by a transaction hash
func (bi *BtcInscriptor) getTransaction(txHash string) (*btcjson.GetTransactionResult, error) {
	hash, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		return nil, err
	}
	tx, err := bi.client.GetTransaction(hash)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// getInscriptionHex returns the raw inscribed hex from the transaction
func (bi *BtcInscriptor) getInscriptionHex(txHex string) (string, error) {
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		log.Errorf("Error decoding hex string: %s", err)
		return "", err
	}
	var targetTx wire.MsgTx

	err = targetTx.Deserialize(bytes.NewReader(txBytes))
	if err != nil {
		log.Infof("Error deserializing transaction: %s", err)
		return "", err
	}
	if len(targetTx.TxIn) < 1 || len(targetTx.TxIn[0].Witness) < 2 {
		log.Infof("Error getting witness data: %s\n", err)
		return "", err
	}
	inscriptionHex := hex.EncodeToString(targetTx.TxIn[0].Witness[1])

	return inscriptionHex, nil
}

// getMessageFromHex cuts the actual message from the inscription hex, ordinals inscription data is expected
func (bi *BtcInscriptor) getMessageFromHex(inscriptionHex string) (string, error) {
	const (
		utfMarker       = "746578742f706c61696e3b636861727365743d7574662d38" // text/plain;charset=utf-8
		utfMarkerLength = 48
	)

	markerIndex := strings.Index(inscriptionHex, utfMarker)
	if markerIndex == -1 {
		return "", fmt.Errorf("inscription hex is invalid")
	}
	messageIndex := markerIndex + utfMarkerLength

	messageHex := inscriptionHex[messageIndex : len(inscriptionHex)-2]
	decodedBytes, err := hex.DecodeString(messageHex)
	if err != nil {
		log.Errorf("Error decoding hex string: %s", err)
		return "", nil
	}
	return string(decodedBytes), nil
}

// removeLengthBytes separates the inscription length from the inscription data, using a separator char
func (bi *BtcInscriptor) removeLengthBytes(decodedMsg, separator string) (string, error) {
	parts := strings.Split(decodedMsg, separator)
	partsNum := len(parts)

	if partsNum < 2 {
		return "", fmt.Errorf("separator not found")
	}

	return parts[1], nil
}
