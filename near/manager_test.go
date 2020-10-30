package near

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/blocktree/openwallet/log"
	"github.com/mr-tron/base58"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/astaxie/beego/config"

	"github.com/stellar/go/keypair"
)

var (
	tw *WalletManager
)

func testNewWalletManager() *WalletManager {
	wm := NewWalletManager()

	//读取配置
	absFile := filepath.Join("../conf", "NEAR.ini")
	//log.Debug("absFile:", absFile)
	c, err := config.NewConfig("ini", absFile)
	if err != nil {
		return nil
	}
	wm.LoadAssetsConfig(c)
	return wm
}

func init() {
	tw = testNewWalletManager()
}
func TestGetBlock(t *testing.T) {
	block, _ := tw.Blockscanner.GetBlockByHeight(20896569, true)
	log.Info(block)
}

func TestUnSerlize(t *testing.T) {
	chunkJson := `{
    "author": "cryptium.poolv1.near",
    "header": {
      "chunk_hash": "Ax9L8x6aPxSrNrWwc1CF3CZYbCVHzvadUbLCnbiay3Vj",
      "prev_block_hash": "Di2pexuWVzuXRdMt7ZuE1p9tZqzvWxNVWZvMjousdVju",
      "outcome_root": "11111111111111111111111111111111",
      "prev_state_root": "CBi6Grxyr62g8NH9i7QkHFuhC3NeiGSFAwCCw9HYUfoX",
      "encoded_merkle_root": "Gw72wvCxcnaiTNoyf5BC844SZMrYmHETYyduwW6LGSUL",
      "encoded_length": 242,
      "height_created": 20896569,
      "height_included": 20896569,
      "shard_id": 0,
      "gas_used": 0,
      "gas_limit": 1000000000000000,
      "rent_paid": "0",
      "validator_reward": "0",
      "balance_burnt": "0",
      "outgoing_receipts_root": "H4Rd6SGeEBTbxkitsCdzfu9xL9HtZ2eHoPCQXUeZ6bW4",
      "tx_root": "43qpNT5wb6KC1pDG3XgVzWwH6JCe2kXyN5dfsccXwBCL",
      "validator_proposals": [],
      "signature": "ed25519:srrKKrEfTPUGTSf8KaQKk9148oeux5nLZwNe5LKiq9WYEVGrWUd2BayhrPUbarpQMVnnNVfSGmwiS17Y3kWsnUQ"
    },
    "transactions": [
      {
        "signer_id": "01.near",
        "public_key": "ed25519:6GxYiNnRLoKkjGeKA68hrfyrJC9tYSamGND5d23aXqRx",
        "nonce": 51,
        "receiver_id": "66qozepy.01.near",
        "actions": [
          "CreateAccount",
          {
            "Transfer": {
              "deposit": "20000000000000000000000"
            }
          },
          {
            "AddKey": {
              "public_key": "ed25519:5hemHmfbbAmNnoX9hzuWgpMTUs9fsDzmRMS8kLfQPCR",
              "access_key": {
                "nonce": 0,
                "permission": "FullAccess"
              }
            }
          }
        ],
        "signature": "ed25519:5oF8kiVhucsmQgbYbRndRHfYut9VVgyNSrUXxi7xpgfH1GhLnAnEX99MNoiJ4RPDaZuAycQDvqMCc2U9CSmvo8rq",
        "hash": "42cMHzr3Cjaig9YtNXPesMYAKroSJVBf3ZuHnS5sGys6"
      }
    ],
    "receipts": []
}`
	chunkResp := ChunkResponse{}
	err := json.Unmarshal([]byte(chunkJson), &chunkResp)
	if err != nil {
		t.Error(err)
	}
}

func TestDecode(t *testing.T) {
	fmt.Println(regexp.Match("^(([a-z\\d]+[\\-_])*[a-z\\d]+\\.)*([a-z\\d]+[\\-_])*[a-z\\d]+$", []byte("234**234")))
}

func TestKeyPair(t *testing.T) {
	//pk58 := "BGCCDDHfysuuVnaNVtEhhqeT4k9Muyem3Kpgq2U1m9HX"
	privateKey := "RbwQoeUlYwnHqeUuNDVv7siB1yzVYS8cAuMriCxUiLPXZDwNqfGH1a9VuufIBnKGgErZPTxvrGvBZ/vmye9yXw=="
	bytes, err := base64.StdEncoding.WithPadding(base64.StdPadding).DecodeString(privateKey)
	if err != nil {
		t.Error(err)
		return
	}
	base58_private_key := base58.Encode(bytes)
	//pubkeys, _ := owcrypt.GenPubkey(decodeBytes, tw.CurveType())
	log.Info("private_key: ", base58_private_key)
	pubKey := "12Q8Danxh9WvVbrnyAZyhoBK2T08b6xrwWf75snvcl8="
	bytes2, err := base64.StdEncoding.WithPadding(base64.StdPadding).DecodeString(pubKey)
	base58_public_key := hex.EncodeToString(bytes2)
	base58_public_key_base58 := base58.Encode(bytes2)

	log.Info("public_key_hex: ", base58_public_key)
	log.Info("public_key_58: ", base58_public_key_base58)
}

func TestAccount(t *testing.T) {

	const (
		account1Addr   = "GAVDK2OHFZ5B257PRTCOFYNGRIWV5JRCD5SINMLQJUMSSVYV4LVHI4CN"
		account1Secret = "SDNKCPIVRCS76DATVQUFXDO73DPSXVJ22YCIS46JOBV3UR47ONWFKEUX"
		//account2Secret = "SBOEFVTSQCFFTHHFAIPLOBMDY32JC4E4KEHR4TKCSUE2O5BSBTHOAANH"
	)

	sender, err := keypair.Parse(account1Secret)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	txid := "5d9d4712a05361619a4608a4e2560bbb6f941a8244364bd61c875bdb3945944a"
	txid = strings.Trim(txid, "\"")
	fmt.Printf("txid: %s\n", txid)
	fmt.Printf("pub: %s\n", sender.Address())

}
