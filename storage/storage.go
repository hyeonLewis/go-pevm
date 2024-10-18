package storage

import (
	"encoding/json"
	"math/big"
	"os"

	"github.com/kaiachain/kaia/blockchain"
	"github.com/kaiachain/kaia/blockchain/state"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/log"
	"github.com/kaiachain/kaia/storage/database"
)

var (
	logger = log.NewModuleLogger(1)
)

const genesisPath = "../data/genesis.json"
const networkId = 8217

type Account struct {
	Balance string            `json:"balance"`
	Nonce   int               `json:"nonce"`
	Storage map[string]string `json:"storage"`
}

type Accounts map[string]Account

func NewInMemoryStorage() database.DBManager {
	return database.NewMemoryDBManager()
}

func InjectGenesis(db database.DBManager) {
	genesis, err := readGenesis(genesisPath)
	if err != nil {
		logger.Info("Failed to read genesis", "error", err)
	}

	_, _, genesisErr := blockchain.SetupGenesisBlock(db, genesis, networkId, false, false)
	if genesisErr != nil {
		logger.Info("Failed to setup genesis", "error", genesisErr)
	}
}

func Has(state *state.StateDB, addr common.Address, key common.Hash) bool {
	value := state.GetState(addr, key)
	return value != (common.Hash{})
}

func CommitPreState(strBn string, state *state.StateDB) common.Hash {
	file, err := os.Open("../data/" + strBn + "/pre_state.json")
	if err != nil {
		logger.Info("Failed to open pre_state.json", "error", err)
	}
	defer file.Close()

	accounts := Accounts{}
	err = json.NewDecoder(file).Decode(&accounts)
	if err != nil {
		logger.Info("Failed to decode pre_state.json", "error", err)
	}

	for addr, account := range accounts {
		state.SetBalance(common.HexToAddress(addr), new(big.Int).SetBytes(common.FromHex(account.Balance)))
		state.SetNonce(common.HexToAddress(addr), uint64(account.Nonce))
		for key, value := range account.Storage {
			state.SetState(common.HexToAddress(addr), common.HexToHash(key), common.HexToHash(value))
		}
	}

	root, err := state.Commit(true)
	if err != nil {
		logger.Info("Failed to commit pre_state", "error", err)
	}

	return root
}

func readGenesis(path string) (*blockchain.Genesis, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	genesis := &blockchain.Genesis{}
	err = json.NewDecoder(file).Decode(genesis)
	if err != nil {
		logger.Info("Failed to decode genesis", "error", err)
		return nil, err
	}

	return genesis, nil
}
