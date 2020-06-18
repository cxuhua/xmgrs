package core

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/cxuhua/xginx"

	"github.com/hashicorp/go-memdb"
	"github.com/stretchr/testify/assert"
)

func TestMemDB(t *testing.T) {
	type Person struct {
		Email string
		Name  string
		Age   int
	}

	// Create the DB schema
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"person": {
				Name: "person",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Email"},
					},
					"age": {
						Name:    "age",
						Unique:  false,
						Indexer: &memdb.IntFieldIndex{Field: "Age"},
					},
				},
			},
		},
	}

	// Create a new data base
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		panic(err)
	}

	// Create a write transaction
	txn := db.Txn(true)

	// Insert some people
	people := []*Person{
		{"dorothy@aol.com", "Dorothy", 53},
		{"joe@aol.com", "Joe", 30},
		{"lucy@aol.com", "Lucy", 35},
		{"tariq@aol.com", "Tariq", 21},
	}
	for _, p := range people {
		if err := txn.Insert("person", p); err != nil {
			panic(err)
		}
	}

	// Commit the transaction
	txn.Commit()

	// Create read-only transaction
	txn = db.Txn(false)
	defer txn.Abort()

	// Lookup by email
	raw, err := txn.First("person", "id", "joe@aol.com")
	if err != nil {
		panic(err)
	}

	// Say hi!
	fmt.Printf("Hello %s!\n", raw.(*Person).Name)

	// List all the people
	it, err := txn.Get("person", "id")
	if err != nil {
		panic(err)
	}

	fmt.Println("All the people:")
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(*Person)
		fmt.Printf("  %s %d\n", p.Email, p.Age)
	}

	// Range scan over people with ages between 25 and 35 inclusive
	it, err = txn.LowerBound("person", "id", "joe@aol.com")
	if err != nil {
		panic(err)
	}

	fmt.Println("People aged 25 - 35:")
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(*Person)
		fmt.Printf("  %s is aged %d\n", p.Name, p.Age)
	}
}

func TestLoadDumpKey(t *testing.T) {
	as := assert.New(t)
	k1 := NewDeterKey()
	s, err := k1.Dump("1111")
	if err != nil {
		t.Fatal(err)
	}
	k2, err := LoadDeterKey(s, "1111")
	as.NoError(err)
	as.Equal(k1.Body, k2.Body)
	as.Equal(k1.Key, k2.Key)
	msg := xginx.Hash256([]byte("dkfsdnf(9343"))
	pri, err := k2.GetPrivateKey()
	as.NoError(err)
	sig, err := pri.Sign(msg)
	as.NoError(err)
	pub := pri.PublicKey()
	vb := pub.Verify(msg, sig)
	as.True(vb, "sign verify error")
}

func TestNewPrivate(t *testing.T) {
	app := InitApp(context.Background())
	defer app.Close()
	err := app.UseTx(func(db IDbImp) error {
		user, err := db.GetUserInfoWithMobile("17716858036")
		if err == nil {
			db.DeleteUser(user.ID)
		}
		user, err = NewUser("17716858036", "xh0714", "1111")
		if err != nil {
			return err
		}
		err = db.InsertUser(user)
		if err != nil {
			return err
		}
		dp, err := user.NewPrivate(db, "测试私钥1", "1111")
		if err != nil {
			return err
		}
		pri, err := db.GetPrivate(dp.ID)
		if err != nil {
			return err
		}
		if !pri.Pkh.Equal(dp.Pkh) {
			return errors.New("pkh error")
		}
		err = db.DeleteUser(user.ID)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}
