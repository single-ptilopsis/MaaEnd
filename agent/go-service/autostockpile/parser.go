package autostockpile

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

const (
	rowMergeTolerance = 70
	colMergeTolerance = 120
)

// ParseOCRCandidates 解析 AutoStockpile 的 OCR token，并输出已过滤的商品候选区域。
func ParseOCRCandidates(ctx *maa.Context, detail *maa.RecognitionDetail, rerunRecognitionName string) (*OCRParseResult, error) {
	allTokens := extractOCRTokens(detail)

	if len(allTokens) == 0 && ctx != nil && detail != nil && detail.Raw != nil && rerunRecognitionName != "" {
		rerunDetail, err := ctx.RunRecognition(rerunRecognitionName, detail.Raw, nil)
		if err != nil {
			log.Error().
				Err(err).
				Str("component", "autostockpile").
				Str("step", "rerun_ocr").
				Str("recognition", rerunRecognitionName).
				Msg("rerun recognition failed")
		} else {
			allTokens = extractOCRTokens(rerunDetail)
		}
	}

	dividerY := detectEffectiveDividerY(allTokens)
	overflowTokens := make([]OCRToken, 0)
	productTokens := make([]OCRToken, 0)

	for _, token := range allTokens {
		if isOverflowMarker(token.Text) {
			overflowTokens = append(overflowTokens, token)
			continue
		}

		if isMarketMarker(token.Text) || isOwnedMarker(token.Text) {
			continue
		}

		if token.CenterY() <= dividerY {
			continue
		}

		if !looksLikeProductToken(token.Text) {
			continue
		}

		productTokens = append(productTokens, token)
	}

	candidates := groupProductTokenRegions(productTokens)

	log.Info().
		Str("component", "autostockpile").
		Int("all_tokens", len(allTokens)).
		Int("product_tokens", len(productTokens)).
		Int("candidate_regions", len(candidates)).
		Int("overflow_markers", len(overflowTokens)).
		Int("divider_y", dividerY).
		Msg("ocr tokens parsed")

	return &OCRParseResult{
		AllTokens:               allTokens,
		ProductTokens:           productTokens,
		CandidateRegions:        candidates,
		OverflowMarkerTokens:    overflowTokens,
		EffectiveProductDivider: dividerY,
	}, nil
}

func extractOCRTokens(detail *maa.RecognitionDetail) []OCRToken {
	if detail == nil {
		return nil
	}

	if detail.Results != nil && len(detail.Results.All) > 0 {
		tokens := make([]OCRToken, 0, len(detail.Results.All))
		for _, result := range detail.Results.All {
			if result == nil {
				continue
			}
			ocrResult, ok := result.AsOCR()
			if !ok || ocrResult == nil || strings.TrimSpace(ocrResult.Text) == "" {
				continue
			}
			tokens = append(tokens, OCRToken{
				Text:  ocrResult.Text,
				X:     ocrResult.Box[0],
				Y:     ocrResult.Box[1],
				W:     ocrResult.Box[2],
				H:     ocrResult.Box[3],
				Score: ocrResult.Score,
			})
		}
		if len(tokens) > 0 {
			return tokens
		}
	}

	if detail.DetailJson != "" {
		return parseOCRTokensFromDetailJSON(detail.DetailJson)
	}

	return nil
}

func parseOCRTokensFromDetailJSON(detailJSON string) []OCRToken {
	var payload any
	if err := json.Unmarshal([]byte(detailJSON), &payload); err != nil {
		return nil
	}

	tokens := make([]OCRToken, 0)
	collectOCRTokens(payload, &tokens)
	return tokens
}

func collectOCRTokens(payload any, tokens *[]OCRToken) {
	switch v := payload.(type) {
	case map[string]any:
		if token, ok := mapToToken(v); ok {
			*tokens = append(*tokens, token)
		}

		for _, key := range []string{"all", "detail", "best", "filtered"} {
			if nested, ok := v[key]; ok {
				collectOCRTokens(nested, tokens)
			}
		}
	case []any:
		for _, item := range v {
			collectOCRTokens(item, tokens)
		}
	}
}

func mapToToken(v map[string]any) (OCRToken, bool) {
	text, ok := v["text"].(string)
	if !ok || strings.TrimSpace(text) == "" {
		return OCRToken{}, false
	}

	boxRaw, ok := v["box"].([]any)
	if !ok || len(boxRaw) < 4 {
		return OCRToken{}, false
	}

	box := [4]int{}
	for i := 0; i < 4; i++ {
		f, ok := boxRaw[i].(float64)
		if !ok {
			return OCRToken{}, false
		}
		box[i] = int(f)
	}

	score := 0.0
	if rawScore, ok := v["score"].(float64); ok {
		score = rawScore
	}

	return OCRToken{
		Text:  text,
		X:     box[0],
		Y:     box[1],
		W:     box[2],
		H:     box[3],
		Score: score,
	}, true
}

func detectEffectiveDividerY(tokens []OCRToken) int {
	dividerY := 0
	for _, token := range tokens {
		if isMarketMarker(token.Text) && token.CenterY() > dividerY {
			dividerY = token.CenterY()
		}
	}
	return dividerY
}

func normalizeMarkerText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}

	builder := strings.Builder{}
	builder.Grow(len(trimmed))
	for _, r := range trimmed {
		switch {
		case unicode.IsSpace(r):
			continue
		case r == '佣':
			builder.WriteRune('拥')
		case r == '物':
			builder.WriteRune('溢')
		case unicode.IsPunct(r):
			continue
		default:
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

func isMarketMarker(text string) bool {
	norm := normalizeMarkerText(text)
	return strings.Contains(norm, "市场")
}

func isOwnedMarker(text string) bool {
	norm := normalizeMarkerText(text)
	if strings.Contains(norm, "当前拥有") {
		return true
	}
	if strings.Contains(norm, "拥有") && hasDigit(norm) {
		return true
	}
	return false
}

func isOverflowMarker(text string) bool {
	norm := normalizeMarkerText(text)
	return strings.Contains(norm, "溢出") || strings.Contains(norm, "即溢出") || strings.Contains(norm, "即出")
}

func hasDigit(text string) bool {
	for _, r := range text {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func looksLikeProductToken(text string) bool {
	norm := normalizeMarkerText(text)
	if norm == "" {
		return false
	}
	if len([]rune(norm)) <= 1 && !hasDigit(norm) {
		return false
	}
	if strings.Contains(norm, "货组") {
		return true
	}
	if strings.ContainsAny(norm, "%xX.") {
		return true
	}
	if strings.Contains(norm, "▲") {
		return true
	}
	return hasDigit(norm)
}

func groupProductTokenRegions(tokens []OCRToken) []OCRCandidateRegion {
	if len(tokens) == 0 {
		return nil
	}

	sorted := make([]OCRToken, len(tokens))
	copy(sorted, tokens)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].CenterY() == sorted[j].CenterY() {
			return sorted[i].CenterX() < sorted[j].CenterX()
		}
		return sorted[i].CenterY() < sorted[j].CenterY()
	})

	type rowCluster struct {
		avgY   int
		tokens []OCRToken
	}
	rows := make([]*rowCluster, 0)

	for _, token := range sorted {
		assigned := false
		for _, row := range rows {
			if absInt(row.avgY-token.CenterY()) <= rowMergeTolerance {
				row.tokens = append(row.tokens, token)
				row.avgY = averageCenterY(row.tokens)
				assigned = true
				break
			}
		}
		if !assigned {
			rows = append(rows, &rowCluster{avgY: token.CenterY(), tokens: []OCRToken{token}})
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].avgY < rows[j].avgY
	})

	regions := make([]OCRCandidateRegion, 0)
	for rowIndex, row := range rows {
		for _, region := range groupRowColumns(row.tokens, rowIndex) {
			regions = append(regions, region)
		}
	}

	return regions
}

func groupRowColumns(tokens []OCRToken, rowIndex int) []OCRCandidateRegion {
	type colCluster struct {
		avgX   int
		tokens []OCRToken
	}

	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].CenterX() < tokens[j].CenterX()
	})

	cols := make([]*colCluster, 0)
	for _, token := range tokens {
		assigned := false
		for _, col := range cols {
			if absInt(col.avgX-token.CenterX()) <= colMergeTolerance {
				col.tokens = append(col.tokens, token)
				col.avgX = averageCenterX(col.tokens)
				assigned = true
				break
			}
		}
		if !assigned {
			cols = append(cols, &colCluster{avgX: token.CenterX(), tokens: []OCRToken{token}})
		}
	}

	sort.Slice(cols, func(i, j int) bool {
		return cols[i].avgX < cols[j].avgX
	})

	regions := make([]OCRCandidateRegion, 0, len(cols))
	for colIndex, col := range cols {
		bounds := calcBounds(col.tokens)
		regions = append(regions, OCRCandidateRegion{
			Row:    rowIndex,
			Column: colIndex,
			Bounds: bounds,
			Tokens: col.tokens,
		})
	}

	return regions
}

func averageCenterY(tokens []OCRToken) int {
	if len(tokens) == 0 {
		return 0
	}
	sum := 0
	for _, token := range tokens {
		sum += token.CenterY()
	}
	return sum / len(tokens)
}

func averageCenterX(tokens []OCRToken) int {
	if len(tokens) == 0 {
		return 0
	}
	sum := 0
	for _, token := range tokens {
		sum += token.CenterX()
	}
	return sum / len(tokens)
}

func calcBounds(tokens []OCRToken) [4]int {
	if len(tokens) == 0 {
		return [4]int{}
	}

	minX, minY := tokens[0].X, tokens[0].Y
	maxX, maxY := tokens[0].X+tokens[0].W, tokens[0].Y+tokens[0].H

	for _, token := range tokens[1:] {
		if token.X < minX {
			minX = token.X
		}
		if token.Y < minY {
			minY = token.Y
		}
		if token.X+token.W > maxX {
			maxX = token.X + token.W
		}
		if token.Y+token.H > maxY {
			maxY = token.Y + token.H
		}
	}

	return [4]int{minX, minY, maxX - minX, maxY - minY}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
