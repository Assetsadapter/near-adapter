package near

import (
	"fmt"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/shopspring/decimal"
	hClient "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/xdr"
	"strconv"
	"strings"
)

const (
	blockchainBucket = "blockchain" // blockchain dataset
	//periodOfTask      = 5 * time.Second // task interval
	maxExtractingSize = 10 // thread count

	fixFeePerOperation = "0.001" //RIA one operation min consume 0.001 RIA
)

//TriamBlockScanner Algo block scanner
type TriamBlockScanner struct {
	*openwallet.BlockScannerBase

	CurrentBlockHeight   uint64         //当前区块高度
	extractingCH         chan struct{}  //扫描工作令牌
	wm                   *WalletManager //钱包管理者
	RescanLastBlockCount uint64         //重扫上N个区块数量
}

//ExtractResult 扫描完成的提取结果
type ExtractResult struct {
	extractData map[string][]*openwallet.TxExtractData
	TxID        string
	BlockHeight uint64
	Success     bool
}

//SaveResult result
type SaveResult struct {
	TxID        string
	BlockHeight uint64
	Success     bool
}

// NewEOSBlockScanner create a block scanner
func NewTriamBlockScanner(wm *WalletManager) *TriamBlockScanner {
	bs := TriamBlockScanner{
		BlockScannerBase: openwallet.NewBlockScannerBase(),
	}

	bs.extractingCH = make(chan struct{}, maxExtractingSize)
	bs.wm = wm

	bs.RescanLastBlockCount = 0

	// set task
	bs.SetTask(bs.ScanBlockTask)

	return &bs
}

//返回主币balance
func (bs *TriamBlockScanner) GetNativeAssetBalance(accountId string) (string, error) {
	accountDetail, err := bs.wm.tclient.AccountDetail(hClient.AccountRequest{AccountID: accountId})
	if err != nil {
		return "0", err
	}
	for _, balance := range accountDetail.Balances {
		if balance.Asset.Type == "native" {
			return balance.Balance, nil
		}
	}
	return "0", nil
}

//返回资产assets balance
func (bs *TriamBlockScanner) GetAssetBalance(accountId, assetsCode, issuerAccountId string) (string, error) {
	assetsCode = strings.ToUpper(assetsCode)
	accountDetail, err := bs.wm.tclient.AccountDetail(hClient.AccountRequest{AccountID: accountId})
	if err != nil {
		return "0", err
	}
	for _, balance := range accountDetail.Balances {
		if balance.Asset.Type != "native" && balance.Asset.Code == assetsCode && balance.Asset.Issuer == issuerAccountId {
			return balance.Balance, nil
		}
	}
	return "0", nil
}

//GetBalanceByAddress 查询地址余额
func (bs *TriamBlockScanner) GetBalanceByAddress(address ...string) ([]*openwallet.Balance, error) {

	addrBalanceArr := make([]*openwallet.Balance, 0)
	for _, a := range address {
		balance, err := bs.GetNativeAssetBalance(a)
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

//GetBlockHeight 获取区块链高度
func (bs *TriamBlockScanner) GetBlockHeight() (uint64, error) {

	// Create an account request
	//hClient.LedgerRequest{Limit:1}
	// Load the account detail from the network
	root, err := bs.wm.tclient.Root()
	if err != nil {
		log.Error(err)
	}
	return uint64(root.HorizonSequence), nil
}

//GetCurrentBlock 获取当前最新区块
func (bs *TriamBlockScanner) GetCurrentBlock() (*Block, error) {

	lastHeight, err := bs.GetBlockHeight()
	if err != nil {
		return nil, err
	}

	return bs.GetBlockByHeight(lastHeight)
}

//GetCurrentBlockHeader 获取当前区块高度
func (bs *TriamBlockScanner) GetCurrentBlockHeader() (*openwallet.BlockHeader, error) {

	block, err := bs.GetCurrentBlock()
	if err != nil {
		return nil, err
	}

	return &openwallet.BlockHeader{Height: block.Height, Hash: block.Hash}, nil
}

//SetRescanBlockHeight 重置区块链扫描高度
func (bs *TriamBlockScanner) SetRescanBlockHeight(height uint64) error {
	height = height - 1
	if height < 0 {
		return fmt.Errorf("block height to rescan must greater than 0.")
	}
	block, err := bs.GetBlockByHeight(height)
	if err != nil {
		return err
	}

	bs.wm.SaveLocalNewBlock(height, block.Hash)

	return nil
}

func (bs *TriamBlockScanner) GetBlockByHeight(height uint64) (*Block, error) {
	r, err := bs.wm.tclient.LedgerDetail(uint32(height))
	if err != nil {
		return nil, err
	}

	block := NewBlock(r)

	return block, nil
}

//GetScannedBlockHeader 获取当前扫描的区块头
func (bs *TriamBlockScanner) GetScannedBlockHeader() (*openwallet.BlockHeader, error) {

	var (
		blockHeader *openwallet.BlockHeader
		blockHeight uint64 = 0
		hash        string
		err         error
	)

	blockHeight, hash = bs.wm.GetLocalNewBlock()

	//如果本地没有记录，查询接口的高度
	if blockHeight == 0 {
		blockHeader, err = bs.GetCurrentBlockHeader()
		if err != nil {

			return nil, err
		}
		blockHeight = blockHeader.Height
		//就上一个区块链为当前区块
		blockHeight = blockHeight - 1

		block, err := bs.GetBlockByHeight(blockHeight)
		if err != nil {
			return nil, err
		}
		hash = block.Hash
	}

	return &openwallet.BlockHeader{Height: blockHeight, Hash: hash}, nil
}

//GetScannedBlockHeight 获取已扫区块高度
func (bs *TriamBlockScanner) GetScannedBlockHeight() uint64 {
	localHeight, _ := bs.wm.GetLocalNewBlock()
	return localHeight
}

//GetGlobalMaxBlockHeight 获取区块链全网最大高度
func (bs *TriamBlockScanner) GetGlobalMaxBlockHeight() uint64 {

	height, err := bs.GetBlockHeight()
	if err != nil {
		return 0
	}

	return height
}

//GetTransaction
func (bs *TriamBlockScanner) GetTransaction(hash string) (*Transaction, error) {
	r, err := bs.wm.client.TransactionByID(hash)
	if err != nil {
		return nil, err
	}
	return NewTransaction(r), nil
}

//ScanBlockTask 扫描任务
func (bs *TriamBlockScanner) ScanBlockTask() {

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
		maxHeight, err := bs.GetBlockHeight()
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

		block, err := bs.GetBlockByHeight(currentHeight)
		if err != nil {
			bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)

			//记录未扫区块
			unscanRecord := NewUnscanRecord(currentHeight, "", err.Error())
			bs.SaveUnscanRecord(unscanRecord)
			bs.wm.Log.Std.Info("block height: %d extract failed.", currentHeight)
			continue
		}

		isFork := false

		//判断hash是否上一区块的hash
		if currentHash != block.PrevBlockHash {

			bs.wm.Log.Std.Info("block has been fork on height: %d.", currentHeight)
			bs.wm.Log.Std.Info("block height: %d local hash = %s ", currentHeight-1, currentHash)
			bs.wm.Log.Std.Info("block height: %d mainnet hash = %s ", currentHeight-1, block.PrevBlockHash)

			bs.wm.Log.Std.Info("delete recharge records on block height: %d.", currentHeight-1)

			//查询本地分叉的区块
			forkBlock, _ := bs.wm.GetLocalBlock(currentHeight - 1)

			//删除上一区块链的所有充值记录
			//bs.DeleteRechargesByHeight(currentHeight - 1)
			//删除上一区块链的未扫记录
			bs.wm.DeleteUnscanRecord(currentHeight - 1)
			currentHeight = currentHeight - 2 //倒退2个区块重新扫描
			if currentHeight <= 0 {
				currentHeight = 1
			}

			localBlock, err := bs.wm.GetLocalBlock(currentHeight)
			if err != nil {
				bs.wm.Log.Std.Error("block scanner can not get local block; unexpected error: %v", err)

				//查找core钱包的RPC
				bs.wm.Log.Info("block scanner prev block height:", currentHeight)

				localBlock, err = bs.GetBlockByHeight(currentHeight)
				if err != nil {
					bs.wm.Log.Std.Error("block scanner can not get prev block; unexpected error: %v", err)
					break
				}

			}

			//重置当前区块的hash
			currentHash = localBlock.Hash

			bs.wm.Log.Std.Info("rescan block on height: %d, hash: %s .", currentHeight, currentHash)

			//重新记录一个新扫描起点
			bs.wm.SaveLocalNewBlock(localBlock.Height, localBlock.Hash)

			isFork = true

			if forkBlock != nil {

				//通知分叉区块给观测者，异步处理
				bs.newBlockNotify(forkBlock, isFork)
			}

		} else {
			err = bs.BatchExtractTransaction(block.Height, block.Hash, block.Time)
			if err != nil {
				bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", err)
			}

			//重置当前区块的hash
			currentHash = block.Hash

			//保存本地新高度
			bs.wm.SaveLocalNewBlock(currentHeight, currentHash)
			bs.wm.SaveLocalBlock(block)

			isFork = false

			//通知新区块给观测者，异步处理
			bs.newBlockNotify(block, isFork)
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
func (bs *TriamBlockScanner) ScanBlock(height uint64) error {

	block, err := bs.scanBlock(height)
	if err != nil {
		return err
	}

	//通知新区块给观测者，异步处理
	bs.newBlockNotify(block, false)

	return nil
}

func (bs *TriamBlockScanner) scanBlock(height uint64) (*Block, error) {

	block, err := bs.GetBlockByHeight(height)
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)

		//记录未扫区块
		unscanRecord := NewUnscanRecord(height, "", err.Error())
		bs.SaveUnscanRecord(unscanRecord)
		bs.wm.Log.Std.Info("block height: %d extract failed.", height)
		return nil, err
	}

	bs.wm.Log.Std.Info("block scanner scanning height: %d ...", block.Height)

	err = bs.BatchExtractTransaction(block.Height, block.Hash, block.Time)
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", err)
	}

	return block, nil
}

//rescanFailedRecord 重扫失败记录
func (bs *TriamBlockScanner) RescanFailedRecord() {

	var (
		blockMap = make(map[uint64][]string)
	)

	list, err := bs.wm.GetUnscanRecords()
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

	for height, txs := range blockMap {

		if height == 0 {
			continue
		}

		var hash string

		bs.wm.Log.Std.Info("block scanner rescanning height: %d ...", height)

		var blockTime int64
		if len(txs) == 0 {

			block, err := bs.GetBlockByHeight(height)
			if err != nil {
				bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)
				continue
			}
			blockTime = block.Time
		}

		err = bs.BatchExtractTransaction(height, hash, blockTime)
		if err != nil {
			bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", err)
			continue
		}

		//删除未扫记录
		bs.wm.DeleteUnscanRecord(height)
	}

	//删除未没有找到交易记录的重扫记录
	bs.wm.DeleteUnscanRecordNotFindTX()
}

//newBlockNotify 获得新区块后，通知给观测者
func (bs *TriamBlockScanner) newBlockNotify(block *Block, isFork bool) {
	header := block.BlockHeader(bs.wm.Symbol())
	header.Fork = isFork
	bs.NewBlockNotify(header)
}

//获取一个区块所有transaction
func (bs *TriamBlockScanner) GetTransactionByBlock(blockHeight uint64) ([]horizon.Transaction, error) {
	ledgerTx, err := bs.wm.tclient.Transactions(hClient.TransactionRequest{ForLedger: uint(blockHeight)})
	if err != nil {
		log.Error(err)
	}
	return ledgerTx.Embedded.Records, nil
}

//获取一个区块所有transaction
func (bs *TriamBlockScanner) GetPayOperationByTx(txHash string) ([]operations.Payment, int, error) {
	ledgeOps, err := bs.wm.tclient.Operations(hClient.OperationRequest{ForTransaction: txHash})
	if err != nil {
		log.Error(err)
	}
	var payOps = make([]operations.Payment, 0)
	for _, op := range ledgeOps.Embedded.Records {
		//log.Infof("from=%s to=%s amount=%s")
		opType := op.GetType()
		if opType == operations.TypeNames[xdr.OperationTypePayment] {
			payOP, OK := op.(operations.Payment)
			if OK {
				payOps = append(payOps, payOP)
			} else {

			}
		}
		continue
	}
	return payOps, len(ledgeOps.Embedded.Records), nil
}

//获取一个区块所有payment Operations 和 所有Operations的数量
func (bs *TriamBlockScanner) GetOperationsByBlock(blockHeight uint64) ([]operations.Payment, int, error) {
	ledgeOps, err := bs.wm.tclient.Operations(hClient.OperationRequest{ForLedger: uint(blockHeight)})
	if err != nil {
		log.Error(err)
	}
	var payOps = make([]operations.Payment, 0)
	for _, op := range ledgeOps.Embedded.Records {
		//log.Infof("from=%s to=%s amount=%s")
		opType := op.GetType()
		if opType == operations.TypeNames[xdr.OperationTypePayment] {
			payOP, OK := op.(operations.Payment)
			if OK {
				payOps = append(payOps, payOP)
			} else {

			}
		}
		continue
	}

	return payOps, len(ledgeOps.Embedded.Records), nil
}

//BatchExtractTransaction 批量提取交易单
//直接获取区块 Payment 操作
func (bs *TriamBlockScanner) BatchExtractTransaction(blockHeight uint64, blockHash string, blockTime int64) error {

	var (
		quit   = make(chan struct{})
		done   = 0 //完成标记
		failed = 0
	)
	//先获取所有的交易
	txs, err := bs.GetTransactionByBlock(blockHeight) //[]operations.Paymeny
	if err != nil {
		bs.wm.Log.Std.Info("newExtractDataNotify unexpected error: %v", err)
		return err
	}
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
				unscanRecord := NewUnscanRecord(height, "", "")
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
	extractWork := func(eblockHeight uint64, eBlockHash string, eBlockTime int64, txs []horizon.Transaction, eProducer chan ExtractResult) {
		for _, tx := range txs {
			bs.extractingCH <- struct{}{}
			go func(mBlockHeight uint64, tx horizon.Transaction, end chan struct{}, mProducer chan<- ExtractResult) {

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
func (bs *TriamBlockScanner) extractRuntime(producer chan ExtractResult, worker chan ExtractResult, quit chan struct{}) {

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
func (bs *TriamBlockScanner) ExtractTransaction(blockHeight uint64, blockHash string, tx horizon.Transaction, scanTargetFunc openwallet.BlockScanTargetFunc) ExtractResult {
	var (
		success = true
		result  = ExtractResult{
			BlockHeight: blockHeight,
			TxID:        tx.Hash,
			extractData: make(map[string][]*openwallet.TxExtractData),
		}
	)
	txPayOperations, _, err := bs.GetPayOperationByTx(tx.Hash)
	if err != nil {
		return result
	}
	if len(txPayOperations) == 0 { //没有payment 退出
		result.Success = true
		return result
	}
	txMemo := tx.Memo //提取交易memo
	decimalFee := decimal.New(1, int32(bs.wm.Config.Decimal))
	decimalFeePayed := decimal.New(int64(tx.FeePaid), 0)
	feePayed := decimalFeePayed.Div(decimalFee).String()
	//提出易单明细
	for _, payment := range txPayOperations {
		if payment.GetType() == operations.TypeNames[xdr.OperationTypePayment] {

			from := payment.From
			to := payment.To

			//bs.wm.Log.Std.Info("block scanner scanning tx: %+v", txid)
			//订阅地址为交易单中的发送者
			accountId, ok1 := scanTargetFunc(openwallet.ScanTarget{
				Address:          from,
				BalanceModelType: openwallet.BalanceModelTypeAddress,
			})
			//订阅地址为交易单中的接收者
			accountId2, ok2 := scanTargetFunc(openwallet.ScanTarget{
				Address:          to,
				BalanceModelType: openwallet.BalanceModelTypeAddress,
			})

			//相同账户
			if accountId == accountId2 && len(accountId) > 0 && len(accountId2) > 0 {
				bs.InitExtractResult(payment, feePayed, txMemo, blockHeight, blockHash, accountId, &result, 0)
			} else {
				if ok1 {
					bs.InitExtractResult(payment, feePayed, txMemo, blockHeight, blockHash, accountId, &result, 1)
				}

				if ok2 {
					bs.InitExtractResult(payment, feePayed, txMemo, blockHeight, blockHash, accountId2, &result, 2)
				}
			}

			success = true

		}
	}

	result.Success = success
	return result

}

//InitTronExtractResult operate = 0: 输入输出提取，1: 输入提取，2：输出提取
func (bs *TriamBlockScanner) InitExtractResult(payment operations.Payment, feePayed string, memo string, blockHeight uint64, blockHash string, sourceKey string, result *ExtractResult, operate int64) {

	txExtractDataArray := result.extractData[sourceKey]
	if txExtractDataArray == nil {
		txExtractDataArray = make([]*openwallet.TxExtractData, 0)
	}

	txExtractData := &openwallet.TxExtractData{}

	status := "1"
	reason := ""

	coin := openwallet.Coin{
		Symbol:     bs.wm.Symbol(),
		IsContract: false,
	}
	amount, _ := decimal.NewFromString(payment.Amount)

	transx := &openwallet.Transaction{
		Fees:        feePayed,
		Coin:        coin,
		BlockHash:   blockHash,
		BlockHeight: blockHeight,
		TxID:        payment.GetTransactionHash(),
		Decimal:     bs.wm.Decimal(),
		Amount:      "",
		IsMemo:      true,
		ConfirmTime: 0,
		From:        []string{payment.From + ":" + amount.String()},
		To:          []string{payment.To + ":" + amount.String()},
		Status:      status,
		Reason:      reason,
	}

	wxID := openwallet.GenTransactionWxID(transx)
	transx.WxID = wxID

	txExtractData.Transaction = transx
	if operate == 0 {
		bs.extractTxInput(payment, blockHeight, blockHash, txExtractData)
		bs.extractTxOutput(payment, memo, blockHeight, blockHash, txExtractData)
	} else if operate == 1 {
		bs.extractTxInput(payment, blockHeight, blockHash, txExtractData)
	} else if operate == 2 {
		bs.extractTxOutput(payment, memo, blockHeight, blockHash, txExtractData)
	}

	txExtractDataArray = append(txExtractDataArray, txExtractData)
	result.extractData[sourceKey] = txExtractDataArray
}

//extractTxInput 提取交易单输入部分,无需手续费，所以只包含1个TxInput
func (bs *TriamBlockScanner) extractTxInput(payment operations.Payment, blockHeight uint64, blockHash string, txExtractData *openwallet.TxExtractData) {
	var coin openwallet.Coin
	if payment.Asset.Type != "native" {
		coin = openwallet.Coin{
			Symbol:     bs.wm.Symbol(),
			IsContract: false,
		}
	} else {
		coin = openwallet.Coin{
			Symbol:     bs.wm.Symbol(),
			IsContract: true,
			Contract: openwallet.SmartContract{
				Address:  payment.Asset.Issuer,
				Symbol:   bs.wm.Symbol(),
				Name:     payment.Asset.Code,
				Token:    payment.Asset.Code,
				Decimals: 0,
			},
		}
	}
	amount, _ := decimal.NewFromString(payment.Amount)

	//主网from交易转账信息，第一个TxInput
	txInput := &openwallet.TxInput{}
	txInput.Recharge.Sid = openwallet.GenTxInputSID(payment.GetTransactionHash(), bs.wm.Symbol(), coin.ContractID, uint64(0))
	txInput.Recharge.TxID = payment.GetTransactionHash()
	txInput.Recharge.Address = payment.From
	txInput.Recharge.Coin = coin
	txInput.Recharge.Amount = amount.String()
	txInput.Recharge.BlockHash = blockHash
	txInput.Recharge.BlockHeight = blockHeight
	txInput.Recharge.Index = 0 //账户模型填0
	txInput.Recharge.CreateAt = int64(0)
	txExtractData.TxInputs = append(txExtractData.TxInputs, txInput)

	//手续费也作为一个输出s 暂时不需要
	//	fees := common.IntToDecimals(int64(transaction.Fee), bs.wm.Decimal())
	//	tmp := *txInput
	//	feeCharge := &tmp
	//	feeCharge.Amount = fees.String()
	//	txExtractData.TxInputs = append(txExtractData.TxInputs, feeCharge)
}

//extractTxOutput 提取交易单输入部分,只有一个TxOutPut
func (bs *TriamBlockScanner) extractTxOutput(payment operations.Payment, memo string, blockHeight uint64, blockHash string, txExtractData *openwallet.TxExtractData) {

	amount, _ := decimal.NewFromString(payment.Amount)
	var coin openwallet.Coin
	if payment.Asset.Type == "native" { //RIO 主链币
		coin = openwallet.Coin{
			Symbol:     bs.wm.Symbol(),
			IsContract: false,
		}
	} else {
		coin = openwallet.Coin{ //资产 assets
			Symbol:     bs.wm.Symbol(),
			IsContract: true,
			Contract: openwallet.SmartContract{
				Address:  payment.Asset.Issuer,
				Symbol:   bs.wm.Symbol(),
				Name:     payment.Asset.Code,
				Token:    payment.Asset.Code,
				Decimals: uint64(bs.wm.Decimal()),
			},
		}

		//token转换 充值数量转换为整数
		tokenDecimal := decimal.New(1, int32(bs.wm.Decimal()))
		amount = amount.Mul(tokenDecimal)

	}

	//主网to交易转账信息,只有一个TxOutPut
	txOutput := &openwallet.TxOutPut{}
	txOutput.Recharge.Sid = openwallet.GenTxOutPutSID(payment.GetTransactionHash(), bs.wm.Symbol(), coin.ContractID, uint64(0))
	txOutput.Recharge.TxID = payment.GetTransactionHash()
	txOutput.Recharge.Address = payment.To
	txOutput.Recharge.Coin = coin
	txOutput.Recharge.IsMemo = true
	txOutput.Recharge.Memo = memo
	txOutput.Recharge.Amount = amount.String()
	txOutput.Recharge.BlockHash = blockHash
	txOutput.Recharge.BlockHeight = blockHeight
	txOutput.Recharge.Index = 0 //账户模型填0
	txOutput.Recharge.CreateAt = int64(0)

	txExtractData.TxOutputs = append(txExtractData.TxOutputs, txOutput)
}

//newExtractDataNotify 发送通知
//发送通知
func (bs *TriamBlockScanner) newExtractDataNotify(height uint64, extractData map[string][]*openwallet.TxExtractData) error {
	for o, _ := range bs.Observers {
		for key, array := range extractData {
			for _, data := range array {
				err := o.BlockExtractDataNotify(key, data)
				if err != nil {
					bs.wm.Log.Error("BlockExtractDataNotify unexpected error:", err)
					//记录未扫区块
					unscanRecord := NewUnscanRecord(height, "", "ExtractData Notify failed.")
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

//ExtractTransactionData
func (bs *TriamBlockScanner) ExtractTransactionData(txid string, scanAddressFunc openwallet.BlockScanTargetFunc) (map[string][]*openwallet.TxExtractData, error) {

	//, err := bs.GetTransaction(txid)
	//if err != nil {
	//	bs.wm.Log.Std.Info("block scanner can not extract transaction data; unexpected error: %v", err)
	//	return nil, err
	//}

	//result := bs.ExtractTransaction(0, "", trx, scanAddressFunc)
	return nil, nil
}

//获取下一个账户的seq
func (bs *TriamBlockScanner) getAccountSeq(account string) int64 {
	accountDetail, _ := bs.wm.tclient.AccountDetail(hClient.AccountRequest{AccountID: account})
	seq, _ := strconv.ParseInt(accountDetail.Sequence, 10, 64)
	return seq
}

//检查账户是否存在
func (bs *TriamBlockScanner) AccountExists(account string) bool {
	_, err := bs.wm.tclient.AccountDetail(hClient.AccountRequest{AccountID: account})
	if err != nil {
		return false
	}
	return true
}
