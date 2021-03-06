package send

import (
	"github.com/astromechio/astrocache/logger"
	"github.com/astromechio/astrocache/model"
	"github.com/astromechio/astrocache/model/blockchain"
	"github.com/astromechio/astrocache/model/requests"
	"github.com/astromechio/astrocache/transport"
	"github.com/pkg/errors"
)

// GetEntireChain requests the entire chain from the master node
func GetEntireChain(masterNode *model.Node) ([]*blockchain.Block, error) {
	url := transport.URLFromAddressAndPath(masterNode.Address, "v1/master/chain")

	blocks := []*blockchain.Block{}
	if err := transport.Get(url, &blocks); err != nil {
		return nil, errors.Wrap(err, "GetEntireChain failed to Get")
	}

	return blocks, nil
}

// GetBlocksAfter requests the entire chain from the master node
func GetBlocksAfter(masterNode *model.Node, afterID string) ([]*blockchain.Block, error) {
	url := transport.URLFromAddressAndPath(masterNode.Address, "v1/master/chain/after/"+afterID)

	blocks := []*blockchain.Block{}
	if err := transport.Get(url, &blocks); err != nil {
		return nil, errors.Wrap(err, "GetBlocksAfter failed to Get")
	}

	return blocks, nil
}

// RequestReservedID reserves a block ID with the master node
func RequestReservedID(masterNode *model.Node, propNID string) (*requests.ReserveIDResponse, error) {
	logger.LogInfo("RequestReservedID requesting block ID from master node")

	req := &requests.ReserveIDRequest{
		ProposingNID: propNID,
	}

	url := transport.URLFromAddressAndPath(masterNode.Address, req.Path())

	resp := &requests.ReserveIDResponse{}
	if err := transport.Post(url, req, &resp); err != nil {
		return nil, errors.Wrap(err, "RequestReservedID failed to Get")
	}

	return resp, nil
}
