package blockchain

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	acrypto "github.com/astromechio/astrocache/crypto"
	"github.com/astromechio/astrocache/model/actions"
	"github.com/pkg/errors"
)

// GenesisBlockID and others are block related consts
const (
	genesisBlockID = "iamthegenesisbutnottheterminator"
)

// Notes:
// ID is the sha256 of the previous block's data [randomly generated until the block is committed]
// Data can be anything, but in the context of astrocache, it is generally JSON encrypted by the global symKey
// ActionType is an astrocache specific field to help with unmarshalling
// Signature is a DSS of the data generated by the mining node's privKey
// PrevID is the ID of the block whose data was hashed to create this block's ID (directly previous in the chain) [will be empty if not committed]

// Block is the base type for the astrocache blockchain
type Block struct {
	ID         string             `json:"id"`
	Data       *acrypto.Message   `json:"data"`
	ActionType string             `json:"actionType,omitempty"`
	Signature  *acrypto.Signature `json:"signature"`
	PrevID     string             `json:"prevId"`
}

// NewBlockWithAction creates a block with JSON from an action
func NewBlockWithAction(encKey *acrypto.SymKey, action actions.Action) (*Block, error) {
	actionJSON := action.JSON()

	block, err := newBlock(encKey, actionJSON)
	if err != nil {
		return nil, errors.Wrap(err, "NewBlockWithAction failed to NewBlock")
	}

	block.ActionType = action.ActionType()

	return block, nil
}

func genesisBlockWithAction(encKey *acrypto.SymKey, action actions.Action) (*Block, error) {
	actionJSON := action.JSON()

	block, err := newBlock(encKey, actionJSON)
	if err != nil {
		return nil, errors.Wrap(err, "genesisBlockWithAction failed to NewBlock")
	}

	block.ActionType = action.ActionType()

	return block, nil
}

func newBlock(encKey *acrypto.SymKey, data []byte) (*Block, error) {
	encData, err := encKey.Encrypt(data)
	if err != nil {
		return nil, err
	}

	newBlock := &Block{
		ID:     generatePendingBlockID(),
		Data:   encData,
		PrevID: "",
	}

	return newBlock, nil
}

func (b *Block) prepareForCommit(sigKey *acrypto.KeyPair, prev *Block) error {
	prevHash, err := prev.Hash()
	if err != nil {
		return errors.Wrap(err, "prepareForCommit failed to prev.Hash")
	}

	signingBody := append(prevHash, b.Data.Data...)

	sig, err := sigKey.Sign(signingBody)
	if err != nil {
		return errors.Wrap(err, "prepareForCommit failed to sigKey.Sign")
	}

	blockID := acrypto.Base64URLEncode(prevHash)

	b.ID = blockID
	b.Signature = sig
	b.PrevID = prev.ID

	return nil
}

// Verify verifies a block's integrity
func (b *Block) Verify(keySet *acrypto.KeySet, prev *Block) error {
	newID := ""
	signingBody := []byte{}

	sigKey := keySet.KeyPairWithKID(b.Signature.KID)
	if sigKey == nil {
		return fmt.Errorf("keyset to verify block signature with KID %s not found", b.Signature.KID)
	}

	// handle the genesis block case
	if prev == nil {
		if sigKey.KID != acrypto.MasterKeyPairKID {
			return fmt.Errorf("attempted to verify genesis block with non-master keyPair with KID %s", sigKey.KID)
		}

		if b.ID != genesisBlockID {
			return errors.New("attempted to verify non-genesis block with nil prev block")
		}

		signingBody = b.Data.Data
		newID = genesisBlockID
	} else {
		prevHash, err := prev.Hash()
		if err != nil {
			return errors.Wrap(err, "Verify failed to prev.Hash")
		}

		signingBody = append(prevHash, b.Data.Data...)
		newID = acrypto.Base64URLEncode(prevHash)
	}

	if b.ID != newID {
		return fmt.Errorf("block ID %s does not match prev.Hash %s", b.ID, newID)
	}

	if result := sigKey.Verify(signingBody, b.Signature); result == acrypto.AstroSigUnverified {
		return errors.New("block data verification failed")
	}

	return nil
}

func (b *Block) isPending() bool {
	return b.PrevID == ""
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

func generatePendingBlockID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)

	return acrypto.Base64URLEncode(bytes)
}
