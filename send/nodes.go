package send

import (
	acrypto "github.com/astromechio/astrocache/crypto"
	"github.com/astromechio/astrocache/execute"
	"github.com/astromechio/astrocache/logger"
	"github.com/astromechio/astrocache/model"
	"github.com/astromechio/astrocache/model/actions"
	"github.com/astromechio/astrocache/model/blockchain"
	"github.com/astromechio/astrocache/model/requests"
	"github.com/pkg/errors"
)

// AddNodeToChain handles adding a new node to the network
func AddNodeToChain(chain *blockchain.Chain, keySet *acrypto.KeySet, verifier *model.Node, action *actions.NodeAdded) (*requests.NewNodeResponse, error) {
	block, err := blockchain.NewBlockWithAction(keySet.GlobalKey, action)
	if err != nil {
		return nil, errors.Wrap(err, "AddNodeToChain failed to NewBlockWithAction")
	}

	prevBlock := chain.LastBlock()
	block.PrevID = prevBlock.ID

	// handle the bootstrap case
	if verifier == nil {
		if err := execute.AddPendingBlock(chain, keySet, block); err != nil {
			return nil, errors.Wrap(err, "AddNodeToChain failed to AddPendingBlock")
		}

		if err := execute.CommitPendingBlock(chain, keySet, block); err != nil {
			return nil, errors.Wrap(err, "AddNodeToChain failed to CommitBlockToChain")
		}
	} else {
		logger.LogWarn("Verifier node block mining not yet implemented")
	}

	resp := &requests.NewNodeResponse{
		Node:         action.Node,
		EncGlobalKey: action.EncGlobalKey,
	}

	return resp, nil
}
