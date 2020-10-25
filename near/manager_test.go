package near

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/blocktree/openwallet/log"
	"github.com/mr-tron/base58"
	"path/filepath"
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
	block, _ := tw.Blockscanner.GetBlockByHeight(21332802, true)
	log.Info(block)
}

func TestDecode(t *testing.T) {
	str := "c5b4f6634bf7de7366bacc2f1fc72a0dcf786e79996224c01781a67988c9dc3b"
	a := []byte(str)
	log.Info(a)
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
