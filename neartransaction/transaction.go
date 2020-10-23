package neartransaction

import (
	"encoding/hex"
	"github.com/Assetsadapter/near-adapter/near"
	"github.com/blocktree/openwallet/common"
	"github.com/mr-tron/base58"
	"math/big"
)

// Transaction struct
type Transaction struct {
	SignerID   string
	PublicKey  []byte
	Nonce      uint64
	ReceiverID string
	BlockHash  []byte
	Signature  string
	Actions    []Transfer
}

type Transfer struct {
	Deposit *big.Int
}

func NewTransaction(from, to, refBlockHash, transferAmount string, nonce uint64) (*Transaction, error) {
	tx := Transaction{}
	tx.SignerID = from
	singerPubKey, err := hex.DecodeString(from)
	if err != nil {
		return nil, err
	}
	tx.ReceiverID = to
	tx.PublicKey = singerPubKey
	tx.Nonce = nonce
	tx.BlockHash, err = base58.Decode(refBlockHash)
	if err != nil {
		return nil, err
	}
	amount := common.StringNumToBigIntWithExp(transferAmount, near.Decimal)
	if err != nil {
		return nil, err
	}
	transferAction := Transfer{Deposit: amount}
	tx.Actions = append(tx.Actions, transferAction)
	return &tx, nil
}

//序列化
//https://github.com/near/near-api-js/blob/79a336931943384708a644d9ca4ce327a1daec07/src/transaction.ts#L148
//[Transaction, { kind: 'struct', fields: [
//['signerId', 'string'],
//['publicKey', PublicKey],
//['nonce', 'u64'],
//['receiverId', 'string'],
//['blockHash', [32]],
//['actions', [Action]]
//]}],
//[Transfer, { kind: 'struct', fields: [
//['deposit', 'u128']
//]}],
func (tx Transaction) serialize() (string, error) {
	bytesData := []byte{}
	//signerId
	bytesData = append(bytesData, uint32ToLittleEndianBytes(uint32(len(tx.SignerID)))...)
	bytesData = append(bytesData, []byte(tx.SignerID)...)

	//publicKey
	bytesData = append(bytesData, byte(0))
	bytesData = append(bytesData, reverseBytes(tx.PublicKey)...)

	//nonce
	bytesData = append(bytesData, uint64ToLittleEndianBytes(uint64(len(tx.SignerID)))...)

	//receiverId
	bytesData = append(bytesData, uint32ToLittleEndianBytes(uint32(len(tx.ReceiverID)))...)
	bytesData = append(bytesData, []byte(tx.ReceiverID)...)

	//blockHash
	bytesData = append(bytesData, tx.BlockHash...)

	//actions
	for _, action := range tx.Actions {
		bytesData = append(bytesData, byte(3))
		bytesData = append(bytesData, action.Deposit.Bytes()...)
	}

	return hex.EncodeToString(bytesData), nil

}
