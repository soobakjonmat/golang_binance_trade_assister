package main

import (
	"bufio"
	"context"
	"fmt"
	"golang_binance_trade_assister/credentials"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
)

const DEFAULT_FIAT_CURRENCY = "USDT"
const toSecond = 1000000000

type tradeVar struct {
	cryptoFullname                string
	priceDP, quantityDP           int
	pOrderBookNum                 *int
	leverage                      float64
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

	client := binance.NewFuturesClient(credentials.API_KEY, credentials.API_SECRET)

	var priceDP, quantityDP int
	exchangeInfo, _ := client.NewExchangeInfoService().Do(context.Background())
	var isValidSymbol bool = false
	for _, item := range exchangeInfo.Symbols {
		if item.Symbol == cryptoFullname {
			isValidSymbol = true
			priceDP = item.PricePrecision
			quantityDP = item.QuantityPrecision
			break
		}
	}
	if !isValidSymbol {
		log.Panicln("Invalid Symbol")
	}

	var leverage float64
	accountInfo, _ := client.NewGetAccountService().Do(context.Background())
	for _, item := range accountInfo.Positions {
		if item.Symbol == cryptoFullname {
			leverage, _ = strconv.ParseFloat(item.Leverage, 64)
		}
	}

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

	return tradeVar{cryptoFullname, priceDP, quantityDP, oBNPointer, leverage, tFPointer, cAFPointer}
}

func enterLong(param tradeVar, client *futures.Client) {
	orderBook, _ := client.NewDepthService().Symbol(param.cryptoFullname).Limit(5).Do(context.Background())
	enteringPrice := orderBook.Bids[*param.pOrderBookNum].Price
	enteringPriceFloat, _, _ := orderBook.Bids[*param.pOrderBookNum].Parse()

	accountInfo, _ := client.NewGetAccountService().Do(context.Background())
	balance := accountInfo.TotalWalletBalance
	balanceFloat, _ := strconv.ParseFloat(balance, 64)
	quantity := (balanceFloat * param.leverage * *param.pTradeFactor) / enteringPriceFloat
	quantityStr := strconv.FormatFloat(quantity, 'f', param.quantityDP, 64)

	fmt.Printf("Placing a long order to buy %v %v at %v\n", quantityStr, param.cryptoFullname, enteringPrice)

	client.NewCreateOrderService().Symbol(param.cryptoFullname).
		Side("BUY").Type("LIMIT").TimeInForce("GTC").Quantity(quantityStr).Price(enteringPrice).Do(context.Background())

}

func enterShort(param tradeVar, client *futures.Client) {
	orderBook, _ := client.NewDepthService().Symbol(param.cryptoFullname).Limit(5).Do(context.Background())
	enteringPrice := orderBook.Asks[*param.pOrderBookNum].Price
	enteringPriceFloat, _, _ := orderBook.Asks[*param.pOrderBookNum].Parse()

	accountInfo, _ := client.NewGetAccountService().Do(context.Background())
	balance := accountInfo.TotalWalletBalance
	balanceFloat, _ := strconv.ParseFloat(balance, 64)
	quantity := (balanceFloat * param.leverage * *param.pTradeFactor) / enteringPriceFloat
	quantityStr := strconv.FormatFloat(quantity, 'f', param.quantityDP, 64)

	fmt.Printf("Placing a short order to sell %v %v at %v\n", quantityStr, param.cryptoFullname, enteringPrice)

	client.NewCreateOrderService().Symbol(param.cryptoFullname).
		Side("SELL").Type("LIMIT").TimeInForce("GTC").Quantity(quantityStr).Price(enteringPrice).Do(context.Background())

}

func closePosition(param tradeVar, client *futures.Client) {
	fmt.Println("Closing position")
}

func createTestOrder(param tradeVar, client *futures.Client) {
	orderBook, _ := client.NewDepthService().Symbol(param.cryptoFullname).Limit(5).Do(context.Background())
	enteringPriceFloat, _, _ := orderBook.Bids[*param.pOrderBookNum].Parse()
	testPriceFloat := enteringPriceFloat / 2
	testPrice := strconv.FormatFloat(testPriceFloat, 'f', param.priceDP, 64)

	accountInfo, _ := client.NewGetAccountService().Do(context.Background())
	balance := accountInfo.TotalWalletBalance
	balanceFloat, _ := strconv.ParseFloat(balance, 64)
	quantity := (balanceFloat * param.leverage * *param.pTradeFactor) / testPriceFloat
	quantityStr := strconv.FormatFloat(quantity, 'f', param.quantityDP, 64)

	fmt.Println("Placing a test order. Cancel orders after.")

	client.NewCreateOrderService().Symbol(param.cryptoFullname).
		Side("BUY").Type("LIMIT").TimeInForce("GTC").Quantity(quantityStr).Price(testPrice).Do(context.Background())
}

func round(num float64, decimal int) float64 {
	return math.Round(num*math.Pow10(decimal)) / math.Pow10(decimal)
}

func testRuntime(repeatNum int, decimal int, target func(tradeVar, *futures.Client), param tradeVar, client *futures.Client) {
	var totalTime float64 = 0
	for i := 0; i < repeatNum; i++ {
		startTime := time.Now()
		fmt.Printf("Loop %v\n", i)
		target(param, client)
		timeTakenNano := (time.Since(startTime) / toSecond)
		timeTakenFloat := float64(timeTakenNano)
		timeTakenSecond := round(timeTakenFloat, decimal)
		totalTime += timeTakenSecond
		fmt.Printf("Time taken: %v\n", timeTakenSecond)
		time.Sleep(1 * time.Second)
	}
	var averageTime = totalTime / float64(repeatNum)
	fmt.Printf("Average time: %v\n", averageTime)
	// make this median time
}

func extendClient(pClient **futures.Client) {
	time.Sleep(30 * time.Minute)
	*pClient = binance.NewFuturesClient(credentials.API_KEY, credentials.API_SECRET)
}

func main() {
	param := SetParams()
	fmt.Println()
	fmt.Println("Crypto name: ", param.cryptoFullname)
	fmt.Println("Price decimal place: ", param.priceDP)
	fmt.Println("Quantity decimal place: ", param.quantityDP)
	fmt.Println("Leverage: ", param.leverage)
	fmt.Println("Order book number: ", *param.pOrderBookNum)
	fmt.Println("Trade factor: ", *param.pTradeFactor)
	fmt.Println("Close amount factor: ", *param.pCloseAmtFactor)
	fmt.Println()

	client := binance.NewFuturesClient(credentials.API_KEY, credentials.API_SECRET)
	pClient := &client
	go extendClient(pClient)

	inputScanner := bufio.NewScanner(os.Stdin)
	var command string
	for {
		fmt.Print("Buy(b)/Sell(s)/Close(c)/Set trading factor(tf)/Set closing amount factor(cf)/Set order book number(o)Test runtime(tr) ")
		inputScanner.Scan()
		command = inputScanner.Text()
		if command == "b" {
			enterLong(param, client)
		} else if command == "s" {
			enterShort(param, client)
		} else if command == "c" {
			closePosition(param, client)
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
		} else if command == "tr" {
			testRuntime(5, 5, createTestOrder, param, client)
		} else {
			fmt.Println("Wrong command")
		}
	}
}
