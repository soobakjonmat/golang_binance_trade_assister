package main

import (
	"bufio"
	"context"
	"fmt"
	"golang_binance_trade_assister/credentials"
	"golang_binance_trade_assister/shared_functions"
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

type tradeVar struct {
	cryptoFullname                string
	quantityDP                    int
	pOrderBookNum                 *int
	leverage                      float64
	pTradeFactor, pCloseAmtFactor *float64
}

func SetParams() tradeVar {
	var cryptoNameLower string
	inputScanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Set current trading crypto: ")
	inputScanner.Scan()
	cryptoNameLower = inputScanner.Text()
	var (
		cryptoNameUpper string = strings.ToUpper(cryptoNameLower)
		cryptoFullname  string = cryptoNameUpper + DEFAULT_FIAT_CURRENCY
	)

	client := binance.NewFuturesClient(credentials.API_KEY, credentials.SECRET_KEY)

	var quantityDP int
	exchangeInfo, _ := client.NewExchangeInfoService().Do(context.Background())
	var isValidSymbol bool = false
	for _, item := range exchangeInfo.Symbols {
		if item.Symbol == cryptoFullname {
			isValidSymbol = true
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
	var orderBookNum int = 1
	var oBNPointer *int = &orderBookNum
	var tradeFactor float64 = 0.2
	var tFPointer *float64 = &tradeFactor
	var closingFactor float64 = 1
	var cFPointer *float64 = &closingFactor

	return tradeVar{cryptoFullname, quantityDP, oBNPointer, leverage, tFPointer, cFPointer}
}

func enterLong(param tradeVar, client *futures.Client) {
	orderBook, _ := client.NewDepthService().Symbol(param.cryptoFullname).Limit(5).Do(context.Background())
	enteringPrice := orderBook.Bids[*param.pOrderBookNum].Price
	enteringPriceFloat, _, _ := orderBook.Bids[*param.pOrderBookNum].Parse()

	accountInfo, _ := client.NewGetAccountService().Do(context.Background())
	balance := accountInfo.TotalWalletBalance
	balanceFloat, _ := strconv.ParseFloat(balance, 64)
	quantity := shared_functions.Round((balanceFloat*param.leverage**param.pTradeFactor)/enteringPriceFloat, param.quantityDP)
	quantityStr := strconv.FormatFloat(quantity, 'f', -1, 64)

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
	quantity := shared_functions.Round((balanceFloat*param.leverage**param.pTradeFactor)/enteringPriceFloat, param.quantityDP)
	quantityStr := strconv.FormatFloat(quantity, 'f', -1, 64)

	fmt.Printf("Placing a short order to sell %v %v at %v\n", quantityStr, param.cryptoFullname, enteringPrice)

	client.NewCreateOrderService().Symbol(param.cryptoFullname).
		Side("SELL").Type("LIMIT").TimeInForce("GTC").Quantity(quantityStr).Price(enteringPrice).Do(context.Background())

}

func closePosition(param tradeVar, client *futures.Client) {
	fmt.Println("Closing position")
	orderBook, _ := client.NewDepthService().Symbol(param.cryptoFullname).Limit(5).Do(context.Background())
	closingPrice := orderBook.Asks[*param.pOrderBookNum].Price

	positionRisk, _ := client.NewGetPositionRiskService().Symbol(param.cryptoFullname).Do(context.Background())
	positionAmtStr := positionRisk[0].PositionAmt
	profit := positionRisk[0].UnRealizedProfit
	positionAmtFloat, _ := strconv.ParseFloat(positionAmtStr, 64)
	quantity := math.Abs(shared_functions.Round(positionAmtFloat**param.pCloseAmtFactor, param.quantityDP))
	quantityStr := strconv.FormatFloat(quantity, 'f', -1, 64)

	if positionAmtFloat > 0 {
		fmt.Printf("Placing a closing order at %v. Profit: %v", closingPrice, profit)
		client.NewCreateOrderService().Symbol(param.cryptoFullname).Side("SELL").Type("LIMIT").TimeInForce("GTC").
			Quantity(quantityStr).Price(closingPrice).Do(context.Background())
	} else {
		fmt.Printf("Placing a closing order at %v. Profit: %v", closingPrice, profit)
		client.NewCreateOrderService().Symbol(param.cryptoFullname).Side("BUY").Type("LIMIT").TimeInForce("GTC").
			Quantity(quantityStr).Price(closingPrice).Do(context.Background())
	}
}

func updateClient(pClient **futures.Client) {
	time.Sleep(30 * time.Minute)
	*pClient = binance.NewFuturesClient(credentials.API_KEY, credentials.SECRET_KEY)
}

func main() {
	param := SetParams()
	fmt.Println()
	fmt.Println("Crypto name: ", param.cryptoFullname)
	fmt.Println("Quantity decimal place: ", param.quantityDP)
	fmt.Println("Leverage: ", param.leverage)
	fmt.Println("Order book number: ", *param.pOrderBookNum)
	fmt.Println("Trade factor: ", *param.pTradeFactor)
	fmt.Println("Close amount factor: ", *param.pCloseAmtFactor)
	fmt.Println()

	client := binance.NewFuturesClient(credentials.API_KEY, credentials.SECRET_KEY)
	pClient := &client
	go updateClient(pClient)

	inputScanner := bufio.NewScanner(os.Stdin)
	var command string
	for {
		fmt.Print("Enter Command: ")
		inputScanner.Scan()
		command = inputScanner.Text()
		if command == "b" {
			enterLong(param, client)
			fmt.Println()
		} else if command == "s" {
			enterShort(param, client)
			fmt.Println()
		} else if command == "c" {
			closePosition(param, client)
			fmt.Println()
		} else if strings.Split(command, " ")[0] == "tf" {
			*param.pTradeFactor, _ = strconv.ParseFloat(strings.Split(command, " ")[1], 64)
			*param.pTradeFactor /= 100
			fmt.Printf("Trade factor: %v\n", *param.pTradeFactor)
			fmt.Println()
		} else if strings.Split(command, " ")[0] == "cf" {
			*param.pCloseAmtFactor, _ = strconv.ParseFloat(strings.Split(command, " ")[1], 64)
			*param.pCloseAmtFactor /= 100
			fmt.Printf("Close amount factor: %v\n", *param.pCloseAmtFactor)
			fmt.Println()
		} else if strings.Split(command, " ")[0] == "o" {
			*param.pOrderBookNum, _ = strconv.Atoi(strings.Split(command, " ")[1])
			fmt.Printf("Order book number: %v\n", *param.pOrderBookNum)
			fmt.Println()
		} else {
			fmt.Println("Wrong command")
			fmt.Println()
		}
	}
}
