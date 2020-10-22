package near

import (
	"github.com/astaxie/beego/config"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
	hClient "github.com/stellar/go/clients/horizonclient"
	oldHClient "github.com/triamnetwork/triam-horizon/clients/horizon"

	"net/http"
)

//CurveType 曲线类型
func (wm *WalletManager) CurveType() uint32 {
	return wm.Config.CurveType
}

//FullName 币种全名
func (wm *WalletManager) FullName() string {
	return "algorand"
}

//Symbol 币种标识
func (wm *WalletManager) Symbol() string {
	return wm.Config.Symbol
}

//Decimal 小数位精度
func (wm *WalletManager) Decimal() int32 {
	return wm.Config.Decimal
}

//BalanceModelType 余额模型类别
func (wm *WalletManager) BalanceModelType() openwallet.BalanceModelType {
	return openwallet.BalanceModelTypeAddress
}

//GetAddressDecode 地址解析器
func (wm *WalletManager) GetAddressDecode() openwallet.AddressDecoder {
	return wm.Decoder
}

//GetTransactionDecoder 交易单解析器
func (wm *WalletManager) GetTransactionDecoder() openwallet.TransactionDecoder {
	return wm.TxDecoder
}

//GetBlockScanner 获取区块链
func (wm *WalletManager) GetBlockScanner() openwallet.BlockScanner {
	return wm.Blockscanner
}

//LoadAssetsConfig 加载外部配置
func (wm *WalletManager) LoadAssetsConfig(c config.Configer) error {

	wm.Config.ServerAPI = c.String("ServerAPI")
	wm.Config.FixFees = c.String("FixFees")
	wm.Config.Network = c.String("Network")
	wm.Config.AddressRetainAmount = c.String("AddressRetainAmount")

	//stellar客户端
	wm.tclient = &hClient.Client{
		HorizonURL: wm.Config.ServerAPI + "/",
		HTTP:       http.DefaultClient,
	}
	//老的traim客户端
	wm.oldTclient = &oldHClient.Client{
		URL:  wm.Config.ServerAPI,
		HTTP: http.DefaultClient,
	}

	return nil
}

//InitAssetsConfig 初始化默认配置
func (wm *WalletManager) InitAssetsConfig() (config.Configer, error) {
	return config.NewConfigData("ini", []byte(wm.Config.DefaultConfig))
}

//GetAssetsLogger 获取资产账户日志工具
func (wm *WalletManager) GetAssetsLogger() *log.OWLogger {
	return wm.Log
}

//GetSmartContractDecoder 获取智能合约解析器
func (wm *WalletManager) GetSmartContractDecoder() openwallet.SmartContractDecoder {
	return wm.ContractDecoder
}
