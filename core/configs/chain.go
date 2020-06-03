package configs

type ChainConfig struct {
	Name  string     `yaml:name`  // Name of the chain (will be used in config print)
	Nodes []string   `yaml:nodes` // Address of the nodes.
	Keys  []ChainKey `yaml:keys`  // Key information
}

type ChainKey struct {
	PrivateKey []byte `yaml:private` // Private key information
	Address    string `yaml:address` // Address that it is from
}
