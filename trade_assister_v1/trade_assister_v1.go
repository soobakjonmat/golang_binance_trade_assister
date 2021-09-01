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
	"github.com/xuri/excelize/v2"
)

const (
	textFilePath  = "/Users/Isac/Desktop/Programming_stuff/DEFAULT_FIAT_CURRENCY.txt"
	excelFilePath = "/Users/Isac/Desktop/Programming_stuff/Trade Record.xlsx"
	iconFilePath  = "/Users/Isac/Desktop/Programming_stuff/golang_binance_trade_assister/gopher.ico"

	HEIGHT_BIG   = 27
	HEIGHT_SMALL = 23
	HEIGHT_TINY  = 18
)

var (
	cryptoFullname        string = ""
	cryptoName            string = ""
	DEFAULT_FIAT_CURRENCY string = ""

	BG_COLOR_24_BGR = 0x201A18
	BG_COLOR_RGB    = wui.RGB(24, 26, 32)
	LONG_COLOR      = wui.RGB(2, 192, 119)
	SHORT_COLOR     = wui.RGB(248, 73, 96)
	WHITE_COLOR     = wui.RGB(255, 255, 255)
	BLACK_COLOR     = wui.RGB(0, 0, 0)

	quantityDP    int
	orderBookIdx  int = 0
	leverage      float64
	tradeFactor   float64 = 0.05
	closingFactor float64 = 1

	specificAmt    string = ""
	useTradeFactor bool   = true

	fiatIndex int

	balanceBefore float64

	client *futures.Client

	accountInfo  *futures.Account
	positionInfo []*futures.PositionRisk

	isInitialized bool = false

	window *wui.Window

	symbolLabel *wui.Label

	overallProfitLabel *wui.Label
	positionLabel      *wui.Label
	profitLabel        *wui.Label
	marginInputLabel   *wui.Label

	tradeFactorDisplay *wui.Label
	tradeFactorEntry   *wui.EditLine

	closingFactorDisplay *wui.Label
	closingFactorEntry   *wui.EditLine

	// Bid: Buy
	// Ask: Sell
	orderBookIdxDisplay *wui.Label
	orderBookIdxEntry   *wui.EditLine

	commandEntry *wui.EditLine

	specificAmtOrderEntry *wui.EditLine

	SYMBOL_FONT, _        = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 31})
	BINANCE_FONT_BIG, _   = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 26})
	BINANCE_FONT_SMALL, _ = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 19})
	BINANCE_FONT_TINY, _  = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 14})
)

func enterLong() {
	if useTradeFactor {
		accountInfo, err := client.NewGetAccountService().Do(context.Background())
		handleError(err)
		balance := accountInfo.Assets[fiatIndex].WalletBalance
		balanceFloat := stringToFloat(balance)

		orderBook, err := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
		handleError(err)
		enteringPrice := orderBook.Bids[orderBookIdx].Price
		enteringPriceFloat, _, _ := orderBook.Bids[orderBookIdx].Parse()

		quantity := shared_functions.Round((balanceFloat*leverage*tradeFactor)/enteringPriceFloat, quantityDP)
		quantityStr := floatToString(quantity)

		client.NewCreateOrderService().Symbol(cryptoFullname).
			Side("BUY").Type("LIMIT").TimeInForce("GTC").Quantity(quantityStr).Price(enteringPrice).Do(context.Background())
	} else {
		orderBook, err := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
		handleError(err)
		enteringPrice := orderBook.Bids[orderBookIdx].Price

		client.NewCreateOrderService().Symbol(cryptoFullname).
			Side("BUY").Type("LIMIT").TimeInForce("GTC").Quantity(specificAmt).Price(enteringPrice).Do(context.Background())
	}
}

func enterShort() {
	if useTradeFactor {
		accountInfo, err := client.NewGetAccountService().Do(context.Background())
		handleError(err)
		balance := accountInfo.Assets[fiatIndex].WalletBalance
		balanceFloat := stringToFloat(balance)

		orderBook, err := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
		handleError(err)
		enteringPrice := orderBook.Asks[orderBookIdx].Price
		enteringPriceFloat, _, _ := orderBook.Asks[orderBookIdx].Parse()

		quantity := shared_functions.Round((balanceFloat*leverage*tradeFactor)/enteringPriceFloat, quantityDP)
		quantityStr := floatToString(quantity)

		client.NewCreateOrderService().Symbol(cryptoFullname).
			Side("SELL").Type("LIMIT").TimeInForce("GTC").Quantity(quantityStr).Price(enteringPrice).Do(context.Background())
	} else {
		orderBook, err := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
		handleError(err)
		enteringPrice := orderBook.Asks[orderBookIdx].Price

		client.NewCreateOrderService().Symbol(cryptoFullname).
			Side("SELL").Type("LIMIT").TimeInForce("GTC").Quantity(specificAmt).Price(enteringPrice).Do(context.Background())
	}
}

func closePosition() {
	if useTradeFactor {
		positionRisk, err := client.NewGetPositionRiskService().Symbol(cryptoFullname).Do(context.Background())
		handleError(err)
		positionAmtFloat := stringToFloat(positionRisk[0].PositionAmt)

		orderBook, err := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
		handleError(err)

		quantity := math.Abs(shared_functions.Round(positionAmtFloat*closingFactor, quantityDP))
		quantityStr := floatToString(quantity)

		if positionAmtFloat > 0 {
			closingPrice := orderBook.Asks[orderBookIdx].Price
			client.NewCreateOrderService().Symbol(cryptoFullname).Side("SELL").Type("LIMIT").TimeInForce("GTC").
				Quantity(quantityStr).Price(closingPrice).Do(context.Background())
		} else {
			closingPrice := orderBook.Bids[orderBookIdx].Price
			client.NewCreateOrderService().Symbol(cryptoFullname).Side("BUY").Type("LIMIT").TimeInForce("GTC").
				Quantity(quantityStr).Price(closingPrice).Do(context.Background())
		}
	} else {
		positionRisk, err := client.NewGetPositionRiskService().Symbol(cryptoFullname).Do(context.Background())
		handleError(err)
		positionAmtFloat := stringToFloat(positionRisk[0].PositionAmt)

		orderBook, err := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
		handleError(err)

		if positionAmtFloat > 0 {
			closingPrice := orderBook.Asks[orderBookIdx].Price
			client.NewCreateOrderService().Symbol(cryptoFullname).Side("SELL").Type("LIMIT").TimeInForce("GTC").
				Quantity(specificAmt).Price(closingPrice).Do(context.Background())
		} else {
			closingPrice := orderBook.Bids[orderBookIdx].Price
			client.NewCreateOrderService().Symbol(cryptoFullname).Side("BUY").Type("LIMIT").TimeInForce("GTC").
				Quantity(specificAmt).Price(closingPrice).Do(context.Background())
		}
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
	} else if tradeFactorEntry.HasFocus() && tradeFactorEntry.Text() != "" {
		useTradeFactor = true
		tradeFactor = stringToFloat(tradeFactorEntry.Text())
		tradeFactorDisplay.SetText(tradeFactorEntry.Text())
		tradeFactorEntry.SetText("")
	} else if closingFactorEntry.HasFocus() && closingFactorEntry.Text() != "" {
		closingFactor = stringToFloat(closingFactorEntry.Text())
		closingFactorDisplay.SetText(closingFactorEntry.Text())
		closingFactorEntry.SetText("")
	} else if orderBookIdxEntry.HasFocus() && orderBookIdxEntry.Text() != "" {
		orderBookIdx, _ = strconv.Atoi(orderBookIdxEntry.Text())
		orderBookIdxDisplay.SetText(orderBookIdxEntry.Text())
		orderBookIdxEntry.SetText("")
	} else if specificAmtOrderEntry.HasFocus() && specificAmtOrderEntry.Text() != "" {
		useTradeFactor = false
		specificAmt = specificAmtOrderEntry.Text()
		tradeFactorDisplay.SetText(specificAmtOrderEntry.Text() + " " + cryptoName)
		specificAmtOrderEntry.SetText("")
	}
}

func updateClient() {
	for {
		time.Sleep(30 * time.Minute)
		client = binance.NewFuturesClient(credentials.API_KEY, credentials.SECRET_KEY)
	}
}

func createNewLabel(text string, xPos int, yPos int, width int, height int, font *wui.Font) *wui.Label {
	newLabel := wui.NewLabel()
	newLabel.SetText(text)
	newLabel.SetSize(width, height)
	newLabel.SetPosition(xPos, yPos)
	newLabel.SetFont(font)
	window.Add(newLabel)
	return newLabel
}

func createNewEditLine(xPos int, yPos int, width int, height int, font *wui.Font) *wui.EditLine {
	newEditLine := wui.NewEditLine()
	newEditLine.SetSize(width, height)
	newEditLine.SetPosition(xPos, yPos)
	newEditLine.SetFont(font)
	window.Add(newEditLine)
	return newEditLine
}

func initialize() {
	window.SetSize(260, 322)

	exchangeInfo, err := client.NewExchangeInfoService().Do(context.Background())
	handleError(err)
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

	accInfo, err := client.NewGetAccountService().Do(context.Background())
	handleError(err)
	for _, item := range accInfo.Positions {
		if item.Symbol == cryptoFullname {
			leverage = stringToFloat(item.Leverage)
		}
	}
	for idx, item := range accInfo.Assets {
		if item.Asset == DEFAULT_FIAT_CURRENCY {
			fiatIndex = idx
		}
	}

	symbolLabel = createNewLabel(cryptoFullname, 0, 0, 130, 33, SYMBOL_FONT)
	symbolLabel.SetAlignment(wui.AlignCenter)
	symbolLabel.SetX(window.InnerWidth()/2 - symbolLabel.Width()/2)

	overallProfitLabel = createNewLabel("Overall Profit: 0.00%", 0, 38, 200, HEIGHT_BIG, BINANCE_FONT_BIG)
	overallProfitLabel.SetX(window.InnerWidth()/2 - overallProfitLabel.Width()/2)
	overallProfitLabel.SetAlignment(wui.AlignCenter)

	positionLabel = createNewLabel("Not in Position", 0, 74, 180, HEIGHT_BIG, BINANCE_FONT_BIG)
	positionLabel.SetX(window.InnerWidth()/2 - positionLabel.Width()/2)
	positionLabel.SetAlignment(wui.AlignCenter)

	marginInputLabel = createNewLabel("Margin Input: nil", 0, 102, 240, HEIGHT_BIG, BINANCE_FONT_BIG)
	marginInputLabel.SetX(window.InnerWidth()/2 - marginInputLabel.Width()/2)
	marginInputLabel.SetAlignment(wui.AlignCenter)

	profitLabel = createNewLabel("Profit: nil", 0, 130, 240, HEIGHT_BIG, BINANCE_FONT_BIG)
	profitLabel.SetX(window.InnerWidth()/2 - profitLabel.Width()/2)
	profitLabel.SetAlignment(wui.AlignCenter)

	labelXPos := 10
	displayEntryWidth := 30
	displayXPos := 164
	entryXPos := 205

	tradeFactorYPos := 160
	tradeFactorLabel := createNewLabel("Trade Factor", labelXPos, tradeFactorYPos, 70, HEIGHT_TINY, BINANCE_FONT_TINY)
	tradeFactorLabel.SetAlignment(wui.AlignCenter)
	tradeFactorStr := floatToString(tradeFactor)
	tradeFactorDisplay = createNewLabel(tradeFactorStr, displayXPos-40, tradeFactorYPos, displayEntryWidth+40, HEIGHT_TINY, BINANCE_FONT_TINY)
	tradeFactorDisplay.SetAlignment(wui.AlignCenter)
	tradeFactorEntry = createNewEditLine(entryXPos, tradeFactorYPos, displayEntryWidth, HEIGHT_TINY, BINANCE_FONT_TINY)

	closingFactorYPos := 180
	closingFactorLabel := createNewLabel("Closing Factor", labelXPos, closingFactorYPos, 80, HEIGHT_TINY, BINANCE_FONT_TINY)
	closingFactorLabel.SetAlignment(wui.AlignCenter)
	closingFactorStr := floatToString(closingFactor)
	closingFactorDisplay = createNewLabel(closingFactorStr, displayXPos, closingFactorYPos, displayEntryWidth, HEIGHT_TINY, BINANCE_FONT_TINY)
	closingFactorDisplay.SetAlignment(wui.AlignCenter)
	closingFactorEntry = createNewEditLine(entryXPos, closingFactorYPos, displayEntryWidth, HEIGHT_TINY, BINANCE_FONT_TINY)

	orderBookIdxYPos := 200
	orderBookIdxLabel := createNewLabel("Order Book Index", labelXPos, orderBookIdxYPos, 95, HEIGHT_TINY, BINANCE_FONT_TINY)
	orderBookIdxLabel.SetAlignment(wui.AlignCenter)
	orderBookIdxStr := strconv.Itoa(orderBookIdx)
	orderBookIdxDisplay = createNewLabel(orderBookIdxStr, displayXPos, orderBookIdxYPos, displayEntryWidth, HEIGHT_TINY, BINANCE_FONT_TINY)
	orderBookIdxDisplay.SetAlignment(wui.AlignCenter)
	orderBookIdxEntry = createNewEditLine(entryXPos, orderBookIdxYPos, displayEntryWidth, HEIGHT_TINY, BINANCE_FONT_TINY)

	commandYPos := 225
	commandLabel := createNewLabel("Command:", 10, commandYPos, 90, HEIGHT_SMALL, BINANCE_FONT_SMALL)
	commandLabel.SetAlignment(wui.AlignCenter)
	commandEntry = createNewEditLine(20, commandYPos+27, 70, HEIGHT_SMALL, BINANCE_FONT_SMALL)

	specificAmtLabel := createNewLabel("Specific Amount:", 130, commandYPos, 110, HEIGHT_SMALL, BINANCE_FONT_SMALL)
	specificAmtLabel.SetAlignment(wui.AlignCenter)
	specificAmtOrderEntry = createNewEditLine(145, commandYPos+27, 70, HEIGHT_SMALL, BINANCE_FONT_SMALL)

	window.SetShortcut(runCommand, wui.KeyReturn)
}

func sendNewService() {
	for {
		acc, err := client.NewGetAccountService().Do(context.Background())
		handleError(err)
		accountInfo = acc

		po, err := client.NewGetPositionRiskService().Symbol(cryptoFullname).Do(context.Background())
		handleError(err)
		positionInfo = po
		time.Sleep(500 * time.Millisecond)
	}
}

func updateInfo() {
	for {
		time.Sleep(200 * time.Millisecond)
		// Balance
		assetInfo := accountInfo.Assets[fiatIndex]
		totalbalanceStr := assetInfo.WalletBalance
		totalBalance := stringToFloat(totalbalanceStr)
		totalBalance = shared_functions.Round(totalBalance, 2)
		// Overall profit
		overallProfit := shared_functions.Round((totalBalance/balanceBefore-1)*100, 2)
		overallProfitStr := floatToString(overallProfit)
		overallProfitLabel.SetText("Overall Profit: " + overallProfitStr + "%")
		// Position info
		positionAmtStr := positionInfo[0].PositionAmt
		positionAmt := stringToFloat(positionAmtStr)
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
			inputMargin := stringToFloat(inputMarginStr)
			marginInputRatio := inputMargin / totalBalance
			// Preventing rounding error
			if marginInputRatio > 1 {
				marginInputRatio = 1
			}
			marginInputPercentage := shared_functions.Round(marginInputRatio*100, 2)
			marginInputPercentageStr := floatToString(marginInputPercentage)
			marginInputLabel.SetText("Margin Input: " + marginInputPercentageStr + "%")
			// profit label
			profitStr := positionInfo[0].UnRealizedProfit
			profit := stringToFloat(profitStr)
			profitRounded := shared_functions.Round(math.Abs(profit), 2)
			profitRoundedStr := floatToString(profitRounded)
			profitPercentage := shared_functions.Round(profit*100/totalBalance, 2)
			profitPercentageStr := floatToString(profitPercentage)
			if profit > 0 {
				profitLabel.SetText("Profit: $" + profitRoundedStr + " (" + profitPercentageStr + "%)")
			} else if profit < 0 {
				profitLabel.SetText("Profit: -$" + profitRoundedStr + " (" + profitPercentageStr + "%)")
			} else {
				profitLabel.SetText("Profit: $0.00 (0.00%)")
			}
		}
	}
}

func handleError(err error) {
	if err != nil {
		log.Panicln(err)
	}
}

func stringToFloat(str string) float64 {
	num, _ := strconv.ParseFloat(str, 64)
	return num
}

func floatToString(num float64) string {
	str := strconv.FormatFloat(num, 'f', -1, 64)
	return str
}

func startUpdateInfo() {
	for {
		time.Sleep(2 * time.Second)
		if isInitialized {
			fmt.Println("Starting updating info")
			go sendNewService()
			go updateInfo()
			break
		}
	}
}

func main() {
	file, _ := os.Open(textFilePath)
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	DEFAULT_FIAT_CURRENCY = scanner.Text()

	client = binance.NewFuturesClient(credentials.API_KEY, credentials.SECRET_KEY)
	go updateClient()

	go startUpdateInfo()

	sheet, _ := excelize.OpenFile(excelFilePath)
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
	balanceBefore = stringToFloat(value)

	window = wui.NewWindow()
	window.SetTitle("Trade Assister")
	window.SetPosition(1260, 200)
	window.SetResizable(false)
	window.SetSize(260, 140)
	window.SetBackground(wui.Color(BG_COLOR_24_BGR))
	icon, _ := wui.NewIconFromFile(iconFilePath)
	window.SetIcon(icon)

	instruction := createNewLabel("Enter Crypto Name:", 0, 20, 160, HEIGHT_SMALL, BINANCE_FONT_SMALL)
	instruction.SetX(window.InnerWidth()/2 - instruction.Width()/2)
	instruction.SetAlignment(wui.AlignCenter)

	cryptoNameEntry := createNewEditLine(0, 60, 90, HEIGHT_SMALL, BINANCE_FONT_SMALL)
	cryptoNameEntry.SetX(window.InnerWidth()/2 - cryptoNameEntry.Width()/2)

	window.SetShortcut(func() {
		if cryptoNameEntry.HasFocus() {
			cryptoName = strings.ToUpper(cryptoNameEntry.Text())
			cryptoFullname = cryptoName + DEFAULT_FIAT_CURRENCY
			window.Remove(instruction)
			window.Remove(cryptoNameEntry)
			initialize()
		}
	}, wui.KeyReturn)

	window.Show()
}
