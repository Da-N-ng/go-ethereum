// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = abi.U256
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// ContractABI is the input ABI used to generate the binding from.
const ContractABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"GetAllAdmin\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"GetLatestCheckpoint\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"},{\"name\":\"\",\"type\":\"bytes32\"},{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_sectionIndex\",\"type\":\"uint256\"}],\"name\":\"GetCheckpoint\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"},{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_sectionIndex\",\"type\":\"uint256\"},{\"name\":\"_hash\",\"type\":\"bytes32\"},{\"name\":\"_sig\",\"type\":\"bytes\"}],\"name\":\"SetCheckpoint\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"GetPending\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"},{\"name\":\"\",\"type\":\"address[]\"},{\"name\":\"\",\"type\":\"bytes32[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_adminlist\",\"type\":\"address[]\"},{\"name\":\"_sectionSize\",\"type\":\"uint256\"},{\"name\":\"_processConfirms\",\"type\":\"uint256\"},{\"name\":\"_sigThreshold\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"index\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"checkpointHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"NewCheckpointEvent\",\"type\":\"event\"}]"

// ContractBin is the compiled bytecode used for deploying new contracts.
const ContractBin = `608060405234801561001057600080fd5b506040516113073803806113078339810180604052608081101561003357600080fd5b81019080805164010000000081111561004b57600080fd5b8281019050602081018481111561006157600080fd5b815185602082028301116401000000008211171561007e57600080fd5b505092919060200180519060200190929190805190602001909291908051906020019092919050505060008090505b845181101561019c5760016004600087848151811015156100ca57fe5b9060200190602002015173ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506005858281518110151561012257fe5b9060200190602002015190806001815401808255809150509060018203906000526020600020016000909192909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505080806001019150506100ad565b508260098190555081600a8190555080600b8190555050505050611142806101c56000396000f3fe608060405234801561001057600080fd5b5060043610610074576000357c01000000000000000000000000000000000000000000000000000000009004806345848dfc146100795780634d6a304c146100d8578063710aeac8146101045780639475a2b91461014d578063fff5f36714610234575b600080fd5b6100816102e2565b6040518080602001828103825283818151815260200191508051906020019060200280838360005b838110156100c45780820151818401526020810190506100a9565b505050509050019250505060405180910390f35b6100e06103c8565b60405180848152602001838152602001828152602001935050505060405180910390f35b6101306004803603602081101561011a57600080fd5b81019080803590602001909291905050506103f1565b604051808381526020018281526020019250505060405180910390f35b61021a6004803603606081101561016357600080fd5b8101908080359060200190929190803590602001909291908035906020019064010000000081111561019457600080fd5b8201836020820111156101a657600080fd5b803590602001918460018302840111640100000000831117156101c857600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f820116905080830192505050505050509192919290505050610425565b604051808215151515815260200191505060405180910390f35b61023c610bae565b604051808481526020018060200180602001838103835285818151815260200191508051906020019060200280838360005b8381101561028957808201518184015260208101905061026e565b50505050905001838103825284818151815260200191508051906020019060200280838360005b838110156102cb5780820151818401526020810190506102b0565b505050509050019550505050505060405180910390f35b6060806005805490506040519080825280602002602001820160405280156103195781602001602082028038833980820191505090505b50905060008090505b6005805490508110156103c05760058181548110151561033e57fe5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16828281518110151561037757fe5b9060200190602002019073ffffffffffffffffffffffffffffffffffffffff16908173ffffffffffffffffffffffffffffffffffffffff16815250508080600101915050610322565b508091505090565b60008060008060006103db6007546103f1565b9150915060075482829450945094505050909192565b6000806006600084815260200190815260200160002054600860008581526020019081526020016000205491509150915091565b600080600460003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205411151561047457600080fd5b600a54600954600186010201431080610494575060095460028501024310155b156104a25760009050610ba7565b600754841480156104d5575060006007541415806104d457506000600860008081526020019081526020016000205414155b5b156104e35760009050610ba7565b60008314156104f55760009050610ba7565b83600080015414151561050b5761050a610d92565b5b60008060020160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905083811415610563576000915050610ba7565b6000808214905084600060020160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508015156108a3576000806003016000848152602001908152602001600020905060008090505b81805490508110156107c9573373ffffffffffffffffffffffffffffffffffffffff16828281548110151561060857fe5b906000526020600020906002020160000160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614156107bc5760008190505b600183805490500381101561074557826001820181548110151561067c57fe5b9060005260206000209060020201838281548110151561069857fe5b90600052602060002090600202016000820160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff168160000160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555060018201816001019080546001816001161561010002031660029004610734929190610ef6565b50905050808060010191505061065c565b5081600183805490500381548110151561075b57fe5b9060005260206000209060020201600080820160006101000a81549073ffffffffffffffffffffffffffffffffffffffff02191690556001820160006107a19190610f7d565b5050600182818180549050039150816107ba9190610fc5565b505b80806001019150506105d7565b506000600301600087815260200190815260200160002060408051908101604052803373ffffffffffffffffffffffffffffffffffffffff168152602001878152509080600181540180825580915050906001820390600052602060002090600202016000909192909190915060008201518160000160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506020820151816001019080519060200190610899929190610ff7565b5050505050610ba0565b6001600060010160008282540192505081905550856000800181905550600080600301600087815260200190815260200160002090508060408051908101604052803373ffffffffffffffffffffffffffffffffffffffff168152602001878152509080600181540180825580915050906001820390600052602060002090600202016000909192909190915060008201518160000160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506020820151816001019080519060200190610993929190610ff7565b50505050600b54818054905010156109b15760019350505050610ba7565b85600660008981526020019081526020016000208190555043600860008981526020019081526020016000208190555086600781905550606060008090505b600b54811015610aef57818382815481101515610a0957fe5b90600052602060002090600202016001016040516020018083805190602001908083835b602083101515610a525780518252602082019150602081019050602083039250610a2d565b6001836020036101000a03801982511681845116808217855250505050505090500182805460018160011615610100020316600290048015610acb5780601f10610aa9576101008083540402835291820191610acb565b820191906000526020600020905b815481529060010190602001808311610ab7575b505092505050604051602081830303815290604052915080806001019150506109f0565b50877ff7aa4ddabff125da62b8692942a8dee5c673822157f230e5520a5b4e92d6929f88836040518083815260200180602001828103825283818151815260200191508051906020019080838360005b83811015610b5a578082015181840152602081019050610b3f565b50505050905090810190601f168015610b875780820380516001836020036101000a031916815260200191505b50935050505060405180910390a2610b9d610d92565b50505b6001925050505b9392505050565b600060608060008090506060600060010154604051908082528060200260200182016040528015610bee5781602001602082028038833980820191505090505b5090506060600060010154604051908082528060200260200182016040528015610c275781602001602082028038833980820191505090505b50905060008090505b600580549050811015610d7c576000806002016000600584815481101515610c5457fe5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050600081141515610d6e57600582815481101515610cd557fe5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff168486815181101515610d0e57fe5b9060200190602002019073ffffffffffffffffffffffffffffffffffffffff16908173ffffffffffffffffffffffffffffffffffffffff1681525050808386815181101515610d5957fe5b90602001906020020181815250506001850194505b508080600101915050610c30565b5060008001548282955095509550505050909192565b60008090505b600580549050811015610ede576000806002016000600584815481101515610dbc57fe5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050600081141515610ed057600060030160008281526020019081526020016000206000610e509190611077565b60006002016000600584815481101515610e6657fe5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600090555b508080600101915050610d98565b50600080600082016000905560018201600090555050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f10610f2f5780548555610f6c565b82800160010185558215610f6c57600052602060002091601f016020900482015b82811115610f6b578254825591600101919060010190610f50565b5b509050610f79919061109b565b5090565b50805460018160011615610100020316600290046000825580601f10610fa35750610fc2565b601f016020900490600052602060002090810190610fc1919061109b565b5b50565b815481835581811115610ff257600202816002028360005260206000209182019101610ff191906110c0565b5b505050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061103857805160ff1916838001178555611066565b82800160010185558215611066579182015b8281111561106557825182559160200191906001019061104a565b5b509050611073919061109b565b5090565b508054600082556002029060005260206000209081019061109891906110c0565b50565b6110bd91905b808211156110b95760008160009055506001016110a1565b5090565b90565b61111391905b8082111561110f57600080820160006101000a81549073ffffffffffffffffffffffffffffffffffffffff02191690556001820160006111069190610f7d565b506002016110c6565b5090565b9056fea165627a7a72305820bfbdb887ef1d682bd04df1a6ff8b36378c800f992557d8d66dc277c079400ef20029`

// DeployContract deploys a new Ethereum contract, binding an instance of Contract to it.
func DeployContract(auth *bind.TransactOpts, backend bind.ContractBackend, _adminlist []common.Address, _sectionSize *big.Int, _processConfirms *big.Int, _sigThreshold *big.Int) (common.Address, *types.Transaction, *Contract, error) {
	parsed, err := abi.JSON(strings.NewReader(ContractABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ContractBin), backend, _adminlist, _sectionSize, _processConfirms, _sigThreshold)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Contract{ContractCaller: ContractCaller{contract: contract}, ContractTransactor: ContractTransactor{contract: contract}, ContractFilterer: ContractFilterer{contract: contract}}, nil
}

// Contract is an auto generated Go binding around an Ethereum contract.
type Contract struct {
	ContractCaller     // Read-only binding to the contract
	ContractTransactor // Write-only binding to the contract
	ContractFilterer   // Log filterer for contract events
}

// ContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type ContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ContractSession struct {
	Contract     *Contract         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ContractCallerSession struct {
	Contract *ContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// ContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ContractTransactorSession struct {
	Contract     *ContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type ContractRaw struct {
	Contract *Contract // Generic contract binding to access the raw methods on
}

// ContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ContractCallerRaw struct {
	Contract *ContractCaller // Generic read-only contract binding to access the raw methods on
}

// ContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ContractTransactorRaw struct {
	Contract *ContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewContract creates a new instance of Contract, bound to a specific deployed contract.
func NewContract(address common.Address, backend bind.ContractBackend) (*Contract, error) {
	contract, err := bindContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Contract{ContractCaller: ContractCaller{contract: contract}, ContractTransactor: ContractTransactor{contract: contract}, ContractFilterer: ContractFilterer{contract: contract}}, nil
}

// NewContractCaller creates a new read-only instance of Contract, bound to a specific deployed contract.
func NewContractCaller(address common.Address, caller bind.ContractCaller) (*ContractCaller, error) {
	contract, err := bindContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ContractCaller{contract: contract}, nil
}

// NewContractTransactor creates a new write-only instance of Contract, bound to a specific deployed contract.
func NewContractTransactor(address common.Address, transactor bind.ContractTransactor) (*ContractTransactor, error) {
	contract, err := bindContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ContractTransactor{contract: contract}, nil
}

// NewContractFilterer creates a new log filterer instance of Contract, bound to a specific deployed contract.
func NewContractFilterer(address common.Address, filterer bind.ContractFilterer) (*ContractFilterer, error) {
	contract, err := bindContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ContractFilterer{contract: contract}, nil
}

// bindContract binds a generic wrapper to an already deployed contract.
func bindContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ContractABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Contract *ContractRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Contract.Contract.ContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Contract *ContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.Contract.ContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Contract *ContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Contract.Contract.ContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Contract *ContractCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Contract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Contract *ContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Contract *ContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Contract.Contract.contract.Transact(opts, method, params...)
}

// GetAllAdmin is a free data retrieval call binding the contract method 0x45848dfc.
//
// Solidity: function GetAllAdmin() constant returns(address[])
func (_Contract *ContractCaller) GetAllAdmin(opts *bind.CallOpts) ([]common.Address, error) {
	var (
		ret0 = new([]common.Address)
	)
	out := ret0
	err := _Contract.contract.Call(opts, out, "GetAllAdmin")
	return *ret0, err
}

// GetAllAdmin is a free data retrieval call binding the contract method 0x45848dfc.
//
// Solidity: function GetAllAdmin() constant returns(address[])
func (_Contract *ContractSession) GetAllAdmin() ([]common.Address, error) {
	return _Contract.Contract.GetAllAdmin(&_Contract.CallOpts)
}

// GetAllAdmin is a free data retrieval call binding the contract method 0x45848dfc.
//
// Solidity: function GetAllAdmin() constant returns(address[])
func (_Contract *ContractCallerSession) GetAllAdmin() ([]common.Address, error) {
	return _Contract.Contract.GetAllAdmin(&_Contract.CallOpts)
}

// GetCheckpoint is a free data retrieval call binding the contract method 0x710aeac8.
//
// Solidity: function GetCheckpoint(uint256 _sectionIndex) constant returns(bytes32, uint256)
func (_Contract *ContractCaller) GetCheckpoint(opts *bind.CallOpts, _sectionIndex *big.Int) ([32]byte, *big.Int, error) {
	var (
		ret0 = new([32]byte)
		ret1 = new(*big.Int)
	)
	out := &[]interface{}{
		ret0,
		ret1,
	}
	err := _Contract.contract.Call(opts, out, "GetCheckpoint", _sectionIndex)
	return *ret0, *ret1, err
}

// GetCheckpoint is a free data retrieval call binding the contract method 0x710aeac8.
//
// Solidity: function GetCheckpoint(uint256 _sectionIndex) constant returns(bytes32, uint256)
func (_Contract *ContractSession) GetCheckpoint(_sectionIndex *big.Int) ([32]byte, *big.Int, error) {
	return _Contract.Contract.GetCheckpoint(&_Contract.CallOpts, _sectionIndex)
}

// GetCheckpoint is a free data retrieval call binding the contract method 0x710aeac8.
//
// Solidity: function GetCheckpoint(uint256 _sectionIndex) constant returns(bytes32, uint256)
func (_Contract *ContractCallerSession) GetCheckpoint(_sectionIndex *big.Int) ([32]byte, *big.Int, error) {
	return _Contract.Contract.GetCheckpoint(&_Contract.CallOpts, _sectionIndex)
}

// GetLatestCheckpoint is a free data retrieval call binding the contract method 0x4d6a304c.
//
// Solidity: function GetLatestCheckpoint() constant returns(uint256, bytes32, uint256)
func (_Contract *ContractCaller) GetLatestCheckpoint(opts *bind.CallOpts) (*big.Int, [32]byte, *big.Int, error) {
	var (
		ret0 = new(*big.Int)
		ret1 = new([32]byte)
		ret2 = new(*big.Int)
	)
	out := &[]interface{}{
		ret0,
		ret1,
		ret2,
	}
	err := _Contract.contract.Call(opts, out, "GetLatestCheckpoint")
	return *ret0, *ret1, *ret2, err
}

// GetLatestCheckpoint is a free data retrieval call binding the contract method 0x4d6a304c.
//
// Solidity: function GetLatestCheckpoint() constant returns(uint256, bytes32, uint256)
func (_Contract *ContractSession) GetLatestCheckpoint() (*big.Int, [32]byte, *big.Int, error) {
	return _Contract.Contract.GetLatestCheckpoint(&_Contract.CallOpts)
}

// GetLatestCheckpoint is a free data retrieval call binding the contract method 0x4d6a304c.
//
// Solidity: function GetLatestCheckpoint() constant returns(uint256, bytes32, uint256)
func (_Contract *ContractCallerSession) GetLatestCheckpoint() (*big.Int, [32]byte, *big.Int, error) {
	return _Contract.Contract.GetLatestCheckpoint(&_Contract.CallOpts)
}

// GetPending is a free data retrieval call binding the contract method 0xfff5f367.
//
// Solidity: function GetPending() constant returns(uint256, address[], bytes32[])
func (_Contract *ContractCaller) GetPending(opts *bind.CallOpts) (*big.Int, []common.Address, [][32]byte, error) {
	var (
		ret0 = new(*big.Int)
		ret1 = new([]common.Address)
		ret2 = new([][32]byte)
	)
	out := &[]interface{}{
		ret0,
		ret1,
		ret2,
	}
	err := _Contract.contract.Call(opts, out, "GetPending")
	return *ret0, *ret1, *ret2, err
}

// GetPending is a free data retrieval call binding the contract method 0xfff5f367.
//
// Solidity: function GetPending() constant returns(uint256, address[], bytes32[])
func (_Contract *ContractSession) GetPending() (*big.Int, []common.Address, [][32]byte, error) {
	return _Contract.Contract.GetPending(&_Contract.CallOpts)
}

// GetPending is a free data retrieval call binding the contract method 0xfff5f367.
//
// Solidity: function GetPending() constant returns(uint256, address[], bytes32[])
func (_Contract *ContractCallerSession) GetPending() (*big.Int, []common.Address, [][32]byte, error) {
	return _Contract.Contract.GetPending(&_Contract.CallOpts)
}

// SetCheckpoint is a paid mutator transaction binding the contract method 0x9475a2b9.
//
// Solidity: function SetCheckpoint(uint256 _sectionIndex, bytes32 _hash, bytes _sig) returns(bool)
func (_Contract *ContractTransactor) SetCheckpoint(opts *bind.TransactOpts, _sectionIndex *big.Int, _hash [32]byte, _sig []byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "SetCheckpoint", _sectionIndex, _hash, _sig)
}

// SetCheckpoint is a paid mutator transaction binding the contract method 0x9475a2b9.
//
// Solidity: function SetCheckpoint(uint256 _sectionIndex, bytes32 _hash, bytes _sig) returns(bool)
func (_Contract *ContractSession) SetCheckpoint(_sectionIndex *big.Int, _hash [32]byte, _sig []byte) (*types.Transaction, error) {
	return _Contract.Contract.SetCheckpoint(&_Contract.TransactOpts, _sectionIndex, _hash, _sig)
}

// SetCheckpoint is a paid mutator transaction binding the contract method 0x9475a2b9.
//
// Solidity: function SetCheckpoint(uint256 _sectionIndex, bytes32 _hash, bytes _sig) returns(bool)
func (_Contract *ContractTransactorSession) SetCheckpoint(_sectionIndex *big.Int, _hash [32]byte, _sig []byte) (*types.Transaction, error) {
	return _Contract.Contract.SetCheckpoint(&_Contract.TransactOpts, _sectionIndex, _hash, _sig)
}

// ContractNewCheckpointEventIterator is returned from FilterNewCheckpointEvent and is used to iterate over the raw logs and unpacked data for NewCheckpointEvent events raised by the Contract contract.
type ContractNewCheckpointEventIterator struct {
	Event *ContractNewCheckpointEvent // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractNewCheckpointEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractNewCheckpointEvent)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractNewCheckpointEvent)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractNewCheckpointEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractNewCheckpointEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractNewCheckpointEvent represents a NewCheckpointEvent event raised by the Contract contract.
type ContractNewCheckpointEvent struct {
	Index          *big.Int
	CheckpointHash [32]byte
	Signature      []byte
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterNewCheckpointEvent is a free log retrieval operation binding the contract event 0xf7aa4ddabff125da62b8692942a8dee5c673822157f230e5520a5b4e92d6929f.
//
// Solidity: event NewCheckpointEvent(uint256 indexed index, bytes32 checkpointHash, bytes signature)
func (_Contract *ContractFilterer) FilterNewCheckpointEvent(opts *bind.FilterOpts, index []*big.Int) (*ContractNewCheckpointEventIterator, error) {

	var indexRule []interface{}
	for _, indexItem := range index {
		indexRule = append(indexRule, indexItem)
	}

	logs, sub, err := _Contract.contract.FilterLogs(opts, "NewCheckpointEvent", indexRule)
	if err != nil {
		return nil, err
	}
	return &ContractNewCheckpointEventIterator{contract: _Contract.contract, event: "NewCheckpointEvent", logs: logs, sub: sub}, nil
}

// WatchNewCheckpointEvent is a free log subscription operation binding the contract event 0xf7aa4ddabff125da62b8692942a8dee5c673822157f230e5520a5b4e92d6929f.
//
// Solidity: event NewCheckpointEvent(uint256 indexed index, bytes32 checkpointHash, bytes signature)
func (_Contract *ContractFilterer) WatchNewCheckpointEvent(opts *bind.WatchOpts, sink chan<- *ContractNewCheckpointEvent, index []*big.Int) (event.Subscription, error) {

	var indexRule []interface{}
	for _, indexItem := range index {
		indexRule = append(indexRule, indexItem)
	}

	logs, sub, err := _Contract.contract.WatchLogs(opts, "NewCheckpointEvent", indexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractNewCheckpointEvent)
				if err := _Contract.contract.UnpackLog(event, "NewCheckpointEvent", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseNewCheckpointEvent is a log parse operation binding the contract event 0xf7aa4ddabff125da62b8692942a8dee5c673822157f230e5520a5b4e92d6929f.
//
// Solidity: event NewCheckpointEvent(uint256 indexed index, bytes32 checkpointHash, bytes signature)
func (_Contract *ContractFilterer) ParseNewCheckpointEvent(log types.Log) (*ContractNewCheckpointEvent, error) {
	event := new(ContractNewCheckpointEvent)
	if err := _Contract.contract.UnpackLog(event, "NewCheckpointEvent", log); err != nil {
		return nil, err
	}
	return event, nil
}
