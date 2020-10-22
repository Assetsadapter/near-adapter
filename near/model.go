package near

import (
	"fmt"
	"github.com/algorand/go-algorand-sdk/client/algod/models"
	"github.com/blocktree/openwallet/common"
	"github.com/blocktree/openwallet/crypto"
	"github.com/blocktree/openwallet/openwallet"
	hProtocol "github.com/stellar/go/protocols/horizon"
)

func NewAddrBalance(b *openwallet.Balance) *AddrBalance {
	obj := AddrBalance{}
	obj.Address = b.Address

	obj.Balance = b.Balance

	return &obj
}

func NewAddrAssetsBalance(b *openwallet.TokenBalance) *AddrBalance {
	obj := AddrBalance{}
	obj.Address = b.Balance.Address

	obj.Balance = b.Balance.Balance

	return &obj
}

type Block struct {
	Hash            string
	PrevBlockHash   string
	Time            int64
	Height          uint64
	TransactionsCnt uint64
	Transactions    []string
}

func NewBlock(block hProtocol.Ledger) *Block {

	obj := Block{}
	//解析json
	obj.Hash = block.Hash
	obj.PrevBlockHash = block.PrevHash
	obj.Height = uint64(block.Sequence)
	return &obj

}

//BlockHeader 区块链头
func (b *Block) BlockHeader(symbol string) *openwallet.BlockHeader {

	obj := openwallet.BlockHeader{}
	//解析json
	obj.Hash = b.Hash
	//obj.Confirmations = b.Confirmations
	//obj.Merkleroot = b.TransactionMerkleRoot
	obj.Previousblockhash = b.PrevBlockHash
	obj.Height = b.Height
	// obj.Version = uint64(b.Version)
	obj.Time = uint64(obj.Time)
	obj.Symbol = symbol

	return &obj
}

type Transaction struct {
	Type        string `json:"type"`
	BlockHash   string
	BlockHeight uint64
	BlockTime   int64
	GenesisID   string                         `json:"genesisID"`
	Fee         uint64                         `json:"fee"`
	TxID        string                         `json:"tx"`
	From        string                         `json:"from"`
	Note        []byte                         `json:"note"`
	Payment     *models.PaymentTransactionType `json:"payment,omitempty"`
}

func NewTransaction(tx models.Transaction) *Transaction {
	obj := Transaction{}
	obj.Type = string(tx.Type)
	obj.BlockHeight = tx.ConfirmedRound
	obj.GenesisID = tx.GenesisID
	obj.Fee = tx.Fee
	obj.TxID = tx.TxID
	obj.From = tx.From
	obj.Payment = tx.Payment
	return &obj
}

//UnscanRecords 扫描失败的区块及交易
type UnscanRecord struct {
	ID          string `storm:"id"` // primary key
	BlockHeight uint64
	TxID        string
	Reason      string
}

func NewUnscanRecord(height uint64, txID, reason string) *UnscanRecord {
	obj := UnscanRecord{}
	obj.BlockHeight = height
	obj.TxID = txID
	obj.Reason = reason
	obj.ID = common.Bytes2Hex(crypto.SHA256([]byte(fmt.Sprintf("%d_%s", height, txID))))
	return &obj
}
