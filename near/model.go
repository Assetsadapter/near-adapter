/*
 * Copyright 2018 The OpenWallet Authors
 * This file is part of the OpenWallet library.
 *
 * The OpenWallet library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The OpenWallet library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 */

package near

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/Assetsadapter/near-adapter/encoding"
	"github.com/Assetsadapter/near-adapter/types"
	"github.com/blocktree/openwallet/common"
	"github.com/blocktree/openwallet/crypto"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/tidwall/gjson"
)

type Asset struct {
	ID                 types.ObjectID `json:"id"`
	Symbol             string         `json:"symbol"`
	Precision          uint8          `json:"precision"`
	Issuer             string         `json:"issuer"`
	DynamicAssetDataID string         `json:"dynamic_asset_data_id"`
}

type BlockHeader struct {
	TransactionMerkleRoot string            `json:"transaction_merkle_root"`
	Previous              string            `json:"previous"`
	Timestamp             types.Time        `json:"timestamp"`
	Witness               string            `json:"witness"`
	Extensions            []json.RawMessage `json:"extensions"`
	WitnessSignature      string            `json:"witness_signature"`
}

func NewBlockHeader(result *gjson.Result) *BlockHeader {
	obj := BlockHeader{}
	json.Unmarshal([]byte(result.Raw), &obj)
	return &obj
}

func (block *BlockHeader) Serialize() ([]byte, error) {
	var b bytes.Buffer
	encoder := encoding.NewEncoder(&b)

	if err := encoder.Encode(block); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (block *BlockHeader) CalculateID() (string, error) {
	var msgBuffer bytes.Buffer

	// Write the serialized transaction.
	rawTx, err := block.Serialize()
	if err != nil {
		return "", err
	}

	if _, err := msgBuffer.Write(rawTx); err != nil {
		return "", errors.Wrap(err, "failed to write serialized block header")
	}

	msgBytes := msgBuffer.Bytes()

	// Compute the digest.
	digest := sha256.Sum224(msgBytes)

	id := hex.EncodeToString(digest[:])
	length := 40
	if len(id) < 40 {
		length = len(id)
	}
	return id[:length], nil
}

// MarshalBlockHeader implements encoding.Marshaller interface.
func (block *BlockHeader) Marshal(encoder *encoding.Encoder) error {

	enc := encoding.NewRollingEncoder(encoder)

	enc.Encode(block.TransactionMerkleRoot)
	enc.Encode(block.Previous)
	enc.Encode(block.Timestamp)
	enc.Encode(block.Witness)
	enc.Encode(block.WitnessSignature)

	// Extensions are not supported yet.
	enc.EncodeUVarint(0)
	return enc.Err()
}

/*
{
	"author": "node0",
	"header": {
		"height": 12513827,
		"epoch_id": "EtT9rBY8Wq536nbsr4W5xK5H7TT8bdBsTkN7HK4izpX5",
		"next_epoch_id": "8fNAmRULjDVGu7X8jdYKfxJCZkBjqaCZbYpMtXAWoPmX",
		"hash": "AhT2VvFjYLjLNPhSJRpCrfCZqa8aZdLCg1wVKCYmPxND",
		"prev_hash": "G1jPWGbJtFLkzpZRsY6nQ8ogAp3t5VLPdFKWxFxNsXNb",
		"prev_state_root": "J1HYYsYbPxRg1EfsxwkwdQashZaR2JZii7RhN7b7UZ1N",
		"chunk_receipts_root": "9ETNjrt6MkwTgSVMMbpukfxRshSD1avBUUa4R4NuqwHv",
		"chunk_headers_root": "HgAs6t6TV5d7NJNBpWwBEdtRcLayrwk8UjnDxSvjBWX9",
		"chunk_tx_root": "7tkzFg8RHBmMw1ncRJZCCZAizgq4rwCftTKYLce8RU8t",
		"outcome_root": "7tkzFg8RHBmMw1ncRJZCCZAizgq4rwCftTKYLce8RU8t",
		"chunks_included": 1,
		"challenges_root": "11111111111111111111111111111111",
		"timestamp": 1597631094062564970,
		"timestamp_nanosec": "1597631094062564970",
		"random_value": "GqQRYTZ5qR9qBkAXHJhiE25kGNbGks8MDGVPgSWPYcu3",
		"validator_proposals": [],
		"chunk_mask": [
			true
		],
		"gas_price": "5000",
		"rent_paid": "0",
		"validator_reward": "0",
		"total_supply": "1033983738573655930552610242529229",
		"challenges_result": [],
		"last_final_block": "CjXbGkEzP85g4pytn9EDM5EpSzJ6AcGHrZ1XNQARsreh",
		"last_ds_final_block": "G1jPWGbJtFLkzpZRsY6nQ8ogAp3t5VLPdFKWxFxNsXNb",
		"next_bp_hash": "7icBuAQWji9eHVug4JcjqcF6Fr2M83nG5L9WaamVF8Hv",
		"block_merkle_root": "DcRQLsXJMEG33gR6WhYtKZf6ZkeuWSvpX6bj7NwaJEcS",
		"approvals": [
			"ed25519:7iVtyFAM658upGBnTVYM4MQhQv6fVetacgWGMsDfbv126kH4ZaDe6HfTnSSRG17e9d4W3fCh6fxn3iwEYcHXC8J",
			"ed25519:66SQFLeESmaSp88BhXmqPCxoMy26LGoxPJDtkYTe2xF8iYHshunoGPgg3wKe6gHfFNkNHp5wzXte9xkQg4ZQE6W9",
			null,
			"ed25519:5ahtLmYJcKZZEyELY2i9yrsD4vHMFkG52ygpJ1KNYLys5axdnt4hogF7tT722ebjCC1ytKRDQSqvkFpum6w2ea7n",
			null,
			null,
			"ed25519:2RWTn7dr2qyVm1inGBcZy9HkcPP4x7K6PYnAvnUvxo6rSGTirMKDJWJWoTwSkBWLt9fYssfpKQeHbpGJRn5tCUoe",
			"ed25519:286JQtkJEWzQQJmed7kt7XmtG4utU2DK6Ao9zhx6cA9iiVvZEEGwnsSsQTGAbuoqVNLKxFCvEsVig6efnb27Vbyh",
			null,
			null,
			"ed25519:41yJrosrc1fS1vFYb4z8di2428qBXC1nse984EJX8FZQvpjrherzpKAYQ7oqhT25X9eGJsUACjwD7xz1saTeNTru",
			"ed25519:2YwGwDfWUPy5JS3MkbzytodrVjZPPQAV9FfPQVpYFWNmaAifydof2hmdLTBqRpaQyWkmqxJzJoJhqCAVBE7zMzwS",
			"ed25519:5eqrGHqEwJkhWucWF8SNkSRpxxDkbhGNhDZTd95sPQ6o5cGuW1NzTD2qRv13r3d6k5vmy5TExwmHKaGvQhkKB3tg",
			null,
			"ed25519:t2Wg6RZxCiVqGG5NZVXpkyT3AswFNuduFoHMTpMgZd3jRPKgZRpFzDJaA2zDdjkzhUr6mvGCrSrVEXGF3ZrMhWX",
			null,
			"ed25519:2KquUy2GiuVf6R36MXgw8D4TsLqYLRdvzaBqS6Gw4C2VeKuZagw32ThYRTEnMRXE8CNqTBeB3QcsXxAB4ErYnKEz",
			"ed25519:5QbYH3DbpSk9NmjizwrmsAY4YL7KQqnpvpJe1Z9KEtkRVznD1DfbQNi8kcpCJcsdd62iMsgxyujafWBWUZo9f4TG",
			"ed25519:o56TrgsuqmiyX9SFv374vUE7td6ocSm7CUCvZ5YNvg4wbSr8hhzTk5cST6sAS9CRUsrWdKFQsHh9djQUdiSsJhM",
			null,
			"ed25519:5ye4YiYA42eWnLmBugxfwMqfbg1VmpeywQ1Rbd3fy95KSz3dsXrTwSozx9V4Jkt4stxWRRJUdCw2aQoJ15JtstKZ",
			"ed25519:cacHjRR8PQWtkrRiPexc4gm9ZPAqYX2MrhSBVRQc23S6X2TqmPsUkxmTAVWqWb4ew8ybLPMhTyVEzVL5ez6hdoY",
			"ed25519:5keJpe6mpCNjwBhKBttTnGjEHZrwxfsi3chxWDQfBc42aRhniVmVP77kkyA9zM1fG75moFmRJGg4JTKe1FJdps6e",
			null,
			null
		],
		"signature": "ed25519:PUhnAf1TWQFrLobxsi7RTbWXZofM4MjjVCmqNGRevUB63cgL2pLLHvZxFBs1E7b4nDGdX2foKcbVfCyJcCtKfj9",
		"latest_protocol_version": 30
	},
	"chunks": [
		{
			"chunk_hash": "9pFX6hSZAtXQ5YjtmWkHVdNTZ58dBFXjeVoMEXrvHR4f",
			"prev_block_hash": "G1jPWGbJtFLkzpZRsY6nQ8ogAp3t5VLPdFKWxFxNsXNb",
			"outcome_root": "11111111111111111111111111111111",
			"prev_state_root": "BhkubqyZpAb6xr3iAzUqrcE8iQ3KGFAWRxZZcf7Bs2ae",
			"encoded_merkle_root": "9zYue7drR1rhfzEEoc4WUXzaYRnRNihvRoGt1BgK7Lkk",
			"encoded_length": 8,
			"height_created": 12513827,
			"height_included": 12513827,
			"shard_id": 0,
			"gas_used": 0,
			"gas_limit": 1000000000000000,
			"rent_paid": "0",
			"validator_reward": "0",
			"balance_burnt": "0",
			"outgoing_receipts_root": "H4Rd6SGeEBTbxkitsCdzfu9xL9HtZ2eHoPCQXUeZ6bW4",
			"tx_root": "11111111111111111111111111111111",
			"validator_proposals": [],
			"signature": "ed25519:5RTZBS94mWMb8C6Mrtc3fgnyfpaeJk7EMP7q2w4x3yQ4qbC6kjWjeioDvd3DEt8mCxowEbVJ76EVao9fwJUKUPeT"
		}
	]
}
*/
type Block struct {
	Height       uint64
	Author       string             `json:"author"`
	Header       Header             `json:"header"`
	Chunks       []BlockChunk       `json:"chunks"`
	Transactions []ChunkTransaction `json:"transactions"`
	Receipts     []Receipt          `json:"receipts"`
}

/*
"header": {
	"height": 12513827,
	"epoch_id": "EtT9rBY8Wq536nbsr4W5xK5H7TT8bdBsTkN7HK4izpX5",
	"next_epoch_id": "8fNAmRULjDVGu7X8jdYKfxJCZkBjqaCZbYpMtXAWoPmX",
	"hash": "AhT2VvFjYLjLNPhSJRpCrfCZqa8aZdLCg1wVKCYmPxND",
	"prev_hash": "G1jPWGbJtFLkzpZRsY6nQ8ogAp3t5VLPdFKWxFxNsXNb",
	"prev_state_root": "J1HYYsYbPxRg1EfsxwkwdQashZaR2JZii7RhN7b7UZ1N",
	"chunk_receipts_root": "9ETNjrt6MkwTgSVMMbpukfxRshSD1avBUUa4R4NuqwHv",
	"chunk_headers_root": "HgAs6t6TV5d7NJNBpWwBEdtRcLayrwk8UjnDxSvjBWX9",
	"chunk_tx_root": "7tkzFg8RHBmMw1ncRJZCCZAizgq4rwCftTKYLce8RU8t",
	"outcome_root": "7tkzFg8RHBmMw1ncRJZCCZAizgq4rwCftTKYLce8RU8t",
	"chunks_included": 1,
	"challenges_root": "11111111111111111111111111111111",
	"timestamp": 1597631094062564970,
	"timestamp_nanosec": "1597631094062564970",
	"random_value": "GqQRYTZ5qR9qBkAXHJhiE25kGNbGks8MDGVPgSWPYcu3",
	"validator_proposals": [],
	"chunk_mask": [
		true
	],
	"gas_price": "5000",
	"rent_paid": "0",
	"validator_reward": "0",
	"total_supply": "1033983738573655930552610242529229",
	"challenges_result": [],
	"last_final_block": "CjXbGkEzP85g4pytn9EDM5EpSzJ6AcGHrZ1XNQARsreh",
	"last_ds_final_block": "G1jPWGbJtFLkzpZRsY6nQ8ogAp3t5VLPdFKWxFxNsXNb",
	"next_bp_hash": "7icBuAQWji9eHVug4JcjqcF6Fr2M83nG5L9WaamVF8Hv",
	"block_merkle_root": "DcRQLsXJMEG33gR6WhYtKZf6ZkeuWSvpX6bj7NwaJEcS",
	"approvals": [
		... string
	],
	"signature": "ed25519:PUhnAf1TWQFrLobxsi7RTbWXZofM4MjjVCmqNGRevUB63cgL2pLLHvZxFBs1E7b4nDGdX2foKcbVfCyJcCtKfj9",
	"latest_protocol_version": 30
}
*/
type Header struct {
	Height            uint64 `json:"height"`
	Hash              string `json:"hash"`
	PrevHash          string `json:"prev_hash"`
	PrevStateRoot     string `json:"prev_state_root"`
	ChunkReceiptsRoot string `json:"chunk_receipts_root"`
	ChunkHeadersRoot  string `json:"chunk_headers_root"`
	ChunkTxRoot       string `json:"chunk_tx_root"`
	ChunksIncluded    uint32 `json:"chunks_included"`
	Timestamp         int64  `json:"timestamp"`
	GasPrice          string `json:"gas_price"`
	LastFinalBlock    string `json:"last_final_block"`
	BlockMerkleRoot   string `json:"block_merkle_root"`
	Signature         string `json:"signature"`
}

/*
{
	"chunk_hash": "9pFX6hSZAtXQ5YjtmWkHVdNTZ58dBFXjeVoMEXrvHR4f",
	"prev_block_hash": "G1jPWGbJtFLkzpZRsY6nQ8ogAp3t5VLPdFKWxFxNsXNb",
	"outcome_root": "11111111111111111111111111111111",
	"prev_state_root": "BhkubqyZpAb6xr3iAzUqrcE8iQ3KGFAWRxZZcf7Bs2ae",
	"encoded_merkle_root": "9zYue7drR1rhfzEEoc4WUXzaYRnRNihvRoGt1BgK7Lkk",
	"encoded_length": 8,
	"height_created": 12513827,
	"height_included": 12513827,
	"shard_id": 0,
	"gas_used": 0,
	"gas_limit": 1000000000000000,
	"rent_paid": "0",
	"validator_reward": "0",
	"balance_burnt": "0",
	"outgoing_receipts_root": "H4Rd6SGeEBTbxkitsCdzfu9xL9HtZ2eHoPCQXUeZ6bW4",
	"tx_root": "11111111111111111111111111111111",
	"validator_proposals": [],
	"signature": "ed25519:5RTZBS94mWMb8C6Mrtc3fgnyfpaeJk7EMP7q2w4x3yQ4qbC6kjWjeioDvd3DEt8mCxowEbVJ76EVao9fwJUKUPeT"
}
*/
type BlockChunk struct {
	ChunkHash         string `json:"chunk_hash"`
	PrevBlockHash     string `json:"prev_block_hash"`
	PrevStateRoot     string `json:"prev_state_root"`
	EncodedMerkleRoot string `json:"encoded_merkle_root"`
	HeightCreated     uint64 `json:"height_created"`
	HeightIncluded    uint64 `json:"height_included"`
	GasUsed           uint64 `json:"gas_used"`
	GasLimit          uint64 `json:"gas_limit"`
	TxRoot            string `json:"tx_root"`
	Signature         string `json:"signature"`
}

/*
{
  "author": "node0",
  "header": {
	  "chunk_hash": "9pFX6hSZAtXQ5YjtmWkHVdNTZ58dBFXjeVoMEXrvHR4f",
	  "prev_block_hash": "G1jPWGbJtFLkzpZRsY6nQ8ogAp3t5VLPdFKWxFxNsXNb",
	  "outcome_root": "11111111111111111111111111111111",
	  "prev_state_root": "BhkubqyZpAb6xr3iAzUqrcE8iQ3KGFAWRxZZcf7Bs2ae",
	  "encoded_merkle_root": "9zYue7drR1rhfzEEoc4WUXzaYRnRNihvRoGt1BgK7Lkk",
	  "encoded_length": 8,
	  "height_created": 12513827,
	  "height_included": 12513827,
	  "shard_id": 0,
	  "gas_used": 0,
	  "gas_limit": 1000000000000000,
	  "rent_paid": "0",
	  "validator_reward": "0",
	  "balance_burnt": "0",
	  "outgoing_receipts_root": "H4Rd6SGeEBTbxkitsCdzfu9xL9HtZ2eHoPCQXUeZ6bW4",
	  "tx_root": "11111111111111111111111111111111",
	  "validator_proposals": [],
	  "signature": "ed25519:5RTZBS94mWMb8C6Mrtc3fgnyfpaeJk7EMP7q2w4x3yQ4qbC6kjWjeioDvd3DEt8mCxowEbVJ76EVao9fwJUKUPeT"
  },
  "transactions": [],
  "receipts": []
}
*/
type Chunk struct {
	Author       string             `json:"author"`
	Header       ChunkHeader        `json:"header"`
	Transactions []ChunkTransaction `json:"transactions"`
	Receipts     []Receipt          `json:"receipts"`
}

func NewChunk(result *gjson.Result) (*Chunk, error) {
	obj := Chunk{}
	err := json.Unmarshal([]byte(result.Raw), &obj)
	if err != nil {
		return nil, err
	}
	return &obj, nil
}

/*
"header": {
  "chunk_hash": "9pFX6hSZAtXQ5YjtmWkHVdNTZ58dBFXjeVoMEXrvHR4f",
  "prev_block_hash": "G1jPWGbJtFLkzpZRsY6nQ8ogAp3t5VLPdFKWxFxNsXNb",
  "outcome_root": "11111111111111111111111111111111",
  "prev_state_root": "BhkubqyZpAb6xr3iAzUqrcE8iQ3KGFAWRxZZcf7Bs2ae",
  "encoded_merkle_root": "9zYue7drR1rhfzEEoc4WUXzaYRnRNihvRoGt1BgK7Lkk",
  "encoded_length": 8,
  "height_created": 12513827,
  "height_included": 12513827,
  "shard_id": 0,
  "gas_used": 0,
  "gas_limit": 1000000000000000,
  "rent_paid": "0",
  "validator_reward": "0",
  "balance_burnt": "0",
  "outgoing_receipts_root": "H4Rd6SGeEBTbxkitsCdzfu9xL9HtZ2eHoPCQXUeZ6bW4",
  "tx_root": "11111111111111111111111111111111",
  "validator_proposals": [],
  "signature": "ed25519:5RTZBS94mWMb8C6Mrtc3fgnyfpaeJk7EMP7q2w4x3yQ4qbC6kjWjeioDvd3DEt8mCxowEbVJ76EVao9fwJUKUPeT"
}
*/
type ChunkHeader struct {
	ChunkHash         string `json:"chunk_hash"`
	PrevBlockHash     string `json:"prev_block_hash"`
	PrevStateRoot     string `json:"prev_state_root"`
	EncodedMerkleRoot string `json:"encoded_merkle_root"`
	HeightCreated     uint64 `json:"height_created"`
	HeightIncluded    uint64 `json:"height_included"`
	GasUsed           uint64 `json:"gas_used"`
	GasLimit          uint64 `json:"gas_limit"`
	TxRoot            string `json:"tx_root"`
	Signature         string `json:"signature"`
}

/*
{
	"signer_id": "zest.near",
	"public_key": "ed25519:C1BYE6i7bDoqccHuXfGTHXQzJsdhjN7MQ6h3iCRxycCY",
	"nonce": 11,
	"receiver_id": "cheese.zest.near",
	"actions": [
		{
			"Transfer": {
				"deposit": "11000000000000000000000000"
			}
		}
	],
	"signature": "ed25519:2vTEREHJRqDREyPaHAu7EgpyWtLqmDeDZ3dT7xPTNaKzs1TJnycVfi2Tqad3pz4xLcdNiMR2m4iN6tYeVa2Evnfc",
	"hash": "GGiCafAAWEnu4cXhEXhyQi7w3PYPBZL7YKRLuzV9ySHm"
}
*/
type ChunkTransaction struct {
	SignerId   string `json:"signer_id"`
	PublicKey  string `json:"public_key"`
	Nonce      uint32 `json:"nonce"`
	ReceiverId string `json:"receiver_id"`
	Actions    []struct {
		Transfer struct {
			Deposit string `json:"deposit"`
		} `json:"transfer"`
	} `json:"actions"`
	Signature string `json:"signature"`
	Hash      string `json:"hash"`
}

/*
"result": {
	"status": {
		"SuccessValue": ""
	},
	"transaction": {
		"signer_id": "hileor-2.testnet",
		"public_key": "ed25519:7EaSeZrgrfsGjg18LxoxvtvPQFacVMJ4LvvMkvTtvBg1",
		"nonce": 2,
		"receiver_id": "hileor.testnet",
		"actions": [
			{
				"Transfer": {
					"deposit": "1000000000000000000000000"
				}
			}
		],
		"signature": "ed25519:4ojsm6FgzWHFhJUn6EaVBY6vYaBqE2eMdFPiLXwTUe4s412TgtjJ4dqYGukq9KH4Jh9iAAVJx1rgY92UWHDjtzeB",
		"hash": "HZFCBNuUDZFd4MirpqeDzbeK4r7NsmMLKE2knMVjm8PF"
	},
	"transaction_outcome": {
		"proof": [],
		"block_hash": "6VFozrYuzsA2mMXZiMtsaNuqrFWSbtse5K7JzGa8446c",
		"id": "HZFCBNuUDZFd4MirpqeDzbeK4r7NsmMLKE2knMVjm8PF",
		"outcome": {
			"logs": [],
			"receipt_ids": [
				"5raVXFwSdwiiQTiFTqm1G2fWindwthw1vq2ZXEVSDefX"
			],
			"gas_burnt": 223182562500,
			"tokens_burnt": "1115912812500000",
			"executor_id": "hileor-2.testnet",
			"status": {
				"SuccessReceiptId": "5raVXFwSdwiiQTiFTqm1G2fWindwthw1vq2ZXEVSDefX"
			}
		}
	},
	"receipts_outcome": [
		{
			"proof": [],
			"block_hash": "98VwnpJmJVfnEaYdE5otRcf6nuC11DD7HoJwpkKUMgW4",
			"id": "5raVXFwSdwiiQTiFTqm1G2fWindwthw1vq2ZXEVSDefX",
			"outcome": {
				"logs": [],
				"receipt_ids": [
					"6ZYW4hAHkudwRXC59bKrxkNQuEjo3Emmic2VyCcbnCsJ"
				],
				"gas_burnt": 223182562500,
				"tokens_burnt": "1115912812500000",
				"executor_id": "hileor.testnet",
				"status": {
					"SuccessValue": ""
				}
			}
		},
		{
			"proof": [],
			"block_hash": "9nJjWeWDc9bQuY22cH5zGuwvoxMpAz7Eir2N7Fs5eyAN",
			"id": "6ZYW4hAHkudwRXC59bKrxkNQuEjo3Emmic2VyCcbnCsJ",
			"outcome": {
				"logs": [],
				"receipt_ids": [],
				"gas_burnt": 0,
				"tokens_burnt": "0",
				"executor_id": "hileor-2.testnet",
				"status": {
					"SuccessValue": ""
				}
			}
		}
	]
}
*/
type Transaction struct {
	Transaction        ChunkTransaction   `json:"transaction"`
	TransactionOutcome TransactionOutcome `json:"transaction_outcome"`
}

func NewTransaction(result *gjson.Result) (*Transaction, error) {
	obj := Transaction{}
	err := json.Unmarshal([]byte(result.Raw), &obj)
	return &obj, err
}

/*
"transaction_outcome": {
	"proof": [],
	"block_hash": "6VFozrYuzsA2mMXZiMtsaNuqrFWSbtse5K7JzGa8446c",
	"id": "HZFCBNuUDZFd4MirpqeDzbeK4r7NsmMLKE2knMVjm8PF",
	"outcome": {
		"logs": [],
		"receipt_ids": [
			"5raVXFwSdwiiQTiFTqm1G2fWindwthw1vq2ZXEVSDefX"
		],
		"gas_burnt": 223182562500,
		"tokens_burnt": "1115912812500000",
		"executor_id": "hileor-2.testnet",
		"status": {
			"SuccessReceiptId": "5raVXFwSdwiiQTiFTqm1G2fWindwthw1vq2ZXEVSDefX"
		}
	}
}
*/
type TransactionOutcome struct {
	BlockHash string  `json:"block_hash"`
	Id        string  `json:"id"`
	Outcome   Outcome `json:"outcome"`
}

type Outcome struct {
	Logs        []string `json:"logs"`
	ReceiptIdes []string `json:"receipt_ides"`
	GasBurnt    string   `json:"gas_burnt"`
	TokensBurnt string   `json:"tokens_burnt"`
	ExecutorId  string   `json:"executor_id"`
	Status      struct {
		SuccessReceiptId string `json:"success_receipt_id"`
	}
}

type Receipt struct {
}

func NewBlock(result *gjson.Result) *Block {
	obj := Block{}
	json.Unmarshal([]byte(result.Raw), &obj)
	obj.Height = uint64(obj.Header.Height)
	return &obj
}

func (block *Block) CalculateID() error {
	header := BlockHeader{}
	header.TransactionMerkleRoot = block.TransactionMerkleRoot
	header.Previous = block.Previous
	header.Timestamp = block.Timestamp
	header.Witness = block.Witness
	header.Extensions = block.Extensions
	header.WitnessSignature = block.WitnessSignature

	id, err := header.CalculateID()
	if err != nil {
		return err
	}
	block.BlockID = id
	return nil
}

//UnscanRecord 扫描失败的区块及交易
type UnscanRecord struct {
	ID          string `storm:"id"` // primary key
	BlockHeight uint64
	TxID        string
	Reason      string
}

//NewUnscanRecord new UnscanRecord
func NewUnscanRecord(height uint64, txID, reason string) *UnscanRecord {
	obj := UnscanRecord{}
	obj.BlockHeight = height
	obj.TxID = txID
	obj.Reason = reason
	obj.ID = common.Bytes2Hex(crypto.SHA256([]byte(fmt.Sprintf("%d_%s", height, txID))))
	return &obj
}

// ParseHeader 区块链头
func ParseHeader(b *Block) *openwallet.BlockHeader {
	obj := openwallet.BlockHeader{}

	//解析josn
	obj.Merkleroot = b.TransactionMerkleRoot
	obj.Hash = b.BlockID
	obj.Previousblockhash = b.Previous
	obj.Height = b.Height
	obj.Time = uint64(b.Timestamp.Unix())
	obj.Symbol = Symbol
	return &obj
}

type BlockStatus struct {
	/*
		{
			"version": {
				"version": "1.8.0-rc.4",
				"build": "fc666218"
			},
			"chain_id": "testnet",
			"protocol_version": 30,
			"latest_protocol_version": 30,
			"rpc_addr": "0.0.0.0:3030",
			"validators": [
			],
			"sync_info": {
				"latest_block_hash": "EygmMywAFMrksCj5jzRtqdR1AvzDkbYjNkkztNdYByUQ",
				"latest_block_height": 12139729,
				"latest_state_root": "AUvR8dLQf9YmZZEu7aSMXBoEvBtf2SzbxLswKWhMYH1P",
				"latest_block_time": "2020-08-13T06:47:32.495371571Z",
				"syncing": false
			},
			"validator_account_id": "nearup-node8"
		}
	*/
	ChainId           string    `json:"chain_id"`
	LatestBlockHash   string    `json:"latest_block_hash"`
	LatestBlockHeight uint64    `json:"latest_block_height"`
	LatestBlockRoot   string    `json:"latest_block_root"`
	LatestBlockTime   time.Time `json:"latest_block_time"`
}

const TimeLayout = `2006-01-02T15:04:05`

func NewBlockChainStatus(result *gjson.Result) *BlockStatus {
	obj := BlockStatus{}
	obj.ChainId = result.Get("chain_id").String()
	obj.LatestBlockHash = result.Get("sync_info.latest_block_hash").String()
	obj.LatestBlockHeight = result.Get("sync_info.latest_block_height").Uint()
	obj.LatestBlockRoot = result.Get("sync_info.latest_state_root").String()
	obj.LatestBlockTime, _ = time.ParseInLocation(TimeLayout, result.Get("sync_info.latest_block_time").String(), time.UTC)
	return &obj
}

type Balance struct {
	AssetID types.ObjectID `json:"asset_id"`
	Amount  string         `json:"amount"`
}

func NewBalance(result *gjson.Result) *Balance {
	arr := result.Array()
	for _, item := range arr {
		obj := Balance{}
		obj.Amount = item.Get("amount").String()
		obj.AssetID = types.MustParseObjectID(item.Get("asset_id").String())
		return &obj
	}
	return nil
}

type BroadcastResponse struct {
	ID string `json:"id"`
}

/*
"result": {
	"amount": "501000000997901753750000000",
	"locked": "0",
	"code_hash": "11111111111111111111111111111111",
	"storage_usage": 264,
	"storage_paid_at": 0,
	"block_height": 12537869,
	"block_hash": "73Syy1Uc4tGzyRhT2uRSUeD2LXgVvWti3wwBNWB8a1yi"
}
*/
type Account struct {
	Amount        string `json:"amount"`
	Locked        string `json:"locked"`
	CodeHash      string `json:"code_hash"`
	StorageUsage  uint32 `json:"storage_usage"`
	StoragePaidAt uint32 `json:"storage_paid_at"`
	BlockHeight   uint64 `json:"block_height"`
	BlockHash     string `json:"block_hash"`
}
