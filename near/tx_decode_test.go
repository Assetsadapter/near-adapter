package near

//
//import (
//	"encoding/hex"
//	"github.com/Assetsadapter/near-adapter/txsigner"
//	"github.com/blocktree/go-owcrypt"
//	"github.com/blocktree/openwallet/log"
//	hClient "github.com/stellar/go/clients/horizonclient"
//	"github.com/stellar/go/keypair"
//	"github.com/stellar/go/txnbuild"
//	"github.com/stellar/go/xdr"
//	oldhClient "github.com/triamnetwork/triam-horizon/clients/horizon"
//	"net/http"
//	"strconv"
//	"testing"
//)
//
////获取下一个账户的seq
//func getAccountSeq(account string) int64 {
//	client := hClient.Client{HorizonURL: "https://testnet-horizon.triamnetwork.com/", HTTP: http.DefaultClient}
//	accountDetail, _ := client.AccountDetail(hClient.AccountRequest{AccountID: account})
//	seq, _ := strconv.ParseInt(accountDetail.Sequence, 10, 64)
//	return seq
//}
//
////获取下一个账户的seq
//func TestExistsAccount(t *testing.T) {
//	client := hClient.Client{HorizonURL: "https://testnet-horizon.triamnetwork.com/", HTTP: http.DefaultClient}
//	accountDetail, err := client.AccountDetail(hClient.AccountRequest{AccountID: "GDJQNT7SMTRZPQBVB46R54J454F2ZCKUJ37YVW6KYD2P2Y2FPLK5ANGQ1"})
//	if err != nil {
//		panic(err)
//	}
//	log.Info(accountDetail)
//}
//
////发送交易
//func postTransaction(tx txnbuild.Transaction) string {
//	client := oldhClient.Client{URL: "https://testnet-horizon.triamnetwork.com", HTTP: http.DefaultClient}
//	txBase64, err := tx.Base64()
//	log.Info(txBase64)
//	if err != nil {
//		panic(err)
//	}
//	a, err := client.SubmitTransaction(txBase64)
//	if err != nil {
//		panic(err)
//	}
//	log.Info(a.Hash)
//	txId := a.Hash
//	return txId
//}
//
////测试创建交易
//func TestBuildPayTransaction(t *testing.T) {
//	privateKey, _ := hex.DecodeString("407f243d382e6754b14307efcde040b6f88904973e1ef306d93743d098d0f945")
//	nexSeq := getAccountSeq("GDJQNT7SMTRZPQBVB46R54J454F2ZCKUJ37YVW6KYD2P2Y2FPLK5ANGQ")
//	txSourceAccount := NewSimpleAccount("GDJQNT7SMTRZPQBVB46R54J454F2ZCKUJ37YVW6KYD2P2Y2FPLK5ANGQ", nexSeq)
//
//	payment := txnbuild.Payment{
//		Destination: "GD2FAPCGEX5L63FENNSN6ZZPIL7NEHVPV2NCTGCALY7HG7KHLDW3QOMM",
//		Amount:      "0.234",
//		Asset:       txnbuild.NativeAsset{},
//	}
//
//	tx := txnbuild.Transaction{
//		BaseFee:       10000,
//		SourceAccount: &txSourceAccount,
//		Operations:    []txnbuild.Operation{&payment},
//		Timebounds:    txnbuild.NewInfiniteTimeout(),
//		Network:       "SAAK5654--ARM-NETWORK--BHC3SQOHPO2GGI--BY-B.A.P--CNEMJQCWPTA--RUBY-AND-BLOCKCHAIN--3KECMPY5L7W--THANKYOU-CS--S542ZHDVHLFV",
//	}
//	tx.Build()
//	if tx.XdrEnvelope == nil {
//		tx.XdrEnvelope = &xdr.TransactionEnvelope{}
//		tx.XdrEnvelope.Tx = tx.XdrTransaction
//	}
//	hash, err := tx.Hash()
//	if err != nil {
//		panic(err)
//	}
//	log.Info("hash=", hash)
//	sig, err := txsigner.Default.SignTransactionHash(hash[:], privateKey, owcrypt.ECC_CURVE_ED25519)
//	log.Info("sig=", sig, "len=", len(sig))
//	kp, _ := keypair.Parse("GDJQNT7SMTRZPQBVB46R54J454F2ZCKUJ37YVW6KYD2P2Y2FPLK5ANGQ")
//	xdrSig := xdr.DecoratedSignature{
//		Hint:      xdr.SignatureHint(kp.Hint()),
//		Signature: xdr.Signature(sig),
//	}
//	tx.XdrEnvelope.Signatures = append(tx.XdrEnvelope.Signatures, xdrSig)
//
//	postTransaction(tx)
//}
//
////测试创建change trust交易
//func TestBuildTrustTransaction(t *testing.T) {
//	privateKey, _ := hex.DecodeString("687c53739e7092a2bb7e7316c0bb91173b204da07a558b51a84cb7779cd0f945")
//	nexSeq := getAccountSeq("GA5M4ILYIQS3QRK4OBFZCUXOQ37SKO3DWZBK7K5CDGRY6UBOHLXJWKVX")
//	txSourceAccount := NewSimpleAccount("GA5M4ILYIQS3QRK4OBFZCUXOQ37SKO3DWZBK7K5CDGRY6UBOHLXJWKVX", nexSeq)
//
//	changeTrust := txnbuild.ChangeTrust{
//		Line: txnbuild.CreditAsset{"WGX", "GB6UCOSZDP45XVB35KA7WH2LKB6IE7H7SZKKAWEHT3G4IKBWFMOWCE23"},
//	}
//
//	tx := txnbuild.Transaction{
//		BaseFee:       10000,
//		SourceAccount: &txSourceAccount,
//		Operations:    []txnbuild.Operation{&changeTrust},
//		Timebounds:    txnbuild.NewInfiniteTimeout(),
//		Network:       "SAAK5654--ARM-NETWORK--BHC3SQOHPO2GGI--BY-B.A.P--CNEMJQCWPTA--RUBY-AND-BLOCKCHAIN--3KECMPY5L7W--THANKYOU-CS--S542ZHDVHLFV",
//	}
//	tx.Build()
//	if tx.XdrEnvelope == nil {
//		tx.XdrEnvelope = &xdr.TransactionEnvelope{}
//		tx.XdrEnvelope.Tx = tx.XdrTransaction
//	}
//	hash, err := tx.Hash()
//	if err != nil {
//		panic(err)
//	}
//	log.Info("hash=", hash)
//	sig, err := txsigner.Default.SignTransactionHash(hash[:], privateKey, owcrypt.ECC_CURVE_ED25519)
//	log.Info("sig=", sig, "len=", len(sig))
//	kp, _ := keypair.Parse("GA5M4ILYIQS3QRK4OBFZCUXOQ37SKO3DWZBK7K5CDGRY6UBOHLXJWKVX")
//	xdrSig := xdr.DecoratedSignature{
//		Hint:      xdr.SignatureHint(kp.Hint()),
//		Signature: xdr.Signature(sig),
//	}
//	tx.XdrEnvelope.Signatures = append(tx.XdrEnvelope.Signatures, xdrSig)
//	//tx.XdrEnvelope.postTransaction(tx)
//}
