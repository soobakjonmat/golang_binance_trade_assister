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
	"github.com/gonutz/wui/v2"
)

const (
	DEFAULT_FIAT_CURRENCY = "USDT"

	BG_COLOR    = 0x201A18
	LONG_COLOR  = 0x77C002
	SHORT_COLOR = 0x6049F8

	HEIGHT_BIG    = 34
	HEIGHT_MEDIUM = 28
	HEIGHT_SMALL  = 23
	HEIGHT_TINY   = 18
)

var (
	cryptoFullname string = ""
	quantityDP     int
	orderBookNum   int = 0
	leverage       float64
	tradeFactor    float64 = 0.1
	closingFactor  float64 = 1
	USDT_index     int

	client             *futures.Client
	window             *wui.Window
	command            string
	commandEntry       *wui.EditLine
	tradeFactorEntry   *wui.EditLine
	tradeFactorDisplay *wui.Label

	BINANCE_FONT_BIG, _    = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 32})
	BINANCE_FONT_MEDIUM, _ = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 26})
	BINANCE_FONT_SMALL, _  = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 18})
	BINANCE_FONT_TINY, _   = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 14})
)

func enterLong() {
	accountInfo, _ := client.NewGetAccountService().Do(context.Background())
	balance := accountInfo.Assets[USDT_index].WalletBalance
	balanceFloat, _ := strconv.ParseFloat(balance, 64)

	orderBook, _ := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
	enteringPrice := orderBook.Bids[orderBookNum].Price
	enteringPriceFloat, _, _ := orderBook.Bids[orderBookNum].Parse()

	quantity := shared_functions.Round((balanceFloat*leverage*tradeFactor)/enteringPriceFloat, quantityDP)
	quantityStr := strconv.FormatFloat(quantity, 'f', -1, 64)

	client.NewCreateOrderService().Symbol(cryptoFullname).
		Side("BUY").Type("LIMIT").TimeInForce("GTC").Quantity(quantityStr).Price(enteringPrice).Do(context.Background())
}

func enterShort() {
	accountInfo, _ := client.NewGetAccountService().Do(context.Background())
	balance := accountInfo.Assets[USDT_index].WalletBalance
	balanceFloat, _ := strconv.ParseFloat(balance, 64)

	orderBook, _ := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
	enteringPrice := orderBook.Asks[orderBookNum].Price
	enteringPriceFloat, _, _ := orderBook.Asks[orderBookNum].Parse()

	quantity := shared_functions.Round((balanceFloat*leverage*tradeFactor)/enteringPriceFloat, quantityDP)
	quantityStr := strconv.FormatFloat(quantity, 'f', -1, 64)

	client.NewCreateOrderService().Symbol(cryptoFullname).
		Side("SELL").Type("LIMIT").TimeInForce("GTC").Quantity(quantityStr).Price(enteringPrice).Do(context.Background())
}

func closePosition() {
	positionRisk, _ := client.NewGetPositionRiskService().Symbol(cryptoFullname).Do(context.Background())
	positionAmtStr := positionRisk[0].PositionAmt
	positionAmtFloat, _ := strconv.ParseFloat(positionAmtStr, 64)

	orderBook, _ := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
	closingPrice := orderBook.Asks[orderBookNum].Price

	quantity := math.Abs(shared_functions.Round(positionAmtFloat*closingFactor, quantityDP))
	quantityStr := strconv.FormatFloat(quantity, 'f', -1, 64)

	if positionAmtFloat > 0 {
		client.NewCreateOrderService().Symbol(cryptoFullname).Side("SELL").Type("LIMIT").TimeInForce("GTC").
			Quantity(quantityStr).Price(closingPrice).Do(context.Background())
	} else {
		client.NewCreateOrderService().Symbol(cryptoFullname).Side("BUY").Type("LIMIT").TimeInForce("GTC").
			Quantity(quantityStr).Price(closingPrice).Do(context.Background())
	}
}

func cancelOrder() {
	client.NewCancelAllOpenOrdersService().Symbol(cryptoFullname).Do(context.Background())
}

func runCommand() {
	// make order or make changes in order
	if commandEntry.HasFocus() {
		command = commandEntry.Text()
		go commandEntry.SetText("")
		if command == "t" {
			testClient := binance.NewClient(credentials.API_KEY, credentials.SECRET_KEY)
			shared_functions.TestRuntime(4, 6, shared_functions.MakeTestOrder, testClient)
		} else if command == "b" {
			enterLong()
		} else if command == "s" {
			enterShort()
		} else if command == "cl" {
			closePosition()
		} else if command == "cc" {
			cancelOrder()
		}
	}
	// update trade factor
	if tradeFactorEntry.HasFocus() {
		num, _ := strconv.ParseFloat(tradeFactorEntry.Text(), 64)
		tradeFactorEntry.SetText("")
		tradeFactor = num / 100
		tradeFactorStr := strconv.FormatFloat(tradeFactor, 'f', -1, 64)
		tradeFactorDisplay.SetText(tradeFactorStr)
		fmt.Printf("Trade Factor: %v", tradeFactor)
	}
}

func updateClient() {
	for {
		time.Sleep(30 * time.Minute)
		client = binance.NewFuturesClient(credentials.API_KEY, credentials.SECRET_KEY)
	}
}

func createNewLabel(font *wui.Font, width int, height int, text string, xPos int, yPos int) *wui.Label {
	newLabel := wui.NewLabel()
	newLabel.SetFont(font)
	newLabel.SetSize(width, height)
	newLabel.SetText(text)
	newLabel.SetX(xPos)
	newLabel.SetY(yPos)
	window.Add(newLabel)
	return newLabel
}

func createNewEditLine(font *wui.Font, width int, height int, xPos int, yPos int) *wui.EditLine {
	newEditLine := wui.NewEditLine()
	newEditLine.SetFont(font)
	newEditLine.SetSize(width, height)
	newEditLine.SetX(xPos)
	newEditLine.SetY(yPos)
	window.Add(newEditLine)
	return newEditLine
}

func main() {
	client = binance.NewFuturesClient(credentials.API_KEY, credentials.SECRET_KEY)
	go updateClient()

	fmt.Print("Enter crypto name: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	cryptoFullname = strings.ToUpper(scanner.Text()) + DEFAULT_FIAT_CURRENCY

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

	accountInfo, _ := client.NewGetAccountService().Do(context.Background())
	for _, item := range accountInfo.Positions {
		if item.Symbol == cryptoFullname {
			leverage, _ = strconv.ParseFloat(item.Leverage, 64)
		}
	}
	for idx, item := range accountInfo.Assets {
		if item.Asset == DEFAULT_FIAT_CURRENCY {
			USDT_index = idx
		}
	}

	window = wui.NewWindow()
	window.SetTitle("Trade Assister")
	window.SetPosition(1260, 200)
	window.SetSize(260, 300)
	window.SetResizable(false)
	window.SetBackground(wui.Color(BG_COLOR))

	symbolLabel := createNewLabel(BINANCE_FONT_BIG, window.InnerWidth(), HEIGHT_BIG, cryptoFullname, 0, 0)
	symbolLabel.SetAlignment(wui.AlignCenter)
	symbolLabel.SetX(window.InnerWidth()/2 - symbolLabel.Width()/2)

	tradeFactorYPos := 170
	tradeFactorLabel := createNewLabel(BINANCE_FONT_TINY, 70, HEIGHT_TINY, "Trade Factor", 10, tradeFactorYPos)
	tradeFactorLabel.SetAlignment(wui.AlignCenter)
	tradeFactorStr := strconv.FormatFloat(tradeFactor, 'f', -1, 64)
	tradeFactorDisplay = createNewLabel(BINANCE_FONT_TINY, 30, HEIGHT_TINY, tradeFactorStr, 164, tradeFactorYPos)
	tradeFactorDisplay.SetAlignment(wui.AlignCenter)
	tradeFactorEntry = createNewEditLine(BINANCE_FONT_TINY, 30, HEIGHT_TINY, 205, tradeFactorYPos)

	commandYPos := 200
	commandLabel := createNewLabel(BINANCE_FONT_SMALL, 120, HEIGHT_SMALL, "Enter Command:", 0, commandYPos)
	commandLabel.SetX(window.InnerWidth()/2 - commandLabel.Width()/2)
	commandLabel.SetAlignment(wui.AlignCenter)

	commandEntry = createNewEditLine(BINANCE_FONT_SMALL, 40, HEIGHT_SMALL, 0, commandYPos+30)
	commandEntry.SetX(window.InnerWidth()/2 - commandEntry.Width()/2)

	window.SetShortcut(runCommand, wui.KeyReturn)
	window.Show()
}
