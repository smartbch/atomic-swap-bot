package htlcsbch

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	_abiJsonStr = `[
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "minStakedValue",
          "type": "uint256"
        },
        {
          "internalType": "uint256",
          "name": "minRetireDelay",
          "type": "uint256"
        }
      ],
      "stateMutability": "nonpayable",
      "type": "constructor"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": true,
          "internalType": "address",
          "name": "_sender",
          "type": "address"
        },
        {
          "indexed": true,
          "internalType": "address",
          "name": "_receiver",
          "type": "address"
        },
        {
          "indexed": false,
          "internalType": "bytes32",
          "name": "_secretLock",
          "type": "bytes32"
        },
        {
          "indexed": false,
          "internalType": "uint256",
          "name": "_unlockTime",
          "type": "uint256"
        },
        {
          "indexed": false,
          "internalType": "uint256",
          "name": "_value",
          "type": "uint256"
        },
        {
          "indexed": false,
          "internalType": "bytes20",
          "name": "_receiverBchPkh",
          "type": "bytes20"
        },
        {
          "indexed": false,
          "internalType": "uint256",
          "name": "_createdTime",
          "type": "uint256"
        },
        {
          "indexed": false,
          "internalType": "uint16",
          "name": "_penaltyBPS",
          "type": "uint16"
        }
      ],
      "name": "Lock",
      "type": "event"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": true,
          "internalType": "bytes32",
          "name": "_secretLock",
          "type": "bytes32"
        }
      ],
      "name": "Refund",
      "type": "event"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": true,
          "internalType": "bytes32",
          "name": "_secretLock",
          "type": "bytes32"
        },
        {
          "indexed": true,
          "internalType": "bytes32",
          "name": "_secretKey",
          "type": "bytes32"
        }
      ],
      "name": "Unlock",
      "type": "event"
    },
    {
      "inputs": [],
      "name": "MIN_RETIRE_DELAY",
      "outputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [],
      "name": "MIN_STAKED_VALUE",
      "outputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "fromIdx",
          "type": "uint256"
        },
        {
          "internalType": "uint256",
          "name": "count",
          "type": "uint256"
        }
      ],
      "name": "getMarketMakers",
      "outputs": [
        {
          "components": [
            {
              "internalType": "address",
              "name": "addr",
              "type": "address"
            },
            {
              "internalType": "uint64",
              "name": "retiredAt",
              "type": "uint64"
            },
            {
              "internalType": "bytes32",
              "name": "intro",
              "type": "bytes32"
            },
            {
              "internalType": "bytes20",
              "name": "bchPkh",
              "type": "bytes20"
            },
            {
              "internalType": "uint16",
              "name": "bchLockTime",
              "type": "uint16"
            },
            {
              "internalType": "uint32",
              "name": "sbchLockTime",
              "type": "uint32"
            },
            {
              "internalType": "uint16",
              "name": "penaltyBPS",
              "type": "uint16"
            },
            {
              "internalType": "uint16",
              "name": "feeBPS",
              "type": "uint16"
            },
            {
              "internalType": "uint256",
              "name": "minSwapAmt",
              "type": "uint256"
            },
            {
              "internalType": "uint256",
              "name": "maxSwapAmt",
              "type": "uint256"
            },
            {
              "internalType": "uint256",
              "name": "stakedValue",
              "type": "uint256"
            },
            {
              "internalType": "address",
              "name": "statusChecker",
              "type": "address"
            },
            {
              "internalType": "bool",
              "name": "unavailable",
              "type": "bool"
            }
          ],
          "internalType": "struct AtomicSwapEther.MarketMaker[]",
          "name": "list",
          "type": "tuple[]"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "bytes32",
          "name": "secretLock",
          "type": "bytes32"
        }
      ],
      "name": "getSwapState",
      "outputs": [
        {
          "internalType": "enum AtomicSwapEther.States",
          "name": "",
          "type": "uint8"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address payable",
          "name": "_receiver",
          "type": "address"
        },
        {
          "internalType": "bytes32",
          "name": "_secretLock",
          "type": "bytes32"
        },
        {
          "internalType": "uint256",
          "name": "_validPeriod",
          "type": "uint256"
        },
        {
          "internalType": "bytes20",
          "name": "_receiverBchPkh",
          "type": "bytes20"
        },
        {
          "internalType": "uint16",
          "name": "_penaltyBPS",
          "type": "uint16"
        },
        {
          "internalType": "bool",
          "name": "_receiverIsMM",
          "type": "bool"
        }
      ],
      "name": "lock",
      "outputs": [],
      "stateMutability": "payable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "",
          "type": "address"
        }
      ],
      "name": "marketMakers",
      "outputs": [
        {
          "internalType": "address",
          "name": "addr",
          "type": "address"
        },
        {
          "internalType": "uint64",
          "name": "retiredAt",
          "type": "uint64"
        },
        {
          "internalType": "bytes32",
          "name": "intro",
          "type": "bytes32"
        },
        {
          "internalType": "bytes20",
          "name": "bchPkh",
          "type": "bytes20"
        },
        {
          "internalType": "uint16",
          "name": "bchLockTime",
          "type": "uint16"
        },
        {
          "internalType": "uint32",
          "name": "sbchLockTime",
          "type": "uint32"
        },
        {
          "internalType": "uint16",
          "name": "penaltyBPS",
          "type": "uint16"
        },
        {
          "internalType": "uint16",
          "name": "feeBPS",
          "type": "uint16"
        },
        {
          "internalType": "uint256",
          "name": "minSwapAmt",
          "type": "uint256"
        },
        {
          "internalType": "uint256",
          "name": "maxSwapAmt",
          "type": "uint256"
        },
        {
          "internalType": "uint256",
          "name": "stakedValue",
          "type": "uint256"
        },
        {
          "internalType": "address",
          "name": "statusChecker",
          "type": "address"
        },
        {
          "internalType": "bool",
          "name": "unavailable",
          "type": "bool"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "bytes32",
          "name": "_secretLock",
          "type": "bytes32"
        }
      ],
      "name": "refund",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "bytes32",
          "name": "_intro",
          "type": "bytes32"
        },
        {
          "internalType": "bytes20",
          "name": "_bchPkh",
          "type": "bytes20"
        },
        {
          "internalType": "uint16",
          "name": "_bchLockTime",
          "type": "uint16"
        },
        {
          "internalType": "uint16",
          "name": "_penaltyBPS",
          "type": "uint16"
        },
        {
          "internalType": "uint16",
          "name": "_feeBPS",
          "type": "uint16"
        },
        {
          "internalType": "uint256",
          "name": "_minSwapAmt",
          "type": "uint256"
        },
        {
          "internalType": "uint256",
          "name": "_maxSwapAmt",
          "type": "uint256"
        },
        {
          "internalType": "address",
          "name": "_statusChecker",
          "type": "address"
        }
      ],
      "name": "registerMarketMaker",
      "outputs": [],
      "stateMutability": "payable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "_delay",
          "type": "uint256"
        }
      ],
      "name": "retireMarketMaker",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "marketMaker",
          "type": "address"
        },
        {
          "internalType": "bool",
          "name": "b",
          "type": "bool"
        }
      ],
      "name": "setUnavailable",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "bytes32",
          "name": "",
          "type": "bytes32"
        }
      ],
      "name": "swaps",
      "outputs": [
        {
          "internalType": "bool",
          "name": "receiverIsMM",
          "type": "bool"
        },
        {
          "internalType": "uint64",
          "name": "startTime",
          "type": "uint64"
        },
        {
          "internalType": "uint64",
          "name": "startHeight",
          "type": "uint64"
        },
        {
          "internalType": "uint32",
          "name": "validPeriod",
          "type": "uint32"
        },
        {
          "internalType": "address payable",
          "name": "sender",
          "type": "address"
        },
        {
          "internalType": "address payable",
          "name": "receiver",
          "type": "address"
        },
        {
          "internalType": "uint96",
          "name": "value",
          "type": "uint96"
        },
        {
          "internalType": "bytes20",
          "name": "receiverBchPkh",
          "type": "bytes20"
        },
        {
          "internalType": "uint16",
          "name": "penaltyBPS",
          "type": "uint16"
        },
        {
          "internalType": "enum AtomicSwapEther.States",
          "name": "state",
          "type": "uint8"
        },
        {
          "internalType": "bytes32",
          "name": "secretKey",
          "type": "bytes32"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "bytes32",
          "name": "_secretLock",
          "type": "bytes32"
        },
        {
          "internalType": "bytes32",
          "name": "_secretKey",
          "type": "bytes32"
        }
      ],
      "name": "unlock",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "bytes32",
          "name": "_intro",
          "type": "bytes32"
        }
      ],
      "name": "updateMarketMaker",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [],
      "name": "withdrawStakedValue",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    }
  ]`
)

var (
	htlcAbi       abi.ABI
	LockEventId   common.Hash
	UnlockEventId common.Hash
	RefundEventId common.Hash
)

/*
   struct MarketMaker {
       address addr;          // EVM address
       uint64  retiredAt;     // retired time
       bytes32 intro;         // introduction
       bytes20 bchPkh;        // BCH P2PKH address
       uint16  bchLockTime;   // BCH HTLC lock time (in blocks)
       uint32  sbchLockTime;  // sBCH HTLC lock time (in seconds)
       uint16  penaltyBPS;    // refund penalty ratio (in BPS)
       uint16  feeBPS;        // service fee ratio (in BPS)
       uint256 minSwapAmt;    //
       uint256 maxSwapAmt;    //
       uint256 stakedValue;   // to prevent spam bots
       address statusChecker; // the one who can set unavailable status
       bool    unavailable;   //
   }
*/

type MarketMakerInfo struct {
	Addr         common.Address
	RetiredAt    uint64
	Intro        [32]byte
	BchPkh       [20]byte
	BchLockTime  uint16
	SbchLockTime uint32
	PenaltyBPS   uint16
	FeeBPS       uint16
	MinSwapAmt   *big.Int
	MaxSwapAmt   *big.Int
	StakedValue  *big.Int
	Checker      common.Address
	Unavailable  bool
}

func init() {
	var err error
	htlcAbi, err = abi.JSON(strings.NewReader(_abiJsonStr))
	if err != nil {
		panic("failed to parse HTLC EVM ABI")
	}

	LockEventId = htlcAbi.Events["Lock"].ID
	UnlockEventId = htlcAbi.Events["Unlock"].ID
	RefundEventId = htlcAbi.Events["Refund"].ID
}

func PackOpen(
	recipient common.Address,
	hashLock common.Hash,
	timeLock uint32,
	bchAddr common.Address,
) ([]byte, error) {
	/*
	   function lock(address payable _receiver,
	                 bytes32 _secretLock,
	                 uint256 _validPeriod,
	                 bytes20 _receiverBchPkh,
	                 uint16  _penaltyBPS,
	                 bool    _receiverIsMM) public payable
	*/
	var penaltyBPS uint16 = 0
	return htlcAbi.Pack("lock",
		recipient, hashLock, big.NewInt(int64(timeLock)), bchAddr, penaltyBPS, false)
}

func PackUnlock(hashLock, secret common.Hash) ([]byte, error) {
	// function unlock(bytes32 _secretLock, bytes32 _secretKey) public
	return htlcAbi.Pack("unlock", hashLock, secret)
}

func PackRefund(hashLock common.Hash) ([]byte, error) {
	// function refund(bytes32 _secretLock) public
	return htlcAbi.Pack("refund", hashLock)
}

func PackGetSwapState(hashLock common.Hash) ([]byte, error) {
	// function getSwapState(bytes32 secretLock) public view returns (States)
	return htlcAbi.Pack("getSwapState", hashLock)
}
func UnpackGetSwapState(data []byte) (uint8, error) {
	result, err := htlcAbi.Unpack("getSwapState", data)
	if err != nil {
		return 0, err
	}
	if len(result) != 1 {
		return 0, fmt.Errorf("no or too many results: %d", len(result))
	}
	n, ok := result[0].(uint8)
	if !ok {
		return 0, fmt.Errorf("failed to cast result to uint8")
	}
	return n, nil
}

func PackGetMarketMaker(addr common.Address) ([]byte, error) {
	// mapping (address => MarketMaker) public marketMakers;
	return htlcAbi.Pack("marketMakers", addr)
}
func UnpackGetMarketMaker(data []byte) (*MarketMakerInfo, error) {
	result, err := htlcAbi.Unpack("marketMakers", data)
	if err != nil {
		return nil, err
	}
	if len(result) != 13 {
		return nil, fmt.Errorf("expected fields: 12, got: %d", len(result))
	}

	ok := false
	mm := &MarketMakerInfo{}

	if mm.Addr, ok = result[0].(common.Address); !ok {
		return nil, fmt.Errorf("failed to cast addr")
	}
	if mm.RetiredAt, ok = result[1].(uint64); !ok {
		return nil, fmt.Errorf("failed to cast retiredAt")
	}
	if mm.Intro, ok = result[2].([32]byte); !ok {
		return nil, fmt.Errorf("failed to cast intro")
	}
	if mm.BchPkh, ok = result[3].([20]byte); !ok {
		return nil, fmt.Errorf("failed to cast bchPkh")
	}
	if mm.BchLockTime, ok = result[4].(uint16); !ok {
		return nil, fmt.Errorf("failed to cast bchLockTime")
	}
	if mm.SbchLockTime, ok = result[5].(uint32); !ok {
		return nil, fmt.Errorf("failed to cast sbchLockTime")
	}
	if mm.PenaltyBPS, ok = result[6].(uint16); !ok {
		return nil, fmt.Errorf("failed to cast penaltyBPS")
	}
	if mm.FeeBPS, ok = result[7].(uint16); !ok {
		return nil, fmt.Errorf("failed to cast feeBPS")
	}
	if mm.MinSwapAmt, ok = result[8].(*big.Int); !ok {
		return nil, fmt.Errorf("failed to cast minSwapAmt")
	}
	if mm.MaxSwapAmt, ok = result[9].(*big.Int); !ok {
		return nil, fmt.Errorf("failed to cast maxSwapAmt")
	}
	if mm.StakedValue, ok = result[10].(*big.Int); !ok {
		return nil, fmt.Errorf("failed to cast stakedValue")
	}
	if mm.Checker, ok = result[11].(common.Address); !ok {
		return nil, fmt.Errorf("failed to cast checker")
	}
	if mm.Unavailable, ok = result[12].(bool); !ok {
		return nil, fmt.Errorf("failed to cast unavailable")
	}

	return mm, nil
}
