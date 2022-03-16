package types

import (
	"crypto/ecdsa"
	"encoding/json"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type EthereumTransactionWithPublicKey struct {
	Tx  *types.DynamicFeeTx
	Pub *ecdsa.PublicKey
}

func (tx EthereumTransactionWithPublicKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Tx  *types.DynamicFeeTx
		Pub []byte
	}{
		Tx:  tx.Tx,
		Pub: crypto.CompressPubkey(tx.Pub),
	})
}

func (tx *EthereumTransactionWithPublicKey) UnmarshalJSON(data []byte) error {
	aux := &struct {
		Tx  *types.DynamicFeeTx
		Pub []byte
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	pub, err := crypto.DecompressPubkey(aux.Pub)
	if err != nil {
		return err
	}
	tx.Tx = aux.Tx
	tx.Pub = pub
	return nil
}
