package market

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// Data 市场数据结构
type Data struct {
	Symbol            string
	CurrentPrice      float64
	PriceChange1h     float64 // 1小时价格变化百分比
	PriceChange4h     float64 // 4小时价格变化百分比
	CurrentEMA20      float64
	CurrentMACD       float64
	CurrentRSI7       float64
	OpenInterest      *OIData
	FundingRate       float64
	IntradaySeries    *IntradayData
	LongerTermContext *LongerTermData
	DailyContext      *DailyData
	WeeklyContext     *WeeklyData
	DerivedMetrics    *DerivedMetrics
}

// OIData Open Interest数据
type OIData struct {
	Latest  float64
	Average float64
}

// IntradayData 日内数据(3分钟间隔)
type IntradayData struct {
	MidPrices   []float64
	EMA20Values []float64
	MACDValues  []float64
	RSI7Values  []float64
	RSI14Values []float64
}

// LongerTermData 长期数据(4小时时间框架)
type LongerTermData struct {
	EMA20         float64
	EMA50         float64
	ATR3          float64
	ATR14         float64
	CurrentVolume float64
	AverageVolume float64
	MACDValues    []float64
	RSI14Values   []float64
}

// DailyData 日线级别数据
type DailyData struct {
	EMA20         float64
	EMA50         float64
	ATR14         float64
	CloseSeries   []float64
	MACDValues    []float64
	RSI14Values   []float64
	AverageVolume float64
	CurrentVolume float64
}

// WeeklyData 周线级别数据
type WeeklyData struct {
	EMA20         float64
	EMA50         float64
	ATR14         float64
	CloseSeries   []float64
	MACDValues    []float64
	RSI14Values   []float64
}

// DerivedMetrics 预计算的趋势指标
type DerivedMetrics struct {
	PriceVsEMA20Pct float64
	EMA20Slope      float64
	MACDSlope       float64
	RSI7Slope       float64
	HTFTrend        string
	VolatilityRatio float64
	DailyTrend      string
	DailyPriceVsEMA float64
	DailyEMA20Slope float64
	WeeklyTrend     string
	WeeklyPriceVsEMA float64
	WeeklyEMA20Slope float64
}

// Kline K线数据
type Kline struct {
	OpenTime  int64
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	CloseTime int64
}

// Get 获取指定代币的市场数据
func Get(symbol string) (*Data, error) {
	// 标准化symbol
	symbol = Normalize(symbol)

	// 获取3分钟K线数据 (最近10个)
	klines3m, err := getKlines(symbol, "3m", 40) // 多获取一些用于计算
	if err != nil {
		return nil, fmt.Errorf("获取3分钟K线失败: %v", err)
	}

	// 获取4小时K线数据 (最近10个)
	klines4h, err := getKlines(symbol, "4h", 60) // 多获取用于计算指标
	if err != nil {
		return nil, fmt.Errorf("获取4小时K线失败: %v", err)
	}

	// 获取日线K线数据 (最近90个)
	klines1d, err := getKlines(symbol, "1d", 90)
	if err != nil {
		return nil, fmt.Errorf("获取日线K线失败: %v", err)
	}

	// 获取周线K线数据 (最近90个)
	klines1w, err := getKlines(symbol, "1w", 90)
	if err != nil {
		return nil, fmt.Errorf("获取周线K线失败: %v", err)
	}

	// 计算当前指标 (基于3分钟最新数据)
	currentPrice := klines3m[len(klines3m)-1].Close
	currentEMA20 := calculateEMA(klines3m, 20)
	currentMACD := calculateMACD(klines3m)
	currentRSI7 := calculateRSI(klines3m, 7)

	// 计算价格变化百分比
	// 1小时价格变化 = 20个3分钟K线前的价格
	priceChange1h := 0.0
	if len(klines3m) >= 21 { // 至少需要21根K线 (当前 + 20根前)
		price1hAgo := klines3m[len(klines3m)-21].Close
		if price1hAgo > 0 {
			priceChange1h = ((currentPrice - price1hAgo) / price1hAgo) * 100
		}
	}

	// 4小时价格变化 = 1个4小时K线前的价格
	priceChange4h := 0.0
	if len(klines4h) >= 2 {
		price4hAgo := klines4h[len(klines4h)-2].Close
		if price4hAgo > 0 {
			priceChange4h = ((currentPrice - price4hAgo) / price4hAgo) * 100
		}
	}

	// 获取OI数据
	oiData, err := getOpenInterestData(symbol)
	if err != nil {
		// OI失败不影响整体,使用默认值
		oiData = &OIData{Latest: 0, Average: 0}
	}

	// 获取Funding Rate
	fundingRate, _ := getFundingRate(symbol)

	// 计算日内系列数据
	intradayData := calculateIntradaySeries(klines3m)

	// 计算长期数据
	longerTermData := calculateLongerTermData(klines4h)

	// 计算日线数据
	dailyData := calculateDailyData(klines1d)

	// 计算周线数据
	weeklyData := calculateWeeklyData(klines1w)

	// 派生趋势指标
	derivedMetrics := calculateDerivedMetrics(currentPrice, currentEMA20, intradayData, longerTermData, dailyData, weeklyData)

	return &Data{
		Symbol:            symbol,
		CurrentPrice:      currentPrice,
		PriceChange1h:     priceChange1h,
		PriceChange4h:     priceChange4h,
		CurrentEMA20:      currentEMA20,
		CurrentMACD:       currentMACD,
		CurrentRSI7:       currentRSI7,
		OpenInterest:      oiData,
		FundingRate:       fundingRate,
		IntradaySeries:    intradayData,
		LongerTermContext: longerTermData,
		DailyContext:      dailyData,
		WeeklyContext:     weeklyData,
		DerivedMetrics:    derivedMetrics,
	}, nil
}

// getKlines 从Binance获取K线数据
func getKlines(symbol, interval string, limit int) ([]Kline, error) {
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/klines?symbol=%s&interval=%s&limit=%d",
		symbol, interval, limit)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawData [][]interface{}
	if err := json.Unmarshal(body, &rawData); err != nil {
		return nil, err
	}

	klines := make([]Kline, len(rawData))
	for i, item := range rawData {
		openTime := int64(item[0].(float64))
		open, _ := parseFloat(item[1])
		high, _ := parseFloat(item[2])
		low, _ := parseFloat(item[3])
		close, _ := parseFloat(item[4])
		volume, _ := parseFloat(item[5])
		closeTime := int64(item[6].(float64))

		klines[i] = Kline{
			OpenTime:  openTime,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			CloseTime: closeTime,
		}
	}

	return klines, nil
}

// calculateEMA 计算EMA
func calculateEMA(klines []Kline, period int) float64 {
	if len(klines) < period {
		return 0
	}

	// 计算SMA作为初始EMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += klines[i].Close
	}
	ema := sum / float64(period)

	// 计算EMA
	multiplier := 2.0 / float64(period+1)
	for i := period; i < len(klines); i++ {
		ema = (klines[i].Close-ema)*multiplier + ema
	}

	return ema
}

// calculateMACD 计算MACD
func calculateMACD(klines []Kline) float64 {
	if len(klines) < 26 {
		return 0
	}

	// 计算12期和26期EMA
	ema12 := calculateEMA(klines, 12)
	ema26 := calculateEMA(klines, 26)

	// MACD = EMA12 - EMA26
	return ema12 - ema26
}

// calculateRSI 计算RSI
func calculateRSI(klines []Kline, period int) float64 {
	if len(klines) <= period {
		return 0
	}

	gains := 0.0
	losses := 0.0

	// 计算初始平均涨跌幅
	for i := 1; i <= period; i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			gains += change
		} else {
			losses += -change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	// 使用Wilder平滑方法计算后续RSI
	for i := period + 1; i < len(klines); i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			avgGain = (avgGain*float64(period-1) + change) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + (-change)) / float64(period)
		}
	}

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// calculateATR 计算ATR
func calculateATR(klines []Kline, period int) float64 {
	if len(klines) <= period {
		return 0
	}

	trs := make([]float64, len(klines))
	for i := 1; i < len(klines); i++ {
		high := klines[i].High
		low := klines[i].Low
		prevClose := klines[i-1].Close

		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)

		trs[i] = math.Max(tr1, math.Max(tr2, tr3))
	}

	// 计算初始ATR
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trs[i]
	}
	atr := sum / float64(period)

	// Wilder平滑
	for i := period + 1; i < len(klines); i++ {
		atr = (atr*float64(period-1) + trs[i]) / float64(period)
	}

	return atr
}

// calculateIntradaySeries 计算日内系列数据
func calculateIntradaySeries(klines []Kline) *IntradayData {
	data := &IntradayData{
		MidPrices:   make([]float64, 0, 10),
		EMA20Values: make([]float64, 0, 10),
		MACDValues:  make([]float64, 0, 10),
		RSI7Values:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
	}

	// 获取最近10个数据点
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		data.MidPrices = append(data.MidPrices, klines[i].Close)

		// 计算每个点的EMA20
		if i >= 19 {
			ema20 := calculateEMA(klines[:i+1], 20)
			data.EMA20Values = append(data.EMA20Values, ema20)
		}

		// 计算每个点的MACD
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}

		// 计算每个点的RSI
		if i >= 7 {
			rsi7 := calculateRSI(klines[:i+1], 7)
			data.RSI7Values = append(data.RSI7Values, rsi7)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	return data
}

// calculateLongerTermData 计算长期数据
func calculateLongerTermData(klines []Kline) *LongerTermData {
	data := &LongerTermData{
		MACDValues:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
	}

	// 计算EMA
	data.EMA20 = calculateEMA(klines, 20)
	data.EMA50 = calculateEMA(klines, 50)

	// 计算ATR
	data.ATR3 = calculateATR(klines, 3)
	data.ATR14 = calculateATR(klines, 14)

	// 计算成交量
	if len(klines) > 0 {
		data.CurrentVolume = klines[len(klines)-1].Volume
		// 计算平均成交量
		sum := 0.0
		for _, k := range klines {
			sum += k.Volume
		}
		data.AverageVolume = sum / float64(len(klines))
	}

	// 计算MACD和RSI序列
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	return data
}

func calculateDailyData(klines []Kline) *DailyData {
	if len(klines) == 0 {
		return nil
	}

	data := &DailyData{
		MACDValues:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
		CloseSeries: make([]float64, 0, len(klines)),
	}

	for _, k := range klines {
		data.CloseSeries = append(data.CloseSeries, k.Close)
	}

	data.EMA20 = calculateEMA(klines, 20)
	data.EMA50 = calculateEMA(klines, 50)
	data.ATR14 = calculateATR(klines, 14)

	// 计算成交量
	sumVol := 0.0
	for _, k := range klines {
		sumVol += k.Volume
	}
	data.AverageVolume = sumVol / float64(len(klines))
	data.CurrentVolume = klines[len(klines)-1].Volume

	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}
		if i >= 14 {
			rsi := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi)
		}
	}

	return data
}

func calculateWeeklyData(klines []Kline) *WeeklyData {
	if len(klines) == 0 {
		return nil
	}

	data := &WeeklyData{
		MACDValues:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
		CloseSeries: make([]float64, 0, len(klines)),
	}

	for _, k := range klines {
		data.CloseSeries = append(data.CloseSeries, k.Close)
	}

	data.EMA20 = calculateEMA(klines, 20)
	data.EMA50 = calculateEMA(klines, 50)
	data.ATR14 = calculateATR(klines, 14)

	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}
		if i >= 14 {
			rsi := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi)
		}
	}

	return data
}

func calculateDerivedMetrics(currentPrice, currentEMA float64, intraday *IntradayData, longer *LongerTermData, daily *DailyData, weekly *WeeklyData) *DerivedMetrics {
	if intraday == nil {
		return nil
	}

	metrics := &DerivedMetrics{}

	if currentEMA > 0 {
		metrics.PriceVsEMA20Pct = ((currentPrice - currentEMA) / currentEMA) * 100
	}

	if len(intraday.EMA20Values) >= 2 {
		first := intraday.EMA20Values[0]
		last := intraday.EMA20Values[len(intraday.EMA20Values)-1]
		if first != 0 {
			metrics.EMA20Slope = (last - first) / first * 100
		} else {
			metrics.EMA20Slope = last - first
		}
	}

	if len(intraday.MACDValues) >= 2 {
		first := intraday.MACDValues[0]
		last := intraday.MACDValues[len(intraday.MACDValues)-1]
		metrics.MACDSlope = last - first
	}

	if len(intraday.RSI7Values) >= 2 {
		first := intraday.RSI7Values[0]
		last := intraday.RSI7Values[len(intraday.RSI7Values)-1]
		metrics.RSI7Slope = last - first
	}

	metrics.HTFTrend = "neutral"
	if longer != nil && len(longer.MACDValues) >= 2 {
		last := longer.MACDValues[len(longer.MACDValues)-1]
		prev := longer.MACDValues[len(longer.MACDValues)-2]
		switch {
		case last > 0 && last >= prev:
			metrics.HTFTrend = "bullish"
		case last < 0 && last <= prev:
			metrics.HTFTrend = "bearish"
		default:
			metrics.HTFTrend = "neutral"
		}
	}

	if longer != nil && longer.ATR14 > 0 {
		metrics.VolatilityRatio = longer.ATR3 / longer.ATR14
	}

	if daily != nil {
		if daily.EMA20 > 0 {
			metrics.DailyPriceVsEMA = ((currentPrice - daily.EMA20) / daily.EMA20) * 100
		}
		if len(daily.MACDValues) >= 2 {
			last := daily.MACDValues[len(daily.MACDValues)-1]
			prev := daily.MACDValues[len(daily.MACDValues)-2]
			switch {
			case last > 0 && last >= prev:
				metrics.DailyTrend = "bullish"
			case last < 0 && last <= prev:
				metrics.DailyTrend = "bearish"
			default:
				metrics.DailyTrend = "neutral"
			}
		} else {
			metrics.DailyTrend = "neutral"
		}

		if len(daily.CloseSeries) >= 20 {
			first := daily.CloseSeries[len(daily.CloseSeries)-20]
			last := daily.CloseSeries[len(daily.CloseSeries)-1]
			if first != 0 {
				metrics.DailyEMA20Slope = (last - first) / first * 100
			}
		}
	}

	if weekly != nil {
		if weekly.EMA20 > 0 {
			metrics.WeeklyPriceVsEMA = ((currentPrice - weekly.EMA20) / weekly.EMA20) * 100
		}
		if len(weekly.MACDValues) >= 2 {
			last := weekly.MACDValues[len(weekly.MACDValues)-1]
			prev := weekly.MACDValues[len(weekly.MACDValues)-2]
			switch {
			case last > 0 && last >= prev:
				metrics.WeeklyTrend = "bullish"
			case last < 0 && last <= prev:
				metrics.WeeklyTrend = "bearish"
			default:
				metrics.WeeklyTrend = "neutral"
			}
		} else {
			metrics.WeeklyTrend = "neutral"
		}
		if len(weekly.CloseSeries) >= 20 {
			first := weekly.CloseSeries[len(weekly.CloseSeries)-20]
			last := weekly.CloseSeries[len(weekly.CloseSeries)-1]
			if first != 0 {
				metrics.WeeklyEMA20Slope = (last - first) / first * 100
			}
		}
	}

	return metrics
}

// getOpenInterestData 获取OI数据
func getOpenInterestData(symbol string) (*OIData, error) {
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/openInterest?symbol=%s", symbol)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OpenInterest string `json:"openInterest"`
		Symbol       string `json:"symbol"`
		Time         int64  `json:"time"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	oi, _ := strconv.ParseFloat(result.OpenInterest, 64)

	return &OIData{
		Latest:  oi,
		Average: oi * 0.999, // 近似平均值
	}, nil
}

// getFundingRate 获取资金费率
func getFundingRate(symbol string) (float64, error) {
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/premiumIndex?symbol=%s", symbol)

	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result struct {
		Symbol          string `json:"symbol"`
		MarkPrice       string `json:"markPrice"`
		IndexPrice      string `json:"indexPrice"`
		LastFundingRate string `json:"lastFundingRate"`
		NextFundingTime int64  `json:"nextFundingTime"`
		InterestRate    string `json:"interestRate"`
		Time            int64  `json:"time"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	rate, _ := strconv.ParseFloat(result.LastFundingRate, 64)
	return rate, nil
}

// Format 格式化输出市场数据
func Format(data *Data) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("current_price = %.2f, current_ema20 = %.3f, current_macd = %.3f, current_rsi (7 period) = %.3f\n\n",
		data.CurrentPrice, data.CurrentEMA20, data.CurrentMACD, data.CurrentRSI7))

	sb.WriteString(fmt.Sprintf("In addition, here is the latest %s open interest and funding rate for perps:\n\n",
		data.Symbol))

	if data.OpenInterest != nil {
		sb.WriteString(fmt.Sprintf("Open Interest: Latest: %.2f Average: %.2f\n\n",
			data.OpenInterest.Latest, data.OpenInterest.Average))
	}

	sb.WriteString(fmt.Sprintf("Funding Rate: %.2e\n\n", data.FundingRate))

	if data.IntradaySeries != nil {
		sb.WriteString("Intraday series (3‑minute intervals, oldest → latest):\n\n")

		if len(data.IntradaySeries.MidPrices) > 0 {
			sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.IntradaySeries.MidPrices)))
		}

		if len(data.IntradaySeries.EMA20Values) > 0 {
			sb.WriteString(fmt.Sprintf("EMA indicators (20‑period): %s\n\n", formatFloatSlice(data.IntradaySeries.EMA20Values)))
		}

		if len(data.IntradaySeries.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.IntradaySeries.MACDValues)))
		}

		if len(data.IntradaySeries.RSI7Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (7‑Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI7Values)))
		}

		if len(data.IntradaySeries.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI14Values)))
		}
	}

	if data.DerivedMetrics != nil {
		sb.WriteString("Derived trend metrics:\n\n")
		sb.WriteString(fmt.Sprintf("price_vs_ema20_pct = %.3f, ema20_slope = %.3f, macd_slope = %.3f, rsi7_slope = %.3f, volatility_ratio = %.3f, htf_trend = %s, daily_trend = %s, daily_price_vs_ema20_pct = %.3f, daily_ema20_slope = %.3f, weekly_trend = %s, weekly_price_vs_ema20_pct = %.3f, weekly_ema20_slope = %.3f\n\n",
			data.DerivedMetrics.PriceVsEMA20Pct,
			data.DerivedMetrics.EMA20Slope,
			data.DerivedMetrics.MACDSlope,
			data.DerivedMetrics.RSI7Slope,
			data.DerivedMetrics.VolatilityRatio,
			data.DerivedMetrics.HTFTrend,
			data.DerivedMetrics.DailyTrend,
			data.DerivedMetrics.DailyPriceVsEMA,
			data.DerivedMetrics.DailyEMA20Slope,
			data.DerivedMetrics.WeeklyTrend,
			data.DerivedMetrics.WeeklyPriceVsEMA,
			data.DerivedMetrics.WeeklyEMA20Slope,
		))
	}

	if data.LongerTermContext != nil {
		sb.WriteString("Longer‑term context (4‑hour timeframe):\n\n")

		sb.WriteString(fmt.Sprintf("20‑Period EMA: %.3f vs. 50‑Period EMA: %.3f\n\n",
			data.LongerTermContext.EMA20, data.LongerTermContext.EMA50))

		sb.WriteString(fmt.Sprintf("3‑Period ATR: %.3f vs. 14‑Period ATR: %.3f\n\n",
			data.LongerTermContext.ATR3, data.LongerTermContext.ATR14))

		sb.WriteString(fmt.Sprintf("Current Volume: %.3f vs. Average Volume: %.3f\n\n",
			data.LongerTermContext.CurrentVolume, data.LongerTermContext.AverageVolume))

		if len(data.LongerTermContext.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.LongerTermContext.MACDValues)))
		}

		if len(data.LongerTermContext.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.LongerTermContext.RSI14Values)))
		}
	}

	if data.DailyContext != nil {
		sb.WriteString("Daily context (1‑day timeframe):\n\n")
		sb.WriteString(fmt.Sprintf("20‑Period EMA: %.3f vs. 50‑Period EMA: %.3f\n\n", data.DailyContext.EMA20, data.DailyContext.EMA50))
		sb.WriteString(fmt.Sprintf("14‑Period ATR: %.3f\n\n", data.DailyContext.ATR14))
		sb.WriteString(fmt.Sprintf("Current Volume: %.3f vs. Average Volume: %.3f\n\n", data.DailyContext.CurrentVolume, data.DailyContext.AverageVolume))

		dailySummary := map[string]interface{}{
			"ema20":      data.DailyContext.EMA20,
			"ema50":      data.DailyContext.EMA50,
			"atr14":      data.DailyContext.ATR14,
			"macd_mult":  truncateSlice(data.DailyContext.MACDValues, 6),
			"rsi14_mult": truncateSlice(data.DailyContext.RSI14Values, 6),
		}
		if jsonBlob, err := json.MarshalIndent(dailySummary, "", "  "); err == nil {
			sb.WriteString("```json\n")
			sb.WriteString(string(jsonBlob))
			sb.WriteString("\n```\n\n")
		}
	}

	if data.WeeklyContext != nil {
		sb.WriteString("Weekly context (1‑week timeframe):\n\n")
		sb.WriteString(fmt.Sprintf("20‑Period EMA: %.3f vs. 50‑Period EMA: %.3f\n\n", data.WeeklyContext.EMA20, data.WeeklyContext.EMA50))
		sb.WriteString(fmt.Sprintf("14‑Period ATR: %.3f\n\n", data.WeeklyContext.ATR14))

		weeklySummary := map[string]interface{}{
			"ema20":     data.WeeklyContext.EMA20,
			"ema50":     data.WeeklyContext.EMA50,
			"atr14":     data.WeeklyContext.ATR14,
			"macd_mult": truncateSlice(data.WeeklyContext.MACDValues, 6),
			"rsi14_mult": truncateSlice(data.WeeklyContext.RSI14Values, 6),
		}
		if jsonBlob, err := json.MarshalIndent(weeklySummary, "", "  "); err == nil {
			sb.WriteString("```json\n")
			sb.WriteString(string(jsonBlob))
			sb.WriteString("\n```\n\n")
		}
	}

	return sb.String()
}

// formatFloatSlice 格式化float64切片为字符串
func formatFloatSlice(values []float64) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = fmt.Sprintf("%.3f", v)
	}
	return "[" + strings.Join(strValues, ", ") + "]"
}

func truncateSlice(values []float64, limit int) []float64 {
	if len(values) <= limit {
		return values
	}
	return values[len(values)-limit:]
}

// Normalize 标准化symbol,确保是USDT交易对
func Normalize(symbol string) string {
	symbol = strings.ToUpper(symbol)
	if strings.HasSuffix(symbol, "USDT") {
		return symbol
	}
	return symbol + "USDT"
}

// parseFloat 解析float值
func parseFloat(v interface{}) (float64, error) {
	switch val := v.(type) {
	case string:
		return strconv.ParseFloat(val, 64)
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}
