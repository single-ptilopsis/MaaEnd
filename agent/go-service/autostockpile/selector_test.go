package autostockpile

import (
	"testing"
	"time"
)

func TestParseSelectionConfigFromNodeJSON(t *testing.T) {
	raw := `{"attach":{"strategy":"Strict","overflow_mode":true,"sunday_mode":false,"fallback_threshold":1200,"price_limits":{"valley_iv_tier1":800,"valley_iv_tier2":1200,"valley_iv_tier3":1500,"wuling_tier1":1500}}}`

	cfg, err := parseSelectionConfigFromNodeJSON(raw)
	if err != nil {
		t.Fatalf("expected parse success, got error: %v", err)
	}

	if cfg.Strategy != "Strict" || !cfg.OverflowMode || cfg.SundayMode {
		t.Fatalf("unexpected base config: %+v", cfg)
	}
	if cfg.FallbackThreshold != 1200 {
		t.Fatalf("unexpected fallback threshold: got %d", cfg.FallbackThreshold)
	}
	if cfg.PriceLimits.ValleyIVTier1 != 800 || cfg.PriceLimits.WulingTier1 != 1500 {
		t.Fatalf("unexpected price limits: %+v", cfg.PriceLimits)
	}
}

func TestParseSelectionConfigFromNodeJSONUsesDefaultFallback(t *testing.T) {
	raw := `{"attach":{"strategy":"Recommend"}}`

	cfg, err := parseSelectionConfigFromNodeJSON(raw)
	if err != nil {
		t.Fatalf("expected parse success, got error: %v", err)
	}

	if cfg.FallbackThreshold != defaultFallbackBuyThreshold {
		t.Fatalf("expected default fallback threshold %d, got %d", defaultFallbackBuyThreshold, cfg.FallbackThreshold)
	}
}

func TestNormalModeSelection(t *testing.T) {
	parsed := &OCRParseResult{
		CandidateRegions: []OCRCandidateRegion{
			newCandidateRegion("天使罐头货组", "950", 100, 200),
			newCandidateRegion("谷地水培肉货组", "1500", 100, 260),
			newCandidateRegion("源石树幼苗货组", "1600", 100, 320),
		},
	}

	result := SelectBestProduct(parsed, testSelectionConfig())
	if !result.Selected {
		t.Fatalf("expected selected=true, got false, reason=%q", result.Reason)
	}
	if result.ProductName != "源石树幼苗货组" {
		t.Fatalf("unexpected product: got %q", result.ProductName)
	}
	if result.Threshold != 1700 || result.CurrentPrice != 1600 || result.Score != 100 {
		t.Fatalf("unexpected score fields: threshold=%d price=%d score=%d", result.Threshold, result.CurrentPrice, result.Score)
	}
	if result.ClickX != 160 || result.ClickY != 332 {
		t.Fatalf("unexpected click center: got (%d,%d)", result.ClickX, result.ClickY)
	}
}

func TestOverflowModeSelection(t *testing.T) {
	cfg := testSelectionConfig()
	cfg.OverflowMode = true

	parsed := &OCRParseResult{
		OverflowMarkerTokens: []OCRToken{{Text: "9小时后+50即将溢出", X: 30, Y: 120, W: 100, H: 20}},
		CandidateRegions: []OCRCandidateRegion{
			newCandidateRegion("天使罐头货组", "1100", 80, 180),
			newCandidateRegion("谷地水培肉货组", "国1500", 90, 240),
			newCandidateRegion("源石树幼苗货组", "1900", 100, 300),
		},
	}

	result := SelectBestProduct(parsed, cfg)
	if !result.Selected {
		t.Fatalf("expected selected=true, got false, reason=%q", result.Reason)
	}
	if result.ProductName != "天使罐头货组" {
		t.Fatalf("expected least-negative score candidate, got %q", result.ProductName)
	}
	if result.Score != -100 {
		t.Fatalf("unexpected score: got %d, want -100", result.Score)
	}
}

func TestSundayModeSelection(t *testing.T) {
	cfg := testSelectionConfig()
	cfg.SundayMode = true

	parsed := &OCRParseResult{
		CandidateRegions: []OCRCandidateRegion{
			newCandidateRegion("天使罐头货组", "1200", 100, 210),
			newCandidateRegion("谷地水培肉货组", "1300", 100, 270),
			newCandidateRegion("源石树幼苗货组", "1650", 100, 330),
		},
	}

	result := selectBestProductAt(parsed, cfg, time.Date(2026, time.March, 8, 12, 0, 0, 0, time.UTC))
	if !result.Selected {
		t.Fatalf("expected selected=true, got false, reason=%q", result.Reason)
	}
	if result.ProductName != "谷地水培肉货组" {
		t.Fatalf("expected highest score candidate, got %q", result.ProductName)
	}
	if result.Score != 100 {
		t.Fatalf("unexpected score: got %d, want 100", result.Score)
	}
}

func TestNoQualifyingProduct(t *testing.T) {
	parsed := &OCRParseResult{
		CandidateRegions: []OCRCandidateRegion{
			newCandidateRegion("天使罐头货组", "1000", 100, 220),
			newCandidateRegion("谷地水培肉货组", "1700", 100, 280),
		},
	}

	result := SelectBestProduct(parsed, testSelectionConfig())
	if result.Selected {
		t.Fatalf("expected selected=false, got true with %q", result.ProductName)
	}
	if result.Reason != "no_qualifying_products" {
		t.Fatalf("unexpected reason: got %q", result.Reason)
	}
}

func TestSelectWithNilParsedResult(t *testing.T) {
	result := SelectBestProduct(nil, testSelectionConfig())
	if result.Selected {
		t.Fatalf("expected selected=false for nil parsed result, got true")
	}
	if result.Reason != "no_qualifying_products" {
		t.Fatalf("expected reason=no_qualifying_products, got %q", result.Reason)
	}
}

func TestSelectWithAllMissingPrices(t *testing.T) {
	parsed := &OCRParseResult{
		CandidateRegions: []OCRCandidateRegion{
			{
				Bounds: [4]int{100, 200, 350, 30},
				Tokens: []OCRToken{
					{Text: "天使罐头货组", X: 100, Y: 200, W: 120, H: 24},
					{Text: "无法解析的价格", X: 270, Y: 201, W: 80, H: 22},
				},
			},
			{
				Bounds: [4]int{100, 260, 350, 30},
				Tokens: []OCRToken{
					{Text: "谷地水培肉货组", X: 100, Y: 260, W: 120, H: 24},
					{Text: "19.6%", X: 270, Y: 261, W: 70, H: 20},
				},
			},
		},
	}

	result := SelectBestProduct(parsed, testSelectionConfig())
	if result.Selected {
		t.Fatalf("expected selected=false when all prices are missing, got true")
	}
	if result.Reason != "no_qualifying_products" {
		t.Fatalf("expected reason=no_qualifying_products, got %q", result.Reason)
	}
}

func TestSelectOverflowModeWithoutMarkers(t *testing.T) {
	cfg := testSelectionConfig()
	cfg.OverflowMode = true

	parsed := &OCRParseResult{
		OverflowMarkerTokens: []OCRToken{},
		CandidateRegions: []OCRCandidateRegion{
			newCandidateRegion("天使罐头货组", "1100", 100, 200),
		},
	}

	result := SelectBestProduct(parsed, cfg)
	if result.Selected {
		t.Fatalf("expected selected=false when overflow mode enabled but no markers detected, got true")
	}
	if result.Reason != "no_qualifying_products" {
		t.Fatalf("expected reason=no_qualifying_products, got %q", result.Reason)
	}
}

func TestSelectSundayModeOnNonSunday(t *testing.T) {
	cfg := testSelectionConfig()
	cfg.SundayMode = true

	parsed := &OCRParseResult{
		CandidateRegions: []OCRCandidateRegion{
			newCandidateRegion("天使罐头货组", "1100", 100, 200),
		},
	}

	nonSundayTime := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)
	result := selectBestProductAt(parsed, cfg, nonSundayTime)
	if result.Selected {
		t.Fatalf("expected selected=false when sunday mode enabled but current day is not sunday, got true")
	}
	if result.Reason != "no_qualifying_products" {
		t.Fatalf("expected reason=no_qualifying_products, got %q", result.Reason)
	}
}

func TestSelectTiebreakerStability(t *testing.T) {
	parsed := &OCRParseResult{
		CandidateRegions: []OCRCandidateRegion{
			newCandidateRegion("边角料积木货组", "1600", 200, 300),
			newCandidateRegion("源石树幼苗货组", "1600", 100, 200),
			newCandidateRegion("谷地水培肉货组", "1300", 150, 250),
		},
	}

	result := SelectBestProduct(parsed, testSelectionConfig())
	if !result.Selected {
		t.Fatalf("expected selected=true, got false")
	}
	if result.ProductName != "源石树幼苗货组" {
		t.Fatalf("expected tiebreaker to select smallest Y (200), then smallest X (100): got %q (x=%d, y=%d)",
			result.ProductName, result.ClickX, result.ClickY)
	}
}

func testSelectionConfig() SelectionConfig {
	return SelectionConfig{
		Strategy:          "Recommend",
		FallbackThreshold: 1000,
		PriceLimits: PriceLimitConfig{
			ValleyIVTier1: 1000,
			ValleyIVTier2: 1400,
			ValleyIVTier3: 1700,
			WulingTier1:   1700,
		},
	}
}

func newCandidateRegion(name, price string, x, y int) OCRCandidateRegion {
	nameToken := OCRToken{Text: name, X: x, Y: y, W: 120, H: 24}
	priceToken := OCRToken{Text: price, X: x + 170, Y: y + 1, W: 80, H: 22}
	percentToken := OCRToken{Text: "19.6%", X: x + 270, Y: y + 1, W: 70, H: 20}
	return OCRCandidateRegion{
		Bounds: [4]int{x, y, 350, 30},
		Tokens: []OCRToken{nameToken, priceToken, percentToken},
	}
}
