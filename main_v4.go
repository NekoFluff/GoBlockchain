package main

// TODO: Signatures and transactions
// TODO: Transaction verification (with above TODO)
// TODO: Merkle Tree (see tampering of transactions)
// TODO: Listen for network transactions
// TODO: Replacement of chain refreshes proof of work mining
// TODO: P2P
// TODO: Adjust isHashValid for ProofOfWork (use algorithm presented in Berkeley videos)
// TODO: Adjust hasing function to perform double hash  (after implementing merkle tree)
// TODO: Broadcast new block once it has been found

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew" // pretty print slices
	"github.com/gorilla/mux"
	"github.com/joho/godotenv" // read .env
)

const difficulty = 1

type Block struct {
	Index     int
	Timestamp string
	BPM       int
	Hash      string
	PrevHash  string
	// Difficulty int
	Nonce string

	//Required for PoS
	Validator string
}

type Message struct {
	BPM int
}

// Required for PoS
var tempBlocks []Block
var candidateBlocks = make(chan Block)
var announcements = make(chan string)
var validators = make(map[string]int)

//-------
var Blockchain []Block
var bcServer chan []Block //bsServer (a.k.a Blockchain Server) is a channel that handles incoming concurrent Blockchains (which will be broadcast to the other nodes)
var mutex = &sync.Mutex{}

func calculateBlockHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash + block.Validator
	return calculateHash(record)
}

/*
	Generates the hash of a string
*/
func calculateHash(s string) string {
	// Create new sha256 instance
	h := sha256.New()

	// Cast string to bytes and write to hashing algorithm as input
	h.Write([]byte(s))

	// Convert input into 256 byte output
	hashed := h.Sum(nil)

	// Convert output into hex string
	return hex.EncodeToString(hashed)
}

/*
	Used in PoW to see if generated hash is valid
*/
func isHashValid(hash string, difficulty int) bool {
	prefix := strings.Repeat("0", difficulty)
	return strings.HasPrefix(hash, prefix)
}

/*
	Generates a new block using the previous block
*/
func generateBlock(previousBlock Block, BPM int, address string) (Block, error) {
	// Create a new block with 0 values
	var newBlock Block

	// Get the time of creating the new block
	t := time.Now()

	// Set values of the new block
	newBlock.Index = previousBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.BPM = BPM
	newBlock.PrevHash = previousBlock.Hash
	// newBlock.Difficulty = difficulty
	newBlock.Validator = address
	newBlock.Hash = calculateBlockHash(newBlock)

	// // Attempt to hash block with given data
	// for i := 0; ; i++ {
	// 	hex := fmt.Sprintf("%x", i)
	// 	newBlock.Nonce = hex

	// 	// If hash isn't valid, keep working at it
	// 	if !isHashValid(calculateHash(newBlock), newBlock.Difficulty) {
	// 		fmt.Println(calculateHash(newBlock), " more work is required!")
	// 		time.Sleep(time.Second)
	// 		continue
	// 	} else { // Otherwise you found a valid block! Congrats!
	// 		fmt.Println(calculateHash(newBlock), " work complete!")
	// 		newBlock.Hash = calculateHash(newBlock)
	// 		break
	// 	}
	// }

	// Using previously set values, get the hash of the new block
	// newBlock.Hash = calculateHash(newBlock)

	// Done!
	return newBlock, nil
}

/*
	Determines if a block is valid given the previous block
*/
func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}

	if calculateBlockHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}

/*
	Sets the current chain to the longest chain
*/
func replaceChain(newBlocks []Block) {
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
}

/*
	Returms an object that conforms to http.Handler interface
*/
func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")
	return muxRouter
}

/*
	Read from the entire blockchain
*/
func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(Blockchain, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	io.WriteString(w, string(bytes))
}

/*
	Write a block to the blockchain
*/
func handleWriteBlock(w http.ResponseWriter, r *http.Request) {
	var m Message

	// Create a json decoder using request Body text
	decoder := json.NewDecoder(r.Body)

	// Determine if there is an error with the request body
	// Attempt to decode into 'm' Message
	if err := decoder.Decode(&m); err != nil {
		respondWithJSON(w, r, http.StatusBadRequest, r.Body)
		return
	}
	defer r.Body.Close() // The request is OK. Data placed into 'm' Message

	// Create a new block from the HTTP request
	mutex.Lock()
	address := time.Now().String()
	newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], m.BPM, address)
	mutex.Unlock()
	if err != nil {
		respondWithJSON(w, r, http.StatusInternalServerError, m)
		return
	}

	// Determine if the block is valid
	if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {

		// Append to current blockchain
		newBlockchain := append(Blockchain, newBlock)

		// Attempt to replace the chain (if a longer one already exists then don't do it)
		replaceChain(newBlockchain)

		// Pretty print latest Blockchain
		spew.Dump(Blockchain)
	}

	respondWithJSON(w, r, http.StatusCreated, newBlock)
}

/*
	Send out actual message to requester
*/
func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
	response, err := json.MarshalIndent(payload, "", "  ")

	// Failed to convert payload into string
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Server Error"))
		return
	}

	// Write the code and the reponse
	w.WriteHeader(code)
	w.Write(response)
}

func run() error {
	// Create a router
	mux := makeMuxRouter()
	httpAddr := os.Getenv("ADDR")
	log.Println("Listening on port ", httpAddr)

	// Get a pointer to the server (ensure you are modifying the same server and not making copies)
	s := &http.Server{
		Addr:           ":" + httpAddr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := s.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

/*
	Handles TCP Connections
*/
func handleTCPConn(conn net.Conn) {
	// Make sure to close the connection once the function finishes
	defer conn.Close()

	println("New connection established")

	// Simulate recieving announcements (Client)
	go func() {
		for {
			msg := <-announcements
			io.WriteString(conn, msg)
		}
	}()

	// Validator address
	var address string

	// Ask for BPM from the user
	io.WriteString(conn, "Enter token balance:")

	// Scan input
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {

		// Attempt to read in a number
		balance, err := strconv.Atoi(scanner.Text())
		if err != nil {
			log.Printf("%v is not a number: %v", scanner.Text(), err)
			return
		}

		// Save the validator's balance
		t := time.Now()
		address = calculateHash(t.String())
		validators[address] = balance
		fmt.Println(validators)
		break
	}

	// Generate blocks in a goroutine
	io.WriteString(conn, "\nEnter a new BPM:")
	go func() {
		for {
			fmt.Println("Hello")
			for scanner.Scan() {
				fmt.Println("Scanned Value")
				// Convert text to number
				bpm, err := strconv.Atoi(scanner.Text())
				if err != nil {
					log.Printf("%v is not a number: %v", scanner.Text(), err)

					// Delete and close the connection if malicious actor tries to enter false information
					delete(validators, address)
					conn.Close()
					//continue
				}

				mutex.Lock()
				oldlastIndex := Blockchain[len(Blockchain)-1]
				mutex.Unlock()

				// Create a new block with the data
				fmt.Println("Generating block")
				newBlock, err := generateBlock(oldlastIndex, bpm, address)
				fmt.Println("Generated block")
				if err != nil {
					log.Println(err)
					continue
				}

				// Check for validity
				fmt.Println("Adding to candidate")
				if isBlockValid(newBlock, oldlastIndex) {
					// newBlockchain := append(Blockchain, newBlock)
					// replaceChain(newBlockchain)
					candidateBlocks <- newBlock
				}
				fmt.Println("added")
				// Put the new blockchain in the channel (bcServer)
				// bcServer <- Blockchain

				// Ask for BPM from the user
				io.WriteString(conn, "\nEnter a new BPM:")
			}
		}
	}()

	// Simulate Receiving Broadcast (Client)
	for {
		// Every 30 seconds, print out the current blockchain
		time.Sleep(30 * time.Second)
		mutex.Lock()
		output, err := json.Marshal(Blockchain)
		if err != nil {
			log.Fatal(err)
		}
		mutex.Unlock()
		io.WriteString(conn, "\n"+string(output)+"\n")
	}

	// // Print out recieved blockchains (Server)
	// for _ = range bcServer {
	// 	println("Server recieved: ")
	// 	spew.Dump(Blockchain)
	// }
	// println("Connection closed")

}

func pickWinner() {
	time.Sleep(30 * time.Second)
	mutex.Lock()
	temp := tempBlocks
	mutex.Unlock()

	// Create a lottery pool of validators
	lotteryPool := []string{}
	if len(temp) > 0 {
	OUTER:
		// Enumerate through temporary blocks and construct a list of validators that have proposed a new block
		for _, block := range temp {
			// If the validator is already in the lottery pool, skip it
			for _, node := range lotteryPool {
				if block.Validator == node {
					continue OUTER
				}
			}

			mutex.Lock()
			setValidators := validators
			mutex.Unlock()

			// Ensure that the validator is valid
			k, ok := setValidators[block.Validator]
			if ok {
				for i := 0; i < k; i++ {
					// For every dollar/coin/etc. the validator put in, they get another vote in the lottery
					lotteryPool = append(lotteryPool, block.Validator)
				}
			}
		} // End OUTER for loop. You have constructed the lottery pool

		// Randomly pick a winner from the pool
		s := rand.NewSource(time.Now().Unix()) // source
		r := rand.New(s)                       // random number generator
		lotteryWinner := lotteryPool[r.Intn(len(lotteryPool))]

		// Add the winner's block to the blockchain and let all the other nodes know
		for _, block := range temp {
			if block.Validator == lotteryWinner {
				mutex.Lock()
				Blockchain = append(Blockchain, block)
				mutex.Unlock()

				// Announce the winner to everyone else! (send it to the announcements)
				for _ = range validators {
					announcements <- "\nWinning validator: " + lotteryWinner + "\n"
				}
				break
			}
		}

		// Reset temp blocks
		mutex.Lock()
		tempBlocks = []Block{}
		mutex.Unlock()
	}

}

/*
	Listens for TCP requests on Port #XXXX
	Calls handleTCPConn() when a request is recieved
*/
func startTCPServer() {
	// Listen for TCP packets at Port #XXXX
	server, err := net.Listen("tcp", ":"+os.Getenv("ADDR"))
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()

	// Create connection once we hear a request
	// Infinite loop blocks defer server.Close()
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleTCPConn(conn)
	}
}

func main() {
	// Attempt to read environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Make (channel of blockchains)
	// bcServer = make(chan []Block)

	// Create a genesis block
	t := time.Now()
	genesisBlock := Block{0, t.String(), 0, "", "", "", ""}
	genesisBlock.Hash = calculateBlockHash(genesisBlock)

	spew.Dump(genesisBlock)
	Blockchain = append(Blockchain, genesisBlock)

	// Every candidate block gets sent to tempBlocks
	go func() {
		for candidate := range candidateBlocks {
			mutex.Lock()
			tempBlocks = append(tempBlocks, candidate)
			mutex.Unlock()
		}
	}()

	// Pick a winner!
	go func() {
		for {
			pickWinner()
		}
	}()

	startTCPServer()
}
