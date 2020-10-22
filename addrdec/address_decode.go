package addrdec

import (
	"crypto/sha512"
	"encoding/base32"
	"encoding/hex"
	"errors"
)

const DigestSize = sha512.Size256
const PublicKeySize = 32
const ChecksumLength = 2

// Digest represents a 32-byte value holding the 256-bit Hash digest.
type Digest [DigestSize]byte
type PublicKey [PublicKeySize]byte

type (
	Address Digest
)

type ChecksumAddress struct {
	shortAddress Address
	checksum     []byte
}

// AddressDecoderV2
type AddressDecoderV2 struct {
}

var (
	Default = AddressDecoderV2{}
)

//NewAddressDecoder 地址解析器
func NewAddressDecoderV2() *AddressDecoderV2 {
	decoder := AddressDecoderV2{}
	return &decoder
}

var (
	ErrorInvalidHashLength = errors.New("Invalid hash length!")
	ErrorInvalidAddress    = errors.New("Invalid address!")
)

//AddressEncode encode address bytes
func (dec *AddressDecoderV2) AddressEncode(address []byte) (string, error) {
	//var pk PublicKey
	//
	//if len(pk) != len(address) {
	//	return "", ErrorInvalidHashLength
	//}

	//for i := range pk {
	//	pk[i] = address[i]
	//}
	return hex.EncodeToString(address), nil
}

var prefix = []byte{0x30}

// String returns a string representation of ChecksumAddress
func (addr *ChecksumAddress) String() string {
	var addrWithChecksum []byte
	addrWithChecksum = append(prefix, addr.shortAddress[:]...)
	addrWithChecksum = append(addrWithChecksum, addr.checksum...)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(addrWithChecksum)
}
