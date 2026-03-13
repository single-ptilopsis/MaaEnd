package autostockpile

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/MaaXYZ/MaaEnd/agent/go-service/pkg/maafocus"
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

const pricePairRowTolerance = 10

type scoredProduct struct {
	nameToken    OCRToken
	productName  string
	canonical    string
	threshold    int
	price        int
	score        int
	clickX       int
	clickY       int
	matchSuccess bool
}

// Run 执行 AutoStockpile 单商品选择逻辑。
func (a *SelectItemAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	if arg == nil {
		log.Error().
			Str("component", "autostockpile").
			Msg("custom action arg is nil")
		return false
	}

	cfg, err := getSelectionConfigFromNode(ctx, arg.CurrentTaskName)
	if err != nil {
		log.Error().
			Err(err).
			Str("component", "autostockpile").
			Str("step", "load_selection_config").
			Msg("invalid selection config")
		return false
	}

	parsed, err := ParseOCRCandidates(ctx, arg.RecognitionDetail, "")
	if err != nil {
		log.Error().
			Err(err).
			Str("component", "autostockpile").
			Str("step", "parse_ocr_candidates").
			Msg("failed to parse ocr candidates")
		return false
	}

	result := SelectBestProduct(parsed, cfg)
	if !result.Selected {
		log.Info().
			Str("component", "autostockpile").
			Str("reason", result.Reason).
			Msg("no qualifying product selected")
		maafocus.NodeActionStarting(ctx, fmt.Sprintf("未找到符合条件的物资 (原因: %s)", result.Reason))
		return true
	}

	_ = ctx.OverridePipeline(map[string]any{
		"AutoStockpileSelectedProductClick": map[string]any{
			"target": []int{result.ClickX, result.ClickY, 1, 1},
		},
	})

	log.Info().
		Str("component", "autostockpile").
		Str("product_name", result.ProductName).
		Str("canonical_name", result.CanonicalName).
		Int("threshold", result.Threshold).
		Int("price", result.CurrentPrice).
		Int("score", result.Score).
		Int("click_x", result.ClickX).
		Int("click_y", result.ClickY).
		Msg("product selected")
	maafocus.NodeActionStarting(ctx, fmt.Sprintf("已选择物资: %s (价格 %d < 阈值 %d, 利润 %d)", result.ProductName, result.CurrentPrice, result.Threshold, result.Score))

	return true
}

// SelectBestProduct 根据 OCR 解析结果与模式配置返回唯一候选商品。
func SelectBestProduct(parsed *OCRParseResult, cfg SelectionConfig) SelectionResult {
	return selectBestProductAt(parsed, cfg, time.Now())
}

func selectBestProductAt(parsed *OCRParseResult, cfg SelectionConfig, now time.Time) SelectionResult {
	if parsed == nil || len(parsed.CandidateRegions) == 0 {
		return SelectionResult{Selected: false, Reason: "no_qualifying_products"}
	}

	overflowDetected := len(parsed.OverflowMarkerTokens) > 0
	sundayMode := isSundayMode(cfg.SundayMode, now)
	allowAll := sundayMode || (cfg.OverflowMode && overflowDetected)

	candidates := make([]scoredProduct, 0, len(parsed.CandidateRegions))
	thresholdCfg := ThresholdConfig{FallbackThreshold: cfg.FallbackThreshold, PriceLimits: cfg.PriceLimits}

	for _, region := range parsed.CandidateRegions {
		nameToken, ok := findProductNameToken(region.Tokens)
		if !ok {
			continue
		}

		priceToken, price, ok := findAdjacentPriceToken(region.Tokens, nameToken)
		if !ok {
			log.Warn().
				Str("component", "autostockpile").
				Str("product_name", nameToken.Text).
				Int("name_x", nameToken.X).
				Int("name_y", nameToken.Y).
				Msg("adjacent price token not found")
			continue
		}

		match := MatchItemTierAndThreshold(nameToken.Text, thresholdCfg)
		score := match.Threshold - price
		if !allowAll && score <= 0 {
			continue
		}

		candidates = append(candidates, scoredProduct{
			nameToken:    nameToken,
			productName:  nameToken.Text,
			canonical:    match.CanonicalName,
			threshold:    match.Threshold,
			price:        price,
			score:        score,
			clickX:       nameToken.X + nameToken.W/2,
			clickY:       nameToken.Y + nameToken.H/2,
			matchSuccess: match.Matched,
		})

		_ = priceToken
	}

	if len(candidates) == 0 {
		log.Info().
			Str("component", "autostockpile").
			Bool("overflow_mode", cfg.OverflowMode).
			Bool("overflow_detected", overflowDetected).
			Bool("sunday_mode", sundayMode).
			Msg("no qualifying products after mode gate")
		return SelectionResult{Selected: false, Reason: "no_qualifying_products"}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].nameToken.Y != candidates[j].nameToken.Y {
			return candidates[i].nameToken.Y < candidates[j].nameToken.Y
		}
		return candidates[i].nameToken.X < candidates[j].nameToken.X
	})

	best := candidates[0]
	return SelectionResult{
		Selected:      true,
		ProductName:   best.productName,
		CanonicalName: best.canonical,
		Threshold:     best.threshold,
		CurrentPrice:  best.price,
		Score:         best.score,
		ClickX:        best.clickX,
		ClickY:        best.clickY,
	}
}

func isSundayMode(enabled bool, now time.Time) bool {
	if !enabled {
		return false
	}
	return now.Weekday() == time.Sunday
}

func findProductNameToken(tokens []OCRToken) (OCRToken, bool) {
	if len(tokens) == 0 {
		return OCRToken{}, false
	}

	sorted := append([]OCRToken(nil), tokens...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].X == sorted[j].X {
			return sorted[i].Y < sorted[j].Y
		}
		return sorted[i].X < sorted[j].X
	})

	for _, token := range sorted {
		if isLikelyProductNameToken(token.Text) {
			return token, true
		}
	}

	return OCRToken{}, false
}

func findAdjacentPriceToken(tokens []OCRToken, nameToken OCRToken) (OCRToken, int, bool) {
	bestIdx := -1
	bestPrice := 0
	bestDistance := 0
	bestRightBias := 0

	for idx, token := range tokens {
		if token.Text == nameToken.Text && token.X == nameToken.X && token.Y == nameToken.Y && token.W == nameToken.W && token.H == nameToken.H {
			continue
		}
		if absInt(token.CenterY()-nameToken.CenterY()) > pricePairRowTolerance {
			continue
		}
		if isLikelyRateToken(token.Text) {
			continue
		}

		price, ok := extractInt(token.Text)
		if !ok {
			continue
		}

		dx := token.CenterX() - nameToken.CenterX()
		rightBias := 1
		if dx >= 0 {
			rightBias = 0
		}
		distance := absInt(dx)

		if bestIdx < 0 || rightBias < bestRightBias || (rightBias == bestRightBias && distance < bestDistance) {
			bestIdx = idx
			bestPrice = price
			bestDistance = distance
			bestRightBias = rightBias
		}
	}

	if bestIdx < 0 {
		return OCRToken{}, 0, false
	}
	return tokens[bestIdx], bestPrice, true
}

func isLikelyProductNameToken(text string) bool {
	norm := strings.TrimSpace(text)
	if norm == "" {
		return false
	}
	if strings.Contains(norm, "货组") {
		return true
	}
	if isLikelyRateToken(norm) {
		return false
	}
	if _, ok := extractInt(norm); ok {
		return false
	}
	for _, r := range norm {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func isLikelyRateToken(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	if strings.ContainsAny(trimmed, ".%▲xX") {
		return true
	}
	return false
}

func extractInt(text string) (int, bool) {
	digits := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, text)

	digits = strings.TrimSpace(digits)
	if digits == "" {
		return 0, false
	}

	num, err := strconv.Atoi(digits)
	if err != nil {
		return 0, false
	}
	return num, true
}
