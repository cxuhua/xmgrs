package core

import (
	"bytes"
	"testing"
)

func TestNewDeterKey(t *testing.T) {
	dk := NewDeterKey()
	s, err := dk.Dump("1212")
	if err != nil {
		t.Fatal(err)
	}
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
	s, err = dk.Dump()
	if err != nil {
		t.Fatal(err)
	}
	pk, err = LoadDeterKey(s)
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
