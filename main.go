package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/dustin/go-humanize"

	"github.com/gorilla/mux"
)

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
}

func main() {
	gRouter := mux.NewRouter()

	gRouter.HandleFunc("/", Home)

	gRouter.HandleFunc("/{address}", GetBalance)

	port := 4001
	if len(os.Args) > 1 {
		port, _ = strconv.Atoi(os.Args[1])
	}

	host := ":"
	if len(os.Args) > 2 {
		host = os.Args[2]
		if host[len(host)-1] != ':' {
			host += ":"
		}
	}

	fmt.Println("Listening on " + host + strconv.Itoa(port))
	http.ListenAndServe(fmt.Sprintf("%s%d", host, port), gRouter)
}

func Home(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Dungeon Airdrop Check</title>
	</head>
	<body>
		<h1>Dungeon Airdrop Check</h1>

		<form action="/"" method="get" id="form">
			<input type="text" id="address" name="address" placeholder="Enter your cosmos address">
			<input type="submit" value="Check">
		</form>

		<script>
			document.getElementById("form").addEventListener("submit", function(e) {
				e.preventDefault();
				const address = document.getElementById("address").value;
				window.location.href = "https://dungeon-airdrop-check.reece.sh/" + address;
			});
		</script>

		<hr />

		<p>Use <a href="https://dungeon-airdrop-check.reece.sh/YOUR_ADDRESS_HERE">https://dungeon-airdrop-check.reece.sh/{address}</a> to check your DGN airdrop allocation.</p>
		<p>( You can use your CosmosHub, Osmosis, Juno, or Noble address from any wallet )</p>

		<p>Source & airdrop logic: <a href="https://github.com/CryptoDungeon/dungeonchain/tree/main/airdrop">https://github.com/CryptoDungeon/dungeonchain</a></p>
	</body>
	</html>
	`
	w.Write([]byte(html))
}

func GetBalance(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	// see if address is in vars, if not, return error
	address, ok := vars["address"]
	if !ok {
		http.Error(w, "address not found. make request with https://dungeon-airdrop-check.reece.sh/{address}", http.StatusBadRequest)
		return
	}

	// bech32 convert to dungeon
	hrp, data, err := bech32.Decode(address)
	if err != nil {
		http.Error(w, "invalid address", http.StatusBadRequest)
		return
	}

	// require hrp is cosmos, dungeon, osmosis, or juno
	if hrp != "cosmos" && hrp != "dungeon" && hrp != "osmosis" && hrp != "juno" && hrp != "noble" {
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
	// w.Write([]byte(outputStr))

	html := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<title>Dungeon Airdrop Check</title>
	</head>
	<body>
		<h1>Dungeon Airdrop Check</h1>
		<p>Address: %s</p>
		<p>Allocation: %s</p>
	</body>
	</html>
	`, address, outputStr)
	w.Write([]byte(html))
}
