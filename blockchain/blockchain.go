package blockchain

import (
	"encoding/hex"
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"log"
	"os"
	"runtime"
)

const (
	dbPath         = "./tmp/blockchain"
	dbManifestPath = "./tmp/blockchain/MANIFEST"
	defaultKey     = "lh"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func DBExists() bool {
	if _, err := os.Stat(dbManifestPath); os.IsNotExist(err) {
		return false
	}
	return true
}

func InitBlockChain(address string) *BlockChain {
	var lastHash []byte

	if DBExists() {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	opts := badger.DefaultOptions(dbPath)

	db, err := badger.Open(opts)
	if err != nil {
		log.Panicln("badger.Open failed on InitBlockChain: ", err)
	}

	lastHashKey := []byte(defaultKey)

	err = db.Update(func(txn *badger.Txn) error {

		_, err := txn.Get(lastHashKey)

		if err == badger.ErrKeyNotFound {

			coinbase := CoinbaseTx(address, "First Transaction from Genesis")
			genesis := Genesis(coinbase)
			fmt.Println("Genesis created")

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

func ContinueBlockChain(address string) *BlockChain {
	if !DBExists() {
		fmt.Println("No existing blockchain found, create one!")
		runtime.Goexit()
	}

	opts := badger.DefaultOptions(dbPath)

	db, err := badger.Open(opts)
	if err != nil {
		log.Panicln("badger.Open failed on ContinueBlockChain: ", err)
	}

	var lastHash []byte
	lastHashKey := []byte(defaultKey)

	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(lastHashKey)
		if err != nil {
			return err
		}
		return item.Value(func(lh []byte) error {
			lastHash = lh
			return nil
		})
	})

	if err != nil {
		log.Panicln("db.Update failed on ContinueBlockChain: ", err)
	}

	return &BlockChain{lastHash, db}
}

func (chain *BlockChain) AddBlock(transactions []*Transaction) {
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

	newBlock := CreateBlock(transactions, lastHash)

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

func (chain *BlockChain) FindUnspentTransactions(address string) []Transaction {
	var unspentTxs []Transaction

	spentTXOs := make(map[string][]int)

	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				if out.CanBeUnlocked(address) {
					unspentTxs = append(unspentTxs, *tx)
				}
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					if in.CanUnlock(address) {
						inTxID := hex.EncodeToString(in.ID)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Out)
					}
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return unspentTxs
}

func (chain *BlockChain) FindUTXO(address string) []TxOutput {
	var UTXOs []TxOutput
	unspentTransactions := chain.FindUnspentTransactions(address)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.CanBeUnlocked(address) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

func (chain *BlockChain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(address)
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			if out.CanBeUnlocked(address) && accumulated < amount {
				accumulated += out.Value
				unspentOuts[txID] = append(unspentOuts[txID], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOuts
}
