package near

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/Assetsadapter/near-adapter/neartransaction"
	"github.com/Assetsadapter/near-adapter/txsigner"
	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/mr-tron/base58"
	"github.com/shopspring/decimal"
	"time"
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
	return decoder.CreateRawSimpleTransaction(wrapper, rawTx)
}

//CreateRawTransaction 创建交易单
func (decoder *TransactionDecoder) CreateRawSimpleTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	var (
		accountID       = rawTx.Account.AccountID
		estimateFees    = decimal.Zero
		findAddrBalance *AddrBalance
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
	for _, v := range rawTx.To {
		amountStr = v
		break
	}

	amountSent, _ := decimal.NewFromString(amountStr)

	//if len(rawTx.FeeRate) > 0 {
	//	estimateFees = common.StringNumToBigIntWithExp(rawTx.FeeRate, decimals)
	//} else {
	//	estimateFees = common.StringNumToBigIntWithExp(decoder.wm.Config.FixFees, decimals)
	//}
	gasPriceStr, err := decoder.wm.Blockscanner.GetGasPrice()
	if err != nil {
		return err
	}
	gasPrice, err := decimal.NewFromString(gasPriceStr)
	if err != nil {
		return err
	}
	estimateFees = gasPrice.Mul(decimal.New(424555062500*2, 1)).Div(decimal.New(1, Decimal))
	log.Info("estimateFees:", estimateFees)
	//Accounts must have enough tokens cover its storage.
	//Storage cost per byte is 0.0001 NEAR and an account with one access key must maintain a balance of at least 0.0182 NEAR. For more details, see
	//账户一般消耗182
	retainedBalance := decimal.NewFromFloat32(0.0182).Add(decimal.NewFromFloat32(182 * 0.0001))
	log.Info("retainedBalance:", retainedBalance)
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
		totalAmount = totalAmount.Add(retainedBalance)

		//余额不足查找下一个地址
		if balanceAmount.Cmp(totalAmount) < 0 {
			continue
		}

		//只要找到一个合适使用的地址余额就停止遍历
		findAddrBalance = NewAddrBalance(resp[0])
		break
	}

	if findAddrBalance == nil {
		return openwallet.Errorf(openwallet.ErrInsufficientBalanceOfAccount, "all address's balance of account is not enough, an address required to retain at least %s algos", retainedBalance)
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

			rawTxHex, err := hex.DecodeString(rawTx.RawHex)
			if err != nil {
				return err
			}
			nearTx := neartransaction.Transaction{}
			if err := json.Unmarshal(rawTxHex, &nearTx); err != nil {
				return openwallet.Errorf(openwallet.ErrCreateRawTransactionFailed, "sign transaction hash failed, unexpected err: %v", err)
			}
			nearTx.Signature = sig
			_, _, err = nearTx.Serialize()
			if err != nil {
				return openwallet.Errorf(openwallet.ErrCreateRawTransactionFailed, "sign transaction hash failed, unexpected err: %v", err)
			}
			rawTx.RawHex = hex.EncodeToString(nearTx.RawTxByte)
			if err != nil {
				return openwallet.Errorf(openwallet.ErrCreateRawTransactionFailed, "raw tx Unmarshal failed=%s", err)
			}
			decoder.wm.Log.Debugf("message: %s", hex.EncodeToString(msg))
			decoder.wm.Log.Debugf("publicKey: %s", hex.EncodeToString(publicKey))
			decoder.wm.Log.Errorf("privateKey: %s", base58.Encode(keyBytes))

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

			rawTxHex, err := hex.DecodeString(rawTx.RawHex)
			if err != nil {
				return openwallet.Errorf(openwallet.ErrVerifyRawTransactionFailed, "raw tx Unmarshal failed=%s", err)
			}

			txBase64 := base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(rawTxHex)
			rawTx.RawHex = txBase64
			break

		}
	}

	return nil
}

//SendRawTransaction 广播交易单
func (decoder *TransactionDecoder) SubmitRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) (*openwallet.Transaction, error) {
	param := []interface{}{rawTx.RawHex}
	result, err := decoder.wm.client.Call("broadcast_tx_commit", param)
	if err != nil {
		return nil, err
	}
	txId := result.Get("transaction.hash").String()
	if txId == "" {
		return nil, errors.New("submit transaction fail")
	}
	log.Infof("Transaction [%s] submitted to the network successfully.", txId)

	rawTx.TxID = txId
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

//汇总币种
func (decoder *TransactionDecoder) CreateSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransaction, error) {
	return decoder.CreateSimpleSummaryRawTransaction(wrapper, sumRawTx)
}

//CreateSummaryRawTransaction 创建RIA汇总交易
func (decoder *TransactionDecoder) CreateSimpleSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransaction, error) {

	var (
		rawTxArray         = make([]*openwallet.RawTransaction, 0)
		accountID          = sumRawTx.Account.AccountID
		minTransfer, _     = decimal.NewFromString(decoder.wm.Config.AddressRetainAmount)
		retainedBalance, _ = decimal.NewFromString(decoder.wm.Config.AddressRetainAmount)

		estimateFees = decimal.Zero
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
	gasPriceStr, err := decoder.wm.Blockscanner.GetGasPrice()
	if err != nil {
		return nil, err
	}
	gasPrice, err := decimal.NewFromString(gasPriceStr)
	if err != nil {
		return nil, err
	}
	estimateFees = gasPrice.Mul(decimal.New(424555062500*2, 1)).Div(decimal.New(1, Decimal))
	log.Info("estimateFees:", estimateFees)
	//Accounts must have enough tokens cover its storage.
	//Storage cost per byte is 0.0001 NEAR and an account with one access key must maintain a balance of at least 0.0182 NEAR. For more details, see
	//账户一般消耗182
	retainedBalance = decimal.NewFromFloat32(0.0182).Add(decimal.NewFromFloat32(182 * 0.0001))
	log.Info("retainedBalance:", retainedBalance)

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

		if summaryAmount.Cmp(decimal.Zero) <= 0 {
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

	decimals := decoder.wm.Decimal()
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
	accountNonce, err := decoder.wm.Blockscanner.GetAccountNonce(addrBalance.Address)
	if err != nil {
		return err
	}
	refBlockHash, err := decoder.wm.Blockscanner.GetLatestRefBlockHash()
	nearTx, err := neartransaction.NewTransaction(addrBalance.Address, destination, refBlockHash, amountStr, accountNonce+1)
	if err != nil {
		return err
	}

	_, hash, err := nearTx.Serialize()
	if err != nil {
		return err
	}
	var buf []byte

	if buf, err = json.Marshal(nearTx); err != nil {
		return err
	}
	rawTx.RawHex = hex.EncodeToString(buf)
	if rawTx.Signatures == nil {
		rawTx.Signatures = make(map[string][]*openwallet.KeySignature)
	}

	signature := openwallet.KeySignature{
		EccType: decoder.wm.Config.CurveType,
		Address: addr,
		Message: hash,
	}
	keySignList = append(keySignList, &signature)

	//固定费用
	feesAmount, _ := decimal.NewFromString("0")
	//主币加上交易费
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
