package pool

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// CoinPoolConfig 币种池配置
type CoinPoolConfig struct {
	APIURL  string
	Timeout time.Duration
}

var coinPoolConfig = CoinPoolConfig{
	APIURL:  "http://43.128.34.180:30006/api/ai500/list?auth=admin123sadasd3r323",
	Timeout: 10 * time.Second,
}

// CoinInfo 币种信息
type CoinInfo struct {
	Pair            string  `json:"pair"`             // 交易对符号（例如：BTCUSDT）
	Score           float64 `json:"score"`            // 当前评分
	StartTime       int64   `json:"start_time"`       // 开始时间（Unix时间戳）
	StartPrice      float64 `json:"start_price"`      // 开始价格
	LastScore       float64 `json:"last_score"`       // 最新评分
	MaxScore        float64 `json:"max_score"`        // 最高评分
	MaxPrice        float64 `json:"max_price"`        // 最高价格
	IncreasePercent float64 `json:"increase_percent"` // 涨幅百分比
	IsAvailable     bool    `json:"-"`                // 是否可交易（内部使用）
}

// CoinPoolAPIResponse API返回的原始数据结构
type CoinPoolAPIResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Coins []CoinInfo `json:"coins"`
		Count int        `json:"count"`
	} `json:"data"`
}

// SetCoinPoolAPI 设置币种池API
func SetCoinPoolAPI(apiURL string) {
	coinPoolConfig.APIURL = apiURL
}

// GetCoinPool 获取币种池列表
func GetCoinPool() ([]CoinInfo, error) {
	client := &http.Client{
		Timeout: coinPoolConfig.Timeout,
	}

	resp, err := client.Get(coinPoolConfig.APIURL)
	if err != nil {
		return nil, fmt.Errorf("请求币种池API失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误 (status %d): %s", resp.StatusCode, string(body))
	}

	// 解析API响应
	var response CoinPoolAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API返回失败状态")
	}

	if len(response.Data.Coins) == 0 {
		return nil, fmt.Errorf("币种列表为空")
	}

	// 设置IsAvailable标志
	coins := response.Data.Coins
	for i := range coins {
		coins[i].IsAvailable = true
	}

	return coins, nil
}

// GetAvailableCoins 获取可用的币种列表（过滤不可用的）
func GetAvailableCoins() ([]string, error) {
	coins, err := GetCoinPool()
	if err != nil {
		return nil, err
	}

	var symbols []string
	for _, coin := range coins {
		if coin.IsAvailable {
			// 确保symbol格式正确（转为大写USDT交易对）
			symbol := normalizeSymbol(coin.Pair)
			symbols = append(symbols, symbol)
		}
	}

	if len(symbols) == 0 {
		return nil, fmt.Errorf("没有可用的币种")
	}

	return symbols, nil
}

// normalizeSymbol 标准化币种符号
func normalizeSymbol(symbol string) string {
	// 移除空格
	symbol = trimSpaces(symbol)

	// 转为大写
	symbol = toUpper(symbol)

	// 确保以USDT结尾
	if !endsWith(symbol, "USDT") {
		symbol = symbol + "USDT"
	}

	return symbol
}

// 辅助函数
func trimSpaces(s string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' {
			result += string(s[i])
		}
	}
	return result
}

func toUpper(s string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c = c - 'a' + 'A'
		}
		result += string(c)
	}
	return result
}

func endsWith(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}
