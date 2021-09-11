package shared_functions

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
)

func HandleError(err error) {
	if err != nil {
		log.Panicln(err)
	}
}

func StringToFloat(str string) float64 {
	num, err := strconv.ParseFloat(str, 64)
	HandleError(err)
	return num
}

func FloatToString(num float64) string {
	str := strconv.FormatFloat(num, 'f', -1, 64)
	return str
}

func Round(num float64, precision int) float64 {
	multiplier := math.Pow10(precision)
	return math.Round(num*multiplier) / multiplier
}

func TestRuntime(repeatNum int, precision int, params ...interface{}) {
	var timeSlice []float64
	for i := 0; i < repeatNum; i++ {
		startTime := time.Now()
		fmt.Println("Loop ", i)

		testFunc := params[0].(func())
		testFunc()

		timeSecond := float64(time.Since(startTime)) / float64(time.Second)
		timeRounded := Round(timeSecond, precision)
		timeSlice = append(timeSlice, timeRounded)
		fmt.Println("Time taken: ", timeRounded)
	}
	sort.Float64s(timeSlice)
	var medianTime float64
	var sliceLength int = len(timeSlice)
	if sliceLength%2 == 0 {
		medianTime = (timeSlice[sliceLength/2] + timeSlice[sliceLength/2-1]) / 2
	} else {
		medianTime = timeSlice[(sliceLength-1)/2]
	}
	fmt.Println("Median time: ", medianTime)
}

func MakeTestOrder(client *binance.Client) {
	orderBook, _ := client.NewDepthService().Symbol("XRPUSDT").Limit(5).Do(context.Background())
	enteringPrice := orderBook.Bids[1].Price
	enteringPriceFloat, _, _ := orderBook.Bids[1].Parse()

	accountInfo, _ := client.NewGetAccountService().Do(context.Background())
	balance := accountInfo.Balances[0].Free
	strconv.ParseFloat(balance, 64)
	quantity := Round((10000)/enteringPriceFloat, 0)
	quantityStr := strconv.FormatFloat(quantity, 'f', -1, 64)

	client.NewCreateOrderService().Symbol("XRPUSDT").Side("BUY").Type("LIMIT").TimeInForce("GTC").
		Quantity(quantityStr).Price(enteringPrice).Test(context.Background())
}
