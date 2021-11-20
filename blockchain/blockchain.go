package blockchain

import (
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"log"
)

const (
	dbPath     = "./tmp/blockchain"
	defaultKey = "lh"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func InitBlockChain() *BlockChain {
	var lastHash []byte

	opts := badger.DefaultOptions(dbPath)

	db, err := badger.Open(opts)
	if err != nil {
		log.Panicln("badger.Open failed on InitBlockChain: ", err)
	}

	lastHashKey := []byte(defaultKey)

	err = db.Update(func(txn *badger.Txn) error {

		_, err := txn.Get(lastHashKey)

		if err == badger.ErrKeyNotFound {
			fmt.Println("No existing blockchain found")

			genesis := Genesis()
			fmt.Println("Genesis proved")

			err = txn.Set(genesis.Hash, genesis.Serialize())
			if err != nil {
				return err
			}

			lastHash = genesis.Hash

			return txn.Set(lastHashKey, genesis.Hash)
		}

		item, err := txn.Get(lastHashKey)
		if err != nil {
			log.Panicln("txn.Get failed on db.Update: ", err)
		}

		return item.Value(func(lh []byte) error {
			lastHash = lh
			return nil
		})

	})

	if err != nil {
		log.Panicln("db.Update failed on InitBlockChain: ", err)
	}

	return &BlockChain{lastHash, db}
}

func (chain *BlockChain) AddBlock(data string) {
	var lastHash []byte

	lastHashKey := []byte(defaultKey)

	err := chain.Database.View(func(txn *badger.Txn) error {

		item, err := txn.Get(lastHashKey)
		if err != nil {
			return err
		}

		err = item.Value(func(lh []byte) error {
			lastHash = lh
			return nil
		})

		return err
	})

	if err != nil {
		log.Panicln("chain.Database.View failed on AddBlock: ", err)
	}

	newBlock := CreateBlock(data, lastHash)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			return err
		}
		return txn.Set(lastHashKey, newBlock.Hash)
	})

	if err != nil {
		log.Panicln("chain.Database.Update failed on AddBlock: ", err)
	}

	chain.LastHash = newBlock.Hash
}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	return &BlockChainIterator{chain.LastHash, chain.Database}
}

func (iter *BlockChainIterator) Next() *Block {
	var block *Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		if err != nil {
			return err
		}
		return item.Value(func(b []byte) error {
			block = Deserialize(b)
			return nil
		})
	})

	if err != nil {
		log.Panicln("iter.Database.View failed on Next: ", err)
	}

	iter.CurrentHash = block.PrevHash

	return block
}
