package btcman

type Config struct {
	// BtcHost is the rpc host of the btc node
	BtcHost string `mapstructure:"BtcHost"`

	// BtcNet is the type of network of the btc node
	BtcNet string `mapstructure:"BtcNet"`

	// BtcWalletName is the wallet name of the btc node
	BtcWalletName string `mapstructure:"BtcWalletName"`

	// BtcWalletPass is the password of the btc wallet for the node
	BtcWalletPass string `mapstructure:"BtcWalletPass"`

	// BtcRpcUser is the username for the rpc service
	BtcRpcUser string `mapstructure:"BtcRpcUser"`

	// BtcRpcPass is the password for the rpc service
	BtcRpcPass string `mapstructure:"BtcRpcPass"`

	// BtcPrivateKey is the private key for the btc node wallet
	BtcPrivateKey string `mapstructure:"BtcPrivateKey"`

	// BtcDisableTLS is a flat that disables the TLS
	BtcDisableTLS bool `mapstructure:"BtcDisableTLS"`
}

func IsValidBtcConfig(cfg *Config) bool {
	return cfg.BtcHost != "" &&
		cfg.BtcRpcPass != "" &&
		cfg.BtcRpcUser != "" &&
		cfg.BtcWalletName != "" &&
		cfg.BtcWalletPass != "" &&
		cfg.BtcPrivateKey != ""
}
