package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strconv"
	"time"
)

type Block struct {
	Timestamp     int64
	Data          []byte
	PrevBlockHash []byte
	Hash          []byte
}

type Blockchain struct {
	blocks []*Block
}

func main() {
	bc := NewBlockchain()

	bc.AddBlock("Send 1 BTC to Alex.")
	bc.AddBlock("Send 2 BTC to Nicholas.")

	for _, block := range bc.blocks {
		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Println()
	}
}

/*
	Sets the hash for the block
*/
func (b *Block) SetHash() {
	timestamp := []byte(strconv.FormatInt(b.Timestamp, 10))

	// Join [][]byte into []byte
	headers := bytes.Join([][]byte{b.PrevBlockHash, b.Data, timestamp}, []byte{})

	// SHA256 hash
	hash := sha256.Sum256(headers)

	// Set hash of block
	b.Hash = hash[:]
}

/*
	Creates a new block given the new data and the previous block hash
*/
func NewBlock(data string, prevBlockHash []byte) *Block {
	block := &Block{time.Now().Unix(), []byte(data), prevBlockHash, []byte{}}
	block.SetHash()
	return block
}

/*
	Adds a block the the blockchain (using passed in data)
*/
func (bc *Blockchain) AddBlock(data string) {
	prevBlock := bc.blocks[len(bc.blocks)-1]
	newBlock := NewBlock(data, prevBlock.Hash)
	bc.blocks = append(bc.blocks, newBlock)
}

func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}

func NewBlockchain() *Blockchain {
	return &Blockchain{[]*Block{NewGenesisBlock()}}
}
