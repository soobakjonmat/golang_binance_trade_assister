package main

import (
	"context"
	"golang_binance_trade_assister/credentials"
	"golang_binance_trade_assister/shared_functions"
	"log"
	"math"

	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/gonutz/wui/v2"
	"github.com/xuri/excelize/v2"
)

const (
	DEFAULT_FIAT_CURRENCY = "BUSD"

	BG_COLOR    = 0x201A18
	LONG_COLOR  = 0x77C002
	SHORT_COLOR = 0x6049F8

	HEIGHT_BIG   = 27
	HEIGHT_SMALL = 23
	HEIGHT_TINY  = 18
)

var (
	cryptoFullname string = ""
	quantityDP     int
	orderBookIdx   int = 0
	leverage       float64
	tradeFactor    float64 = 0.1
	closingFactor  float64 = 1

	fiatIndex int

	balanceBefore float64

	client *futures.Client

	window *wui.Window

	symbolLabel *wui.Label

	overallProfitLabel *wui.Label
	positionLabel      *wui.Label
	profitLabel        *wui.Label
	marginInputLabel   *wui.Label

	tradeFactorLabel   *wui.Label
	tradeFactorDisplay *wui.Label
	tradeFactorEntry   *wui.EditLine

	closingFactorLabel   *wui.Label
	closingFactorDisplay *wui.Label
	closingFactorEntry   *wui.EditLine

	orderBookIdxLabel   *wui.Label
	orderBookIdxDisplay *wui.Label
	orderBookIdxEntry   *wui.EditLine

	commandLabel *wui.Label
	commandEntry *wui.EditLine

	SYMBOL_FONT, _        = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 31})
	BINANCE_FONT_BIG, _   = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 26})
	BINANCE_FONT_SMALL, _ = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 19})
	BINANCE_FONT_TINY, _  = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 14})
)

func enterLong() {
	accountInfo, _ := client.NewGetAccountService().Do(context.Background())
	balance := accountInfo.Assets[fiatIndex].WalletBalance
	balanceFloat, _ := strconv.ParseFloat(balance, 64)

	orderBook, _ := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
	enteringPrice := orderBook.Bids[orderBookIdx].Price
	enteringPriceFloat, _, _ := orderBook.Bids[orderBookIdx].Parse()

	quantity := shared_functions.Round((balanceFloat*leverage*tradeFactor)/enteringPriceFloat, quantityDP)
	quantityStr := strconv.FormatFloat(quantity, 'f', -1, 64)

	client.NewCreateOrderService().Symbol(cryptoFullname).
		Side("BUY").Type("LIMIT").TimeInForce("GTC").Quantity(quantityStr).Price(enteringPrice).Do(context.Background())
}

func enterShort() {
	accountInfo, _ := client.NewGetAccountService().Do(context.Background())
	balance := accountInfo.Assets[fiatIndex].WalletBalance
	balanceFloat, _ := strconv.ParseFloat(balance, 64)

	orderBook, _ := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
	enteringPrice := orderBook.Asks[orderBookIdx].Price
	enteringPriceFloat, _, _ := orderBook.Asks[orderBookIdx].Parse()

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
	closingPrice := orderBook.Asks[orderBookIdx].Price

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
		command := commandEntry.Text()
		go commandEntry.SetText("")
		if command == "t" {
			testClient := binance.NewClient(credentials.API_KEY, credentials.SECRET_KEY)
			shared_functions.TestRuntime(11, 6, shared_functions.MakeTestOrder, testClient)
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
	if tradeFactorEntry.HasFocus() {
		num, _ := strconv.ParseFloat(tradeFactorEntry.Text(), 64)
		tradeFactorEntry.SetText("")
		tradeFactor = num / 100
		tradeFactorStr := strconv.FormatFloat(tradeFactor, 'f', -1, 64)
		tradeFactorDisplay.SetText(tradeFactorStr)
	}
	if closingFactorEntry.HasFocus() {
		num, _ := strconv.ParseFloat(tradeFactorEntry.Text(), 64)
		closingFactorEntry.SetText("")
		closingFactor = num / 100
		closingFactorStr := strconv.FormatFloat(closingFactor, 'f', -1, 64)
		closingFactorDisplay.SetText(closingFactorStr)
	}
	if orderBookIdxEntry.HasFocus() {
		num, _ := strconv.Atoi(orderBookIdxEntry.Text())
		orderBookIdxEntry.SetText("")
		orderBookIdx = num
		orderBookIdxStr := strconv.Itoa(orderBookIdx)
		tradeFactorDisplay.SetText(orderBookIdxStr)
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

func initialize() {
	window.SetSize(260, 322)

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
			fiatIndex = idx
		}
	}

	symbolLabel = createNewLabel(SYMBOL_FONT, 130, 33, cryptoFullname, 0, 0)
	symbolLabel.SetAlignment(wui.AlignCenter)
	symbolLabel.SetX(window.InnerWidth()/2 - symbolLabel.Width()/2)

	overallProfitLabel = createNewLabel(BINANCE_FONT_BIG, 200, HEIGHT_BIG, "Overall Profit: 0.00%", 0, 38)
	overallProfitLabel.SetX(window.InnerWidth()/2 - overallProfitLabel.Width()/2)
	overallProfitLabel.SetAlignment(wui.AlignCenter)

	positionLabel = createNewLabel(BINANCE_FONT_BIG, 180, HEIGHT_BIG, "Not in Position", 0, 74)
	positionLabel.SetX(window.InnerWidth()/2 - positionLabel.Width()/2)
	positionLabel.SetAlignment(wui.AlignCenter)

	marginInputLabel = createNewLabel(BINANCE_FONT_BIG, 180, HEIGHT_BIG, "Margin Input: nil", 0, 102)
	marginInputLabel.SetX(window.InnerWidth()/2 - marginInputLabel.Width()/2)
	marginInputLabel.SetAlignment(wui.AlignCenter)

	profitLabel = createNewLabel(BINANCE_FONT_BIG, 180, HEIGHT_BIG, "Profit: nil", 0, 130)
	profitLabel.SetX(window.InnerWidth()/2 - profitLabel.Width()/2)
	profitLabel.SetAlignment(wui.AlignCenter)

	labelXPos := 10
	displayEntryWidth := 30
	displayXPos := 164
	entryXPos := 205

	tradeFactorYPos := 160
	tradeFactorLabel = createNewLabel(BINANCE_FONT_TINY, 70, HEIGHT_TINY, "Trade Factor", labelXPos, tradeFactorYPos)
	tradeFactorLabel.SetAlignment(wui.AlignCenter)
	tradeFactorStr := strconv.FormatFloat(tradeFactor, 'f', -1, 64)
	tradeFactorDisplay = createNewLabel(BINANCE_FONT_TINY, displayEntryWidth, HEIGHT_TINY, tradeFactorStr, displayXPos, tradeFactorYPos)
	tradeFactorDisplay.SetAlignment(wui.AlignCenter)
	tradeFactorEntry = createNewEditLine(BINANCE_FONT_TINY, displayEntryWidth, HEIGHT_TINY, entryXPos, tradeFactorYPos)

	closingFactorYPos := 180
	closingFactorLabel = createNewLabel(BINANCE_FONT_TINY, 80, HEIGHT_TINY, "Closing Factor", labelXPos, closingFactorYPos)
	closingFactorLabel.SetAlignment(wui.AlignCenter)
	closingFactorStr := strconv.FormatFloat(closingFactor, 'f', -1, 64)
	closingFactorDisplay = createNewLabel(BINANCE_FONT_TINY, displayEntryWidth, HEIGHT_TINY, closingFactorStr, displayXPos, closingFactorYPos)
	closingFactorDisplay.SetAlignment(wui.AlignCenter)
	closingFactorEntry = createNewEditLine(BINANCE_FONT_TINY, displayEntryWidth, HEIGHT_TINY, entryXPos, closingFactorYPos)

	orderBookIdxYPos := 200
	orderBookIdxLabel = createNewLabel(BINANCE_FONT_TINY, 95, HEIGHT_TINY, "Order Book Index", labelXPos, orderBookIdxYPos)
	orderBookIdxLabel.SetAlignment(wui.AlignCenter)
	orderBookIdxStr := strconv.Itoa(orderBookIdx)
	orderBookIdxDisplay = createNewLabel(BINANCE_FONT_TINY, displayEntryWidth, HEIGHT_TINY, orderBookIdxStr, displayXPos, orderBookIdxYPos)
	orderBookIdxDisplay.SetAlignment(wui.AlignCenter)
	orderBookIdxEntry = createNewEditLine(BINANCE_FONT_TINY, displayEntryWidth, HEIGHT_TINY, entryXPos, orderBookIdxYPos)

	commandYPos := 225
	commandLabel = createNewLabel(BINANCE_FONT_SMALL, 120, HEIGHT_SMALL, "Enter Command:", 0, commandYPos)
	commandLabel.SetX(window.InnerWidth()/2 - commandLabel.Width()/2)
	commandLabel.SetAlignment(wui.AlignCenter)

	commandEntry = createNewEditLine(BINANCE_FONT_SMALL, 70, HEIGHT_SMALL, 0, commandYPos+27)
	commandEntry.SetX(window.InnerWidth()/2 - commandEntry.Width()/2)

	window.SetShortcut(runCommand, wui.KeyReturn)
}

func updateInfo() {
	for {
		// Balance
		accountInfo, _ := client.NewGetAccountService().Do(context.Background())
		assetInfo := accountInfo.Assets[fiatIndex]
		totalbalanceStr := assetInfo.WalletBalance
		totalBalance, _ := strconv.ParseFloat(totalbalanceStr, 64)
		totalBalance = shared_functions.Round(totalBalance, 2)
		// Overall profit
		overallProfit := shared_functions.Round((totalBalance/balanceBefore-1)*100, 2)
		overallProfitStr := strconv.FormatFloat(overallProfit, 'f', -1, 64)
		overallProfitLabel.SetText("Overall Profit: " + overallProfitStr + "%")
		// Position info
		positionInfo, _ := client.NewGetPositionRiskService().Symbol(cryptoFullname).Do(context.Background())
		positionAmtStr := positionInfo[0].PositionAmt
		positionAmt, _ := strconv.ParseFloat(positionAmtStr, 64)
		if positionAmt == 0 {
			positionLabel.SetText("Not in Position")
			marginInputLabel.SetText("Margin Input: nil")
			profitLabel.SetText("Profit: nil")
		} else {
			// Position label
			if positionAmt > 0 {
				positionLabel.SetText("Long")
			} else {
				positionLabel.SetText("Short")
			}
			// Margin input label
			inputMarginStr := assetInfo.InitialMargin
			inputMargin, _ := strconv.ParseFloat(inputMarginStr, 64)
			marginInputRatio := inputMargin / totalBalance
			// Preventing rounding error
			if marginInputRatio > 1 {
				marginInputRatio = 1
			}
			marginInputPercentage := shared_functions.Round(marginInputRatio*100, 2)
			marginInputPercentageStr := strconv.FormatFloat(marginInputPercentage, 'f', -1, 64)
			marginInputLabel.SetText("Margin Input: " + marginInputPercentageStr + "%")
			// profit label
			profitStr := positionInfo[0].UnRealizedProfit
			profit, _ := strconv.ParseFloat(profitStr, 64)
			profitRounded := shared_functions.Round(math.Abs(profit), 2)
			profitRoundedStr := strconv.FormatFloat(profitRounded, 'f', -1, 64)
			profitPercentage := shared_functions.Round(profit*100/totalBalance, 2)
			profitPercentageStr := strconv.FormatFloat(profitPercentage, 'f', -1, 64)
			if profit > 0 {
				profitLabel.SetText("Profit: $" + profitRoundedStr + " (" + profitPercentageStr + "%)")
			} else if profit < 0 {
				profitLabel.SetText("Profit: -$" + profitRoundedStr + " (" + profitPercentageStr + "%)")
			} else {
				profitLabel.SetText("Profit: $0.00 (0.00%)")
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func main() {
	client = binance.NewFuturesClient(credentials.API_KEY, credentials.SECRET_KEY)
	go updateClient()

	sheet, _ := excelize.OpenFile("/Users/Isac/Desktop/Programming_stuff/Trade Record.xlsx")
	cols, _ := sheet.GetCols("Balance Record")
	targetIndex := 0
	for idx, rowCell := range cols[0] {
		if rowCell == "" {
			targetIndex = idx
			break
		}
	}
	targetIndexStr := strconv.Itoa(targetIndex)
	value, _ := sheet.GetCellValue("Balance Record", "B"+targetIndexStr)
	balanceBefore, _ = strconv.ParseFloat(value, 64)

	window = wui.NewWindow()
	window.SetTitle("Trade Assister")
	window.SetPosition(1260, 200)
	window.SetResizable(false)
	window.SetSize(260, 140)
	window.SetBackground(wui.Color(BG_COLOR))

	instruction := createNewLabel(BINANCE_FONT_SMALL, 160, HEIGHT_SMALL, "Enter Crypto Name:", 0, 20)
	instruction.SetX(window.InnerWidth()/2 - instruction.Width()/2)
	instruction.SetAlignment(wui.AlignCenter)

	cryptoNameEntry := createNewEditLine(BINANCE_FONT_SMALL, 90, HEIGHT_SMALL, 0, 60)
	cryptoNameEntry.SetX(window.InnerWidth()/2 - cryptoNameEntry.Width()/2)

	window.SetShortcut(func() {
		if cryptoNameEntry.HasFocus() {
			cryptoFullname = strings.ToUpper(cryptoNameEntry.Text()) + DEFAULT_FIAT_CURRENCY
			window.Remove(instruction)
			window.Remove(cryptoNameEntry)
			initialize()
			go updateInfo()
		}
	}, wui.KeyReturn)

	window.Show()
}
