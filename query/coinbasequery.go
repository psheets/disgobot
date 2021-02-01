package query

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type CoinbaseData struct {
	Amount string `json:"amount"`
}

// Response
type CoinbaseResponse struct {
	Data CoinbaseData `json:"data"`
}

func GetCrypt(cur string) string {
	url := fmt.Sprintf("https://api.coinbase.com/v2/prices/%s-USD/spot", cur)
	resp, _ := http.Get(url)
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	var res CoinbaseResponse
	json.Unmarshal(body, &res)

	return res.Data.Amount

}
