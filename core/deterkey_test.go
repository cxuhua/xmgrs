package core

import (
	"encoding/gob"
	"log"
	"testing"
)

func TestNewDeterKey(t *testing.T) {
	dk := NewDeterKey()
	log.Println(dk)
	gob.NewDecoder()
}
