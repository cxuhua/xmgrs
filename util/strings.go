package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/cxuhua/xginx"
)

var (
	//Hex 输出16机制字符串
	Hex = hex.EncodeToString
)

//ScriptToStr 解析输出交易脚本
func ScriptToStr(script xginx.Script) string {
	if script.Len() == 0 {
		return ""
	}
	if script.IsCoinBase() {
		return fmt.Sprintf("COINBASE %s", Hex(script[1:]))
	}
	if script.IsWitness() {
		wits, err := script.ToWitness()
		if err != nil {
			return fmt.Sprintf("ERROR %s", Hex(script))
		}
		return fmt.Sprintf("WITNESS %d %d %d ADDRESS=%s {%s} %s", wits.Num, wits.Less, wits.Arb, wits.Address(), string(wits.Exec), Hex(script[1:]))
	}
	if script.IsLocked() {
		lcks, err := script.ToLocked()
		if err != nil {
			return fmt.Sprintf("ERROR %s", Hex(script))
		}
		return fmt.Sprintf("LOCKED ADDRESS=%s {%s} %s", lcks.Address(), string(lcks.Exec), Hex(script[1:]))
	}
	if script.IsTxScript() {
		txs, err := script.ToTxScript()
		if err != nil {
			return fmt.Sprintf("ERROR %s", Hex(script))
		}
		return fmt.Sprintf("TXSCRIPT {%s} %s", string(txs.Exec), Hex(script[1:]))
	}
	return "TYPE ERROR"
}

//RemoveRepeat 移除重复的字符串
func RemoveRepeat(vs []string) []string {
	ms := map[string]bool{}
	for _, v := range vs {
		ms[v] = true
	}
	vs = []string{}
	for k := range ms {
		vs = append(vs, k)
	}
	return vs
}

//NonceStr 生成随机字符串
func NonceStr(n ...int) string {
	l := 8
	if len(n) > 0 && n[0] > 0 {
		l = n[0]
	}
	v := make([]byte, l)
	_, err := rand.Read(v)
	if err != nil {
		panic(err)
	}
	return xginx.B58Encode(v, xginx.BitcoinAlphabet)
}
