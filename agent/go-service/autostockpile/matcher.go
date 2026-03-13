package autostockpile

import (
	"strings"
	"sync"
	"unicode"
)

import "github.com/rs/zerolog/log"

const (
	maxItemNameEditDistance     = 2
	unknownTierID               = "unknown"
	defaultFallbackBuyThreshold = 1000
)

// PriceLimitConfig 表示 custom_action_param.price_limits 的阈值配置。
type PriceLimitConfig struct {
	ValleyIVTier1 int `json:"valley_iv_tier1"`
	ValleyIVTier2 int `json:"valley_iv_tier2"`
	ValleyIVTier3 int `json:"valley_iv_tier3"`
	WulingTier1   int `json:"wuling_tier1"`
}

// ThresholdConfig 表示 custom_action_param 中与阈值决策相关的配置。
type ThresholdConfig struct {
	FallbackThreshold int              `json:"fallback_threshold"`
	PriceLimits       PriceLimitConfig `json:"price_limits"`
}

// ItemMatchResult 表示 OCR 商品名与阈值解析结果。
type ItemMatchResult struct {
	OCRName       string
	CanonicalName string
	TierID        string
	EditDistance  int
	Threshold     int
	Matched       bool
}

type canonicalItem struct {
	Name string
	Tier string
	Raw  string
	Norm string
}

var (
	canonicalItemsOnce sync.Once
	canonicalItems     []canonicalItem
	canonicalItemsErr  error
	noiseNormalizer    = strings.NewReplacer(
		"建痘", "避瘴",
		"避瘴", "避瘴",
		"票笋", "夏草",
		"夏草", "夏草",
		"纺", "幼",
		"幼", "幼",
		"晨", "悬",
		"悬", "悬",
		"警", "誓",
		"誓", "誓",
		"福", "镐",
		"镐", "镐",
		"术", "木",
		"木", "木",
		"快", "侠",
		"侠", "侠",
		"潭", "瘴",
		"瘴", "瘴",
		"度", "陵",
		"陵", "陵",
		"佣", "拥",
		"拥", "拥",
	)
	knownNoisePairs = [][2]string{
		{"幼", "纺"},
		{"悬", "晨"},
		{"誓", "警"},
		{"镐", "福"},
		{"木", "术"},
		{"侠", "快"},
		{"瘴", "潭"},
		{"陵", "度"},
		{"拥", "佣"},
		{"避瘴", "建痘"},
		{"夏草", "票笋"},
	}
)

// MatchItemTierAndThreshold 将 OCR 商品名映射到 canonical item，并返回对应 tier 与买入阈值。
// 当无法匹配到任何商品时，返回 unknown tier 和兜底阈值（默认 1000）。
func MatchItemTierAndThreshold(ocrName string, cfg ThresholdConfig) ItemMatchResult {
	fallback := resolveFallbackThreshold(cfg.FallbackThreshold)
	result := ItemMatchResult{
		OCRName:      ocrName,
		TierID:       unknownTierID,
		EditDistance: -1,
		Threshold:    fallback,
		Matched:      false,
	}

	items, err := loadCanonicalItems()
	if err != nil {
		log.Warn().
			Err(err).
			Str("component", "autostockpile").
			Str("ocr_name", ocrName).
			Int("fallback_threshold", fallback).
			Msg("canonical item map unavailable")
		return result
	}

	normOCR := normalizeItemName(ocrName)
	rawOCR := cleanHanText(ocrName)
	if normOCR == "" {
		log.Warn().
			Str("component", "autostockpile").
			Str("ocr_name", ocrName).
			Int("fallback_threshold", fallback).
			Msg("ocr name empty after normalization")
		return result
	}

	bestIdx := -1
	bestDistance := maxItemNameEditDistance + 1
	for i, item := range items {
		distance := levenshteinDistanceWithCutoff(normOCR, item.Norm, maxItemNameEditDistance)
		if distance < bestDistance {
			bestDistance = distance
			bestIdx = i
		}
	}

	if bestIdx < 0 || bestDistance > maxItemNameEditDistance {
		log.Info().
			Str("component", "autostockpile").
			Str("ocr_name", ocrName).
			Str("normalized_ocr_name", normOCR).
			Int("max_distance", maxItemNameEditDistance).
			Int("fallback_threshold", fallback).
			Msg("item fuzzy match missed")
		return result
	}

	matched := items[bestIdx]
	if bestDistance == maxItemNameEditDistance && !hasKnownNoiseEvidence(rawOCR, matched.Raw) {
		log.Info().
			Str("component", "autostockpile").
			Str("ocr_name", ocrName).
			Str("normalized_ocr_name", normOCR).
			Str("best_canonical_name", matched.Name).
			Int("best_edit_distance", bestDistance).
			Int("fallback_threshold", fallback).
			Msg("item fuzzy match rejected")
		return result
	}

	threshold := resolveTierThreshold(matched.Tier, cfg)

	result.CanonicalName = matched.Name
	result.TierID = matched.Tier
	result.EditDistance = bestDistance
	result.Threshold = threshold
	result.Matched = true

	log.Info().
		Str("component", "autostockpile").
		Str("ocr_name", ocrName).
		Str("normalized_ocr_name", normOCR).
		Str("canonical_name", matched.Name).
		Str("tier_id", matched.Tier).
		Int("edit_distance", bestDistance).
		Int("threshold", threshold).
		Msg("item fuzzy match hit")

	return result
}

func loadCanonicalItems() ([]canonicalItem, error) {
	canonicalItemsOnce.Do(func() {
		data, err := LoadItemValueChangeMap()
		if err != nil {
			canonicalItemsErr = err
			return
		}

		items := make([]canonicalItem, 0, len(data.ZhCn))
		for name, tier := range data.ZhCn {
			raw := cleanHanText(name)
			items = append(items, canonicalItem{
				Name: name,
				Tier: tier,
				Raw:  raw,
				Norm: normalizeItemName(name),
			})
		}
		canonicalItems = items
	})

	if canonicalItemsErr != nil {
		return nil, canonicalItemsErr
	}
	return canonicalItems, nil
}

func resolveFallbackThreshold(raw int) int {
	if raw > 0 {
		return raw
	}
	return defaultFallbackBuyThreshold
}

func resolveTierThreshold(tierID string, cfg ThresholdConfig) int {
	switch tierID {
	case "ValleyIVTier1":
		if cfg.PriceLimits.ValleyIVTier1 > 0 {
			return cfg.PriceLimits.ValleyIVTier1
		}
	case "ValleyIVTier2":
		if cfg.PriceLimits.ValleyIVTier2 > 0 {
			return cfg.PriceLimits.ValleyIVTier2
		}
	case "ValleyIVTier3":
		if cfg.PriceLimits.ValleyIVTier3 > 0 {
			return cfg.PriceLimits.ValleyIVTier3
		}
	case "WulingTier1":
		if cfg.PriceLimits.WulingTier1 > 0 {
			return cfg.PriceLimits.WulingTier1
		}
	}

	return resolveFallbackThreshold(cfg.FallbackThreshold)
}

func normalizeItemName(text string) string {
	return noiseNormalizer.Replace(cleanHanText(text))
}

func cleanHanText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}

	builder := strings.Builder{}
	builder.Grow(len(trimmed))
	for _, r := range trimmed {
		if unicode.Is(unicode.Han, r) {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

func hasKnownNoiseEvidence(ocrRaw, canonicalRaw string) bool {
	for _, pair := range knownNoisePairs {
		left, right := pair[0], pair[1]
		if strings.Contains(ocrRaw, left) && strings.Contains(canonicalRaw, right) {
			return true
		}
		if strings.Contains(ocrRaw, right) && strings.Contains(canonicalRaw, left) {
			return true
		}
	}
	return false
}

func levenshteinDistanceWithCutoff(a, b string, cutoff int) int {
	ra := []rune(a)
	rb := []rune(b)
	la := len(ra)
	lb := len(rb)

	if absInt(la-lb) > cutoff {
		return cutoff + 1
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		rowMin := curr[0]

		for j := 1; j <= lb; j++ {
			cost := 0
			if ra[i-1] != rb[j-1] {
				cost = 1
			}

			deletion := prev[j] + 1
			insertion := curr[j-1] + 1
			substitution := prev[j-1] + cost

			curr[j] = minInt(deletion, insertion, substitution)
			if curr[j] < rowMin {
				rowMin = curr[j]
			}
		}

		if rowMin > cutoff {
			return cutoff + 1
		}

		prev, curr = curr, prev
	}

	if prev[lb] > cutoff {
		return cutoff + 1
	}
	return prev[lb]
}

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
