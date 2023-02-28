package service

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const (
	ValidateSignatureError int = -40001
	ParseXmlError          int = -40002
	ComputeSignatureError  int = -40003
	IllegalAesKey          int = -40004
	ValidateCorpidError    int = -40005
	EncryptAESError        int = -40006
	DecryptAESError        int = -40007
	IllegalBuffer          int = -40008
	EncodeBase64Error      int = -40009
	DecodeBase64Error      int = -40010
	GenXmlError            int = -40010
	ParseJsonError         int = -40012
	GenJsonError           int = -40013
	IllegalProtocolType    int = -40014
)

type ProtocolType int

const (
	XmlType ProtocolType = 1
)

type CryptError struct {
	ErrCode int
	ErrMsg  string
}

func NewCryptError(err_code int, err_msg string) *CryptError {
	return &CryptError{ErrCode: err_code, ErrMsg: err_msg}
}

type WXBizMsg4Recv struct {
	Tousername string `xml:"ToUserName"`
	Encrypt    string `xml:"Encrypt"`
	Agentid    string `xml:"AgentID"`
}

type CDATA struct {
	Value string `xml:",cdata"`
}

type WXBizMsg4Send struct {
	XMLName   xml.Name `xml:"xml"`
	Encrypt   CDATA    `xml:"Encrypt"`
	Signature CDATA    `xml:"MsgSignature"`
	Timestamp string   `xml:"TimeStamp"`
	Nonce     CDATA    `xml:"Nonce"`
}

func NewWXBizMsg4Send(encrypt, signature, timestamp, nonce string) *WXBizMsg4Send {
	return &WXBizMsg4Send{Encrypt: CDATA{Value: encrypt}, Signature: CDATA{Value: signature}, Timestamp: timestamp, Nonce: CDATA{Value: nonce}}
}

type ProtocolProcessor interface {
	parse(src_data []byte) (*WXBizMsg4Recv, *CryptError)
	serialize(msg_send *WXBizMsg4Send) ([]byte, *CryptError)
}

type WXBizMsgCrypt struct {
	token              string
	encoding_aeskey    string
	receiver_id        string
	protocol_processor ProtocolProcessor
}

type XmlProcessor struct {
}

func (self *XmlProcessor) parse(src_data []byte) (*WXBizMsg4Recv, *CryptError) {
	var msg4_recv WXBizMsg4Recv
	err := xml.Unmarshal(src_data, &msg4_recv)
	if nil != err {
		return nil, NewCryptError(ParseXmlError, "xml to msg fail")
	}
	return &msg4_recv, nil
}

func (self *XmlProcessor) serialize(msg4_send *WXBizMsg4Send) ([]byte, *CryptError) {
	xml_msg, err := xml.Marshal(msg4_send)
	if nil != err {
		return nil, NewCryptError(GenXmlError, err.Error())
	}
	return xml_msg, nil
}

func NewWXBizMsgCrypt(token, encoding_aeskey, receiver_id string, protocol_type ProtocolType) *WXBizMsgCrypt {
	var protocol_processor ProtocolProcessor
	if protocol_type != XmlType {
		panic("unsupport protocal")
	} else {
		protocol_processor = new(XmlProcessor)
	}

	return &WXBizMsgCrypt{token: token, encoding_aeskey: (encoding_aeskey + "="), receiver_id: receiver_id, protocol_processor: protocol_processor}
}

func (self *WXBizMsgCrypt) randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func (self *WXBizMsgCrypt) pKCS7Padding(plaintext string, block_size int) []byte {
	padding := block_size - (len(plaintext) % block_size)
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	var buffer bytes.Buffer
	buffer.WriteString(plaintext)
	buffer.Write(padtext)
	return buffer.Bytes()
}

func (self *WXBizMsgCrypt) pKCS7Unpadding(plaintext []byte, block_size int) ([]byte, *CryptError) {
	plaintext_len := len(plaintext)
	if nil == plaintext || plaintext_len == 0 {
		return nil, NewCryptError(DecryptAESError, "pKCS7Unpadding error nil or zero")
	}
	if plaintext_len%block_size != 0 {
		return nil, NewCryptError(DecryptAESError, "pKCS7Unpadding text not a multiple of the block size")
	}
	padding_len := int(plaintext[plaintext_len-1])
	return plaintext[:plaintext_len-padding_len], nil
}

func (self *WXBizMsgCrypt) cbcEncrypter(plaintext string) ([]byte, *CryptError) {
	aeskey, err := base64.StdEncoding.DecodeString(self.encoding_aeskey)
	if nil != err {
		return nil, NewCryptError(DecodeBase64Error, err.Error())
	}
	const block_size = 32
	pad_msg := self.pKCS7Padding(plaintext, block_size)

	block, err := aes.NewCipher(aeskey)
	if err != nil {
		return nil, NewCryptError(EncryptAESError, err.Error())
	}

	ciphertext := make([]byte, len(pad_msg))
	iv := aeskey[:aes.BlockSize]

	mode := cipher.NewCBCEncrypter(block, iv)

	mode.CryptBlocks(ciphertext, pad_msg)
	base64_msg := make([]byte, base64.StdEncoding.EncodedLen(len(ciphertext)))
	base64.StdEncoding.Encode(base64_msg, ciphertext)

	return base64_msg, nil
}

func (self *WXBizMsgCrypt) cbcDecrypter(base64_encrypt_msg string) ([]byte, *CryptError) {
	aeskey, err := base64.StdEncoding.DecodeString(self.encoding_aeskey)
	if nil != err {
		return nil, NewCryptError(DecodeBase64Error, err.Error())
	}

	encrypt_msg, err := base64.StdEncoding.DecodeString(base64_encrypt_msg)
	if nil != err {
		return nil, NewCryptError(DecodeBase64Error, err.Error())
	}

	block, err := aes.NewCipher(aeskey)
	if err != nil {
		return nil, NewCryptError(DecryptAESError, err.Error())
	}

	if len(encrypt_msg) < aes.BlockSize {
		return nil, NewCryptError(DecryptAESError, "encrypt_msg size is not valid")
	}

	iv := aeskey[:aes.BlockSize]

	if len(encrypt_msg)%aes.BlockSize != 0 {
		return nil, NewCryptError(DecryptAESError, "encrypt_msg not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	mode.CryptBlocks(encrypt_msg, encrypt_msg)

	return encrypt_msg, nil
}

func (self *WXBizMsgCrypt) calSignature(timestamp, nonce, data string) string {
	sort_arr := []string{self.token, timestamp, nonce, data}
	sort.Strings(sort_arr)
	var buffer bytes.Buffer
	for _, value := range sort_arr {
		buffer.WriteString(value)
	}

	sha := sha1.New()
	sha.Write(buffer.Bytes())
	signature := fmt.Sprintf("%x", sha.Sum(nil))
	return string(signature)
}

func (self *WXBizMsgCrypt) ParsePlainText(plaintext []byte) ([]byte, uint32, []byte, []byte, *CryptError) {
	const block_size = 32
	plaintext, err := self.pKCS7Unpadding(plaintext, block_size)
	if nil != err {
		return nil, 0, nil, nil, err
	}

	text_len := uint32(len(plaintext))
	if text_len < 20 {
		return nil, 0, nil, nil, NewCryptError(IllegalBuffer, "plain is to small 1")
	}
	random := plaintext[:16]
	msg_len := binary.BigEndian.Uint32(plaintext[16:20])
	if text_len < (20 + msg_len) {
		return nil, 0, nil, nil, NewCryptError(IllegalBuffer, "plain is to small 2")
	}

	msg := plaintext[20 : 20+msg_len]
	receiver_id := plaintext[20+msg_len:]

	return random, msg_len, msg, receiver_id, nil
}

func (self *WXBizMsgCrypt) VerifyURL(msg_signature, timestamp, nonce, echostr string) ([]byte, *CryptError) {
	signature := self.calSignature(timestamp, nonce, echostr)

	if strings.Compare(signature, msg_signature) != 0 {
		return nil, NewCryptError(ValidateSignatureError, "signature not equal")
	}

	plaintext, err := self.cbcDecrypter(echostr)
	if nil != err {
		return nil, err
	}

	_, _, msg, receiver_id, err := self.ParsePlainText(plaintext)
	if nil != err {
		return nil, err
	}

	if len(self.receiver_id) > 0 && strings.Compare(string(receiver_id), self.receiver_id) != 0 {
		fmt.Println(string(receiver_id), self.receiver_id, len(receiver_id), len(self.receiver_id))
		return nil, NewCryptError(ValidateCorpidError, "receiver_id is not equil")
	}

	return msg, nil
}

func (self *WXBizMsgCrypt) EncryptMsg(reply_msg, timestamp, nonce string) ([]byte, *CryptError) {
	rand_str := self.randString(16)
	var buffer bytes.Buffer
	buffer.WriteString(rand_str)

	msg_len_buf := make([]byte, 4)
	binary.BigEndian.PutUint32(msg_len_buf, uint32(len(reply_msg)))
	buffer.Write(msg_len_buf)
	buffer.WriteString(reply_msg)
	buffer.WriteString(self.receiver_id)

	tmp_ciphertext, err := self.cbcEncrypter(buffer.String())
	if nil != err {
		return nil, err
	}
	ciphertext := string(tmp_ciphertext)

	signature := self.calSignature(timestamp, nonce, ciphertext)

	msg4_send := NewWXBizMsg4Send(ciphertext, signature, timestamp, nonce)
	return self.protocol_processor.serialize(msg4_send)
}

func (self *WXBizMsgCrypt) DecryptMsg(msg_signature, timestamp, nonce string, post_data []byte) ([]byte, *CryptError) {
	msg4_recv, crypt_err := self.protocol_processor.parse(post_data)
	if nil != crypt_err {
		return nil, crypt_err
	}

	signature := self.calSignature(timestamp, nonce, msg4_recv.Encrypt)

	if strings.Compare(signature, msg_signature) != 0 {
		return nil, NewCryptError(ValidateSignatureError, "signature not equal")
	}

	plaintext, crypt_err := self.cbcDecrypter(msg4_recv.Encrypt)
	if nil != crypt_err {
		return nil, crypt_err
	}

	_, _, msg, receiver_id, crypt_err := self.ParsePlainText(plaintext)
	if nil != crypt_err {
		return nil, crypt_err
	}

	if len(self.receiver_id) > 0 && strings.Compare(string(receiver_id), self.receiver_id) != 0 {
		return nil, NewCryptError(ValidateCorpidError, "receiver_id is not equil")
	}

	return msg, nil
}
