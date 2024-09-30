package indexer

type IndexClienter interface {
	Ping() error
	GetTxIDs(publicKey string) (*GetTxIDsResponse, error)
	GetTx(txID string) (*GetTxResponse, error)
	ListUnspent(publicKey string) (*ListUnspentResponse, error)
	Shutdown()
}
