package stock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/billylkc/stocklib/util"
)

func HketTest() error {
	// https://invest.hket.com/markets
	// https://invest.hket.com/market-store/board_meeting/earnings_forecasts_bycode.html

	var foo Foo
	err := getJson("http://example.com", foo)
	if err != nil {
		return err
	}
	fmt.Println("go")

	fmt.Println(util.PrettyPrint(foo))

	return nil
}

type Foo struct {
	Bar string
}

func getJson(url string, target interface{}) error {
	var myClient = &http.Client{Timeout: 10 * time.Second}
	r, err := myClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}
