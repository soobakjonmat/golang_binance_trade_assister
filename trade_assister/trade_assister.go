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
)

const (
	DEFAULT_FIAT_CURRENCY = "USDT"

	BG_COLOR    = 0x201A18
	LONG_COLOR  = 0x77C002
	SHORT_COLOR = 0x6049F8
)

var (
	cryptoFullname string
	quantityDP     int
	orderBookNum   int = 0
	leverage       float64
	tradeFactor    float64 = 0.1
	closingFactor  float64 = 1
	client         *futures.Client
	window         *wui.Window

	FONT_LARGE, _     = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans"})
	FONT_MID_LARGE, _ = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans"})
	FONT_MEDIUM, _    = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans"})
	FONT_SMALL, _     = wui.NewFont(wui.FontDesc{Name: "IBM Plex Sans"})
)

func init() {
	window := wui.NewWindow()
	window.SetTitle("Trade Assister")
	window.SetPosition(1290, 200)
	window.SetSize(260, 300)
	window.SetResizable(false)
	window.SetBackground(wui.Color(BG_COLOR))
	window.SetFont(FONT_LARGE)

	instruction := wui.NewLabel()
	instruction.SetFont(FONT_LARGE)
	instruction.SetSize(160, 30)
	instruction.SetText("Enter crypto name:")
	instruction.SetAlignment(wui.AlignCenter)
	instruction.SetX(window.InnerWidth()/2 - instruction.Width()/2)
	instruction.SetY(window.InnerHeight()*1/3 - instruction.Height()/2)
	instruction.SetAnchors(wui.AnchorCenter, wui.AnchorCenter)
	window.Add(instruction)

	entry := wui.NewEditLine()
	entry.SetSize(160, 30)
	entry.SetX(window.InnerWidth()/2 - entry.Width()/2)
	entry.SetY(window.InnerHeight()/2 - entry.Height()/2)
	window.Add(entry)

	window.SetShortcut(func() {
		if entry.HasFocus() {
			cryptoFullname = strings.ToUpper(entry.Text()) + DEFAULT_FIAT_CURRENCY
			window.Remove(instruction)
			window.Remove(entry)
		}
	}, wui.KeyReturn)
	window.Show()
}

func initialize() {
	client := binance.NewFuturesClient(credentials.API_KEY, credentials.SECRET_KEY)

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
}

func enterLong() {
	orderBook, _ := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
	enteringPrice := orderBook.Bids[orderBookNum].Price
	enteringPriceFloat, _, _ := orderBook.Bids[orderBookNum].Parse()

	accountInfo, _ := client.NewGetAccountService().Do(context.Background())
	balance := accountInfo.TotalWalletBalance
	balanceFloat, _ := strconv.ParseFloat(balance, 64)
	quantity := shared_functions.Round((balanceFloat*leverage*tradeFactor)/enteringPriceFloat, quantityDP)
	quantityStr := strconv.FormatFloat(quantity, 'f', -1, 64)

	client.NewCreateOrderService().Symbol(cryptoFullname).
		Side("BUY").Type("LIMIT").TimeInForce("GTC").Quantity(quantityStr).Price(enteringPrice).Do(context.Background())

}

func enterShort() {
	orderBook, _ := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
	enteringPrice := orderBook.Asks[orderBookNum].Price
	enteringPriceFloat, _, _ := orderBook.Asks[orderBookNum].Parse()

	accountInfo, _ := client.NewGetAccountService().Do(context.Background())
	balance := accountInfo.TotalWalletBalance
	balanceFloat, _ := strconv.ParseFloat(balance, 64)
	quantity := shared_functions.Round((balanceFloat*leverage*tradeFactor)/enteringPriceFloat, quantityDP)
	quantityStr := strconv.FormatFloat(quantity, 'f', -1, 64)

	client.NewCreateOrderService().Symbol(cryptoFullname).
		Side("SELL").Type("LIMIT").TimeInForce("GTC").Quantity(quantityStr).Price(enteringPrice).Do(context.Background())

}

func closePosition() {
	orderBook, _ := client.NewDepthService().Symbol(cryptoFullname).Limit(5).Do(context.Background())
	closingPrice := orderBook.Asks[orderBookNum].Price

	positionRisk, _ := client.NewGetPositionRiskService().Symbol(cryptoFullname).Do(context.Background())
	positionAmtStr := positionRisk[0].PositionAmt
	positionAmtFloat, _ := strconv.ParseFloat(positionAmtStr, 64)
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

func updateClient() {
	time.Sleep(30 * time.Minute)
	client = binance.NewFuturesClient(credentials.API_KEY, credentials.SECRET_KEY)
}

func main() {
	// client = binance.NewFuturesClient(credentials.API_KEY, credentials.SECRET_KEY)
	// go updateClient()
}
