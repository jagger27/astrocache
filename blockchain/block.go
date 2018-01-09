package blockchain

import (
	"crypto/sha256"

	acrypto "github.com/astromechio/astrocache/crypto"
)

// Notes:
// ID is the sha256 of the previous block's data
// Data can be anything, but in the context of astrocache, it is generally JSON encrypted by the global symKey
// Signature is a DSS of the data generated by the mining node's pubkey
// PrevID is the ID of the block whose data was hashed to create this block's ID (directly previous in the chain)

// Block is the base type for the astrocache blockchain
type Block struct {
	ID        string             `json:"id"`
	Data      *acrypto.Message   `json:"data"`
	Signature *acrypto.Signature `json:"signature"`
	PrevID    string             `json:"prevId"`
}

// NewBlock creates a new block
func NewBlock(key *acrypto.AstroKeyPair, data []byte, prev *Block) (*Block, error) {
	prevHash, err := prev.Hash()
	if err != nil {
		return nil, err
	}

	newID := acrypto.Base64URLEncode(prevHash)
	encData, err := key.Encrypt(data)
	if err != nil {
		return nil, err
	}

	sig, err := key.Sign(encData.Data)
	if err != nil {
		return nil, err
	}

	newBlock := &Block{
		ID:        newID,
		Data:      encData,
		Signature: sig,
		PrevID:    prev.ID,
	}

	return newBlock, nil
}

// Hash computes the sha256 of the block's data
func (b *Block) Hash() ([]byte, error) {
	h := sha256.New()
	_, err := h.Write(b.Data.Data)
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}