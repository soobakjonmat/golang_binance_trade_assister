package main

import (
	"fmt"
	"strings"
	"constants"
)

func init(){
	var cryptoNameLower string
	fmt.Println("Current trading crypto: ")
	fmt.Scan(&cryptoNameLower)
	var cryptoNameUpper string = strings.ToUpper(cryptoNameLower)
	var cryptoFullname string = cryptoNameUpper + constants.DEFAULT_FIAT_CURRENCY

	
}