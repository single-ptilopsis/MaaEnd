package autostockpile

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
)

var (
	// Interface implementation assertions for compile-time verification
	_ maa.CustomActionRunner      = &SelectItemAction{}
	_ maa.CustomRecognitionRunner = &ItemValueChangeRecognition{}
)

// SelectItemAction handles item selection based on recognition results
type SelectItemAction struct{}

// ItemValueChangeRecognition handles recognition of item value changes
type ItemValueChangeRecognition struct{}

// RecognitionResult represents the output of item value change recognition
type RecognitionResult struct {
	Overflow bool        `json:"overflow"`
	Sunday   bool        `json:"sunday"`
	Goods    []GoodsItem `json:"Goods"`
}

// GoodsItem represents a single recognized good from the recognition result
type GoodsItem struct {
	Name  string `json:"name"`  // Image template path
	Tier  string `json:"tier"`  // e.g., "WulingTier1"
	Price int    `json:"price"` // Current price of the good
}

// SelectionResult represents the item selection decision result
type SelectionResult struct {
	Selected      bool
	ProductName   string
	CanonicalName string
	Threshold     int
	CurrentPrice  int
	Score         int
	Reason        string
}

// SelectionConfig represents the configuration for AutoStockpile selection strategy
type SelectionConfig struct {
	Strategy          string           `json:"strategy"`
	OverflowMode      bool             `json:"overflow_mode"`
	SundayMode        bool             `json:"sunday_mode"`
	FallbackThreshold int              `json:"fallback_threshold"`
	PriceLimits       PriceLimitConfig `json:"price_limits"`
}

// PriceLimitConfig represents the tier-based price thresholds for buying decisions
type PriceLimitConfig struct {
	ValleyIVTier1 int `json:"valley_iv_tier1"`
	ValleyIVTier2 int `json:"valley_iv_tier2"`
	ValleyIVTier3 int `json:"valley_iv_tier3"`
	WulingTier1   int `json:"wuling_tier1"`
}

// ThresholdConfig represents the threshold configuration for item matching and pricing
type ThresholdConfig struct {
	FallbackThreshold int              `json:"fallback_threshold"`
	PriceLimits       PriceLimitConfig `json:"price_limits"`
}

// ItemMatchResult represents the result of matching an OCR item name to a canonical item
type ItemMatchResult struct {
	OCRName       string
	CanonicalName string
	TierID        string
	EditDistance  int
	Threshold     int
	Matched       bool
}

// Utility Functions

// absInt returns the absolute value of an integer
func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// minInt returns the minimum of three integers
func minInt(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// Constants

const (
	defaultFallbackBuyThreshold = 1000
)
