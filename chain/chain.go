package chain

import (
	"math/big"

	"github.com/hyeonLewis/go-pevm/constants"
	"github.com/kaiachain/kaia/blockchain"
	"github.com/kaiachain/kaia/blockchain/state"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/blockchain/vm"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/consensus"
	"github.com/kaiachain/kaia/consensus/istanbul"
	istanbulBackend "github.com/kaiachain/kaia/consensus/istanbul/backend"
	"github.com/kaiachain/kaia/governance"
	"github.com/kaiachain/kaia/log"
	"github.com/kaiachain/kaia/params"
	"github.com/kaiachain/kaia/storage/database"
)

var (
	logger = log.NewModuleLogger(0)
)

// generateGovernaceDataForTest returns governance.Engine for test.
func newGovernance(db database.DBManager) governance.Engine {
	return governance.NewMixedEngine(&params.ChainConfig{
		ChainID:       big.NewInt(2018),
		UnitPrice:     25000000000,
		DeriveShaImpl: 0,
		Istanbul: &params.IstanbulConfig{
			Epoch:          istanbul.DefaultConfig.Epoch,
			ProposerPolicy: uint64(istanbul.DefaultConfig.ProposerPolicy),
			SubGroupSize:   istanbul.DefaultConfig.SubGroupSize,
		},
		Governance: params.GetDefaultGovernanceConfig(),
	}, db)
}

func newEngine(db database.DBManager) consensus.Istanbul {
	engine := istanbulBackend.New(&istanbulBackend.BackendOpts{
		IstanbulConfig: istanbul.DefaultConfig,
		Rewardbase:     constants.DefaultRewardBase,
		PrivateKey:     constants.ValidatorPrivateKey,
		DB:             db,
		Governance:     newGovernance(db),
		NodeType:       common.CONSENSUSNODE,
	})

	return engine
}

func NewBlockchain(db database.DBManager, config *params.ChainConfig) *blockchain.BlockChain {
	engine := newEngine(db)

	chain, err := blockchain.NewBlockChain(db, nil, config, engine, vm.Config{})
	if err != nil {
		logger.Info("Failed to create blockchain", "error", err)
	}

	return chain
}

type Task struct {
	config *params.ChainConfig
	state  *state.StateDB
	header *types.Header
}

func NewTask(config *params.ChainConfig, state *state.StateDB, header *types.Header) *Task {
	return &Task{
		config: config,
		state:  state,
		header: header,
	}
}

func (env *Task) CommitTransaction(tx *types.Transaction, bc *blockchain.BlockChain, rewardbase common.Address, vmConfig *vm.Config) (error, []*types.Log) {
	snap := env.state.Snapshot()

	receipt, _, err := bc.ApplyTransaction(env.config, &rewardbase, env.state, env.header, tx, &env.header.GasUsed, vmConfig)
	if err != nil {
		if err != vm.ErrInsufficientBalance && err != vm.ErrTotalTimeLimitReached {
			tx.MarkUnexecutable(true)
		}
		env.state.RevertToSnapshot(snap)
		return err, nil
	}

	return nil, receipt.Logs
}
