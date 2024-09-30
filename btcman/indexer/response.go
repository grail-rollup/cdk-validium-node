package indexer

type baseResponse struct {
	ID      string `json:"id"`
	JsonRPC string `json:"jsonrpc"`
}

type transactionIDResponse struct {
	Height int    `json:"height"`
	TxHash string `json:"tx_hash"`
}

type GetTxIDsResponse struct {
	baseResponse
	Result []transactionIDResponse `json:"result"`
}

type GetTxResponse struct {
	baseResponse
	Result string `json:"result"`
}

type PingResponse struct {
	baseResponse
	Result interface{} `json:"result"`
}

type subscribeToScriptHashResponse struct {
	baseResponse
	Result string `json:"result"`
}

type UTXOResponse struct {
	TxPos  int    `json:"tx_pos"`
	Value  int64  `json:"value"`
	TxHash string `json:"tx_hash"`
	Height int    `json:"height"`
}

type ListUnspentResponse struct {
	baseResponse
	Result []UTXOResponse `json:"result"`
}
