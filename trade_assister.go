package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const DEFAULT_FIAT_CURRENCY = "USDT"
const baseURL = "https://fapi.binance.com/fapi/v1/"

type tradeVar struct {
	cryptoFullname                string
	priceDP, balanceDP, leverage  int
	pOrderBookNum                 *int
	pTradeFactor, pCloseAmtFactor *float64
}

func SetParams() tradeVar {
	var cryptoNameLower string
	fmt.Print("Set current trading crypto: ")
	fmt.Scanln(&cryptoNameLower)
	var (
		cryptoNameUpper string = strings.ToUpper(cryptoNameLower)
		cryptoFullname  string = cryptoNameUpper + DEFAULT_FIAT_CURRENCY
	)

	var (
		priceDP       int
		balanceDP     int
		validSymbol   bool = false
		wholeJsonResp      = make(map[string]interface{})
	)
	resp, _ := http.Get(baseURL + "exchangeInfo")
	json.NewDecoder(resp.Body).Decode(&wholeJsonResp)

	symbolsArray := wholeJsonResp["symbols"].([]interface{})
	for _, element := range symbolsArray {
		symbolArray := element.(map[string]interface{})
		symbolValue := symbolArray["symbol"]
		if symbolValue == cryptoFullname {
			validSymbol = true
			priceDP = int(symbolArray["baseAssetPrecision"].(float64))
			balanceDP = int(symbolArray["quantityPrecision"].(float64))
			break
		}
	}
	if !validSymbol {
		invalidError := errors.New("the given symbol is invalid")
		log.Panic(invalidError)
	}

	var leverage int
	fmt.Print("Set leverage: ")
	fmt.Scanln(&leverage)

	var orderBookNum int
	fmt.Print("Set order book number: ")
	fmt.Scanln(&orderBookNum)
	var oBNPointer *int = &orderBookNum

	var tradeFactor float64
	fmt.Print("Set trading factor: (Try not to set it over 20) ")
	fmt.Scanln(&tradeFactor)
	tradeFactor /= 100
	var tFPointer *float64 = &tradeFactor

	var closeAmtFactor float64
	fmt.Print("Set close amount factor: ")
	fmt.Scanln(&closeAmtFactor)
	closeAmtFactor /= 100
	var cAFPointer *float64 = &closeAmtFactor

	return tradeVar{cryptoFullname, priceDP, balanceDP, leverage, oBNPointer, tFPointer, cAFPointer}
}

func enterLong() {
	fmt.Println("Entering long")

}

func enterShort() {
	fmt.Println("Entering short")
}

func closePosition() {
	fmt.Println("Closing position")
}

func main() {
	param := SetParams()
	inputScanner := bufio.NewScanner(os.Stdin)

	fmt.Println()
	fmt.Println("Crypto name: ", param.cryptoFullname)
	fmt.Println("Price decimal place: ", param.priceDP)
	fmt.Println("Balance decimal place: ", param.balanceDP)
	fmt.Println("Leverage: ", param.leverage)
	fmt.Println("Order book number: ", *param.pOrderBookNum)
	fmt.Println("Trade factor: ", *param.pTradeFactor)
	fmt.Println("Close amount factor: ", *param.pCloseAmtFactor)
	fmt.Println()

	var command string
	for {
		fmt.Print("Buy(b)/Sell(s)/Close(c)/Set trading factor(tf)/Set closing amount factor(cf)/Set order book number(o) ")
		inputScanner.Scan()
		command = inputScanner.Text()
		if command == "b" {
			enterLong()
		} else if command == "s" {
			enterShort()
		} else if command == "c" {
			closePosition()
		} else if command == "tf" {
			fmt.Print("Set trading factor: (Try not to set it over 20) ")
			inputScanner.Scan()
			*param.pTradeFactor, _ = strconv.ParseFloat(inputScanner.Text(), 64)
			*param.pTradeFactor /= 100
		} else if command == "cf" {
			fmt.Print("Set close amount factor: ")
			inputScanner.Scan()
			*param.pCloseAmtFactor, _ = strconv.ParseFloat(inputScanner.Text(), 64)
			*param.pCloseAmtFactor /= 100
		} else if command == "o" {
			fmt.Print("Set order book number: ")
			inputScanner.Scan()
			*param.pOrderBookNum, _ = strconv.Atoi(inputScanner.Text())
		} else {
			fmt.Println("Wrong command")
		}
	}
}
