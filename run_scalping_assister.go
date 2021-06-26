package main

import (
	"fmt"
	"strings"
)

func init() {
	const DEFAULT_FIAT_CURRENCY = "USDT"

	type decimals struct {
		price int
		balance int
	}
	DecimalPlaces := map[string]decimals{
		"ada": {price: 4, balance: 0},
		"xrp": {price: 4, balance: 1},
	}

	var cryptoNameLower string
	fmt.Print("Current trading crypto: ")
	fmt.Scan(&cryptoNameLower)
	var cryptoNameUpper string = strings.ToUpper(cryptoNameLower)
	var cryptoFullname string = cryptoNameUpper + DEFAULT_FIAT_CURRENCY
	fmt.Println(cryptoFullname)
	var priceDecimalPlace int = DecimalPlaces[cryptoNameLower].price
	var balanceDecimalPlace int = DecimalPlaces[cryptoNameLower].balance
	fmt.Println(priceDecimalPlace)
	fmt.Println(balanceDecimalPlace)
}

func main() {
	fmt.Println("main function executed")
}