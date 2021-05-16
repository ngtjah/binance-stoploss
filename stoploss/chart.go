package stoploss

import (
	"bytes"
	"fmt"
	"github.com/wcharczuk/go-chart/v2"
	"io/ioutil"
	"log"
	"time"
)

func (tlg *Trailing) Chart(exchange string, pair string) {

	var stopSeries []float64
	var priceSeries []float64
	var timeSeries []time.Time

	minPrice := tlg.buyPrice - (tlg.buyPrice * tlg.maxLossStopFactor)
	maxPrice := tlg.buyPrice * (1 + tlg.limitSellFactor)
	now := time.Now().Local()

	for i := minPrice; i < maxPrice; i += 0.01 {
		timeSeries = append(timeSeries, now)
		stopSeries = append(stopSeries, tlg.computeSellStop(i))
		priceSeries = append(priceSeries, i)
		// advance now
		now = now.Add(5 * time.Second)
	}

	graph := chart.Chart{
		//YAxis: chart.YAxis{
		//	Range: &chart.ContinuousRange{
		//		Min: minPrice,
		//		Max: maxPrice,
		//	},
		//},
		XAxis: chart.XAxis{
			ValueFormatter: chart.TimeMinuteValueFormatter,
		},
		Series: []chart.Series{
			chart.TimeSeries{
				XValues: timeSeries,
				YValues: stopSeries,
			},
			chart.TimeSeries{
				XValues: timeSeries,
				YValues: priceSeries,
			},
		},
	}

	buffer := bytes.NewBuffer([]byte{})
	err := graph.Render(chart.PNG, buffer)
	if err != nil {
		fmt.Printf("Error rendering chart: %v", err)
	}

	// write file
	err = ioutil.WriteFile("chart.png", buffer.Bytes(), 0644)
	if err != nil {
		log.Fatalf("Error writing file: %v", err)
	}
}

//import (
//    "fmt"
//    tm "github.com/buger/goterm"
//)
//
//func (tlg *Trailing) Chart(exchange string, pair string) {
//    chart := tm.NewLineChart(100, 30)
//
//    data := new(tm.DataTable)
//    data.AddColumn(fmt.Sprintf("%s %s price", exchange, pair))
//    data.AddColumn("MarketPrice")
//    data.AddColumn("StopPrice")
//
//    minPrice := tlg.buyPrice - (tlg.buyPrice * (tlg.maxLossStopFactor))
//    maxPrice := tlg.buyPrice * (1 + tlg.limitSellFactor)
//
//    for i := minPrice; i < maxPrice; i += 0.1 {
//       data.AddRow(i, i, tlg.computeSellStop(i))
//    }
//
//    fmt.Printf("Chart: %s - %s\n", exchange, pair)
//
//    //chart.Flags = tm.DRAW_INDEPENDENT
//    chart.Flags = tm.DRAW_RELATIVE
//    tlg.logger.Println(chart.Draw(data))
//    //_, err := tlg.logger.Println(chart.Draw(data))
//    //if err != nil {
//    //    tlg.logger.Printf("Error building chart: %s\n", err)
//    //}
//}

//import (
//    "github.com/guptarohit/asciigraph"
//)
//
//func (tlg *Trailing) Chart(exchange string, pair string) {
//
//    var data []float64
//
//    maxPrice := tlg.buyPrice * (1 + tlg.limitSellFactor)
//    for i := tlg.buyPrice; i < maxPrice; i += 0.01 {
//       data = append(data, tlg.computeSellStop(i))
//    }
//
//    graph := asciigraph.Plot(data, asciigraph.Height(10), asciigraph.Width(50))
//
//    tlg.logger.Println(graph)
//}
