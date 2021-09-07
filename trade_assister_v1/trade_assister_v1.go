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
	HEIGHT_BIG   = 27
	HEIGHT_SMALL = 23
	HEIGHT_TINY  = 18

	BG_COLOR_24_BGR = 0x201A18
)

var (
	cryptoFullname        string = ""
	cryptoName            string = ""
	default_fiat_currency string = ""

	quantityDP    int
	orderBookIdx  int = 0
	leverage      float64
	tradeFactor   float64 = 0.05
	closingFactor float64 = 1

	useTradeFactor   bool = true
	useClosingFactor bool = true
	useSpecificPrice bool = false

	tradeAmt   string
	closingAmt string

	fiatIndex int

	balanceBefore float64

	client *futures.Client

	accountInfo  *futures.Account
	positionInfo []*futures.PositionRisk
	positionAmt  float64

	isInitialized bool = false
	startTime     time.Time
	initialProfit float64

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

	// Bid: Buy
	// Ask: Sell
	orderBookIdxDisplay *wui.Label
	orderBookLeftBtn    *wui.Button
	orderBookRightBtn   *wui.Button

	commandEntry       *wui.EditLine
	specificPriceLabel *wui.Label
	specificPriceEntry *wui.EditLine

	SYMBOL_FONT, _        = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 31})
	BINANCE_FONT_BIG, _   = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 26})
	BINANCE_FONT_SMALL, _ = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 19})
	BINANCE_FONT_TINY, _  = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans", Height: 14})
)

func enterLong() {
	var enteringPrice string
	if useSpecificPrice {
		enteringPrice = specificPriceLabel.Text()
	} else {
		if orderBookIdx == 0 {
			priceList, err := client.NewListPricesService().Symbol(cryptoFullname).Do(context.Background())
			handleError(err)
			enteringPrice = priceList[0].Price
		} else {
			orderBook, err := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
			handleError(err)
			enteringPrice = orderBook.Bids[orderBookIdx-1].Price
		}
	}
	if useTradeFactor {
		enteringPriceFloat := stringToFloat(enteringPrice)
		accountInfo, err := client.NewGetAccountService().Do(context.Background())
		handleError(err)
		balance := accountInfo.Assets[fiatIndex].WalletBalance
		balanceFloat := stringToFloat(balance)

		quantity := shared_functions.Round((balanceFloat*leverage*tradeFactor)/enteringPriceFloat, quantityDP)
		quantityStr := floatToString(quantity)

		client.NewCreateOrderService().Symbol(cryptoFullname).
			Side("BUY").Type("LIMIT").TimeInForce("GTC").Quantity(quantityStr).Price(enteringPrice).Do(context.Background())
	} else {
		client.NewCreateOrderService().Symbol(cryptoFullname).
			Side("BUY").Type("LIMIT").TimeInForce("GTC").Quantity(tradeAmt).Price(enteringPrice).Do(context.Background())
	}
}

func enterShort() {
	var enteringPrice string
	if useSpecificPrice {
		enteringPrice = specificPriceLabel.Text()
	} else {
		if orderBookIdx == 0 {
			priceList, err := client.NewListPricesService().Symbol(cryptoFullname).Do(context.Background())
			handleError(err)
			enteringPrice = priceList[0].Price
		} else {
			orderBook, err := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
			handleError(err)
			enteringPrice = orderBook.Asks[orderBookIdx-1].Price
		}
	}
	if useTradeFactor {
		enteringPriceFloat := stringToFloat(enteringPrice)
		accountInfo, err := client.NewGetAccountService().Do(context.Background())
		handleError(err)
		balance := accountInfo.Assets[fiatIndex].WalletBalance
		balanceFloat := stringToFloat(balance)

		quantity := shared_functions.Round((balanceFloat*leverage*tradeFactor)/enteringPriceFloat, quantityDP)
		quantityStr := floatToString(quantity)

		client.NewCreateOrderService().Symbol(cryptoFullname).
			Side("SELL").Type("LIMIT").TimeInForce("GTC").Quantity(quantityStr).Price(enteringPrice).Do(context.Background())
	} else {
		client.NewCreateOrderService().Symbol(cryptoFullname).
			Side("SELL").Type("LIMIT").TimeInForce("GTC").Quantity(tradeAmt).Price(enteringPrice).Do(context.Background())
	}
}

func closePosition() {
	positionRisk, err := client.NewGetPositionRiskService().Symbol(cryptoFullname).Do(context.Background())
	handleError(err)
	positionAmtFloat := stringToFloat(positionRisk[0].PositionAmt)

	orderBook, err := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
	handleError(err)

	var closingPrice string
	if useSpecificPrice {
		closingPrice = specificPriceLabel.Text()
	} else {
		if orderBookIdx == 0 {
			priceList, err := client.NewListPricesService().Symbol(cryptoFullname).Do(context.Background())
			handleError(err)
			closingPrice = priceList[0].Price
		} else {
			if positionAmtFloat > 0 {
				closingPrice = orderBook.Asks[orderBookIdx-1].Price
			} else {
				closingPrice = orderBook.Bids[orderBookIdx-1].Price
			}
		}
	}
	if positionAmtFloat > 0 {
		if useClosingFactor {
			quantity := math.Abs(shared_functions.Round(positionAmtFloat*closingFactor, quantityDP))
			quantityStr := floatToString(quantity)
			client.NewCreateOrderService().Symbol(cryptoFullname).Side("SELL").Type("LIMIT").TimeInForce("GTC").
				Quantity(quantityStr).Price(closingPrice).Do(context.Background())
		} else {
			client.NewCreateOrderService().Symbol(cryptoFullname).Side("SELL").Type("LIMIT").TimeInForce("GTC").
				Quantity(closingAmt).Price(closingPrice).Do(context.Background())
		}
	} else {
		if useClosingFactor {
			quantity := math.Abs(shared_functions.Round(positionAmtFloat*closingFactor, quantityDP))
			quantityStr := floatToString(quantity)
			client.NewCreateOrderService().Symbol(cryptoFullname).Side("BUY").Type("LIMIT").TimeInForce("GTC").
				Quantity(quantityStr).Price(closingPrice).Do(context.Background())
		} else {
			client.NewCreateOrderService().Symbol(cryptoFullname).Side("BUY").Type("LIMIT").TimeInForce("GTC").
				Quantity(closingAmt).Price(closingPrice).Do(context.Background())
		}
	}
}

func cancelOrder() {
	client.NewCancelAllOpenOrdersService().Symbol(cryptoFullname).Do(context.Background())
}

func runCommand() {
	// make order or make changes in order
	if commandEntry.HasFocus() && commandEntry.Text() != "" {
		orderTime := time.Now()
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
		timeTaken := time.Since(orderTime)
		fmt.Println("Time taken:", timeTaken)
	} else if tradeFactorEntry.HasFocus() && tradeFactorEntry.Text() != "" {
		tradeFactor = stringToFloat(tradeFactorEntry.Text())
		tradeAmt = tradeFactorEntry.Text()
		if tradeFactor <= 1 {
			useTradeFactor = true
			tradeFactorLabel.SetText("Trade Factor")
			tradeFactorDisplay.SetText(tradeFactorEntry.Text())
		} else {
			useTradeFactor = false
			tradeFactorLabel.SetText("Trade Amount")
			tradeFactorDisplay.SetText(tradeFactorEntry.Text() + " " + cryptoName)
		}
		tradeFactorEntry.SetText("")
	} else if closingFactorEntry.HasFocus() && closingFactorEntry.Text() != "" {
		closingFactor = stringToFloat(closingFactorEntry.Text())
		closingAmt = closingFactorDisplay.Text()
		if closingFactor <= 1 {
			useClosingFactor = true
			closingFactorLabel.SetText("Closing Factor")
			closingFactorDisplay.SetText(closingFactorEntry.Text())
		} else {
			useClosingFactor = false
			closingFactorLabel.SetText("Closing Amount")
			closingFactorDisplay.SetText(closingFactorEntry.Text() + " " + cryptoName)
		}
		closingFactorEntry.SetText("")
	} else if specificPriceEntry.HasFocus() && specificPriceEntry.Text() != "" {
		if specificPriceEntry.Text() == "re" {
			useSpecificPrice = false
			specificPriceLabel.SetText("Specific Price")
		} else {
			useSpecificPrice = true
			specificPriceLabel.SetText(specificPriceEntry.Text())
		}
		specificPriceEntry.SetText("")
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
	newLabel.SetPosition(xPos, yPos)
	newLabel.SetSize(width, height)
	newLabel.SetFont(font)
	window.Add(newLabel)
	return newLabel
}

func createNewButton(text string, xPos int, yPos int, width int, height int, font *wui.Font) *wui.Button {
	newButton := wui.NewButton()
	newButton.SetText(text)
	newButton.SetPosition(xPos, yPos)
	newButton.SetSize(width, height)
	newButton.SetFont(font)
	window.Add(newButton)
	return newButton
}

func createNewEditLine(xPos int, yPos int, width int, height int, font *wui.Font) *wui.EditLine {
	newEditLine := wui.NewEditLine()
	newEditLine.SetPosition(xPos, yPos)
	newEditLine.SetSize(width, height)
	newEditLine.SetFont(font)
	window.Add(newEditLine)
	return newEditLine
}

func getCenterXPos(target ...interface{}) int {
	label, isLabel := target[0].(*wui.Label)
	editLine, isEditLine := target[0].(*wui.EditLine)
	if isLabel {
		return (window.InnerWidth() - label.Width()) / 2
	} else if isEditLine {
		return (window.InnerWidth() - editLine.Width()) / 2
	}
	return 0
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
		if item.Asset == default_fiat_currency {
			fiatIndex = idx
		}
	}

	// Record initial profit
	totalbalanceStr := accInfo.Assets[fiatIndex].WalletBalance
	totalBalance := shared_functions.Round(stringToFloat(totalbalanceStr), 2)
	initialProfit = shared_functions.Round((totalBalance/balanceBefore-1)*100, 2)

	// Lables and Entries
	symbolLabel = createNewLabel(cryptoFullname, 0, 0, 130, 33, SYMBOL_FONT)
	symbolLabel.SetAlignment(wui.AlignCenter)
	symbolLabel.SetX(getCenterXPos(symbolLabel))

	overallProfitLabel = createNewLabel("Overall Profit: 0.00%", 0, 38, 200, HEIGHT_BIG, BINANCE_FONT_BIG)
	overallProfitLabel.SetX(getCenterXPos(overallProfitLabel))
	overallProfitLabel.SetAlignment(wui.AlignCenter)

	positionLabel = createNewLabel("Not in Position", 0, 74, 180, HEIGHT_BIG, BINANCE_FONT_BIG)
	positionLabel.SetX(getCenterXPos(positionLabel))
	positionLabel.SetAlignment(wui.AlignCenter)

	marginInputLabel = createNewLabel("Margin Input: nil", 0, 102, 240, HEIGHT_BIG, BINANCE_FONT_BIG)
	marginInputLabel.SetX(getCenterXPos(marginInputLabel))
	marginInputLabel.SetAlignment(wui.AlignCenter)

	profitLabel = createNewLabel("Profit: nil", 0, 130, 240, HEIGHT_BIG, BINANCE_FONT_BIG)
	profitLabel.SetX(getCenterXPos(profitLabel))
	profitLabel.SetAlignment(wui.AlignCenter)

	labelXPos := 10
	labelWidth := 95
	displayEntryWidth := 70
	displayXPos := 124
	entryXPos := 205

	tradeFactorYPos := 160
	tradeFactorLabel = createNewLabel("Trade Factor", labelXPos, tradeFactorYPos, labelWidth, HEIGHT_TINY, BINANCE_FONT_TINY)
	tradeFactorLabel.SetAlignment(wui.AlignCenter)
	tradeFactorStr := floatToString(tradeFactor)
	tradeFactorDisplay = createNewLabel(tradeFactorStr, displayXPos, tradeFactorYPos, displayEntryWidth, HEIGHT_TINY, BINANCE_FONT_TINY)
	tradeFactorDisplay.SetAlignment(wui.AlignCenter)
	tradeFactorEntry = createNewEditLine(entryXPos, tradeFactorYPos, displayEntryWidth, HEIGHT_TINY, BINANCE_FONT_TINY)

	closingFactorYPos := 180
	closingFactorLabel = createNewLabel("Closing Factor", labelXPos, closingFactorYPos, labelWidth, HEIGHT_TINY, BINANCE_FONT_TINY)
	closingFactorLabel.SetAlignment(wui.AlignCenter)
	closingFactorStr := floatToString(closingFactor)
	closingFactorDisplay = createNewLabel(closingFactorStr, displayXPos, closingFactorYPos, displayEntryWidth, HEIGHT_TINY, BINANCE_FONT_TINY)
	closingFactorDisplay.SetAlignment(wui.AlignCenter)
	closingFactorEntry = createNewEditLine(entryXPos, closingFactorYPos, displayEntryWidth, HEIGHT_TINY, BINANCE_FONT_TINY)

	orderBookIdxYPos := 200
	orderBookIdxLabel := createNewLabel("Order Book Index", labelXPos, orderBookIdxYPos, labelWidth, HEIGHT_TINY, BINANCE_FONT_TINY)
	orderBookIdxLabel.SetAlignment(wui.AlignCenter)
	orderBookIdxStr := strconv.Itoa(orderBookIdx)
	orderBookIdxDisplay = createNewLabel(orderBookIdxStr, entryXPos-40, orderBookIdxYPos, displayEntryWidth-30, HEIGHT_TINY, BINANCE_FONT_TINY)
	orderBookIdxDisplay.SetAlignment(wui.AlignCenter)
	orderBookBtnWidth := 30
	orderBookLeftBtn = createNewButton("-", displayXPos, orderBookIdxYPos, orderBookBtnWidth, HEIGHT_TINY, BINANCE_FONT_TINY)
	orderBookLeftBtn.SetOnClick(func() {
		if orderBookIdx > 0 {
			orderBookIdx--
			orderBookIdxDisplay.SetText(strconv.Itoa(orderBookIdx))
		}
	})
	orderBookRightBtn = createNewButton("+", entryXPos+10, orderBookIdxYPos, orderBookBtnWidth, HEIGHT_TINY, BINANCE_FONT_TINY)
	orderBookRightBtn.SetOnClick(func() {
		if orderBookIdx < 5 {
			orderBookIdx++
			orderBookIdxDisplay.SetText(strconv.Itoa(orderBookIdx))
		}
	})

	commandYPos := 225
	commandXPos := 15
	commandLabel := createNewLabel("Command:", commandXPos, commandYPos, 90, HEIGHT_SMALL, BINANCE_FONT_SMALL)
	commandLabel.SetAlignment(wui.AlignCenter)
	commandEntry = createNewEditLine(commandXPos+12, commandYPos+27, 70, HEIGHT_SMALL, BINANCE_FONT_SMALL)

	specificPriceLabel = createNewLabel("Specific Price:", commandXPos+113, commandYPos, 105, HEIGHT_SMALL, BINANCE_FONT_SMALL)
	specificPriceLabel.SetAlignment(wui.AlignCenter)
	specificPriceEntry = createNewEditLine(commandXPos+130, commandYPos+27, 70, HEIGHT_SMALL, BINANCE_FONT_SMALL)

	isInitialized = true
	startTime = time.Now()

	window.SetShortcut(runCommand, wui.KeyReturn)
}

func getNewService() {
	for {
		acc, err := client.NewGetAccountService().Do(context.Background())
		handleError(err)
		accountInfo = acc

		po, err := client.NewGetPositionRiskService().Symbol(cryptoFullname).Do(context.Background())
		handleError(err)
		positionInfo = po
		positionAmtStr := positionInfo[0].PositionAmt
		positionAmt = stringToFloat(positionAmtStr)

		time.Sleep(500 * time.Millisecond)
	}
}

func updateInfo() {
	for {
		time.Sleep(200 * time.Millisecond)
		// Balance
		assetInfo := accountInfo.Assets[fiatIndex]
		totalbalanceStr := assetInfo.WalletBalance
		totalBalance := shared_functions.Round(stringToFloat(totalbalanceStr), 2)
		// Overall profit
		overallProfit := shared_functions.Round((totalBalance/balanceBefore-1)*100, 2)
		overallProfitStr := floatToString(overallProfit)
		overallProfitLabel.SetText("Overall Profit: " + overallProfitStr + "%")
		// Position info
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
	num, err := strconv.ParseFloat(str, 64)
	handleError(err)
	return num
}

func floatToString(num float64) string {
	str := strconv.FormatFloat(num, 'f', -1, 64)
	return str
}

func startUpdateInfo() {
	for {
		time.Sleep(200 * time.Millisecond)
		if isInitialized {
			go getNewService()
			time.Sleep(1 * time.Second)
			go updateInfo()
			break
		}
	}
}

func main() {
	file, _ := os.Open(credentials.TextFilePath)
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	default_fiat_currency = scanner.Text()

	client = binance.NewFuturesClient(credentials.API_KEY, credentials.SECRET_KEY)
	go updateClient()

	go startUpdateInfo()

	sheet, _ := excelize.OpenFile(credentials.ExcelFilePath)
	cols, _ := sheet.GetCols(credentials.SheetName)
	targetIndex := 0
	for idx, rowCell := range cols[0] {
		if rowCell == "" {
			targetIndex = idx
			break
		}
	}
	targetIndexStr := strconv.Itoa(targetIndex)
	value, _ := sheet.GetCellValue(credentials.SheetName, "B"+targetIndexStr)
	balanceBefore = stringToFloat(value)

	window = wui.NewWindow()
	window.SetTitle("Trade Assister")
	window.SetPosition(1260, 200)
	window.SetResizable(false)
	window.SetSize(260, 140)
	window.SetBackground(wui.Color(BG_COLOR_24_BGR))
	icon, _ := wui.NewIconFromFile(credentials.IconFilePath)
	window.SetIcon(icon)

	instruction := createNewLabel("Enter Crypto Name:", 0, 20, 160, HEIGHT_SMALL, BINANCE_FONT_SMALL)
	instruction.SetX(getCenterXPos(instruction))
	instruction.SetAlignment(wui.AlignCenter)

	cryptoNameEntry := createNewEditLine(0, 60, 90, HEIGHT_SMALL, BINANCE_FONT_SMALL)
	cryptoNameEntry.SetX(getCenterXPos(cryptoNameEntry))

	window.SetShortcut(func() {
		if cryptoNameEntry.HasFocus() {
			cryptoName = strings.ToUpper(cryptoNameEntry.Text())
			cryptoFullname = cryptoName + default_fiat_currency
			window.Remove(instruction)
			window.Remove(cryptoNameEntry)
			initialize()
		}
	}, wui.KeyReturn)

	window.Show()

	// Compare initialProfit and finishingProfit and record time spent
	accInfo, err := client.NewGetAccountService().Do(context.Background())
	handleError(err)
	totalbalanceStr := accInfo.Assets[fiatIndex].WalletBalance
	totalBalance := shared_functions.Round(stringToFloat(totalbalanceStr), 2)
	finishingProfit := shared_functions.Round((totalBalance/balanceBefore-1)*100, 2)
	if isInitialized && initialProfit != finishingProfit {
		fmt.Println("Recording Spent Time")
		timeSpent := int(time.Since(startTime) / time.Second)
		recordedTimeStr, _ := sheet.GetCellValue(credentials.SheetName, "G2")
		recordedTime, _ := strconv.Atoi(recordedTimeStr)
		totalTimeSpent := timeSpent + recordedTime
		sheet.SetCellValue(credentials.SheetName, "G2", totalTimeSpent)
		sheet.Save()
	}
}
