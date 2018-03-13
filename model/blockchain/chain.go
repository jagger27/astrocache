package blockchain

import (
	"fmt"

	"github.com/astromechio/astrocache/logger"

	acrypto "github.com/astromechio/astrocache/crypto"
	"github.com/pkg/errors"
)

// Chain represents a blockchain
type Chain struct {
	Blocks         []*Block
	Proposed       *Block
	WorkerChan     chan (*NewBlockJob) // WorkerChan is used by verifier_chainworker as the synchronization method for mining and verifying new blocks
	ActionChan     chan (*Block)       // ActionChan decrypts blocks and applies actions
	DistributeChan chan (*Block)       // DistributeChan loads blocks needed to be distributed to workers
	ProposedChan   chan (*Block)       // ProposedChan is used when a goroutine needs to know the next time a block is proposed. It may or may not be nil
	CommittedChan  chan (*Block)       // CommittedChan is used when a goroutine needs to know the next time a block is committed. It may or may not be nil
}

// NewBlockJob represents the intent to add a new block
type NewBlockJob struct {
	Block      *Block
	ResultChan chan (error)
	Check      bool
}

// AddNewBlock queues a new block job
func (c *Chain) AddNewBlock(block *Block) chan (error) {
	errChan := make(chan error, 1)

	job := &NewBlockJob{
		Block:      block,
		ResultChan: errChan,
		Check:      true,
	}

	c.WorkerChan <- job
	return errChan
}

// AddNewBlockUnchecked adds a new block without verifier check
func (c *Chain) AddNewBlockUnchecked(block *Block) chan (error) {
	errChan := make(chan error, 1)

	job := &NewBlockJob{
		Block:      block,
		ResultChan: errChan,
		Check:      false,
	}

	c.WorkerChan <- job
	return errChan
}

// SetProposedBlock checks and then sets the proposed block
func (c *Chain) SetProposedBlock(block *Block) error {
	prevBlock := c.LastBlock()

	if prevBlock == nil {
		if block.ID != genesisBlockID {
			return fmt.Errorf("SetProposedBlock tried to propose block with nil prevBlock and non-genesis ID %q", block.ID)
		}
	} else {
		if prevBlock.ID != block.PrevID {
			return fmt.Errorf("AddPendingBlock failed to add block: block.PrevID (%s) did not match prevBlock.ID (%s)", block.PrevID, prevBlock.ID)
		}
	}

	c.Proposed = block

	if c.ProposedChan != nil {
		c.ProposedChan <- block
		c.ProposedChan = nil
	}

	return nil
}

// CommitProposedBlock verifies and commits a block
func (c *Chain) CommitProposedBlock(keySet *acrypto.KeySet) error {
	prevBlock := c.LastBlock()

	// Verify handles the genesis case
	if err := c.Proposed.Verify(keySet, prevBlock); err != nil {
		return errors.Wrap(err, "CommitProposedBlock failed to block.Verify")
	}

	logger.LogInfo(fmt.Sprintf("*** Committing bock with ID %s ***", c.Proposed.ID))

	c.Blocks = append(c.Blocks, c.Proposed)

	c.ActionChan <- c.Proposed // send the block to be executed

	if c.CommittedChan != nil {
		c.CommittedChan <- c.Proposed
		c.CommittedChan = nil
	}

	c.Proposed = nil

	return nil
}

// LoadFromBlocks loads a chain from a block array
func (c *Chain) LoadFromBlocks(blocks []*Block) error {
	if len(c.Blocks) > 0 {
		return fmt.Errorf("LoadFromBlocks attempted to load chain with %d existing blocks", len(c.Blocks))
	}

	for i := range blocks {
		errChan := c.AddNewBlockUnchecked(blocks[i])
		if err := <-errChan; err != nil {
			return err
		}
	}

	return nil
}

// GetNextProposed blocks until the next block is proposed
func (c *Chain) GetNextProposed() chan *Block {
	logger.LogInfo("Waiting for next proposed block")

	if c.ProposedChan != nil {
		return nil
	}

	notifChan := make(chan *Block, 1)
	c.ProposedChan = notifChan

	return notifChan
}

// GetNextCommitted blocks until the next block is proposed
func (c *Chain) GetNextCommitted() chan *Block {
	logger.LogInfo("Waiting for next committed block")

	if c.CommittedChan != nil {
		return nil
	}

	notifChan := make(chan *Block, 1)
	c.CommittedChan = notifChan

	return notifChan
}

// EmptyChain creates an enpty chain
func EmptyChain() *Chain {
	chain := &Chain{
		Blocks:         []*Block{},
		WorkerChan:     make(chan *NewBlockJob, 2),
		ActionChan:     make(chan *Block),
		DistributeChan: make(chan *Block),
	}

	return chain
}

// BrandNewChain creates a fresh chain using the master keyPair
func BrandNewChain(masterKeyPair *acrypto.KeyPair, globalKey *acrypto.SymKey, blockData []byte, actionType string) (*Chain, error) {
	if masterKeyPair.KID != acrypto.MasterKeyPairKID {
		return nil, fmt.Errorf("attempted to create new chain with non-master keyPair")
	}

	if globalKey.KID != acrypto.GlobalSymKeyKID {
		return nil, fmt.Errorf("attempted to create new chain with non-global symKey")
	}

	genesis, err := genesisBlockWithData(globalKey, blockData, actionType)
	if err != nil {
		return nil, errors.Wrap(err, "BrandNewChain failed to NewBlockWithAction")
	}

	chain := EmptyChain()

	// if this fails in the worker, we'll have to catch it and fatal
	chain.AddNewBlock(genesis)

	return chain, nil
}

// LastBlock returns the last block in the chain
func (c *Chain) LastBlock() *Block {
	if len(c.Blocks) == 0 {
		return nil
	}

	return c.Blocks[len(c.Blocks)-1]
}
