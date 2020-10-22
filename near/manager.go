package near

import (
	"github.com/algorand/go-algorand-sdk/client/algod"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
	hClient "github.com/stellar/go/clients/horizonclient"
	oldhClient "github.com/triamnetwork/triam-horizon/clients/horizon"
)

type WalletManager struct {
	openwallet.AssetsAdapterBase

	Config          *WalletConfig                   // 节点配置
	Decoder         openwallet.AddressDecoder       //地址编码器
	TxDecoder       openwallet.TransactionDecoder   //交易单编码器
	Log             *log.OWLogger                   //日志工具
	ContractDecoder openwallet.SmartContractDecoder //智能合约解析器
	Blockscanner    *TriamBlockScanner              //区块扫描器
	client          *algod.Client                   //algod client

	tclient    *hClient.Client    //stellar horizonclient client
	oldTclient *oldhClient.Client //triam horizonclient client
}

func NewWalletManager() *WalletManager {
	wm := WalletManager{}
	wm.Config = NewConfig(Symbol)
	wm.Blockscanner = NewTriamBlockScanner(&wm)
	wm.Decoder = NewAddressDecoder(&wm)
	wm.TxDecoder = NewTransactionDecoder(&wm)
	wm.ContractDecoder = &toeknDecoder{wm: &wm}
	wm.Log = log.NewOWLogger(wm.Symbol())
	return &wm
}
