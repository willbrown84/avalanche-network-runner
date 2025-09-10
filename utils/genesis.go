// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package utils

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ava-labs/avalanchego/upgrade"
	avago_constants "github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/crypto/bls/signer/localsigner"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/vms/platformvm/signer"
	"github.com/ava-labs/subnet-evm/params"
	"github.com/ava-labs/subnet-evm/params/extras"
	subnetevmutils "github.com/ava-labs/subnet-evm/utils"
)

const (
	// difference between unlock schedule locktime and startime in original genesis
	genesisLocktimeStartimeDelta    = 2836800
	hexa0Str                        = "0x0"
	defaultLocalCChainFundedAddress = "8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
	defaultLocalCChainFundedBalance = "0x295BE96E64066972000000"
	allocationCommonEthAddress      = "0xb3d82b1367d362de99ab59a658165aff520cbd4d"
	stakingAddr                     = "X-custom1g65uqn6t77p656w64023nh8nd9updzmxwd59gh"
	walletAddr                      = "X-custom18jma8ppw3nhx5r4ap8clazz0dps7rv5u9xde7p"
)

func generateCchainGenesis() ([]byte, error) {
	chainConfig := &params.ChainConfig{
		ChainID:             big.NewInt(43112),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
	}
	agoUpgrade := upgrade.GetConfig(avago_constants.LocalID)
	params.GetExtra(chainConfig).NetworkUpgrades = extras.NetworkUpgrades{
		SubnetEVMTimestamp: subnetevmutils.NewUint64(0),
		DurangoTimestamp:   subnetevmutils.TimeToNewUint64(agoUpgrade.DurangoTime),
		EtnaTimestamp:      subnetevmutils.TimeToNewUint64(agoUpgrade.EtnaTime),
		FortunaTimestamp:   nil, // Fortuna is optional and has no effect on Subnet-EVM
		GraniteTimestamp:   nil, // Granite is optional and has no effect on Subnet-EVM
	}
	cChainGenesisMap := map[string]interface{}{}
	cChainGenesisMap["config"] = chainConfig
	cChainGenesisMap["timestamp"] = upgrade.InitiallyActiveTime.Unix()
	cChainGenesisMap["nonce"] = hexa0Str
	cChainGenesisMap["extraData"] = "0x00"
	cChainGenesisMap["gasLimit"] = "0x5f5e100"
	cChainGenesisMap["difficulty"] = hexa0Str
	cChainGenesisMap["mixHash"] = "0x0000000000000000000000000000000000000000000000000000000000000000"
	cChainGenesisMap["coinbase"] = "0x0000000000000000000000000000000000000000"
	cChainGenesisMap["alloc"] = map[string]interface{}{
		defaultLocalCChainFundedAddress: map[string]interface{}{
			"balance": defaultLocalCChainFundedBalance,
		},
	}
	cChainGenesisMap["number"] = hexa0Str
	cChainGenesisMap["gasUsed"] = hexa0Str
	cChainGenesisMap["parentHash"] = "0x0000000000000000000000000000000000000000000000000000000000000000"
	return json.Marshal(cChainGenesisMap)
}

func GenerateGenesis(
	networkID uint32,
	nodeKeys []*NodeKeys,
) ([]byte, error) {
	genesisMap := map[string]interface{}{}

	// cchain
	cChainGenesisBytes, err := generateCchainGenesis()
	if err != nil {
		return nil, err
	}
	genesisMap["cChainGenesis"] = string(cChainGenesisBytes)

	// pchain genesis
	genesisMap["networkID"] = networkID
	startTime := time.Now().Unix()
	genesisMap["startTime"] = startTime
	initialStakers := []map[string]interface{}{}

	for _, keys := range nodeKeys {
		nodeID, err := ToNodeID(keys.StakingKey, keys.StakingCert)
		if err != nil {
			return nil, fmt.Errorf("couldn't get node ID: %w", err)
		}
		blsSk, err := localsigner.FromBytes(keys.BlsKey)
		if err != nil {
			return nil, err
		}
		p, err := signer.NewProofOfPossession(blsSk)
		if err != nil {
			return nil, err
		}
		pk, err := formatting.Encode(formatting.HexNC, p.PublicKey[:])
		if err != nil {
			return nil, err
		}
		pop, err := formatting.Encode(formatting.HexNC, p.ProofOfPossession[:])
		if err != nil {
			return nil, err
		}
		initialStaker := map[string]interface{}{
			"delegationFee": 1000000,
			"nodeID":        nodeID,
			"rewardAddress": walletAddr,
			"signer": map[string]interface{}{
				"proofOfPossession": pop,
				"publicKey":         pk,
			},
		}
		initialStakers = append(initialStakers, initialStaker)
	}

	genesisMap["initialStakeDuration"] = 31536000
	genesisMap["initialStakeDurationOffset"] = 5400
	genesisMap["initialStakers"] = initialStakers
	lockTime := startTime + genesisLocktimeStartimeDelta
	allocations := []interface{}{}
	alloc := map[string]interface{}{
		"avaxAddr":      walletAddr,
		"ethAddr":       allocationCommonEthAddress,
		"initialAmount": 300000000000000000,
		"unlockSchedule": []interface{}{
			map[string]interface{}{"amount": 20000000000000000},
			map[string]interface{}{"amount": 10000000000000000, "locktime": lockTime},
		},
	}
	allocations = append(allocations, alloc)
	alloc = map[string]interface{}{
		"avaxAddr":      stakingAddr,
		"ethAddr":       allocationCommonEthAddress,
		"initialAmount": 0,
		"unlockSchedule": []interface{}{
			map[string]interface{}{"amount": 10000000000000000, "locktime": lockTime},
		},
	}
	allocations = append(allocations, alloc)
	genesisMap["allocations"] = allocations
	genesisMap["initialStakedFunds"] = []interface{}{
		stakingAddr,
	}
	genesisMap["message"] = "{{ fun_quote }}"

	return json.MarshalIndent(genesisMap, "", " ")
}
