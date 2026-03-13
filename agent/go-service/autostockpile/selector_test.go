package autostockpile

import (
	"testing"
)

// TestResolveTierThreshold tests tier threshold resolution logic
func TestResolveTierThreshold(t *testing.T) {
	testCases := []struct {
		name              string
		tierID            string
		cfg               SelectionConfig
		expectedThreshold int
	}{
		{
			name:   "ValleyIVTier1 with configured limit",
			tierID: "ValleyIVTier1",
			cfg: SelectionConfig{
				FallbackThreshold: 1000,
				PriceLimits: PriceLimitConfig{
					ValleyIVTier1: 1000,
					ValleyIVTier2: 1400,
					ValleyIVTier3: 1700,
					WulingTier1:   1700,
				},
			},
			expectedThreshold: 1000,
		},
		{
			name:   "ValleyIVTier2 with configured limit",
			tierID: "ValleyIVTier2",
			cfg: SelectionConfig{
				FallbackThreshold: 1000,
				PriceLimits: PriceLimitConfig{
					ValleyIVTier1: 1000,
					ValleyIVTier2: 1400,
					ValleyIVTier3: 1700,
					WulingTier1:   1700,
				},
			},
			expectedThreshold: 1400,
		},
		{
			name:   "ValleyIVTier3 with configured limit",
			tierID: "ValleyIVTier3",
			cfg: SelectionConfig{
				FallbackThreshold: 1000,
				PriceLimits: PriceLimitConfig{
					ValleyIVTier1: 1000,
					ValleyIVTier2: 1400,
					ValleyIVTier3: 1700,
					WulingTier1:   1700,
				},
			},
			expectedThreshold: 1700,
		},
		{
			name:   "WulingTier1 with configured limit",
			tierID: "WulingTier1",
			cfg: SelectionConfig{
				FallbackThreshold: 1000,
				PriceLimits: PriceLimitConfig{
					ValleyIVTier1: 1000,
					ValleyIVTier2: 1400,
					ValleyIVTier3: 1700,
					WulingTier1:   1700,
				},
			},
			expectedThreshold: 1700,
		},
		{
			name:   "Unknown tier uses fallback",
			tierID: "UnknownTier",
			cfg: SelectionConfig{
				FallbackThreshold: 1200,
				PriceLimits: PriceLimitConfig{
					ValleyIVTier1: 1000,
					ValleyIVTier2: 1400,
					ValleyIVTier3: 1700,
					WulingTier1:   1700,
				},
			},
			expectedThreshold: 1200,
		},
		{
			name:   "ValleyIVTier1 with zero limit uses fallback",
			tierID: "ValleyIVTier1",
			cfg: SelectionConfig{
				FallbackThreshold: 1500,
				PriceLimits: PriceLimitConfig{
					ValleyIVTier1: 0,
					ValleyIVTier2: 1400,
					ValleyIVTier3: 1700,
					WulingTier1:   1700,
				},
			},
			expectedThreshold: 1500,
		},
		{
			name:   "Empty tier string uses fallback",
			tierID: "",
			cfg: SelectionConfig{
				FallbackThreshold: 800,
				PriceLimits:       PriceLimitConfig{},
			},
			expectedThreshold: 800,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			threshold := resolveTierThreshold(tc.tierID, tc.cfg)
			if threshold != tc.expectedThreshold {
				t.Errorf("expected threshold=%d, got %d", tc.expectedThreshold, threshold)
			}
		})
	}
}

// TestResolveFallbackThreshold tests fallback threshold resolution logic
func TestResolveFallbackThreshold(t *testing.T) {
	testCases := []struct {
		name              string
		rawFallback       int
		expectedThreshold int
	}{
		{
			name:              "Positive fallback value",
			rawFallback:       1200,
			expectedThreshold: 1200,
		},
		{
			name:              "Zero fallback uses default",
			rawFallback:       0,
			expectedThreshold: defaultFallbackBuyThreshold,
		},
		{
			name:              "Negative fallback uses default",
			rawFallback:       -100,
			expectedThreshold: defaultFallbackBuyThreshold,
		},
		{
			name:              "Very large fallback value",
			rawFallback:       999999,
			expectedThreshold: 999999,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			threshold := resolveFallbackThreshold(tc.rawFallback)
			if threshold != tc.expectedThreshold {
				t.Errorf("expected threshold=%d, got %d", tc.expectedThreshold, threshold)
			}
		})
	}
}

// TestSelectBestProduct_ScoreCalculation tests score = threshold - price logic
func TestSelectBestProduct_ScoreCalculation(t *testing.T) {
	cfg := SelectionConfig{
		FallbackThreshold: 1000,
		PriceLimits: PriceLimitConfig{
			ValleyIVTier1: 1000,
			ValleyIVTier2: 1400,
			ValleyIVTier3: 1700,
			WulingTier1:   1700,
		},
	}

	testCases := []struct {
		name          string
		result        RecognitionResult
		allowAll      bool
		expectedScore int
		expectedName  string
		shouldSelect  bool
	}{
		{
			name: "Positive score: threshold 1700, price 1600",
			result: RecognitionResult{
				Goods: []GoodsItem{
					{Name: "源石树幼苗货组.Tier3.png", Tier: "ValleyIVTier3", Price: 1600},
				},
			},
			allowAll:      false,
			expectedScore: 100,
			expectedName:  "源石树幼苗货组.Tier3.png",
			shouldSelect:  true,
		},
		{
			name: "Zero score: threshold 1000, price 1000",
			result: RecognitionResult{
				Goods: []GoodsItem{
					{Name: "天使罐头货组.Tier1.png", Tier: "ValleyIVTier1", Price: 1000},
				},
			},
			allowAll:     false,
			shouldSelect: false,
		},
		{
			name: "Negative score: threshold 1000, price 1100",
			result: RecognitionResult{
				Goods: []GoodsItem{
					{Name: "天使罐头货组.Tier1.png", Tier: "ValleyIVTier1", Price: 1100},
				},
			},
			allowAll:     false,
			shouldSelect: false,
		},
		{
			name: "Large positive score: threshold 1700, price 1200",
			result: RecognitionResult{
				Goods: []GoodsItem{
					{Name: "边角料积木货组.Tier3.png", Tier: "ValleyIVTier3", Price: 1200},
				},
			},
			allowAll:      false,
			expectedScore: 500,
			expectedName:  "边角料积木货组.Tier3.png",
			shouldSelect:  true,
		},
		{
			name: "AllowAll mode: negative score accepted",
			result: RecognitionResult{
				Goods: []GoodsItem{
					{Name: "天使罐头货组.Tier1.png", Tier: "ValleyIVTier1", Price: 1100},
				},
			},
			allowAll:      true,
			expectedScore: -100,
			expectedName:  "天使罐头货组.Tier1.png",
			shouldSelect:  true,
		},
		{
			name: "Multiple goods: select highest score",
			result: RecognitionResult{
				Goods: []GoodsItem{
					{Name: "天使罐头货组.Tier1.png", Tier: "ValleyIVTier1", Price: 950},
					{Name: "谷地水培肉货组.Tier2.png", Tier: "ValleyIVTier2", Price: 1300},
					{Name: "源石树幼苗货组.Tier3.png", Tier: "ValleyIVTier3", Price: 1500},
				},
			},
			allowAll:      false,
			expectedScore: 200,
			expectedName:  "源石树幼苗货组.Tier3.png",
			shouldSelect:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			selection := SelectBestProduct(tc.result, cfg, tc.allowAll)
			if selection.Selected != tc.shouldSelect {
				t.Errorf("expected selected=%v, got %v", tc.shouldSelect, selection.Selected)
			}
			if tc.shouldSelect {
				if selection.Score != tc.expectedScore {
					t.Errorf("expected score=%d, got %d", tc.expectedScore, selection.Score)
				}
				if selection.ProductName != tc.expectedName {
					t.Errorf("expected name=%q, got %q", tc.expectedName, selection.ProductName)
				}
			}
		})
	}
}

// TestSelectBestProduct_EmptyGoods tests behavior with empty goods list
func TestSelectBestProduct_EmptyGoods(t *testing.T) {
	cfg := SelectionConfig{
		FallbackThreshold: 1000,
		PriceLimits: PriceLimitConfig{
			ValleyIVTier1: 1000,
		},
	}

	testCases := []struct {
		name           string
		result         RecognitionResult
		allowAll       bool
		expectedReason string
	}{
		{
			name: "Empty goods list",
			result: RecognitionResult{
				Goods: []GoodsItem{},
			},
			allowAll:       false,
			expectedReason: "no_goods",
		},
		{
			name: "Nil goods (zero value)",
			result: RecognitionResult{
				Goods: nil,
			},
			allowAll:       false,
			expectedReason: "no_goods",
		},
		{
			name: "Empty goods list with allowAll",
			result: RecognitionResult{
				Goods: []GoodsItem{},
			},
			allowAll:       true,
			expectedReason: "no_goods",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			selection := SelectBestProduct(tc.result, cfg, tc.allowAll)
			if selection.Selected {
				t.Errorf("expected selected=false, got true")
			}
			if selection.Reason != tc.expectedReason {
				t.Errorf("expected reason=%q, got %q", tc.expectedReason, selection.Reason)
			}
		})
	}
}

// TestSelectBestProduct_AllNegativeScores tests all goods have score ≤ 0
func TestSelectBestProduct_AllNegativeScores(t *testing.T) {
	cfg := SelectionConfig{
		FallbackThreshold: 1000,
		PriceLimits: PriceLimitConfig{
			ValleyIVTier1: 1000,
			ValleyIVTier2: 1400,
			ValleyIVTier3: 1700,
		},
	}

	testCases := []struct {
		name           string
		result         RecognitionResult
		allowAll       bool
		expectedReason string
		shouldSelect   bool
	}{
		{
			name: "All negative scores, allowAll=false",
			result: RecognitionResult{
				Goods: []GoodsItem{
					{Name: "天使罐头货组.Tier1.png", Tier: "ValleyIVTier1", Price: 1100},
					{Name: "谷地水培肉货组.Tier2.png", Tier: "ValleyIVTier2", Price: 1500},
					{Name: "源石树幼苗货组.Tier3.png", Tier: "ValleyIVTier3", Price: 1800},
				},
			},
			allowAll:       false,
			expectedReason: "no_qualifying_products",
			shouldSelect:   false,
		},
		{
			name: "All zero scores, allowAll=false",
			result: RecognitionResult{
				Goods: []GoodsItem{
					{Name: "天使罐头货组.Tier1.png", Tier: "ValleyIVTier1", Price: 1000},
					{Name: "谷地水培肉货组.Tier2.png", Tier: "ValleyIVTier2", Price: 1400},
				},
			},
			allowAll:       false,
			expectedReason: "no_qualifying_products",
			shouldSelect:   false,
		},
		{
			name: "All negative scores, allowAll=true selects least negative",
			result: RecognitionResult{
				Goods: []GoodsItem{
					{Name: "天使罐头货组.Tier1.png", Tier: "ValleyIVTier1", Price: 1100},
					{Name: "谷地水培肉货组.Tier2.png", Tier: "ValleyIVTier2", Price: 1450},
					{Name: "源石树幼苗货组.Tier3.png", Tier: "ValleyIVTier3", Price: 1800},
				},
			},
			allowAll:     true,
			shouldSelect: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			selection := SelectBestProduct(tc.result, cfg, tc.allowAll)
			if selection.Selected != tc.shouldSelect {
				t.Errorf("expected selected=%v, got %v", tc.shouldSelect, selection.Selected)
			}
			if !tc.shouldSelect && selection.Reason != tc.expectedReason {
				t.Errorf("expected reason=%q, got %q", tc.expectedReason, selection.Reason)
			}
			if tc.shouldSelect && tc.allowAll {
				if selection.ProductName != "谷地水培肉货组.Tier2.png" {
					t.Errorf("expected least negative score (closest to 0), got %q", selection.ProductName)
				}
				if selection.Score != -50 {
					t.Errorf("expected score=-50 (least negative), got %d", selection.Score)
				}
			}
		})
	}
}

// TestSelectBestProduct_SortingStability tests tiebreaker logic
func TestSelectBestProduct_SortingStability(t *testing.T) {
	cfg := SelectionConfig{
		FallbackThreshold: 1000,
		PriceLimits: PriceLimitConfig{
			ValleyIVTier1: 1000,
			ValleyIVTier2: 1400,
			ValleyIVTier3: 1700,
		},
	}

	testCases := []struct {
		name         string
		result       RecognitionResult
		expectedName string
		reason       string
	}{
		{
			name: "Same score, different prices: prefer lower price",
			result: RecognitionResult{
				Goods: []GoodsItem{
					{Name: "边角料积木货组.Tier3.png", Tier: "ValleyIVTier3", Price: 1600},
					{Name: "源石树幼苗货组.Tier3.png", Tier: "ValleyIVTier3", Price: 1500},
				},
			},
			expectedName: "源石树幼苗货组.Tier3.png",
			reason:       "Same score (100), lower price (1500 < 1600) wins",
		},
		{
			name: "Same score and price, different tiers: prefer lower tier string",
			result: RecognitionResult{
				Goods: []GoodsItem{
					{Name: "边角料积木货组.Tier3.png", Tier: "ValleyIVTier3", Price: 1600},
					{Name: "武侠电影货组.Tier1.png", Tier: "WulingTier1", Price: 1600},
				},
			},
			expectedName: "边角料积木货组.Tier3.png",
			reason:       "Same score (100) and price (1600), tier 'ValleyIVTier3' < 'WulingTier1' lexically",
		},
		{
			name: "Same score, price, tier: stable sort preserves order",
			result: RecognitionResult{
				Goods: []GoodsItem{
					{Name: "源石树幼苗货组.Tier3.png", Tier: "ValleyIVTier3", Price: 1600},
					{Name: "边角料积木货组.Tier3.png", Tier: "ValleyIVTier3", Price: 1600},
				},
			},
			expectedName: "源石树幼苗货组.Tier3.png",
			reason:       "Same score, price, tier; stable sort preserves input order (first wins)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			selection := SelectBestProduct(tc.result, cfg, false)
			if !selection.Selected {
				t.Fatalf("expected selected=true, got false")
			}
			if selection.ProductName != tc.expectedName {
				t.Errorf("%s\nexpected name=%q, got %q", tc.reason, tc.expectedName, selection.ProductName)
			}
		})
	}
}

// TestParseSelectionConfigFromNodeJSON tests config parsing
func TestParseSelectionConfigFromNodeJSON(t *testing.T) {
	testCases := []struct {
		name        string
		rawJSON     string
		expectError bool
		validate    func(*testing.T, SelectionConfig)
	}{
		{
			name:        "Complete config",
			rawJSON:     `{"attach":{"strategy":"Strict","overflow_mode":true,"sunday_mode":false,"fallback_threshold":1200,"price_limits":{"valley_iv_tier1":800,"valley_iv_tier2":1200,"valley_iv_tier3":1500,"wuling_tier1":1500}}}`,
			expectError: false,
			validate: func(t *testing.T, cfg SelectionConfig) {
				if cfg.Strategy != "Strict" || !cfg.OverflowMode || cfg.SundayMode {
					t.Errorf("unexpected base config: %+v", cfg)
				}
				if cfg.FallbackThreshold != 1200 {
					t.Errorf("unexpected fallback threshold: got %d", cfg.FallbackThreshold)
				}
				if cfg.PriceLimits.ValleyIVTier1 != 800 || cfg.PriceLimits.WulingTier1 != 1500 {
					t.Errorf("unexpected price limits: %+v", cfg.PriceLimits)
				}
			},
		},
		{
			name:        "Minimal config with defaults",
			rawJSON:     `{"attach":{"strategy":"Recommend"}}`,
			expectError: false,
			validate: func(t *testing.T, cfg SelectionConfig) {
				if cfg.FallbackThreshold != defaultFallbackBuyThreshold {
					t.Errorf("expected default fallback threshold %d, got %d", defaultFallbackBuyThreshold, cfg.FallbackThreshold)
				}
				if cfg.Strategy != "Recommend" {
					t.Errorf("expected strategy=Recommend, got %s", cfg.Strategy)
				}
			},
		},
		{
			name:        "Invalid JSON",
			rawJSON:     `{invalid json}`,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := parseSelectionConfigFromNodeJSON(tc.rawJSON)
			if tc.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tc.validate != nil {
					tc.validate(t, cfg)
				}
			}
		})
	}
}
