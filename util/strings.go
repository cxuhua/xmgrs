package util

import (
	"crypto/rand"

	"github.com/cxuhua/xginx"
)

//移除重复的字符串
func RemoveRepeat(vs []string) []string {
	ms := map[string]bool{}
	for _, v := range vs {
		ms[v] = true
	}
	vs = []string{}
	for k, _ := range ms {
		vs = append(vs, k)
	}
	return vs
}

//生成随机字符串
func NonceStr(n ...int) string {
	l := 8
	if len(n) > 0 {
		l = n[0]
	}
	v := make([]byte, l)
	rand.Read(v)
	return xginx.B58Encode(v, xginx.BitcoinAlphabet)
}
