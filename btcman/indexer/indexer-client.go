package indexer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/0xPolygonHermez/zkevm-node/log"

	"golang.org/x/crypto/ripemd160"
)

type IndexClient struct {
	conn   *net.Conn
	ticker *time.Ticker
}

func NewIndexClient(serverAddress string) (IndexClienter, error) {
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		return nil, fmt.Errorf("error connecting to server: %s", err)
	}

	pingInterval := 2 * time.Minute
	ticker := time.NewTicker(pingInterval)
	indexClient := IndexClient{
		conn:   &conn,
		ticker: ticker,
	}
	go func() {
		for {
			select {
			case <-indexClient.ticker.C:
				indexClient.Ping()
			}
		}
	}()
	log.Info("Indexer: New electrum indexer instantiated")
	return &indexClient, nil
}

func (ic *IndexClient) Ping() error {
	const method string = "server.ping"
	params := []interface{}{}

	var pingReponse PingResponse
	err := request(*ic.conn, method, params, &pingReponse)
	if err != nil {
		return fmt.Errorf("error pinging electrum server: %s", err)
	}
	log.Info("Indexer: pinging electrum server to keep the connection open")
	return nil
}

func (ic *IndexClient) GetTx(txID string) (*GetTxResponse, error) {
	const method string = "blockchain.transaction.get"
	params := []interface{}{txID}
	var getTxResponse GetTxResponse
	err := request(*ic.conn, method, params, &getTxResponse)
	if err != nil {
		return nil, fmt.Errorf("error getting raw transaction by ID: %s", err)
	}
	return &getTxResponse, nil
}

func (ic *IndexClient) GetTxIDs(publicKey string) (*GetTxIDsResponse, error) {

	const method string = "blockchain.scripthash.get_history"
	scriptHash, err := publicKeyToScriptHash(publicKey)
	if err != nil {
		return nil, err
	}
	params := []interface{}{scriptHash}

	// subscribe
	ic.subscribeToScriptHash(scriptHash)

	var getTxIDsResponse GetTxIDsResponse
	err = request(*ic.conn, method, params, &getTxIDsResponse)
	if err != nil {
		return nil, fmt.Errorf("error getting transaction IDs by publicKey: %s", err)
	}

	return &getTxIDsResponse, nil
}

func (ic *IndexClient) ListUnspent(publicKey string) (*ListUnspentResponse, error) {
	scriptHash, err := publicKeyToScriptHash(publicKey)
	if err != nil {
		return nil, fmt.Errorf("error listing unspent utxos: %s", err)
	}
	_, err = ic.subscribeToScriptHash(scriptHash)
	if err != nil {
		return nil, fmt.Errorf("error listing unspent utxos: %s", err)
	}

	const method string = "blockchain.scripthash.listunspent"
	params := []interface{}{scriptHash}
	var listUnspentResponse ListUnspentResponse

	err = request(*ic.conn, method, params, &listUnspentResponse)
	if err != nil {
		return nil, fmt.Errorf("error listing unspent utxos: %s", err)
	}
	return &listUnspentResponse, nil
}

func (ic *IndexClient) Shutdown() {
	(*ic.conn).Close()
	ic.ticker.Stop()
	log.Info("Indexer: electrum indexer disconnected")
}

func (ic *IndexClient) subscribeToScriptHash(scriptHash string) (string, error) {

	const method string = "blockchain.scripthash.subscribe"
	params := []interface{}{scriptHash}

	var getTxResponse subscribeToScriptHashResponse
	err := request(*ic.conn, method, params, &getTxResponse)
	if err != nil {
		return "", fmt.Errorf("error getting raw transaction by ID: %s", err)
	}
	log.Infof("Indexer: subscribed to script hash: %s", scriptHash)
	return getTxResponse.Result, nil
}

func request(conn net.Conn, method string, params []interface{}, unMarshallObject interface{}) error {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      "curltext",
		"method":  method,
		"params":  params,
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {

		return fmt.Errorf("error marshaling request to JSON: %s", err)
	}

	_, err = conn.Write(append(requestJSON, '\n'))
	if err != nil {
		return fmt.Errorf("error sending request: %s", err)
	}

	var responseBuffer bytes.Buffer
	buffer := make([]byte, 4096)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() != "EOF" {
				return fmt.Errorf("error reading response: %s", err)
			}
			break
		}
		responseBuffer.Write(buffer[:n])
		if bytes.HasSuffix(responseBuffer.Bytes(), []byte("}\n")) {
			break
		}
	}
	err = json.Unmarshal(responseBuffer.Bytes(), unMarshallObject)
	if err != nil {
		return fmt.Errorf("error unmarshaling response: %s", err)
	}
	return nil
}

func publicKeyToScriptHash(publicKey string) (string, error) {
	sha256Hashed, err := calculateSHA256(publicKey)
	if err != nil {
		return "", fmt.Errorf("error hashing public key with sha256: %s", err)
	}
	pkHash := calculateRIPEMD160(sha256Hashed)

	pubScript := fmt.Sprintf("0014%s", hex.EncodeToString(pkHash))

	scriptHash, err := calculateSHA256(pubScript)
	if err != nil {
		return "", fmt.Errorf("error hashing public key with sha256: %s", err)
	}

	bigEndianBytes := make([]byte, len(scriptHash))
	err = convertEndianess(scriptHash, bigEndianBytes)
	if err != nil {
		return "", fmt.Errorf("error changing endianess: %s", err)
	}
	return hex.EncodeToString(bigEndianBytes), nil
}

/*//////////////////////////////////////////////////////////////
                            UTILS
//////////////////////////////////////////////////////////////*/

func calculateSHA256(data string) ([]byte, error) {
	dataBytes, err := hex.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("error decoding public key string: %s", err)
	}
	hash := sha256.New()

	// Write data to the hash
	hash.Write(dataBytes)

	// Calculate the hash
	sum := hash.Sum(nil)

	// Convert the hash to a hexadecimal string
	return sum, nil
}

func calculateRIPEMD160(data []byte) []byte {
	hash := ripemd160.New()
	hash.Write(data)
	sum := hash.Sum(nil)

	return sum
}

func convertEndianess(src []byte, dst []byte) error {
	if len(src) != len(dst) {
		return fmt.Errorf("source and destination slices must be of the same length")
	}
	for i := range src {
		dst[len(src)-1-i] = src[i]
	}
	return nil
}
