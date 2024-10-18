package multiversion

import (
	"math/big"
	"strings"

	"github.com/kaiachain/kaia/blockchain/state"
	"github.com/kaiachain/kaia/common"
)

type ExtraKey common.Hash

var (
	CodeCopyKey     = common.BytesToHash([]byte("codecopy"))
	CodeHashKey     = common.BytesToHash([]byte("codehash"))
	CodeSizeKey     = common.BytesToHash([]byte("codesize"))
	BalanceKey      = common.BytesToHash([]byte("balance"))
	SelfDestructKey = common.BytesToHash([]byte("selfdestruct"))
)

type StorageKey string

func ToStorageKey(addr common.Address, slot common.Hash) StorageKey {
	return StorageKey(addr.String() + ":" + slot.String())
}

func (k StorageKey) Bytes() []byte {
	return []byte(k)
}

func (k StorageKey) Address() common.Address {
	split := strings.Split(string(k), ":")
	if len(split) < 1 {
		return common.Address{}
	}
	return common.HexToAddress(split[0])
}

func (k StorageKey) Slot() common.Hash {
	split := strings.Split(string(k), ":")
	if len(split) < 2 {
		return common.Hash{}
	}
	return common.HexToHash(split[1])
}

func (k StorageKey) String() string {
	return string(k)
}

func (k StorageKey) GetValue(stateDB *state.StateDB) common.Hash {
	switch k.Slot() {
	case CodeCopyKey, CodeHashKey:
		codeHash := stateDB.GetCodeHash(k.Address())
		return codeHash
	case CodeSizeKey:
		codeSize := stateDB.GetCodeSize(k.Address())
		return common.BigToHash(big.NewInt(int64(codeSize)))
	case BalanceKey:
		balance := stateDB.GetBalance(k.Address())
		return common.BigToHash(balance)
	case SelfDestructKey:
		return common.Hash{}
	default:
		return stateDB.GetState(k.Address(), k.Slot())
	}
}
