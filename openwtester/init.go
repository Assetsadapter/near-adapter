package openwtester

import (
	"github.com/Assetsadapter/near-adapter/near"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openw"
)

func init() {
	//注册钱包管理工具
	log.Notice("Wallet Manager Load Successfully.")
	cache := near.NewCacheManager()

	openw.RegAssets(near.Symbol, near.NewWalletManager(&cache))
}
