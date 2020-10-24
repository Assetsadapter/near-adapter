package neartransaction

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"github.com/blocktree/openwallet/common"
	"github.com/juju/errors"
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
	Signature  []byte
	Actions    []Transfer
	RawTxHex   string
	RawTxByte  []byte
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
	amount := common.StringNumToBigIntWithExp(transferAmount, 24)
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
func (tx *Transaction) Serialize() (string, string, error) {
	bytesData := []byte{}
	//signerId
	bytesData = append(bytesData, uint32ToLittleEndianBytes(uint32(len(tx.SignerID)))...)
	bytesData = append(bytesData, []byte(tx.SignerID)...)

	//publicKey
	bytesData = append(bytesData, byte(0))
	bytesData = append(bytesData, tx.PublicKey...)

	//nonce
	bytesData = append(bytesData, uint64ToLittleEndianBytes(tx.Nonce)...)

	//receiverId
	bytesData = append(bytesData, uint32ToLittleEndianBytes(uint32(len(tx.ReceiverID)))...)
	bytesData = append(bytesData, []byte(tx.ReceiverID)...)

	//blockHash
	bytesData = append(bytesData, tx.BlockHash...)

	//actions
	bytesData = append(bytesData, uint32ToLittleEndianBytes(uint32(len(tx.Actions)))...)
	for _, action := range tx.Actions {
		bytesData = append(bytesData, byte(3))
		amountOriginBytes := reverseBytes(action.Deposit.Bytes())
		amountResult := common.RightPadBytes(amountOriginBytes, 16)
		bytesData = append(bytesData, amountResult...)
	}
	if len(tx.Signature) > 0 {
		bytesData = append(bytesData, byte(0))
		bytesData = append(bytesData, tx.Signature...)
	}

	rawTxHex := hex.EncodeToString(bytesData)
	tx.RawTxHex = rawTxHex
	tx.RawTxByte = bytesData
	hash, err := tx.digest()
	if err != nil {
		return "", "", err
	}
	return rawTxHex, hash, nil
}

func (tx Transaction) digest() (string, error) {
	var msgBuffer bytes.Buffer
	if _, err := msgBuffer.Write(tx.RawTxByte); err != nil {
		return "", errors.Annotate(err, "Write [trx]")
	}
	digest := sha256.Sum256(msgBuffer.Bytes())
	return hex.EncodeToString(digest[:]), nil
}
