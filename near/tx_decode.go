package near

import (
	"encoding/base64"
	"encoding/hex"
	"github.com/Assetsadapter/near-adapter/near/txnbuild"
	"github.com/pkg/errors"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"strconv"
	"strings"
	"time"

	"github.com/Assetsadapter/near-adapter/txsigner"
	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/shopspring/decimal"
)

// txidPrefix is prepended to a transaction when computing its txid
var txidPrefix = []byte("TX")

type TransactionDecoder struct {
	openwallet.TransactionDecoderBase
	wm *WalletManager //钱包管理者
}

//NewTransactionDecoder 交易单解析器
func NewTransactionDecoder(wm *WalletManager) *TransactionDecoder {
	decoder := TransactionDecoder{}
	decoder.wm = wm
	return &decoder
}
func (decoder *TransactionDecoder) CreateRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {
	if !rawTx.Coin.IsContract {
		return decoder.CreateRawSimpleTransaction(wrapper, rawTx)
	}
	return decoder.CreateRawAssetsTransaction(wrapper, rawTx)

}

//CreateRawTransaction 创建交易单
func (decoder *TransactionDecoder) CreateRawSimpleTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	var (
		accountID       = rawTx.Account.AccountID
		estimateFees    = decimal.Zero
		findAddrBalance *AddrBalance
		retainAmount    = decoder.wm.Config.AddressRetainAmount
	)

	//获取wallet
	addresses, err := wrapper.GetAddressList(0, -1, "AccountID", accountID)
	if err != nil {
		return err
	}

	if len(addresses) == 0 {
		return openwallet.Errorf(openwallet.ErrCreateRawTransactionFailed, "[%s] have not addresses", accountID)
	}

	var amountStr string
	var destAddr string
	for k, v := range rawTx.To {
		destAddr = k
		amountStr = v
		break
	}
	//检查目标账户是否存在
	if !decoder.wm.Blockscanner.AccountExists(destAddr) {
		return openwallet.Errorf(10000, "account not exists")

	}
	amountSent, _ := decimal.NewFromString(amountStr)
	forceRetainAmount, _ := decimal.NewFromString(decoder.wm.Config.AddressRetainAmount)

	//if len(rawTx.FeeRate) > 0 {
	//	estimateFees = common.StringNumToBigIntWithExp(rawTx.FeeRate, decimals)
	//} else {
	//	estimateFees = common.StringNumToBigIntWithExp(decoder.wm.Config.FixFees, decimals)
	//}
	estimateFees, _ = decimal.NewFromString("0.001")

	for _, addr := range addresses {
		resp, _ := decoder.wm.Blockscanner.GetBalanceByAddress(addr.Address)
		if len(resp) == 0 {
			continue
		}
		balanceAmount, _ := decimal.NewFromString(resp[0].ConfirmBalance)
		if err != nil {
			continue
		}

		//总消耗数量 = 转账数量 + 手续费
		totalAmount := decimal.Zero
		totalAmount = totalAmount.Add(amountSent)
		totalAmount = totalAmount.Add(estimateFees)
		totalAmount = totalAmount.Add(forceRetainAmount)

		//余额不足查找下一个地址
		if balanceAmount.Cmp(totalAmount) < 0 {
			continue
		}

		//只要找到一个合适使用的地址余额就停止遍历
		findAddrBalance = NewAddrBalance(resp[0])
		break
	}

	if findAddrBalance == nil {
		return openwallet.Errorf(openwallet.ErrInsufficientBalanceOfAccount, "all address's balance of account is not enough, an address required to retain at least %s algos", retainAmount)
	}
	//检查源账户是否存在
	if !decoder.wm.Blockscanner.AccountExists(findAddrBalance.Address) {
		return openwallet.Errorf(10000, "account not exists")

	}

	//最后创建交易单
	err = decoder.createRawTransaction(
		wrapper,
		rawTx,
		findAddrBalance,
	)
	if err != nil {
		return err
	}

	return nil

}

//CreateRawTransaction 创建交易单
func (decoder *TransactionDecoder) CreateRawAssetsTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	var (
		accountID       = rawTx.Account.AccountID
		findAddrBalance *AddrBalance
		retainAmount    = decoder.wm.Config.AddressRetainAmount
	)

	//获取wallet
	addresses, err := wrapper.GetAddressList(0, -1, "AccountID", accountID)
	if err != nil {
		return err
	}

	if len(addresses) == 0 {
		return openwallet.Errorf(openwallet.ErrCreateRawTransactionFailed, "[%s] have not addresses", accountID)
	}

	var amountStr string
	var destAddr string

	for k, v := range rawTx.To {
		amountStr = v
		destAddr = k
		break
	}

	amountSent, _ := decimal.NewFromString(amountStr)
	//检查目标账户是否存在
	if !decoder.wm.Blockscanner.AccountExists(destAddr) {
		return openwallet.Errorf(10000, "account not exists")

	}

	for _, addr := range addresses {
		resp, _ := decoder.wm.ContractDecoder.GetTokenBalanceByAddress(rawTx.Coin.Contract, addr.Address)

		if len(resp) == 0 {
			continue
		}
		balanceAmount, _ := decimal.NewFromString(resp[0].Balance.Balance)
		if err != nil {
			continue
		}

		//总消耗数量 = 转账数量 + 手续费
		totalAmount := amountSent

		//余额不足查找下一个地址
		if balanceAmount.Cmp(totalAmount) < 0 {
			continue
		}

		//只要找到一个合适使用的地址余额就停止遍历
		findAddrBalance = NewAddrBalance(resp[0].Balance)
		break
	}

	if findAddrBalance == nil {
		return openwallet.Errorf(openwallet.ErrInsufficientBalanceOfAccount, "all address's balance of account is not enough, an address required to retain at least %s algos", retainAmount)
	}

	//最后创建交易单
	err = decoder.createRawTransaction(
		wrapper,
		rawTx,
		findAddrBalance,
	)
	if err != nil {
		return err
	}

	return nil

}

//SignRawTransaction 签名交易单
func (decoder *TransactionDecoder) SignRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	var (
		txn xdr.TransactionEnvelope
	)

	if rawTx.Signatures == nil || len(rawTx.Signatures) == 0 {
		return openwallet.Errorf(openwallet.ErrCreateRawTransactionFailed, "transaction signature is empty")
	}

	key, err := wrapper.HDKey()
	if err != nil {
		return err
	}

	keySignatures := rawTx.Signatures[rawTx.Account.AccountID]
	if keySignatures != nil {
		for _, keySignature := range keySignatures {

			childKey, err := key.DerivedKeyWithPath(keySignature.Address.HDPath, keySignature.EccType)
			keyBytes, err := childKey.GetPrivateKeyBytes()
			if err != nil {
				return err
			}

			publicKey, _ := hex.DecodeString(keySignature.Address.PublicKey)

			msg, err := hex.DecodeString(keySignature.Message)
			if err != nil {
				return openwallet.Errorf(openwallet.ErrCreateRawTransactionFailed, "decoder transaction hash failed, unexpected err: %v", err)
			}

			sig, err := txsigner.Default.SignTransactionHash(msg, keyBytes, keySignature.EccType)
			if err != nil {
				return openwallet.Errorf(openwallet.ErrCreateRawTransactionFailed, "sign transaction hash failed, unexpected err: %v", err)
			}

			rawXdr, _ := hex.DecodeString(rawTx.RawHex)
			err = txn.UnmarshalBinary(rawXdr)
			if err != nil {
				return openwallet.Errorf(openwallet.ErrCreateRawTransactionFailed, "raw tx Unmarshal failed=%s", err)
			}

			decoder.wm.Log.Debugf("message: %s", hex.EncodeToString(msg))
			decoder.wm.Log.Debugf("publicKey: %s", hex.EncodeToString(publicKey))
			decoder.wm.Log.Debugf("nonce : %s", keySignature.Nonce)
			decoder.wm.Log.Debugf("signature: %s", hex.EncodeToString(sig))

			keySignature.Signature = hex.EncodeToString(sig)
		}
	}

	decoder.wm.Log.Info("transaction hash sign success")

	rawTx.Signatures[rawTx.Account.AccountID] = keySignatures

	return nil
}

//VerifyRawTransaction 验证交易单，验证交易单并返回加入签名后的交易单
func (decoder *TransactionDecoder) VerifyRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	var (
		txn xdr.TransactionEnvelope
	)

	if rawTx.Signatures == nil || len(rawTx.Signatures) == 0 {
		//this.wm.Log.Std.Error("len of signatures error. ")
		return openwallet.Errorf(openwallet.ErrVerifyRawTransactionFailed, "transaction signature is empty")
	}

	//支持多重签名
	for accountID, keySignatures := range rawTx.Signatures {
		decoder.wm.Log.Debug("accountID Signatures:", accountID)
		for _, keySignature := range keySignatures {

			messsage, _ := hex.DecodeString(keySignature.Message)
			signature, _ := hex.DecodeString(keySignature.Signature)
			publicKey, _ := hex.DecodeString(keySignature.Address.PublicKey)

			// 验证签名
			ret := owcrypt.Verify(publicKey, nil, 0, messsage, uint16(len(messsage)), signature, keySignature.EccType)
			if ret != owcrypt.SUCCESS {
				return openwallet.Errorf(openwallet.ErrVerifyRawTransactionFailed, "transaction verify failed")
			}

			rawXdr, _ := hex.DecodeString(rawTx.RawHex)
			err := txn.UnmarshalBinary(rawXdr)
			if err != nil {
				return openwallet.Errorf(openwallet.ErrVerifyRawTransactionFailed, "raw tx Unmarshal failed=%s", err)
			}
			kp, _ := keypair.Parse(keySignature.Address.Address)

			xdrSig := xdr.DecoratedSignature{
				Hint:      xdr.SignatureHint(kp.Hint()),
				Signature: xdr.Signature(signature),
			}
			txn.Signatures = append(txn.Signatures, xdrSig)

			// Encode the SignedTxn
			rawTx.IsCompleted = true
			txXdr, err := txn.MarshalBinary()
			if err != nil {
				return err
			}
			txBase64 := base64.StdEncoding.EncodeToString(txXdr)
			rawTx.RawHex = txBase64
			break

		}
	}

	return nil
}

//SendRawTransaction 广播交易单
func (decoder *TransactionDecoder) SubmitRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) (*openwallet.Transaction, error) {
	resp, err := decoder.wm.oldTclient.SubmitTransaction(rawTx.RawHex)
	if err != nil {
		return nil, err
	}
	if resp.Hash == "" {
		return nil, errors.New("submit transaction fail")
	}
	log.Infof("Transaction [%s] submitted to the network successfully.", resp.Hash)

	rawTx.TxID = resp.Hash
	rawTx.IsSubmit = true

	decimals := decoder.wm.Decimal()

	//记录一个交易单
	tx := &openwallet.Transaction{
		From:       rawTx.TxFrom,
		To:         rawTx.TxTo,
		Amount:     rawTx.TxAmount,
		Coin:       rawTx.Coin,
		TxID:       rawTx.TxID,
		Decimal:    decimals,
		AccountID:  rawTx.Account.AccountID,
		Fees:       rawTx.Fees,
		SubmitTime: time.Now().Unix(),
	}

	tx.WxID = openwallet.GenTransactionWxID(tx)

	return tx, nil
}

//GetRawTransactionFeeRate 获取交易单的费率
func (decoder *TransactionDecoder) GetRawTransactionFeeRate() (feeRate string, unit string, err error) {
	suggestedFeeRate, err := decoder.wm.client.SuggestedFee()
	return strconv.FormatUint(suggestedFeeRate.Fee, 10), decoder.wm.Config.Symbol, err
}

//汇总币种
func (decoder *TransactionDecoder) CreateSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransaction, error) {
	if !sumRawTx.Coin.IsContract {
		return decoder.CreateSimpleSummaryRawTransaction(wrapper, sumRawTx)
	}
	return decoder.CreateAssetsSummaryRawTransaction(wrapper, sumRawTx)
}

//CreateSummaryRawTransaction 创建RIA汇总交易
func (decoder *TransactionDecoder) CreateSimpleSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransaction, error) {

	var (
		rawTxArray         = make([]*openwallet.RawTransaction, 0)
		accountID          = sumRawTx.Account.AccountID
		minTransfer, _     = decimal.NewFromString(decoder.wm.Config.AddressRetainAmount)
		retainedBalance, _ = decimal.NewFromString(decoder.wm.Config.AddressRetainAmount)
		estimateFees, _    = decimal.NewFromString("0.001")
	)

	if minTransfer.Cmp(retainedBalance) < 0 {
		return nil, openwallet.Errorf(openwallet.ErrInsufficientBalanceOfAccount, "mini transfer amount must be greater than address retained balance")
	}

	//获取wallet
	addresses, err := wrapper.GetAddressList(sumRawTx.AddressStartIndex, sumRawTx.AddressLimit,
		"AccountID", sumRawTx.Account.AccountID)
	if err != nil {
		return nil, err
	}

	if len(addresses) == 0 {
		return nil, openwallet.Errorf(openwallet.ErrCreateRawTransactionFailed, "[%s] have not addresses", accountID)
	}

	for _, addr := range addresses {

		balance, _ := decoder.wm.Blockscanner.GetBalanceByAddress(addr.Address)
		if len(balance) == 0 {
			continue
		}

		//检查余额是否超过最低转账
		addrBalance_BI, _ := decimal.NewFromString(balance[0].Balance)

		if addrBalance_BI.Cmp(minTransfer) < 0 || addrBalance_BI.Cmp(decimal.Zero) <= 0 {
			continue
		}
		//计算汇总数量 = 余额 - 保留余额 - 减去手续费
		summaryAmount := addrBalance_BI.Sub(retainedBalance).Sub(estimateFees)

		if addrBalance_BI.Cmp(decimal.Zero) <= 0 {
			continue
		}

		decoder.wm.Log.Debugf("balance: %v", addrBalance_BI.String())
		decoder.wm.Log.Debugf("fees: %v", estimateFees)
		decoder.wm.Log.Debugf("sumAmount: %v", summaryAmount)

		//创建一笔交易单
		rawTx := &openwallet.RawTransaction{
			Coin:    sumRawTx.Coin,
			Account: sumRawTx.Account,
			To: map[string]string{
				sumRawTx.SummaryAddress: summaryAmount.String(),
			},
			Required: 1,
		}

		findAddrBalance := NewAddrBalance(balance[0])

		createErr := decoder.createRawTransaction(
			wrapper,
			rawTx,
			findAddrBalance,
		)
		if createErr != nil {
			return nil, createErr
		}

		//创建成功，添加到队列
		rawTxArray = append(rawTxArray, rawTx)

	}

	return rawTxArray, nil
}

//汇总资产
func (decoder *TransactionDecoder) CreateAssetsSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransaction, error) {

	var (
		rawTxArray         = make([]*openwallet.RawTransaction, 0)
		accountID          = sumRawTx.Account.AccountID
		minTransfer, _     = decimal.NewFromString(decoder.wm.Config.AddressRetainAmount)
		retainedBalance, _ = decimal.NewFromString(decoder.wm.Config.AddressRetainAmount)
		estimateFees, _    = decimal.NewFromString("0.001")
	)

	if minTransfer.Cmp(retainedBalance) < 0 {
		return nil, openwallet.Errorf(openwallet.ErrInsufficientBalanceOfAccount, "mini transfer amount must be greater than address retained balance")
	}

	//获取wallet
	addresses, err := wrapper.GetAddressList(sumRawTx.AddressStartIndex, sumRawTx.AddressLimit,
		"AccountID", sumRawTx.Account.AccountID)
	if err != nil {
		return nil, err
	}

	if len(addresses) == 0 {
		return nil, openwallet.Errorf(openwallet.ErrCreateRawTransactionFailed, "[%s] have not addresses", accountID)
	}

	for _, addr := range addresses {

		//获取assets余额
		resp, _ := decoder.wm.ContractDecoder.GetTokenBalanceByAddress(sumRawTx.Coin.Contract, addr.Address)

		if len(resp) == 0 {
			continue
		}
		assetsBalance := resp[0]
		assetsBalanceAmount, _ := decimal.NewFromString(assetsBalance.Balance.Balance)
		if err != nil {
			continue
		}
		//余额是否大于0
		if assetsBalanceAmount.Cmp(decimal.Zero) <= 0 {
			continue
		}

		decoder.wm.Log.Debugf("tokenCoinBalance: %v", assetsBalanceAmount.String())
		decoder.wm.Log.Debugf("fees: %v", estimateFees)
		decoder.wm.Log.Debugf("sumAmount: %v", assetsBalanceAmount)

		//创建一笔交易单
		rawTx := &openwallet.RawTransaction{
			Coin:    sumRawTx.Coin,
			Account: sumRawTx.Account,
			To: map[string]string{
				sumRawTx.SummaryAddress: assetsBalanceAmount.String(),
			},

			Required: 1,
		}

		findAddrBalance := NewAddrAssetsBalance(assetsBalance)

		//查询主币交易费是否足够
		mainCoinBalance, _ := decoder.wm.Blockscanner.GetBalanceByAddress(addr.Address)
		if len(mainCoinBalance) == 0 {
			continue
		}

		//检查主币余额是否够交易费和超过最低转账限额
		mainCoinAddrBalance_BI, _ := decimal.NewFromString(mainCoinBalance[0].Balance)

		if mainCoinAddrBalance_BI.Cmp(minTransfer) < 0 || mainCoinAddrBalance_BI.Cmp(decimal.Zero) <= 0 {
			continue
		}

		//计算剩下主币数量
		leftMainCoinAmount := mainCoinAddrBalance_BI.Sub(retainedBalance).Sub(estimateFees)

		//检查主币 是否够转账交易费
		if leftMainCoinAmount.Cmp(decimal.Zero) < 0 {
			return nil, openwallet.Errorf(openwallet.ErrInsufficientFees, "main coin fee Insufficient ")
		}

		createErr := decoder.createRawTransaction(
			wrapper,
			rawTx,
			findAddrBalance,
		)

		if createErr != nil {
			return nil, createErr
		}

		//创建成功，添加到队列
		rawTxArray = append(rawTxArray, rawTx)

	}

	return rawTxArray, nil
}

func NewSimpleAccount2(accountID string, sequence int64) txnbuild.SimpleAccount {
	return txnbuild.SimpleAccount{accountID, sequence}
}

//createRawTransaction
func (decoder *TransactionDecoder) createRawTransaction(
	wrapper openwallet.WalletDAI,
	rawTx *openwallet.RawTransaction,
	addrBalance *AddrBalance,
) error {

	var (
		accountTotalSent = decimal.Zero
		txFrom           = make([]string, 0)
		txTo             = make([]string, 0)
		keySignList      = make([]*openwallet.KeySignature, 0)
		amountStr        string
		destination      string
	)

	decimals := int32(0)
	if rawTx.Coin.IsContract {
		decimals = int32(rawTx.Coin.Contract.Decimals)
	} else {
		decimals = decoder.wm.Decimal()
	}

	for k, v := range rawTx.To {
		destination = k
		amountStr = v
		break
	}

	//计算账户的实际转账amount
	accountTotalSentAddresses, findErr := wrapper.GetAddressList(0, -1, "AccountID", rawTx.Account.AccountID, "Address", destination)
	if findErr != nil || len(accountTotalSentAddresses) == 0 {
		amountDec, _ := decimal.NewFromString(amountStr)
		accountTotalSent = accountTotalSent.Add(amountDec)
	}

	addr, err := wrapper.GetAddress(addrBalance.Address)
	if err != nil {
		return err
	}
	currentSeq := decoder.wm.Blockscanner.getAccountSeq(addrBalance.Address)
	txSourceAccount := NewSimpleAccount2(addrBalance.Address, currentSeq)

	//存在直接转账
	//txn, err := transaction.MakePaymentTxn(addrBalance.Address, destination, suggestedParams.Fee, amount.Uint64(), suggestedParams.LastRound, suggestedParams.LastRound+validRounds, []byte(""), "", suggestedParams.GenesisID, suggestedParams.GenesisHash)
	var payment txnbuild.Payment
	if !rawTx.Coin.IsContract { //RIA 主币
		payment = txnbuild.Payment{
			Destination: destination,
			Amount:      amountStr,
			Asset:       txnbuild.NativeAsset{},
		}
	} else { //资产Assets
		payment = txnbuild.Payment{
			Destination: destination,
			Amount:      amountStr,
			Asset:       txnbuild.CreditAsset{Code: strings.ToUpper(rawTx.Coin.Contract.Token), Issuer: rawTx.Coin.Contract.Address},
		}
	}
	var memoText string
	if len(rawTx.ExtParam) != 0 {
		memoText = rawTx.GetExtParam().Get("memo").String()
	}
	var tx txnbuild.Transaction
	if len(memoText) != 0 {
		tx = txnbuild.Transaction{
			BaseFee:       10000,
			SourceAccount: &txSourceAccount,
			Operations:    []txnbuild.Operation{&payment},
			Timebounds:    txnbuild.NewInfiniteTimeout(),
			Network:       decoder.wm.Config.Network,
			Memo:          txnbuild.MemoText(memoText),
		}
	} else {
		tx = txnbuild.Transaction{
			BaseFee:       10000,
			SourceAccount: &txSourceAccount,
			Operations:    []txnbuild.Operation{&payment},
			Timebounds:    txnbuild.NewInfiniteTimeout(),
			Network:       decoder.wm.Config.Network,
		}
	}
	err = tx.Build()
	if err != nil {
		return err
	}

	if tx.XdrEnvelope == nil {
		tx.XdrEnvelope = &xdr.TransactionEnvelope{}
		tx.XdrEnvelope.Tx = tx.XdrTransaction
	}
	hash, err := tx.Hash()
	if err != nil {
		return err
	}

	txXdr, _ := tx.XdrEnvelope.MarshalBinary()
	rawTx.RawHex = hex.EncodeToString(txXdr)
	if rawTx.Signatures == nil {
		rawTx.Signatures = make(map[string][]*openwallet.KeySignature)
	}

	signature := openwallet.KeySignature{
		EccType: decoder.wm.Config.CurveType,
		Address: addr,
		Message: hex.EncodeToString(hash[:]),
	}
	keySignList = append(keySignList, &signature)

	//固定费用
	feesAmount, _ := decimal.NewFromString("0.001")
	//主币加上交易费
	if !rawTx.Coin.IsContract {
		accountTotalSent = accountTotalSent.Add(feesAmount)
	}
	accountTotalSent = decimal.Zero.Sub(accountTotalSent)

	rawTx.Signatures[rawTx.Account.AccountID] = keySignList
	rawTx.FeeRate = feesAmount.String()
	rawTx.Fees = feesAmount.String()
	rawTx.IsBuilt = true
	rawTx.TxAmount = accountTotalSent.StringFixed(decimals)
	rawTx.TxFrom = txFrom
	rawTx.TxTo = txTo

	return nil
}

//CreateSummaryRawTransactionWithError 创建汇总交易，返回能原始交易单数组（包含带错误的原始交易单）
func (decoder *TransactionDecoder) CreateSummaryRawTransactionWithError(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransactionWithError, error) {
	raTxWithErr := make([]*openwallet.RawTransactionWithError, 0)
	rawTxs, err := decoder.CreateSummaryRawTransaction(wrapper, sumRawTx)
	if err != nil {
		return nil, err
	}
	for _, tx := range rawTxs {
		raTxWithErr = append(raTxWithErr, &openwallet.RawTransactionWithError{
			RawTx: tx,
			Error: nil,
		})
	}
	return raTxWithErr, nil
}
