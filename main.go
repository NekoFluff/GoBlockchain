package main

// TODO: Signatures and transactions
// TODO: Merkle Tree
// TODO: Prrof of Work
// TODO: Listen for network transactions
// TODO: Replacement of chain refreshes proof of work mining

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew" // pretty print slices
	"github.com/gorilla/mux"          // web handler
	"github.com/joho/godotenv"        // read .env
)

type Block struct {
	Index     int
	Timestamp string
	BPM       int
	Hash      string
	PrevHash  string
}
type Message struct {
	BPM int
}

var Blockchain []Block

/*
	Generates the hash of a block
*/
func calculateHash(block Block) string {
	// Convert data to string
	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash

	// Create new sha256 instance
	h := sha256.New()

	// Cast string to bytes and write to hashing algorithm as input
	h.Write([]byte(record))

	// Convert input into 256 byte output
	hashed := h.Sum(nil)

	// Convert output into hex string
	return hex.EncodeToString(hashed)
}

/*
	Generates a new block using the previous block
*/
func generateBlock(previousBlock Block, BPM int) (Block, error) {
	// Create a new block with 0 values
	var newBlock Block

	// Get the time of creating the new block
	t := time.Now()

	// Set values of the new block
	newBlock.Index = previousBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.BPM = BPM
	newBlock.PrevHash = previousBlock.Hash

	// Using previously set values, get the hash of the new block
	newBlock.Hash = calculateHash(newBlock)

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

	if calculateHash(newBlock) != newBlock.Hash {
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
	Retirms object that conforms to http.Handler interface
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
	newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], m.BPM)
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

func main() {
	// Attempt to read environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	// goroutine to run server?
	go func() {
		t := time.Now()
		genesisBlock := Block{0, t.String(), 0, "", ""}
		spew.Dump(genesisBlock)
		Blockchain = append(Blockchain, genesisBlock)
	}()

	// Run the server
	log.Fatal(run())
}
