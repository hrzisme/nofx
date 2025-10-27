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
	Symbol      string  `json:"symbol"`       // 币种符号（例如：BTCUSDT）
	Price       float64 `json:"price"`        // 当前价格
	Volume24h   float64 `json:"volume_24h"`   // 24小时交易量
	Change24h   float64 `json:"change_24h"`   // 24小时涨跌幅
	IsAvailable bool    `json:"is_available"` // 是否可交易
}

// CoinPoolResponse API响应结构
type CoinPoolResponse struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    []CoinInfo `json:"data"`
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

	// 尝试多种解析方式
	// 方式1: 标准响应格式
	var poolResp CoinPoolResponse
	if err := json.Unmarshal(body, &poolResp); err == nil {
		if poolResp.Code == 0 || poolResp.Code == 200 {
			return poolResp.Data, nil
		}
	}

	// 方式2: 直接数组格式
	var coins []CoinInfo
	if err := json.Unmarshal(body, &coins); err == nil {
		return coins, nil
	}

	// 方式3: 简单的字符串数组（symbol列表）
	var symbols []string
	if err := json.Unmarshal(body, &symbols); err == nil {
		coins := make([]CoinInfo, len(symbols))
		for i, symbol := range symbols {
			coins[i] = CoinInfo{
				Symbol:      symbol,
				IsAvailable: true,
			}
		}
		return coins, nil
	}

	// 方式4: 对象格式 {"data": [...]}
	var objResp struct {
		Data interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &objResp); err == nil && objResp.Data != nil {
		// 将data转换为JSON再解析
		dataJSON, _ := json.Marshal(objResp.Data)

		// 尝试解析为CoinInfo数组
		if err := json.Unmarshal(dataJSON, &coins); err == nil {
			return coins, nil
		}

		// 尝试解析为字符串数组
		if err := json.Unmarshal(dataJSON, &symbols); err == nil {
			coins := make([]CoinInfo, len(symbols))
			for i, symbol := range symbols {
				coins[i] = CoinInfo{
					Symbol:      symbol,
					IsAvailable: true,
				}
			}
			return coins, nil
		}
	}

	return nil, fmt.Errorf("无法解析API响应: %s", string(body))
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
			symbol := normalizeSymbol(coin.Symbol)
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
