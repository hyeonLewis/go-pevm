// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package test

import (
	"errors"
	"math/big"
	"strings"

	kaia "github.com/kaiachain/kaia"
	"github.com/kaiachain/kaia/accounts/abi"
	"github.com/kaiachain/kaia/accounts/abi/bind"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = kaia.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// TestMetaData contains all meta data concerning the Test contract.
var TestMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"addrArr\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"arr\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"bytesArr\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"len\",\"type\":\"uint256\"}],\"name\":\"setAddrArr\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"len\",\"type\":\"uint256\"}],\"name\":\"setAllArrays\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"len\",\"type\":\"uint256\"}],\"name\":\"setArr\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"len\",\"type\":\"uint256\"}],\"name\":\"setBytesArr\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Sigs: map[string]string{
		"14208d8f": "addrArr(uint256)",
		"71e5ee5f": "arr(uint256)",
		"a942a1fa": "bytesArr(uint256)",
		"74e6e2a8": "setAddrArr(uint256)",
		"c90462a6": "setAllArrays(uint256)",
		"08255318": "setArr(uint256)",
		"8ed469a8": "setBytesArr(uint256)",
	},
	Bin: "0x6080604052348015600e575f80fd5b506105168061001c5f395ff3fe608060405234801561000f575f80fd5b506004361061007a575f3560e01c806374e6e2a81161005857806374e6e2a8146100e45780638ed469a8146100f7578063a942a1fa1461010a578063c90462a61461012a575f80fd5b8063082553181461007e57806314208d8f1461009357806371e5ee5f146100c3575b5f80fd5b61009161008c36600461033c565b61013d565b005b6100a66100a136600461033c565b610183565b6040516001600160a01b0390911681526020015b60405180910390f35b6100d66100d136600461033c565b6101ab565b6040519081526020016100ba565b6100916100f236600461033c565b6101c9565b61009161010536600461033c565b610221565b61011d61011836600461033c565b610278565b6040516100ba9190610353565b61009161013836600461033c565b61031e565b5f5b8181101561017f575f8054600181810183559180527f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563018290550161013f565b5050565b60018181548110610192575f80fd5b5f918252602090912001546001600160a01b0316905081565b5f81815481106101b9575f80fd5b5f91825260209091200154905081565b5f5b8181101561017f576001805480820182555f8290527fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf60180546001600160a01b0319166001600160a01b038416179055016101cb565b5f5b8181101561017f5760028160405160200161024091815260200190565b60408051601f1981840301815291905281546001810183555f92835260209092209091019061026f9082610420565b50600101610223565b60028181548110610287575f80fd5b905f5260205f20015f91509050805461029f9061039c565b80601f01602080910402602001604051908101604052809291908181526020018280546102cb9061039c565b80156103165780601f106102ed57610100808354040283529160200191610316565b820191905f5260205f20905b8154815290600101906020018083116102f957829003601f168201915b505050505081565b6103278161013d565b610330816101c9565b61033981610221565b50565b5f6020828403121561034c575f80fd5b5035919050565b602081525f82518060208401528060208501604085015e5f604082850101526040601f19601f83011684010191505092915050565b634e487b7160e01b5f52604160045260245ffd5b600181811c908216806103b057607f821691505b6020821081036103ce57634e487b7160e01b5f52602260045260245ffd5b50919050565b601f82111561041b57805f5260205f20601f840160051c810160208510156103f95750805b601f840160051c820191505b81811015610418575f8155600101610405565b50505b505050565b815167ffffffffffffffff81111561043a5761043a610388565b61044e81610448845461039c565b846103d4565b602080601f831160018114610481575f841561046a5750858301515b5f19600386901b1c1916600185901b1785556104d8565b5f85815260208120601f198616915b828110156104af57888601518255948401946001909101908401610490565b50858210156104cc57878501515f19600388901b60f8161c191681555b505060018460011b0185555b50505050505056fea26469706673582212209ed12e98a8c9f8b1658d551cc17b11b26861bd061698a4290589a5358f1a6c8564736f6c63430008190033",
}

// TestABI is the input ABI used to generate the binding from.
// Deprecated: Use TestMetaData.ABI instead.
var TestABI = TestMetaData.ABI

// TestBinRuntime is the compiled bytecode used for adding genesis block without deploying code.
const TestBinRuntime = `608060405234801561000f575f80fd5b506004361061007a575f3560e01c806374e6e2a81161005857806374e6e2a8146100e45780638ed469a8146100f7578063a942a1fa1461010a578063c90462a61461012a575f80fd5b8063082553181461007e57806314208d8f1461009357806371e5ee5f146100c3575b5f80fd5b61009161008c36600461033c565b61013d565b005b6100a66100a136600461033c565b610183565b6040516001600160a01b0390911681526020015b60405180910390f35b6100d66100d136600461033c565b6101ab565b6040519081526020016100ba565b6100916100f236600461033c565b6101c9565b61009161010536600461033c565b610221565b61011d61011836600461033c565b610278565b6040516100ba9190610353565b61009161013836600461033c565b61031e565b5f5b8181101561017f575f8054600181810183559180527f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563018290550161013f565b5050565b60018181548110610192575f80fd5b5f918252602090912001546001600160a01b0316905081565b5f81815481106101b9575f80fd5b5f91825260209091200154905081565b5f5b8181101561017f576001805480820182555f8290527fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf60180546001600160a01b0319166001600160a01b038416179055016101cb565b5f5b8181101561017f5760028160405160200161024091815260200190565b60408051601f1981840301815291905281546001810183555f92835260209092209091019061026f9082610420565b50600101610223565b60028181548110610287575f80fd5b905f5260205f20015f91509050805461029f9061039c565b80601f01602080910402602001604051908101604052809291908181526020018280546102cb9061039c565b80156103165780601f106102ed57610100808354040283529160200191610316565b820191905f5260205f20905b8154815290600101906020018083116102f957829003601f168201915b505050505081565b6103278161013d565b610330816101c9565b61033981610221565b50565b5f6020828403121561034c575f80fd5b5035919050565b602081525f82518060208401528060208501604085015e5f604082850101526040601f19601f83011684010191505092915050565b634e487b7160e01b5f52604160045260245ffd5b600181811c908216806103b057607f821691505b6020821081036103ce57634e487b7160e01b5f52602260045260245ffd5b50919050565b601f82111561041b57805f5260205f20601f840160051c810160208510156103f95750805b601f840160051c820191505b81811015610418575f8155600101610405565b50505b505050565b815167ffffffffffffffff81111561043a5761043a610388565b61044e81610448845461039c565b846103d4565b602080601f831160018114610481575f841561046a5750858301515b5f19600386901b1c1916600185901b1785556104d8565b5f85815260208120601f198616915b828110156104af57888601518255948401946001909101908401610490565b50858210156104cc57878501515f19600388901b60f8161c191681555b505060018460011b0185555b50505050505056fea26469706673582212209ed12e98a8c9f8b1658d551cc17b11b26861bd061698a4290589a5358f1a6c8564736f6c63430008190033`

// TestFuncSigs maps the 4-byte function signature to its string representation.
// Deprecated: Use TestMetaData.Sigs instead.
var TestFuncSigs = TestMetaData.Sigs

// TestBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use TestMetaData.Bin instead.
var TestBin = TestMetaData.Bin

// DeployTest deploys a new Kaia contract, binding an instance of Test to it.
func DeployTest(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Test, error) {
	parsed, err := TestMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(TestBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Test{TestCaller: TestCaller{contract: contract}, TestTransactor: TestTransactor{contract: contract}, TestFilterer: TestFilterer{contract: contract}}, nil
}

// Test is an auto generated Go binding around a Kaia contract.
type Test struct {
	TestCaller     // Read-only binding to the contract
	TestTransactor // Write-only binding to the contract
	TestFilterer   // Log filterer for contract events
}

// TestCaller is an auto generated read-only Go binding around a Kaia contract.
type TestCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestTransactor is an auto generated write-only Go binding around a Kaia contract.
type TestTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestFilterer is an auto generated log filtering Go binding around a Kaia contract events.
type TestFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestSession is an auto generated Go binding around a Kaia contract,
// with pre-set call and transact options.
type TestSession struct {
	Contract     *Test             // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TestCallerSession is an auto generated read-only Go binding around a Kaia contract,
// with pre-set call options.
type TestCallerSession struct {
	Contract *TestCaller   // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// TestTransactorSession is an auto generated write-only Go binding around a Kaia contract,
// with pre-set transact options.
type TestTransactorSession struct {
	Contract     *TestTransactor   // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TestRaw is an auto generated low-level Go binding around a Kaia contract.
type TestRaw struct {
	Contract *Test // Generic contract binding to access the raw methods on
}

// TestCallerRaw is an auto generated low-level read-only Go binding around a Kaia contract.
type TestCallerRaw struct {
	Contract *TestCaller // Generic read-only contract binding to access the raw methods on
}

// TestTransactorRaw is an auto generated low-level write-only Go binding around a Kaia contract.
type TestTransactorRaw struct {
	Contract *TestTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTest creates a new instance of Test, bound to a specific deployed contract.
func NewTest(address common.Address, backend bind.ContractBackend) (*Test, error) {
	contract, err := bindTest(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Test{TestCaller: TestCaller{contract: contract}, TestTransactor: TestTransactor{contract: contract}, TestFilterer: TestFilterer{contract: contract}}, nil
}

// NewTestCaller creates a new read-only instance of Test, bound to a specific deployed contract.
func NewTestCaller(address common.Address, caller bind.ContractCaller) (*TestCaller, error) {
	contract, err := bindTest(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TestCaller{contract: contract}, nil
}

// NewTestTransactor creates a new write-only instance of Test, bound to a specific deployed contract.
func NewTestTransactor(address common.Address, transactor bind.ContractTransactor) (*TestTransactor, error) {
	contract, err := bindTest(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TestTransactor{contract: contract}, nil
}

// NewTestFilterer creates a new log filterer instance of Test, bound to a specific deployed contract.
func NewTestFilterer(address common.Address, filterer bind.ContractFilterer) (*TestFilterer, error) {
	contract, err := bindTest(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TestFilterer{contract: contract}, nil
}

// bindTest binds a generic wrapper to an already deployed contract.
func bindTest(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TestMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Test *TestRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Test.Contract.TestCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Test *TestRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Test.Contract.TestTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Test *TestRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Test.Contract.TestTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Test *TestCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Test.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Test *TestTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Test.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Test *TestTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Test.Contract.contract.Transact(opts, method, params...)
}

// AddrArr is a free data retrieval call binding the contract method 0x14208d8f.
//
// Solidity: function addrArr(uint256 ) view returns(address)
func (_Test *TestCaller) AddrArr(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Test.contract.Call(opts, &out, "addrArr", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AddrArr is a free data retrieval call binding the contract method 0x14208d8f.
//
// Solidity: function addrArr(uint256 ) view returns(address)
func (_Test *TestSession) AddrArr(arg0 *big.Int) (common.Address, error) {
	return _Test.Contract.AddrArr(&_Test.CallOpts, arg0)
}

// AddrArr is a free data retrieval call binding the contract method 0x14208d8f.
//
// Solidity: function addrArr(uint256 ) view returns(address)
func (_Test *TestCallerSession) AddrArr(arg0 *big.Int) (common.Address, error) {
	return _Test.Contract.AddrArr(&_Test.CallOpts, arg0)
}

// Arr is a free data retrieval call binding the contract method 0x71e5ee5f.
//
// Solidity: function arr(uint256 ) view returns(uint256)
func (_Test *TestCaller) Arr(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Test.contract.Call(opts, &out, "arr", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Arr is a free data retrieval call binding the contract method 0x71e5ee5f.
//
// Solidity: function arr(uint256 ) view returns(uint256)
func (_Test *TestSession) Arr(arg0 *big.Int) (*big.Int, error) {
	return _Test.Contract.Arr(&_Test.CallOpts, arg0)
}

// Arr is a free data retrieval call binding the contract method 0x71e5ee5f.
//
// Solidity: function arr(uint256 ) view returns(uint256)
func (_Test *TestCallerSession) Arr(arg0 *big.Int) (*big.Int, error) {
	return _Test.Contract.Arr(&_Test.CallOpts, arg0)
}

// BytesArr is a free data retrieval call binding the contract method 0xa942a1fa.
//
// Solidity: function bytesArr(uint256 ) view returns(bytes)
func (_Test *TestCaller) BytesArr(opts *bind.CallOpts, arg0 *big.Int) ([]byte, error) {
	var out []interface{}
	err := _Test.contract.Call(opts, &out, "bytesArr", arg0)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// BytesArr is a free data retrieval call binding the contract method 0xa942a1fa.
//
// Solidity: function bytesArr(uint256 ) view returns(bytes)
func (_Test *TestSession) BytesArr(arg0 *big.Int) ([]byte, error) {
	return _Test.Contract.BytesArr(&_Test.CallOpts, arg0)
}

// BytesArr is a free data retrieval call binding the contract method 0xa942a1fa.
//
// Solidity: function bytesArr(uint256 ) view returns(bytes)
func (_Test *TestCallerSession) BytesArr(arg0 *big.Int) ([]byte, error) {
	return _Test.Contract.BytesArr(&_Test.CallOpts, arg0)
}

// SetAddrArr is a paid mutator transaction binding the contract method 0x74e6e2a8.
//
// Solidity: function setAddrArr(uint256 len) returns()
func (_Test *TestTransactor) SetAddrArr(opts *bind.TransactOpts, len *big.Int) (*types.Transaction, error) {
	return _Test.contract.Transact(opts, "setAddrArr", len)
}

// SetAddrArr is a paid mutator transaction binding the contract method 0x74e6e2a8.
//
// Solidity: function setAddrArr(uint256 len) returns()
func (_Test *TestSession) SetAddrArr(len *big.Int) (*types.Transaction, error) {
	return _Test.Contract.SetAddrArr(&_Test.TransactOpts, len)
}

// SetAddrArr is a paid mutator transaction binding the contract method 0x74e6e2a8.
//
// Solidity: function setAddrArr(uint256 len) returns()
func (_Test *TestTransactorSession) SetAddrArr(len *big.Int) (*types.Transaction, error) {
	return _Test.Contract.SetAddrArr(&_Test.TransactOpts, len)
}

// SetAllArrays is a paid mutator transaction binding the contract method 0xc90462a6.
//
// Solidity: function setAllArrays(uint256 len) returns()
func (_Test *TestTransactor) SetAllArrays(opts *bind.TransactOpts, len *big.Int) (*types.Transaction, error) {
	return _Test.contract.Transact(opts, "setAllArrays", len)
}

// SetAllArrays is a paid mutator transaction binding the contract method 0xc90462a6.
//
// Solidity: function setAllArrays(uint256 len) returns()
func (_Test *TestSession) SetAllArrays(len *big.Int) (*types.Transaction, error) {
	return _Test.Contract.SetAllArrays(&_Test.TransactOpts, len)
}

// SetAllArrays is a paid mutator transaction binding the contract method 0xc90462a6.
//
// Solidity: function setAllArrays(uint256 len) returns()
func (_Test *TestTransactorSession) SetAllArrays(len *big.Int) (*types.Transaction, error) {
	return _Test.Contract.SetAllArrays(&_Test.TransactOpts, len)
}

// SetArr is a paid mutator transaction binding the contract method 0x08255318.
//
// Solidity: function setArr(uint256 len) returns()
func (_Test *TestTransactor) SetArr(opts *bind.TransactOpts, len *big.Int) (*types.Transaction, error) {
	return _Test.contract.Transact(opts, "setArr", len)
}

// SetArr is a paid mutator transaction binding the contract method 0x08255318.
//
// Solidity: function setArr(uint256 len) returns()
func (_Test *TestSession) SetArr(len *big.Int) (*types.Transaction, error) {
	return _Test.Contract.SetArr(&_Test.TransactOpts, len)
}

// SetArr is a paid mutator transaction binding the contract method 0x08255318.
//
// Solidity: function setArr(uint256 len) returns()
func (_Test *TestTransactorSession) SetArr(len *big.Int) (*types.Transaction, error) {
	return _Test.Contract.SetArr(&_Test.TransactOpts, len)
}

// SetBytesArr is a paid mutator transaction binding the contract method 0x8ed469a8.
//
// Solidity: function setBytesArr(uint256 len) returns()
func (_Test *TestTransactor) SetBytesArr(opts *bind.TransactOpts, len *big.Int) (*types.Transaction, error) {
	return _Test.contract.Transact(opts, "setBytesArr", len)
}

// SetBytesArr is a paid mutator transaction binding the contract method 0x8ed469a8.
//
// Solidity: function setBytesArr(uint256 len) returns()
func (_Test *TestSession) SetBytesArr(len *big.Int) (*types.Transaction, error) {
	return _Test.Contract.SetBytesArr(&_Test.TransactOpts, len)
}

// SetBytesArr is a paid mutator transaction binding the contract method 0x8ed469a8.
//
// Solidity: function setBytesArr(uint256 len) returns()
func (_Test *TestTransactorSession) SetBytesArr(len *big.Int) (*types.Transaction, error) {
	return _Test.Contract.SetBytesArr(&_Test.TransactOpts, len)
}
