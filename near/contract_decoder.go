package near

import (
	"errors"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
	"math/big"
)

type toeknDecoder struct {
	*openwallet.SmartContractDecoderBase
	wm *WalletManager
}

type AddrBalance struct {
	Address      string
	Balance      string
	TokenBalance string
	Index        int
}

func (this *AddrBalance) GetAddress() string {
	return this.Address
}

func (this *AddrBalance) ValidTokenBalance() bool {
	if this.Balance == "" {
		return false
	}
	return true
}

type AddrBalanceInf interface {
	SetTokenBalance(b *big.Int)
	GetAddress() string
	ValidTokenBalance() bool
}

func (this *toeknDecoder) GetTokenBalanceByAddress(contract openwallet.SmartContract, address ...string) ([]*openwallet.TokenBalance, error) {
	threadControl := make(chan int, 20)
	defer close(threadControl)
	resultChan := make(chan *openwallet.TokenBalance, 1024)
	defer close(resultChan)
	done := make(chan int, 1)
	var tokenBalanceList []*openwallet.TokenBalance
	count := len(address)

	go func() {
		//		log.Debugf("in save thread.")
		for i := 0; i < count; i++ {
			balance := <-resultChan
			if balance != nil {
				tokenBalanceList = append(tokenBalanceList, balance)
			}
			//log.Debugf("got one balance.")
		}
		done <- 1
	}()

	queryBalance := func(address string) {
		threadControl <- 1
		var balance *openwallet.TokenBalance
		defer func() {
			resultChan <- balance
			<-threadControl
		}()

		//		log.Debugf("in query thread.")
		accountBalance, err := this.wm.Blockscanner.GetAssetBalance(address, contract.Token, contract.Address)
		if err != nil {
			log.Errorf("get address[%v] assets  balance failed, err=%v", address, err)
			return
		}

		balance = &openwallet.TokenBalance{
			Contract: &contract,
			Balance: &openwallet.Balance{
				Address:          address,
				Symbol:           contract.Symbol,
				Balance:          accountBalance,
				ConfirmBalance:   accountBalance,
				UnconfirmBalance: "0",
			},
		}
	}

	for i := range address {
		go queryBalance(address[i])
	}

	<-done

	if len(tokenBalanceList) != count {
		log.Error("unknown errors occurred .")
		return nil, errors.New("unknown errors occurred ")
	}
	return tokenBalanceList, nil
}
