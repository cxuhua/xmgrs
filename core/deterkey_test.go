package core

import (
	"log"
	"testing"
)

func TestNewDeterKey(t *testing.T) {
	dk := NewDeterKey()
	log.Println(dk)
}
