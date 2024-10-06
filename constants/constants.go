package constants

import (
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
)

var (
	DefaultRewardBase = common.HexToAddress("0x0000000000000000000000000000000000000000")

	ValidatorPrivateKey, _ = crypto.HexToECDSA("b2d2c7b2c54d52099048a4df70734985629b0f1ff5eb4118e0f3b08abaae0792")
	ValidatorAddress       = common.HexToAddress("0x2a4f7f1bf3374e2566d0a868246f8f141fc4328d")

	RandomPrivateKey, _ = crypto.GenerateKey()
	RandomAddress       = crypto.PubkeyToAddress(RandomPrivateKey.PublicKey)
)
