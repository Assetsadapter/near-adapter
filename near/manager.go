package near

import (
	"encoding/json"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/shopspring/decimal"
)

type WalletManager struct {
	openwallet.AssetsAdapterBase

	Config          *WalletConfig                   // 节点配置
	Decoder         openwallet.AddressDecoder       //地址编码器
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
	wm.Decoder = NewAddressDecoder(&wm)
	wm.TxDecoder = NewTransactionDecoder(&wm)
	//wm.ContractDecoder = &toeknDecoder{wm: &wm}
	wm.Log = log.NewOWLogger(wm.Symbol())
	return &wm
}

func (wm *WalletManager) GetBlockByHeight(height uint64, getTxs bool) (*Block, error) {
	param := []interface{}{height}
	result, err := wm.client.Call("block", param)
	if err != nil {
		return nil, err
	}
	block := Block{}
	resultJson := result.Raw
	err = json.Unmarshal([]byte(resultJson), &block)
	if err != nil {
		return nil, err
	}
	//获取chunck 里的txs
	if getTxs {
		for _, chunk := range block.Chunks {
			chunkResponse, err := wm.GetTxByChunk(chunk.ChunkHash)
			if err != nil {
				return nil, err
			}
			for _, tx := range chunkResponse.Transactions {
				if len(tx.Actions) > 0 {
					value := tx.Actions[0].Transfer["deposit"]
					formatValue, err := decimal.NewFromString(value)
					if err != nil {
						return nil, err
					}
					formatValueDecimal := formatValue.Div(decimal.New(1, wm.Decimal()))
					txTransfer := TxTransfer{From: tx.SignerID, To: tx.ReceiverID, TxId: tx.Hash, Value: formatValueDecimal.String()}
					block.TxTransfer = append(block.TxTransfer, txTransfer)
				}
			}
		}
	}
	return &block, nil
}

//获取含有transfer action 的 tx
func (wm *WalletManager) GetTxByChunk(chunkHash string) (*ChunkResponse, error) {
	param := []interface{}{chunkHash}
	result, err := wm.client.Call("chunk", param)
	if err != nil {
		return nil, err
	}
	chunkResp := ChunkResponse{}
	resultJson := result.Raw
	err = json.Unmarshal([]byte(resultJson), &chunkResp)
	if err != nil {
		return nil, err
	}

	//获取chunk 里的txs
	return &chunkResp, nil
}
