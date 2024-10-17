package multiversion

import (
	"strings"

	"github.com/kaiachain/kaia/common"
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
	return common.HexToAddress(split[0])
}

func (k StorageKey) Slot() common.Hash {
	split := strings.Split(string(k), ":")
	return common.HexToHash(split[1])
}

func (k StorageKey) String() string {
	return string(k)
}
