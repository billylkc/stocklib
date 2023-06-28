package main

import (
	"fmt"

	"github.com/billylkc/stocklib/stock"
)

func main() {
	err := stock.HketTest()
	if err != nil {
		fmt.Println(err)

	}

}
