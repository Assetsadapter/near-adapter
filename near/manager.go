package near

import (
	"github.com/Assetsadapter/near-adapter/address_decode"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
)

type WalletManager struct {
	openwallet.AssetsAdapterBase

	Config          *WalletConfig                   // 节点配置
	Decoder         openwallet.AddressDecoder       //地址编码器
	DecoderV2       openwallet.AddressDecoderV2     //地址编码器
	TxDecoder       openwallet.TransactionDecoder   //交易单编码器
	Log             *log.OWLogger                   //日志工具
	ContractDecoder openwallet.SmartContractDecoder //智能合约解析器
	Blockscanner    *NearBlockScanner               //区块扫描器
	client          *Client                         //algod client
}

func NewWalletManager() *WalletManager {
	wm := WalletManager{}
	wm.Config = NewConfig(Symbol)
	wm.Blockscanner = NewNearBlockScanner(&wm)
	wm.Decoder = address_decode.NewAddressDecoder()
	wm.DecoderV2 = address_decode.NewAddressDecoderV2()
	wm.TxDecoder = NewTransactionDecoder(&wm)
	//wm.ContractDecoder = &toeknDecoder{wm: &wm}
	wm.Log = log.NewOWLogger(wm.Symbol())
	return &wm
}
