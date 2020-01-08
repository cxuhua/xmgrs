package core

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"

	"github.com/cxuhua/xginx"
)

//确定性私钥地址
type DeterKey struct {
	Root  []byte `bson:"root"` //私钥内容
	Key   []byte `bson:"key"`  //密钥编码
	Index uint32 `bson:"idx"`  //自增索引
}

//加载key
func LoadDeterKey(s string) (*DeterKey, error) {
	data, err := xginx.HashLoad(s)
	if err != nil {
		return nil, err
	}
	dk := &DeterKey{
		Root: data[:32],
		Key:  data[32:],
	}
	return dk, nil
}

func (k DeterKey) GetId() string {
	pkh := k.GetPks().Hash()
	return GetPrivateId(pkh)
}

func (k DeterKey) GetPks() xginx.PKBytes {
	return k.GetPrivateKey().PublicKey().GetPks()
}

func (k DeterKey) GetPrivateKey() *xginx.PrivateKey {
	pri, err := xginx.NewPrivateKeyWithBytes(k.Root)
	if err != nil {
		panic(err)
	}
	return pri
}

//备份密钥
func (k DeterKey) Dump() string {
	data := append([]byte{}, k.Root...)
	data = append(data, k.Key...)
	return xginx.HashDump(data)
}

func (k DeterKey) String() string {
	return fmt.Sprintf("Root=%s,Key=%s,Idx=%d", hex.EncodeToString(k.Root), hex.EncodeToString(k.Key), k.Index)
}

//派生一个密钥
func (k *DeterKey) New(idx uint32) *DeterKey {
	h := hmac.New(func() hash.Hash {
		return sha512.New()
	}, k.Key)
	_, err := h.Write(k.Root)
	if err != nil {
		panic(err)
	}
	err = binary.Write(h, binary.BigEndian, idx)
	if err != nil {
		panic(err)
	}
	b := h.Sum(nil)
	if len(b) != 64 {
		panic(errors.New("hmac sha512 sum error"))
	}
	return &DeterKey{
		Root: b[:32],
		Key:  b[32:],
	}
}

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
	k.Root = pri.Bytes()
	k.Key = key
	return k
}
