package core

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"

	"github.com/cxuhua/xginx"
	"github.com/cxuhua/xmgrs/util"
)

//DeterKey 确定性私钥地址
type DeterKey struct {
	Body []byte `bson:"body"` //私钥内容 32 bytes
	Key  []byte `bson:"key"`  //派生新私钥密钥 32 bytes,派生下个密钥使用
}

//LoadDeterKey 加载key
func LoadDeterKey(s string, pass ...string) (*DeterKey, error) {
	data, err := xginx.HashLoad(s, pass...)
	if err != nil {
		return nil, err
	}
	if len(data) != 64 {
		return nil, errors.New("data len error")
	}
	dk := &DeterKey{
		Body: data[:32],
		Key:  data[32:],
	}
	return dk, nil
}

//GetID 获取私钥id
func (k DeterKey) GetID() string {
	pkh := k.GetPks().Hash()
	return GetPrivateID(pkh)
}

//GetPks 获取公钥内容
func (k DeterKey) GetPks() xginx.PKBytes {
	pri, err := k.GetPrivateKey()
	if err != nil {
		panic(err)
	}
	return pri.PublicKey().GetPks()
}

//GetPrivateKey 获取私钥
func (k DeterKey) GetPrivateKey() (*xginx.PrivateKey, error) {
	return xginx.NewPrivateKeyWithBytes(k.Body)
}

//Dump 备份密钥
func (k DeterKey) Dump(pass ...string) (string, error) {
	data := append([]byte{}, k.Body...)
	data = append(data, k.Key...)
	return xginx.HashDump(data, pass...)
}

func (k DeterKey) String() string {
	return fmt.Sprintf("Body=%s,Key=%s", util.Hex(k.Body), util.Hex(k.Key))
}

//New 派生一个密钥
func (k *DeterKey) New(idx uint32) (*DeterKey, error) {
	h := hmac.New(func() hash.Hash {
		return sha512.New()
	}, k.Key)
	_, err := h.Write(k.Body)
	if err != nil {
		return nil, err
	}
	err = binary.Write(h, xginx.Endian, idx)
	if err != nil {
		return nil, err
	}
	b := h.Sum(nil)
	if len(b) != 64 {
		panic(errors.New("hmac sha512 sum error"))
	}
	return &DeterKey{
		Body: b[:32],
		Key:  b[32:],
	}, nil
}

//NewDeterKey 创建一个确定性私钥
func NewDeterKey() *DeterKey {
	pri, err := xginx.NewPrivateKey()
	if err != nil {
		panic(err)
	}
	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		panic(err)
	}
	k := &DeterKey{}
	k.Body = pri.Bytes()
	k.Key = key
	return k
}
