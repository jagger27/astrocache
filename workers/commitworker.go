package workers

import (
	"fmt"
	"os"
	"time"

	"github.com/astromechio/astrocache/config"
	"github.com/astromechio/astrocache/logger"
	"github.com/astromechio/astrocache/model/blockchain"
	"github.com/astromechio/astrocache/send"
	"github.com/pkg/errors"
)

// CommitWorker runs on a goroutine and manages the adding of new blocks in an atomic manner
func CommitWorker(app *config.App) {
	if app.Chain == nil {
		logger.LogError(errors.New("CommitWorker received nil chain, terminating"))
		os.Exit(1)
	}

	if app.KeySet == nil {
		logger.LogError(errors.New("CommitWorker received nil keySet, terminating"))
		os.Exit(1)
	}

	chain := app.Chain

	logger.LogInfo("starting commit worker")

	for true {
		blockJob := <-chain.CommitChan

		logger.LogInfo("CommitWorker got commit job")

		if blockJob.Block == nil {
			logger.LogWarn("CommitWorker received nil block, continuing..")
			continue
		}

		if err := commitBlock(blockJob, app); err != nil {
			logger.LogError(errors.Wrap(err, "CommitWorker failed to checkBlock"))
			blockJob.ResultChan <- errors.Wrap(err, "CommitWorker failed to checkBlock")

			chain.Proposed = nil

			chain.CommittedChan <- nil
			go loadMissingBlocks(app)

			continue
		}

		blockJob.ResultChan <- nil

		chain.ActionChan <- blockJob.Block // send the block to be executed

		logger.LogInfo("CommitWorker completed commit job, reporting committed")
		chain.CommittedChan <- blockJob.Block // notify other goroutines that something was committed
		logger.LogInfo("CommitWorker completed commit job, reported committed")
	}
}

func commitBlock(job *blockchain.NewBlockJob, app *config.App) error {
	chain := app.Chain

	logger.LogInfo(fmt.Sprintf("commitBlock committing block with ID %q", job.Block.ID))

	if chain.Proposed == nil || !job.Block.IsSameAsBlock(chain.Proposed) {
		return fmt.Errorf("commitBlock tried to commit a non-proposed block")
	}

	last := chain.LastBlock()
	if last != nil && job.Block.IsSameAsBlock(last) {
		logger.LogWarn("Tried committing duplicate block, skipping...")
		return nil
	}

	// Verify handles the genesis case
	if err := job.Block.Verify(app.KeySet, last); err != nil {
		return errors.Wrap(err, "commitBlock failed to block.Verify")
	}

	logger.LogInfo(fmt.Sprintf("*** Committing bock with ID %q ***", job.Block.ID))

	chain.Blocks = append(chain.Blocks, chain.Proposed)

	chain.Proposed = nil

	return nil
}

func loadMissingBlocks(app *config.App) {
	lastBlock := app.Chain.LastBlock()

	logger.LogInfo(fmt.Sprintf("loadMissingBlocks attempting to load missing blocks after %q", lastBlock.ID))

	var missing []*blockchain.Block

	for true {
		var err error
		missing, err = send.GetBlocksAfter(app.NodeList.Master, lastBlock.ID)
		if err != nil {
			logger.LogError(errors.Wrap(err, "loadMissingBlocks failed to GetBlocksAfter"))
			return
		}

		if missing == nil {
			logger.LogInfo(fmt.Sprintf("loadMissingBlocks received no blocks after %q", lastBlock.ID))
		} else if len(missing) > 0 {
			break
		}

		<-time.After(time.Second * 1)
	}

	for i := range missing {
		logger.LogInfo(fmt.Sprintf("loadMissingBlocks loading missing block with ID %q", missing[i].ID))

		errChan := app.Chain.VerifyProposedBlock(missing[i], "")
		if err := <-errChan; err != nil {
			logger.LogError(errors.Wrap(err, "loadMissingBlocks failed to AddNewBlockUnchecked for block with ID "+missing[i].ID))
			return
		}
	}

	logger.LogInfo(fmt.Sprintf("loadMissingBlocks loaded %d missing blocks", len(missing)))
}
