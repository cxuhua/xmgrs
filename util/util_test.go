package util

import (
	"log"
	"math/big"
	"net"
	"testing"

	"github.com/cxuhua/xginx"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestLookUpSrv(t *testing.T) {
	name, srvs, err := net.LookupSRV("mongodb", "tcp", "rs.xginx.com")
	if err != nil {
		panic(err)
	}
	log.Println(name, len(srvs))
	for _, srv := range srvs {
		log.Println(srv)
	}
}

func TestBigInt(t *testing.T) {
	a, err := primitive.ParseDecimal128("-121212121212121212121212121212")
	if err != nil {
		panic(err)
	}
	log.Println(a.BigInt())

	b := new(big.Int).SetInt64(11)
	a, _ = primitive.ParseDecimal128FromBigInt(b, 10)
	log.Println(a.BigInt())
}

func TestAes(t *testing.T) {
	key := "jzxc972198hasdhsad^^027302173102"
	block := xginx.NewAESCipher([]byte(key))
	s := "skdfjslnxvc97934734"
	db, err := xginx.AesEncrypt(block, []byte(s))
	if err != nil {
		t.Fatal(err)
	}
	d, err := xginx.AesDecrypt(block, db)
	if err != nil {
		t.Fatal(err)
	}
	if s != string(d) {
		t.Fatal("dec enc failed")
	}
}

func TestRemoveRepeat(t *testing.T) {
	vs := []string{"111", "222", "111"}
	vs = RemoveRepeat(vs)
	if len(vs) != 2 {
		t.Fatal("error")
	}
}
