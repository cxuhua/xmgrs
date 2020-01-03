package util

import (
	"testing"
)

func TestAes(t *testing.T) {
	key := "jzxc972198hasdhsad^^027302173102"
	block := NewAESCipher([]byte(key))
	s := "skdfjslnxvc97934734"
	db, err := AesEncrypt(block, []byte(s))
	if err != nil {
		t.Fatal(err)
	}
	d, err := AesDecrypt(block, db)
	if err != nil {
		t.Fatal(err)
	}
	if s != string(d) {
		t.Fatal("dec enc failed")
	}
}
