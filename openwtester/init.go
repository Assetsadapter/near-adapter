package openwtester

import (
	"github.com/Assetsadapter/bitshares-adapter/bitshares"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openw"
)

func init() {
	//注册钱包管理工具
	log.Notice("Wallet Manager Load Successfully.")
	cache := bitshares.NewCacheManager()

	openw.RegAssets(bitshares.Symbol, bitshares.NewWalletManager(&cache))
}
