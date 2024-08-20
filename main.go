package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/dustin/go-humanize"

	"github.com/gorilla/mux"
)

// var balances = map[string]uint64{
// 	"alice": 100,
// 	"bob":   200,
// }

//go:embed network/dungeon-1/genesis.json
var genesis []byte

type Coin struct {
	Denom  string
	Amount string
}

type Balance struct {
	Address string `json:"address"`
	Coins   []Coin `json:"coins"`
}

var balances = make(map[string][]Coin)

func init() {
	// parse out genesis and get JUST the balances in json in app_state.bank.balances which is an array of objects
	var g map[string]interface{}
	err := json.Unmarshal(genesis, &g)
	if err != nil {
		panic(err)
	}

	// get the balances
	bals := g["app_state"].(map[string]interface{})["bank"].(map[string]interface{})["balances"].([]interface{})
	for _, b := range bals {
		bal := b.(map[string]interface{})
		address := bal["address"].(string)
		coins := bal["coins"].([]interface{})
		var c []Coin
		for _, coin := range coins {
			c = append(c, Coin{
				Denom:  coin.(map[string]interface{})["denom"].(string),
				Amount: coin.(map[string]interface{})["amount"].(string),
			})
		}
		balances[address] = c
	}

	// fmt.Println(balances)
	// panic(1929)
}

func main() {
	gRouter := mux.NewRouter()

	gRouter.HandleFunc("/{address}", GetBalance)

	port := ":4001"
	fmt.Println("Listening on " + port)
	http.ListenAndServe(port, gRouter)
}

func GetBalance(w http.ResponseWriter, r *http.Request) {

	address := mux.Vars(r)["address"]

	// bech32 convert to dungeon
	hrp, data, err := bech32.Decode(address)
	if err != nil {
		http.Error(w, "invalid address", http.StatusBadRequest)
		return
	}

	// require hrp is cosmos, dungeon, osmosis, or juno
	if hrp != "cosmos" && hrp != "dungeon" && hrp != "osmosis" && hrp != "juno" {
		http.Error(w, "invalid address", http.StatusBadRequest)
		return
	}

	// convert to bech32
	address, err = bech32.Encode("dungeon", data)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// given an address, return the balance
	// address := r.URL.Query().Get("address")
	balance, ok := balances[address]
	if !ok {
		http.Error(w, "address not found", http.StatusNotFound)
		return
	}

	if len(balance) == 0 {
		http.Error(w, "no balance allocation", http.StatusNotFound)
		return
	}

	amtInt, err := strconv.Atoi(balance[0].Amount)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if amtInt < 0 {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	outputStr := fmt.Sprintf("%s %s", humanize.Comma(int64(amtInt/1_000_000)), "DGN")
	w.Write([]byte(outputStr))
}
