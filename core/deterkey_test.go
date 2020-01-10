package core

import (
	"bytes"
	"testing"
)

func TestNewDeterKey(t *testing.T) {
	dk := NewDeterKey()
	s := dk.Dump("1212")
	pk, err := LoadDeterKey(s, "1212")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dk.Root, pk.Root) {
		t.Fatal("root not equal")
	}
	if !bytes.Equal(dk.Key, pk.Key) {
		t.Fatal("key not equal")
	}
}
