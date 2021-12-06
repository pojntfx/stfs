package config

type StateConfig struct {
	Drive    string
	Metadata string
}

type PipeConfig struct {
	Compression string
	Encryption  string
	Signature   string
}

type CryptoConfig struct {
	Recipient interface{}
	Identity  interface{}
	Password  string
}
