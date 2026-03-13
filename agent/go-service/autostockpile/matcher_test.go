package autostockpile

import (
	"strings"
	"testing"
)

func TestFuzzyMatchTier(t *testing.T) {
	cfg := ThresholdConfig{
		FallbackThreshold: 1000,
		PriceLimits: PriceLimitConfig{
			ValleyIVTier1: 1000,
			ValleyIVTier2: 1400,
			ValleyIVTier3: 1700,
			WulingTier1:   1700,
		},
	}

	testCases := []struct {
		name              string
		ocrName           string
		expectedCanonical string
		expectedTier      string
		expectedThreshold int
	}{
		{
			name:              "single char noise 幼↔纺",
			ocrName:           "源石树纺苗货组",
			expectedCanonical: "源石树幼苗货组",
			expectedTier:      "ValleyIVTier3",
			expectedThreshold: 1700,
		},
		{
			name:              "single char noise 悬↔晨 with missing char",
			ocrName:           "晨空兽骨货组",
			expectedCanonical: "悬空兽骨雕货组",
			expectedTier:      "ValleyIVTier1",
			expectedThreshold: 1000,
		},
		{
			name:              "double char noise 誓↔警 and 镐↔福",
			ocrName:           "警戒者矿福货组",
			expectedCanonical: "誓戒者矿镐货组",
			expectedTier:      "ValleyIVTier3",
			expectedThreshold: 1700,
		},
		{
			name:              "single char noise 木↔术",
			ocrName:           "巫术矿钻货组",
			expectedCanonical: "巫木矿钻货组",
			expectedTier:      "ValleyIVTier1",
			expectedThreshold: 1000,
		},
		{
			name:              "single char noise 侠↔快",
			ocrName:           "武快电影货组",
			expectedCanonical: "武侠电影货组",
			expectedTier:      "WulingTier1",
			expectedThreshold: 1700,
		},
		{
			name:              "single char noise 瘴↔潭",
			ocrName:           "岳研避潭茶货组",
			expectedCanonical: "岳研避瘴茶货组",
			expectedTier:      "WulingTier1",
			expectedThreshold: 1700,
		},
		{
			name:              "single char noise 陵↔度",
			ocrName:           "武度冻梨货组",
			expectedCanonical: "武陵冻梨货组",
			expectedTier:      "WulingTier1",
			expectedThreshold: 1700,
		},
		{
			name:              "multi char noise 避瘴↔建痘",
			ocrName:           "岳研建痘茶货组",
			expectedCanonical: "岳研避瘴茶货组",
			expectedTier:      "WulingTier1",
			expectedThreshold: 1700,
		},
		{
			name:              "multi char noise 夏草↔票笋",
			ocrName:           "冬虫票笋货组",
			expectedCanonical: "冬虫夏草货组",
			expectedTier:      "WulingTier1",
			expectedThreshold: 1700,
		},
		{
			name:              "exact valley tier2",
			ocrName:           "谷地水培肉货组",
			expectedCanonical: "谷地水培肉货组",
			expectedTier:      "ValleyIVTier2",
			expectedThreshold: 1400,
		},
		{
			name:              "exact valley tier3",
			ocrName:           "边角料积木货组",
			expectedCanonical: "边角料积木货组",
			expectedTier:      "ValleyIVTier3",
			expectedThreshold: 1700,
		},
		{
			name:              "exact valley tier1",
			ocrName:           "天使罐头货组",
			expectedCanonical: "天使罐头货组",
			expectedTier:      "ValleyIVTier1",
			expectedThreshold: 1000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := MatchItemTierAndThreshold(tc.ocrName, cfg)

			if !got.Matched {
				t.Fatalf("expected matched=true, got false for %q", tc.ocrName)
			}
			if got.CanonicalName != tc.expectedCanonical {
				t.Fatalf("canonical mismatch: got %q, want %q", got.CanonicalName, tc.expectedCanonical)
			}
			if got.TierID != tc.expectedTier {
				t.Fatalf("tier mismatch: got %q, want %q", got.TierID, tc.expectedTier)
			}
			if got.Threshold != tc.expectedThreshold {
				t.Fatalf("threshold mismatch: got %d, want %d", got.Threshold, tc.expectedThreshold)
			}
			if got.EditDistance < 0 || got.EditDistance > 2 {
				t.Fatalf("edit distance out of expected range [0,2]: %d", got.EditDistance)
			}
		})
	}
}

func TestUnknownItemFallback(t *testing.T) {
	cfg := ThresholdConfig{
		FallbackThreshold: 1000,
		PriceLimits: PriceLimitConfig{
			ValleyIVTier1: 1000,
			ValleyIVTier2: 1400,
			ValleyIVTier3: 1700,
			WulingTier1:   1700,
		},
	}

	got := MatchItemTierAndThreshold("镜点圆具货组", cfg)
	if got.Matched {
		t.Fatalf("expected unknown item to remain unmatched, got canonical=%q", got.CanonicalName)
	}
	if got.TierID != unknownTierID {
		t.Fatalf("expected tier=%q, got %q", unknownTierID, got.TierID)
	}
	if got.Threshold != 1000 {
		t.Fatalf("expected fallback threshold=1000, got %d", got.Threshold)
	}
	if got.EditDistance != -1 {
		t.Fatalf("expected edit distance=-1 for unmatched case, got %d", got.EditDistance)
	}
}

func TestMatchEmptyString(t *testing.T) {
	cfg := ThresholdConfig{
		FallbackThreshold: 1000,
		PriceLimits: PriceLimitConfig{
			ValleyIVTier1: 1000,
			ValleyIVTier2: 1400,
			ValleyIVTier3: 1700,
			WulingTier1:   1700,
		},
	}

	got := MatchItemTierAndThreshold("", cfg)
	if got.Matched {
		t.Fatalf("expected empty string to remain unmatched, got matched=true")
	}
	if got.TierID != unknownTierID {
		t.Fatalf("expected tier=%q, got %q", unknownTierID, got.TierID)
	}
	if got.Threshold != 1000 {
		t.Fatalf("expected fallback threshold=1000, got %d", got.Threshold)
	}
	if got.EditDistance != -1 {
		t.Fatalf("expected edit distance=-1 for unmatched case, got %d", got.EditDistance)
	}
}

func TestMatchASCIIOnly(t *testing.T) {
	cfg := ThresholdConfig{
		FallbackThreshold: 1000,
		PriceLimits: PriceLimitConfig{
			ValleyIVTier1: 1000,
			ValleyIVTier2: 1400,
			ValleyIVTier3: 1700,
			WulingTier1:   1700,
		},
	}

	got := MatchItemTierAndThreshold("ABC123XYZ", cfg)
	if got.Matched {
		t.Fatalf("expected ASCII-only input to remain unmatched, got matched=true")
	}
	if got.TierID != unknownTierID {
		t.Fatalf("expected tier=%q, got %q", unknownTierID, got.TierID)
	}
	if got.Threshold != 1000 {
		t.Fatalf("expected fallback threshold=1000, got %d", got.Threshold)
	}
}

func TestMatchVeryLongName(t *testing.T) {
	cfg := ThresholdConfig{
		FallbackThreshold: 1000,
		PriceLimits: PriceLimitConfig{
			ValleyIVTier1: 1000,
			ValleyIVTier2: 1400,
			ValleyIVTier3: 1700,
			WulingTier1:   1700,
		},
	}

	longName := strings.Repeat("测试", 30) + "货组"
	got := MatchItemTierAndThreshold(longName, cfg)

	if got.Matched {
		t.Fatalf("expected very long name to remain unmatched, got matched=true")
	}
	if got.TierID != unknownTierID {
		t.Fatalf("expected tier=%q, got %q", unknownTierID, got.TierID)
	}
	if got.Threshold != 1000 {
		t.Fatalf("expected fallback threshold=1000, got %d", got.Threshold)
	}
}
