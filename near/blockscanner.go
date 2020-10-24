package near

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/btcsuite/btcutil/base58"
	"github.com/shopspring/decimal"
	"strings"
)

const (
	blockchainBucket = "blockchain" // blockchain dataset
	//periodOfTask      = 5 * time.Second // task interval
	maxExtractingSize = 10 // thread count

	fixFeePerOperation = "0.001" //RIA one operation min consume 0.001 RIA
)

type NearBlockScanner struct {
	*openwallet.BlockScannerBase

	CurrentBlockHeight   uint64         //当前区块高度
	extractingCH         chan struct{}  //扫描工作令牌
	wm                   *WalletManager //钱包管理者
	RescanLastBlockCount uint64         //重扫上N个区块数量
}

//
////ExtractResult 扫描完成的提取结果
type ExtractResult struct {
	extractData map[string][]*openwallet.TxExtractData
	TxID        string
	BlockHeight uint64
	Success     bool
}

//
////SaveResult result
type SaveResult struct {
	TxID        string
	BlockHeight uint64
	Success     bool
}

//
//// NewEOSBlockScanner create a block scanner
func NewNearBlockScanner(wm *WalletManager) *NearBlockScanner {
	bs := NearBlockScanner{
		BlockScannerBase: openwallet.NewBlockScannerBase(),
	}

	bs.extractingCH = make(chan struct{}, maxExtractingSize)
	bs.wm = wm

	bs.RescanLastBlockCount = 0

	// set task
	bs.SetTask(bs.ScanBlockTask)

	return &bs
}

//GetBalanceByAddress 查询地址余额
func (bs *NearBlockScanner) GetBalanceByAddress(address ...string) ([]*openwallet.Balance, error) {

	addrBalanceArr := make([]*openwallet.Balance, 0)
	for _, a := range address {
		balance, err := bs.GetAccountBalance(a)
		if err == nil {

			obj := &openwallet.Balance{
				Symbol:           bs.wm.Symbol(),
				Address:          a,
				Balance:          balance,
				UnconfirmBalance: "0",
				ConfirmBalance:   balance,
			}

			addrBalanceArr = append(addrBalanceArr, obj)
		}

	}

	return addrBalanceArr, nil
}

//GetCurrentBlockHeader 获取当前区块高度
func (bs *NearBlockScanner) GetCurrentBlockHeader() (*openwallet.BlockHeader, error) {

	result, err := bs.wm.client.Get("/status", nil)
	if err != nil {
		log.Error(err)
	}
	status := Status{}
	if err := json.Unmarshal([]byte(result.Raw), &status); err != nil {
		return nil, err
	}

	return &openwallet.BlockHeader{Height: uint64(status.SyncInfo.LatestBlockHeight), Hash: status.SyncInfo.LatestBlockHash}, nil
}

//SetRescanBlockHeight 重置区块链扫描高度
func (bs *NearBlockScanner) SetRescanBlockHeight(height uint64) error {
	height = height - 1
	if height < 0 {
		return fmt.Errorf("block height to rescan must greater than 0.")
	}
	block, err := bs.GetBlockByHeight(height, false)
	if err != nil {
		return err
	}

	bs.SaveLocalNewBlock(height, block.Header.Hash)

	return nil
}

//GetLocalNewBlock 获取本地记录的区块高度和hash
func (bs *NearBlockScanner) GetLocalNewBlock() (uint64, string, error) {

	if bs.BlockchainDAI == nil {
		return 0, "", fmt.Errorf("Blockchain DAI is not setup ")
	}

	header, err := bs.BlockchainDAI.GetCurrentBlockHead(bs.wm.Symbol())
	if err != nil {
		return 0, "", err
	}

	return header.Height, header.Hash, nil
}

//GetScannedBlockHeader 获取已扫高度区块头
func (bs *NearBlockScanner) GetScannedBlockHeader() (*openwallet.BlockHeader, error) {

	var (
		blockHeight uint64 = 0
		hash        string
		err         error
	)

	blockHeight, hash, err = bs.GetLocalNewBlock()
	if err != nil {
		bs.wm.Log.Errorf("get local new block failed, err=%v", err)
		return nil, err
	}

	//如果本地没有记录，查询接口的高度
	if blockHeight == 0 {
		blockHeight, err = bs.GetCurrentBlock()
		if err != nil {
			bs.wm.Log.Errorf("NEAR GetBlockHeight failed,err = %v", err)
			return nil, err
		}

		//就上一个区块链为当前区块
		blockHeight = blockHeight - 1
		block, err := bs.GetBlockByHeight(blockHeight, false)
		if err != nil {
			bs.wm.Log.Errorf("get block spec by block number failed, err=%v", err)
			return nil, err
		}

		hash = block.Header.Hash
	}

	return &openwallet.BlockHeader{Height: blockHeight, Hash: hash}, nil
}

//GetScannedBlockHeight 获取已扫区块高度
func (bs *NearBlockScanner) GetScannedBlockHeight() uint64 {
	localHeight, _, _ := bs.wm.Blockscanner.GetLocalNewBlock()
	return localHeight
}

//GetGlobalMaxBlockHeight 获取区块链全网最大高度
func (bs *NearBlockScanner) GetGlobalMaxBlockHeight() uint64 {

	height, err := bs.GetCurrentBlock()
	if err != nil {
		return 0
	}

	return height
}

//GetTransaction
//func (bs *NearBlockScanner) GetTransaction(hash string) (*Transaction, error) {
//	r, err := bs.wm.client.TransactionByID(hash)
//	if err != nil {
//		return nil, err
//	}
//	return NewTransaction(r), nil
//}
//SaveLocalNewBlock 记录区块高度和hash到本地
func (bs *NearBlockScanner) SaveLocalNewBlock(blockHeight uint64, blockHash string) error {

	if bs.BlockchainDAI == nil {
		return fmt.Errorf("Blockchain DAI is not setup ")
	}

	header := &openwallet.BlockHeader{
		Hash:   blockHash,
		Height: blockHeight,
		Fork:   false,
		Symbol: bs.wm.Symbol(),
	}

	//bs.wm.Log.Std.Info("block scanner Save Local New Block: %v", header)

	return bs.BlockchainDAI.SaveCurrentBlockHead(header)
}

//ScanBlockTask 扫描任务
func (bs *NearBlockScanner) ScanBlockTask() {

	//获取本地区块高度
	blockHeader, err := bs.GetScannedBlockHeader()
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get new block height; unexpected error: %v", err)
		return
	}

	currentHeight := blockHeader.Height
	currentHash := blockHeader.Hash

	for {

		if !bs.Scanning {
			//区块扫描器已暂停，马上结束本次任务
			return
		}

		//获取最大高度
		maxHeight, err := bs.GetCurrentBlock()
		if err != nil {
			//下一个高度找不到会报异常
			bs.wm.Log.Std.Info("block scanner can not get rpc-server block height; unexpected error: %v", err)
			break
		}

		//是否已到最新高度
		if currentHeight >= maxHeight {
			bs.wm.Log.Std.Info("block scanner has scanned full chain data. Current height: %d", maxHeight)
			break
		}

		//继续扫描下一个区块
		currentHeight = currentHeight + 1

		bs.wm.Log.Std.Info("block scanner scanning height: %d ...", currentHeight)

		block, err := bs.GetBlockByHeight(currentHeight, true)
		if err != nil {
			bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)

			//记录未扫区块
			unscanRecord := openwallet.NewUnscanRecord(currentHeight, "", "ExtractData Notify failed.", bs.wm.Symbol())
			bs.SaveUnscanRecord(unscanRecord)
			bs.wm.Log.Std.Info("block height: %d extract failed.", currentHeight)
			continue
		}

		isFork := false

		//判断hash是否上一区块的hash
		if currentHash != block.Header.PrevHash {

			bs.wm.Log.Std.Info("block has been fork on height: %d.", currentHeight)
			bs.wm.Log.Std.Info("block height: %d local hash = %s ", currentHeight-1, currentHash)
			bs.wm.Log.Std.Info("block height: %d mainnet hash = %s ", currentHeight-1, block.Header.PrevHash)

			bs.wm.Log.Std.Info("delete recharge records on block height: %d.", currentHeight-1)

			//查询本地分叉的区块
			forkBlock, _ := bs.GetLocalBlock(currentHeight - 1)

			//删除上一区块链的所有充值记录
			//bs.DeleteRechargesByHeight(currentHeight - 1)
			//删除上一区块链的未扫记录
			bs.DeleteUnscanRecord(currentHeight - 1)
			currentHeight = currentHeight - 2 //倒退2个区块重新扫描
			if currentHeight <= 0 {
				currentHeight = 1
			}

			localBlockHeader, err := bs.GetLocalBlock(currentHeight)
			if err != nil {
				bs.wm.Log.Std.Error("block scanner can not get local block; unexpected error: %v", err)

				//查找core钱包的RPC
				bs.wm.Log.Info("block scanner prev block height:", currentHeight)

				block, err = bs.GetBlockByHeight(currentHeight, false)
				if err != nil {
					bs.wm.Log.Std.Error("block scanner can not get prev block; unexpected error: %v", err)
					break
				}
				localBlockHeader = &block.Header

			}

			//重置当前区块的hash
			currentHash = localBlockHeader.PrevHash

			bs.wm.Log.Std.Info("rescan block on height: %d, hash: %s .", currentHeight, currentHash)

			//重新记录一个新扫描起点
			bs.SaveLocalNewBlock(uint64(localBlockHeader.Height), localBlockHeader.Hash)
			bs.SaveLocalBlock(localBlockHeader)

			isFork = true

			if forkBlock != nil {

				//通知分叉区块给观测者，异步处理
				bs.newBlockNotify(forkBlock, isFork)
			}

		} else {
			err = bs.BatchExtractTransaction(block.Header.Height, block.Header.Hash, block.TxTransfer, 0)
			if err != nil {
				bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", err)
			}

			//重置当前区块的hash
			currentHash = block.Header.Hash

			//保存本地新高度
			bs.SaveLocalNewBlock(currentHeight, currentHash)
			bs.SaveLocalBlock(&block.Header)

			isFork = false

			//通知新区块给观测者，异步处理
			bs.newBlockNotify(&block.Header, isFork)
		}

	}

	//重扫前N个块，为保证记录找到
	for i := currentHeight - bs.RescanLastBlockCount; i < currentHeight; i++ {
		bs.scanBlock(i)
	}

	//重扫失败区块
	bs.RescanFailedRecord()

}

//ScanBlock 扫描指定高度区块
func (bs *NearBlockScanner) ScanBlock(height uint64) error {

	block, err := bs.scanBlock(height)
	if err != nil {
		return err
	}

	//通知新区块给观测者，异步处理
	bs.newBlockNotify(&block.Header, false)

	return nil
}

func (bs *NearBlockScanner) scanBlock(height uint64) (*Block, error) {

	block, err := bs.GetBlockByHeight(height, true)
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)

		//记录未扫区块
		unscanRecord := openwallet.NewUnscanRecord(height, "", err.Error(), bs.wm.Symbol())
		bs.SaveUnscanRecord(unscanRecord)
		bs.wm.Log.Std.Info("block height: %d extract failed.", height)
		return nil, err
	}

	bs.wm.Log.Std.Info("block scanner scanning height: %d ...", block.Header.Height)

	err = bs.BatchExtractTransaction(block.Header.Height, block.Header.Hash, block.TxTransfer, 0)
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", err)
	}

	return block, nil
}

//rescanFailedRecord 重扫失败记录
func (bs *NearBlockScanner) RescanFailedRecord() {

	var (
		blockMap = make(map[uint64][]string)
	)

	list, err := bs.GetUnscanRecords()
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get rescan data; unexpected error: %v", err)
	}

	//组合成批处理
	for _, r := range list {

		if _, exist := blockMap[r.BlockHeight]; !exist {
			blockMap[r.BlockHeight] = make([]string, 0)
		}

		if len(r.TxID) > 0 {
			arr := blockMap[r.BlockHeight]
			arr = append(arr, r.TxID)

			blockMap[r.BlockHeight] = arr
		}
	}

	for height := range blockMap {

		if height == 0 {
			continue
		}

		bs.wm.Log.Std.Info("block scanner rescanning height: %d ...", height)

		block, err := bs.GetBlockByHeight(height, true)
		if err != nil {
			bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)
			continue
		}

		err = bs.BatchExtractTransaction(block.Header.Height, block.Header.Hash, block.TxTransfer, 0)
		if err != nil {
			bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", err)
			continue
		}
		//删除未扫记录
		bs.wm.Blockscanner.DeleteUnscanRecord(height)
	}

	//删除未没有找到交易记录的重扫记录
	bs.wm.Blockscanner.DeleteUnscanRecordNotFindTX()
}

//DeleteUnscanRecordNotFindTX 删除未没有找到交易记录的重扫记录
func (bs *NearBlockScanner) DeleteUnscanRecordNotFindTX() error {

	//删除找不到交易单
	reason := "[-5]No information available about transaction"

	if bs.BlockchainDAI == nil {
		return fmt.Errorf("Blockchain DAI is not setup ")
	}

	list, err := bs.BlockchainDAI.GetUnscanRecords(bs.wm.Symbol())
	if err != nil {
		return err
	}

	for _, r := range list {
		if strings.HasPrefix(r.Reason, reason) {
			bs.BlockchainDAI.DeleteUnscanRecordByID(r.ID, bs.wm.Symbol())
		}
	}
	return nil
}

//newBlockNotify 获得新区块后，通知给观测者
func (bs *NearBlockScanner) newBlockNotify(blockHeader *BlockHeader, isFork bool) {
	obj := openwallet.BlockHeader{}
	//解析json
	obj.Hash = blockHeader.Hash
	//obj.Confirmations = b.Confirmations
	obj.Previousblockhash = blockHeader.PrevHash
	obj.Height = blockHeader.Height
	obj.Fork = isFork
	obj.Symbol = bs.wm.Symbol()
	bs.NewBlockNotify(&obj)
}

//BatchExtractTransaction 批量提取交易单
//直接获取区块 Payment 操作
func (bs *NearBlockScanner) BatchExtractTransaction(blockHeight uint64, blockHash string, txs []TxTransfer, blockTime int64) error {

	var (
		quit   = make(chan struct{})
		done   = 0 //完成标记
		failed = 0
	)

	shouldDone := len(txs) //需要完成的总数
	if len(txs) == 0 {     //没交易直接退出
		return nil
	}

	//生产通道
	producer := make(chan ExtractResult)
	defer close(producer)

	//消费通道
	worker := make(chan ExtractResult)
	defer close(worker)

	//保存工作
	saveWork := func(height uint64, result chan ExtractResult) {
		//回收创建的地址
		for gets := range result {

			if gets.Success {

				notifyErr := bs.newExtractDataNotify(height, gets.extractData)
				//saveErr := bs.SaveRechargeToWalletDB(height, gets.Recharges)
				if notifyErr != nil {
					failed++ //标记保存失败数
					bs.wm.Log.Std.Info("newExtractDataNotify unexpected error: %v", notifyErr)
				}

			} else {
				//记录未扫区块
				unscanRecord := openwallet.NewUnscanRecord(height, "", "", bs.wm.Symbol())
				bs.SaveUnscanRecord(unscanRecord)
				bs.wm.Log.Std.Info("block height: %d extract failed.", height)
				failed++ //标记保存失败数
			}
			//累计完成的线程数
			done++
			if done == shouldDone {
				//bs.wm.Log.Std.Info("done = %d, shouldDone = %d ", done, len(txs))
				close(quit) //关闭通道，等于给通道传入nil
			}
		}
	}

	//提取工作
	extractWork := func(eblockHeight uint64, eBlockHash string, eBlockTime int64, txs []TxTransfer, eProducer chan ExtractResult) {
		for _, tx := range txs {
			bs.extractingCH <- struct{}{}
			go func(mBlockHeight uint64, tx TxTransfer, end chan struct{}, mProducer chan<- ExtractResult) {

				//导出提出的交易
				mProducer <- bs.ExtractTransaction(mBlockHeight, eBlockHash, tx, bs.ScanTargetFunc)
				//释放
				<-end

			}(eblockHeight, tx, bs.extractingCH, eProducer)
		}
	}

	/*	开启导出的线程	*/

	//独立线程运行消费
	go saveWork(blockHeight, worker)

	//独立线程运行生产
	go extractWork(blockHeight, blockHash, blockTime, txs, producer)

	//以下使用生产消费模式
	bs.extractRuntime(producer, worker, quit)

	if failed > 0 {
		return fmt.Errorf("block scanner saveWork failed")
	} else {
		return nil
	}

	//return nil
}

//extractRuntime 提取运行时
func (bs *NearBlockScanner) extractRuntime(producer chan ExtractResult, worker chan ExtractResult, quit chan struct{}) {

	var (
		values = make([]ExtractResult, 0)
	)

	for {

		var activeWorker chan<- ExtractResult
		var activeValue ExtractResult

		//当数据队列有数据时，释放顶部，传输给消费者
		if len(values) > 0 {
			activeWorker = worker
			activeValue = values[0]

		}

		select {

		//生成者不断生成数据，插入到数据队列尾部
		case pa := <-producer:
			values = append(values, pa)
		case <-quit:
			//退出
			//bs.wm.Log.Std.Info("block scanner have been scanned!")
			return
		case activeWorker <- activeValue:
			//wm.Log.Std.Info("Get %d", len(activeValue))
			values = values[1:]
		}
	}

}

//提取交易单
func (bs *NearBlockScanner) ExtractTransaction(blockHeight uint64, blockHash string, tx TxTransfer, scanTargetFunc openwallet.BlockScanTargetFunc) ExtractResult {
	var (
		success = true
		result  = ExtractResult{
			BlockHeight: blockHeight,
			TxID:        tx.TxId,
			extractData: make(map[string][]*openwallet.TxExtractData),
		}
	)

	decimalFee := decimal.New(1, int32(bs.wm.Config.Decimal))
	decimalFeePayed := decimal.New(int64(0), 0)
	feePayed := decimalFeePayed.Div(decimalFee).String()
	//提出易单明细
	accountId, ok1 := scanTargetFunc(openwallet.ScanTarget{
		Address:          tx.From,
		BalanceModelType: openwallet.BalanceModelTypeAddress,
	})
	//订阅地址为交易单中的接收者
	accountId2, ok2 := scanTargetFunc(openwallet.ScanTarget{
		Address:          tx.To,
		BalanceModelType: openwallet.BalanceModelTypeAddress,
	})

	//相同账户
	if accountId == accountId2 && len(accountId) > 0 && len(accountId2) > 0 {
		bs.InitExtractResult(tx, feePayed, blockHeight, blockHash, accountId, &result, 0)
	} else {
		if ok1 {
			bs.InitExtractResult(tx, feePayed, blockHeight, blockHash, accountId, &result, 1)
		}

		if ok2 {
			bs.InitExtractResult(tx, feePayed, blockHeight, blockHash, accountId2, &result, 2)
		}
	}

	success = true

	result.Success = success
	return result

}

//InitTronExtractResult operate = 0: 输入输出提取，1: 输入提取，2：输出提取
func (bs *NearBlockScanner) InitExtractResult(tx TxTransfer, feePayed string, blockHeight uint64, blockHash string, sourceKey string, result *ExtractResult, operate int64) {

	txExtractDataArray := result.extractData[sourceKey]
	if txExtractDataArray == nil {
		txExtractDataArray = make([]*openwallet.TxExtractData, 0)
	}

	txExtractData := &openwallet.TxExtractData{}

	reason := ""

	coin := openwallet.Coin{
		Symbol:     bs.wm.Symbol(),
		IsContract: false,
	}
	amount, _ := decimal.NewFromString(tx.Value)

	transx := &openwallet.Transaction{
		Fees:        feePayed,
		Coin:        coin,
		BlockHash:   blockHash,
		BlockHeight: blockHeight,
		TxID:        tx.TxId,
		Decimal:     bs.wm.Decimal(),
		Amount:      "",
		IsMemo:      true,
		ConfirmTime: 0,
		From:        []string{tx.From + ":" + amount.String()},
		To:          []string{tx.To + ":" + amount.String()},
		Status:      tx.Status,
		Reason:      reason,
	}

	wxID := openwallet.GenTransactionWxID(transx)
	transx.WxID = wxID

	txExtractData.Transaction = transx
	if operate == 0 {
		bs.extractTxInput(tx, blockHeight, blockHash, txExtractData)
		bs.extractTxOutput(tx, blockHeight, blockHash, txExtractData)
	} else if operate == 1 {
		bs.extractTxInput(tx, blockHeight, blockHash, txExtractData)
	} else if operate == 2 {
		bs.extractTxOutput(tx, blockHeight, blockHash, txExtractData)
	}

	txExtractDataArray = append(txExtractDataArray, txExtractData)
	result.extractData[sourceKey] = txExtractDataArray
}

//extractTxInput 提取交易单输入部分,无需手续费，所以只包含1个TxInput
func (bs *NearBlockScanner) extractTxInput(tx TxTransfer, blockHeight uint64, blockHash string, txExtractData *openwallet.TxExtractData) {
	coin := openwallet.Coin{
		Symbol:     bs.wm.Symbol(),
		IsContract: false,
	}

	amount, _ := decimal.NewFromString(tx.Value)

	//主网from交易转账信息，第一个TxInput
	txInput := &openwallet.TxInput{}
	txInput.Recharge.Sid = openwallet.GenTxInputSID(tx.TxId, bs.wm.Symbol(), coin.ContractID, uint64(0))
	txInput.Recharge.TxID = tx.TxId
	txInput.Recharge.Address = tx.From
	txInput.Recharge.Coin = coin
	txInput.Recharge.Amount = amount.String()
	txInput.Recharge.BlockHash = blockHash
	txInput.Recharge.BlockHeight = blockHeight
	txInput.Recharge.Index = 0 //账户模型填0
	txInput.Recharge.CreateAt = int64(0)
	txExtractData.TxInputs = append(txExtractData.TxInputs, txInput)

}

//extractTxOutput 提取交易单输入部分,只有一个TxOutPut
func (bs *NearBlockScanner) extractTxOutput(tx TxTransfer, blockHeight uint64, blockHash string, txExtractData *openwallet.TxExtractData) {

	amount, _ := decimal.NewFromString(tx.Value)
	coin := openwallet.Coin{
		Symbol:     bs.wm.Symbol(),
		IsContract: false,
	}

	//主网to交易转账信息,只有一个TxOutPut
	txOutput := &openwallet.TxOutPut{}
	txOutput.Recharge.Sid = openwallet.GenTxOutPutSID(tx.TxId, bs.wm.Symbol(), coin.ContractID, uint64(0))
	txOutput.Recharge.TxID = tx.TxId
	txOutput.Recharge.Address = tx.To
	txOutput.Recharge.Coin = coin
	txOutput.Recharge.IsMemo = false
	txOutput.Recharge.Amount = amount.String()
	txOutput.Recharge.BlockHash = blockHash
	txOutput.Recharge.BlockHeight = blockHeight
	txOutput.Recharge.Index = 0 //账户模型填0
	txOutput.Recharge.CreateAt = int64(0)

	txExtractData.TxOutputs = append(txExtractData.TxOutputs, txOutput)
}

//newExtractDataNotify 发送通知
//发送通知
func (bs *NearBlockScanner) newExtractDataNotify(height uint64, extractData map[string][]*openwallet.TxExtractData) error {
	for o, _ := range bs.Observers {
		for key, array := range extractData {
			for _, data := range array {
				err := o.BlockExtractDataNotify(key, data)
				if err != nil {
					bs.wm.Log.Error("BlockExtractDataNotify unexpected error:", err)
					//记录未扫区块
					unscanRecord := openwallet.NewUnscanRecord(height, "", "ExtractData Notify failed.", bs.wm.Symbol())
					err = bs.SaveUnscanRecord(unscanRecord)
					if err != nil {
						bs.wm.Log.Std.Error("block height: %d, save unscan record failed. unexpected error: %v", height, err.Error())
					}
				}
			}
		}
	}
	return nil
}

func (bs *NearBlockScanner) GetBlockByHeight(height uint64, getTxs bool) (*Block, error) {
	param := []interface{}{height}
	result, err := bs.wm.client.Call("block", param)
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
			chunkResponse, err := bs.GetTxByChunk(chunk.ChunkHash)
			if err != nil {
				return nil, err
			}
			for _, tx := range chunkResponse.Transactions {
				if len(tx.Actions) > 0 {
					value := tx.Actions[0].Transfer["deposit"]
					if "" == value {
						continue
					}
					formatValue, err := decimal.NewFromString(value)
					if err != nil {
						return nil, err
					}
					formatValueDecimal := formatValue.Div(decimal.New(1, bs.wm.Decimal()))
					txStatus, err := bs.GetTxStatus(tx.Hash, tx.SignerID)
					txTransfer := TxTransfer{From: tx.SignerID, To: tx.ReceiverID, TxId: tx.Hash, Value: formatValueDecimal.String(), Status: txStatus}
					block.TxTransfer = append(block.TxTransfer, txTransfer)
				}
			}
		}
	}
	return &block, nil
}

//GetBlockHeight 获取区块链高度
func (bs *NearBlockScanner) GetCurrentBlock() (uint64, error) {

	result, err := bs.wm.client.Get("/status", nil)
	if err != nil {
		log.Error(err)
	}
	status := Status{}
	if err := json.Unmarshal([]byte(result.Raw), &status); err != nil {
		return 0, err
	}
	return uint64(status.SyncInfo.LatestBlockHeight), nil
}

func (bs *NearBlockScanner) GetLatestRefBlockHash() (string, error) {
	currentBlockHeight, err := bs.GetCurrentBlock()
	if err != nil {
		return "0", err
	}

	block, err := bs.GetBlockByHeight(currentBlockHeight-500, false)
	if err != nil {
		return "0", err
	}
	return block.Header.Hash, nil
}

//获取含有transfer action 的 tx
func (bs *NearBlockScanner) GetTxByChunk(chunkHash string) (*ChunkResponse, error) {
	param := []interface{}{chunkHash}
	result, err := bs.wm.client.Call("chunk", param)
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

//获取含有transfer action 的 tx
func (bs *NearBlockScanner) GetTxStatus(txId, senderId string) (string, error) {
	param := []interface{}{txId, senderId}
	result, err := bs.wm.client.Call("tx", param)
	if err != nil {
		return "0", err
	}
	txResp := TransactionStatus{}
	resultJson := result.Raw
	err = json.Unmarshal([]byte(resultJson), &txResp)
	if err != nil {
		return "0", err
	}
	if _, exists := txResp.Status["SuccessValue"]; exists {
		return "1", nil
	}
	//获取chunk 里的txs
	return "0", nil
}

//获取含有transfer action 的 tx
func (bs *NearBlockScanner) GetAccountBalance(accountId string) (string, error) {
	param := map[string]interface{}{"request_type": "view_account", "finality": "final", "account_id": accountId}
	result, err := bs.wm.client.Call2("query", param)
	if err != nil {
		return "0", err
	}
	accountResp := AccountResponse{}
	resultJson := result.Raw
	err = json.Unmarshal([]byte(resultJson), &accountResp)
	if err != nil {
		return "0", err
	}
	formatValue, err := decimal.NewFromString(accountResp.Amount)
	if err != nil {
		return "0", err
	}
	formatValueDecimal := formatValue.Div(decimal.New(1, bs.wm.Decimal()))
	return formatValueDecimal.String(), nil
}

//获取含有transfer action 的 tx
func (bs *NearBlockScanner) GetAccountNonce(accountId string) (uint64, error) {
	hexBytes, err := hex.DecodeString(accountId)
	if err != nil {
		return 0, nil
	}
	publicKey := "ed25519:" + base58.Encode(hexBytes)
	param := map[string]interface{}{"request_type": "view_access_key", "finality": "final", "account_id": accountId, "public_key": publicKey}
	result, err := bs.wm.client.Call2("query", param)
	if err != nil {
		return 0, err
	}
	if err != nil {
		return 0, err
	}
	accessKeyResp := AccessKeyResponse{}
	resultJson := result.Raw
	err = json.Unmarshal([]byte(resultJson), &accessKeyResp)
	if err != nil {
		return 0, err
	}
	return accessKeyResp.Nonce, nil
}

//ExtractTransactionData
func (bs *NearBlockScanner) ExtractTransactionData(txid string, scanAddressFunc openwallet.BlockScanTargetFunc) (map[string][]*openwallet.TxExtractData, error) {

	//, err := bs.GetTransaction(txid)
	//if err != nil {
	//	bs.wm.Log.Std.Info("block scanner can not extract transaction data; unexpected error: %v", err)
	//	return nil, err
	//}

	//result := bs.ExtractTransaction(0, "", trx, scanAddressFunc)
	return nil, nil
}

//Run 运行
func (bs *NearBlockScanner) Run() error {

	bs.BlockScannerBase.Run()

	return nil
}

////Stop 停止扫描
func (bs *NearBlockScanner) Stop() error {

	bs.BlockScannerBase.Stop()

	return nil
}

//Pause 暂停扫描
func (bs *NearBlockScanner) Pause() error {

	bs.BlockScannerBase.Pause()

	return nil
}

//Restart 继续扫描
func (bs *NearBlockScanner) Restart() error {

	bs.BlockScannerBase.Restart()

	return nil
}

/******************* 使用insight socket.io 监听区块 *******************/

//setupSocketIO 配置socketIO监听新区块
func (bs *NearBlockScanner) setupSocketIO() error {
	return nil
}

//SupportBlockchainDAI 支持外部设置区块链数据访问接口
//@optional
func (bs *NearBlockScanner) SupportBlockchainDAI() bool {
	return true
}
