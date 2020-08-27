package configs

// Chain configuration the contains the information about the blockchain
type ChainConfig struct {
	Name  string     `yaml:name`      // Name of the chain (will be used in config print)
	Nodes []string   `yaml:nodes`     // Address of the nodes.
	Keys  []ChainKey `yaml:keys,flow` // Key information
}
