package shared_functions

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/adshao/go-binance/v2"
)

func Round(num float64, precision int) float64 {
	multiplier := math.Pow10(precision)
	return math.Round(num*multiplier) / multiplier
}

func TestRuntime(repeatNum int, precision int, params ...interface{}) {
	var timeSlice []float64
	for i := 0; i < repeatNum; i++ {
		startTime := time.Now()
		fmt.Printf("Loop %v\n", i)

		test := params[0].(func(*binance.Client))
		test(params[1].(*binance.Client))

		timeSecond := float64(time.Since(startTime)) / float64(time.Second)
		timeRounded := Round(timeSecond, precision)
		timeSlice = append(timeSlice, timeRounded)
		fmt.Printf("Time taken: %v\n", timeRounded)
	}
	sort.Float64s(timeSlice)
	var medianTime float64
	var sliceLength int = len(timeSlice)
	if sliceLength%2 == 0 {
		medianTime = (timeSlice[sliceLength/2] + timeSlice[sliceLength/2-1]) / 2
	} else {
		medianTime = timeSlice[(sliceLength-1)/2]
	}
	fmt.Printf("Median time: %v\n", medianTime)
}
