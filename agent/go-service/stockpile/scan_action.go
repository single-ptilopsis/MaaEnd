package stockpile

import (
	"encoding/json"
	"fmt"
	"image"
	"strconv"
	"strings"
	"time"

	"github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

// StockpileScanAction sets up the Pipeline to scan the product at the
// current grid position, identified by row/col in custom_action_param.
var _ maa.CustomActionRunner = &StockpileScanAction{}

// StockpileScanAction is the entry point for each grid-cell scan cycle.
type StockpileScanAction struct{}

func (a *StockpileScanAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	rowIdx, col := 1, 1
	if arg.CustomActionParam != "" {
		var params struct {
			Row int `json:"row"`
			Col int `json:"col"`
		}
		if err := json.Unmarshal([]byte(arg.CustomActionParam), &params); err != nil {
			log.Error().
				Err(err).
				Str("component", "Stockpile").
				Str("param", arg.CustomActionParam).
				Msg("failed to parse scan param")
			return false
		}
		if params.Row >= 1 && params.Row <= 3 && params.Col >= 1 && params.Col <= 8 {
			rowIdx, col = params.Row, params.Col
		}
	}
	setScanPos(rowIdx, col)

	// Override the OCR node to target the current grid position.
	// The ROI nodes follow the Resell naming convention shared across
	// the same elastic-demand goods page.
	pricePipelineName := fmt.Sprintf("StockpileROIProductRow%dCol%dPrice", rowIdx, col)
	_ = ctx.OverridePipeline(map[string]any{
		"StockpileScanPrice": map[string]any{
			"recognition": "Or",
			"any_of":      []string{pricePipelineName},
		},
	})

	// Move mouse away to avoid blocking OCR recognition.
	if controller := ctx.GetTasker().GetController(); controller != nil {
		moveMouseSafe(controller)
	}
	return true
}

// StockpileScanPriceAction extracts the OCR-recognised price and stores a
// ProductRecord with the corresponding threshold diff.
var _ maa.CustomActionRunner = &StockpileScanPriceAction{}

// StockpileScanPriceAction is executed after OCR recognises a product price.
type StockpileScanPriceAction struct{}

func (a *StockpileScanPriceAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	row, col := getScanPos()
	region := getCurrentRegion()
	t := getThresholds()

	text := extractOCRText(arg.RecognitionDetail)
	if text == "" {
		log.Info().
			Str("component", "Stockpile").
			Int("row", row).
			Int("col", col).
			Msg("no OCR text, skipping")
		scanOverrideNext(ctx, arg.CurrentTaskName, row, col, false)
		return true
	}

	price, ok := extractNumbersFromText(text)
	if !ok {
		log.Info().
			Str("component", "Stockpile").
			Str("text", text).
			Int("row", row).
			Int("col", col).
			Msg("no valid number in OCR text, skipping")
		scanOverrideNext(ctx, arg.CurrentTaskName, row, col, false)
		return true
	}

	fluctuation := GetProductFluctuation(region, row, col)
	threshold := GetThreshold(t, region, fluctuation)
	diff := threshold - price

	record := ProductRecord{
		Row:       row,
		Col:       col,
		Price:     price,
		Threshold: threshold,
		Diff:      diff,
	}
	appendRecord(record)

	log.Info().
		Str("component", "Stockpile").
		Int("row", row).
		Int("col", col).
		Int("price", price).
		Int("threshold", threshold).
		Int("diff", diff).
		Msg("product scanned")

	return true
}

// StockpileScanSkipEmptyAction handles the case where no product
// is found at the current grid position.
var _ maa.CustomActionRunner = &StockpileScanSkipEmptyAction{}

// StockpileScanSkipEmptyAction skips the current position and moves to the next.
type StockpileScanSkipEmptyAction struct{}

func (a *StockpileScanSkipEmptyAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	row, col := getScanPos()
	log.Info().
		Str("component", "Stockpile").
		Int("row", row).
		Int("col", col).
		Msg("no product at position, skipping")
	scanOverrideNext(ctx, arg.CurrentTaskName, row, col, true)
	return true
}

// StockpileScanNextAction advances to the next grid position or enters
// the decision stage when all positions have been scanned.
var _ maa.CustomActionRunner = &StockpileScanNextAction{}

// StockpileScanNextAction moves scanning to the next grid cell.
type StockpileScanNextAction struct{}

func (a *StockpileScanNextAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	row, col := getScanPos()
	scanOverrideNext(ctx, arg.CurrentTaskName, row, col, false)
	return true
}

// scanOverrideNext computes the next scan position and overrides the
// Pipeline's next node accordingly.
func scanOverrideNext(ctx *maa.Context, currentTask string, row, col int, breakRow bool) {
	nextRow, nextCol, done := computeNextScanPos(row, col, breakRow)
	if done {
		ctx.OverrideNext(currentTask, []maa.NextItem{{Name: "StockpileDecide"}})
		return
	}
	_ = ctx.OverridePipeline(map[string]any{
		"StockpileScan": map[string]any{
			"custom_action_param": map[string]any{
				"row": nextRow,
				"col": nextCol,
			},
		},
	})
	ctx.OverrideNext(currentTask, []maa.NextItem{{Name: "StockpileScan"}})
}

// computeNextScanPos calculates the next row/col position in the 3×8 grid.
func computeNextScanPos(row, col int, breakRow bool) (nextRow, nextCol int, done bool) {
	if breakRow {
		if row < 3 {
			return row + 1, 1, false
		}
		return 0, 0, true
	}
	if col < 8 {
		return row, col + 1, false
	}
	if row < 3 {
		return row + 1, 1, false
	}
	return 0, 0, true
}

// --- Utility helpers ---

// extractNumbersFromText picks all digit characters from a string and
// converts the result to an integer.
func extractNumbersFromText(text string) (int, bool) {
	var digitsOnly []byte
	for i := 0; i < len(text); i++ {
		if text[i] >= '0' && text[i] <= '9' {
			digitsOnly = append(digitsOnly, text[i])
		}
	}
	if len(digitsOnly) > 0 {
		if num, err := strconv.Atoi(string(digitsOnly)); err == nil {
			return num, true
		}
	}
	return 0, false
}

// moveMouseSafe moves the mouse cursor to a safe corner (10, 10) so
// it does not block OCR recognition on the product grid.
func moveMouseSafe(controller *maa.Controller) {
	controller.PostTouchMove(0, 10, 10, 0)
	// Brief sleep to ensure move completes before next OCR screenshot.
	time.Sleep(50 * time.Millisecond)
}

// extractOCRText retrieves the best OCR text from a RecognitionDetail,
// recursively traversing CombinedResult for Or/And nodes.
func extractOCRText(detail *maa.RecognitionDetail) string {
	if detail == nil {
		return ""
	}
	if detail.Results != nil {
		for _, results := range [][]*maa.RecognitionResult{
			{detail.Results.Best},
			detail.Results.Filtered,
			detail.Results.All,
		} {
			if len(results) > 0 && results[0] != nil {
				if ocrResult, ok := results[0].AsOCR(); ok && ocrResult.Text != "" {
					return ocrResult.Text
				}
			}
		}
	}
	if len(detail.CombinedResult) > 0 {
		for _, child := range detail.CombinedResult {
			if text := extractOCRText(child); text != "" {
				return text
			}
		}
	}
	if detail.DetailJson != "" {
		if text, _, _, _, _ := extractOCRFromDetailJson(detail.DetailJson); text != "" {
			return text
		}
	}
	return ""
}

// extractOCRFromDetailJson is a fallback that parses the raw DetailJson
// to extract the best OCR text and bounding box.
func extractOCRFromDetailJson(detailJson string) (text string, boxX, boxY, boxW, boxH int) {
	var orStruct struct {
		Detail []struct {
			Detail struct {
				Best struct {
					Text string `json:"text"`
					Box  []int  `json:"box"`
				} `json:"best"`
			} `json:"detail"`
		} `json:"detail"`
	}
	if err := json.Unmarshal([]byte(detailJson), &orStruct); err == nil && len(orStruct.Detail) > 0 {
		b := orStruct.Detail[0].Detail.Best
		if b.Text != "" && len(b.Box) >= 4 {
			return b.Text, b.Box[0], b.Box[1], b.Box[2], b.Box[3]
		}
	}
	var ocrStruct struct {
		Best struct {
			Text string `json:"text"`
			Box  []int  `json:"box"`
		} `json:"best"`
	}
	if err := json.Unmarshal([]byte(detailJson), &ocrStruct); err == nil && ocrStruct.Best.Text != "" && len(ocrStruct.Best.Box) >= 4 {
		b := ocrStruct.Best.Box
		return ocrStruct.Best.Text, b[0], b[1], b[2], b[3]
	}
	return "", 0, 0, 0, 0
}

// ocrAndParseQuota performs OCR on the quota region and parses
// current/max quota and the upcoming increment.
func ocrAndParseQuota(ctx *maa.Context, img image.Image) (x, y, hoursLater, b int) {
	x, y, hoursLater, b = -1, -1, -1, -1

	detail1, err := ctx.RunRecognition("StockpileROIQuotaCurrent", img, nil)
	if err != nil {
		log.Error().
			Err(err).
			Str("component", "Stockpile").
			Msg("failed to OCR quota current")
		return
	}
	if text := extractOCRText(detail1); text != "" {
		log.Info().
			Str("component", "Stockpile").
			Str("ocrText", text).
			Msg("quota current OCR result")
		parts := strings.Split(text, "/")
		if len(parts) >= 2 {
			if val, ok := extractNumbersFromText(parts[0]); ok {
				x = val
			}
			if val, ok := extractNumbersFromText(parts[1]); ok {
				y = val
			}
		}
	}

	// Try hours format: "a小时后+b"
	if detail2h, err2 := ctx.RunRecognition("StockpileROIQuotaNextAddHours", img, nil); err2 != nil {
		log.Error().
			Err(err2).
			Str("component", "Stockpile").
			Msg("failed to OCR quota next (hours)")
	} else if text := extractOCRText(detail2h); text != "" {
		parts := strings.Split(text, "+")
		if len(parts) >= 2 {
			if val, ok := extractNumbersFromText(parts[0]); ok {
				hoursLater = val
			}
			if val, ok := extractNumbersFromText(parts[1]); ok {
				b = val
			}
			return
		}
	}

	// Try minutes format: "a分钟后+b"
	if detail2m, err2 := ctx.RunRecognition("StockpileROIQuotaNextAddMinutes", img, nil); err2 != nil {
		log.Error().
			Err(err2).
			Str("component", "Stockpile").
			Msg("failed to OCR quota next (minutes)")
	} else if text := extractOCRText(detail2m); text != "" {
		parts := strings.Split(text, "+")
		if len(parts) >= 2 {
			if val, ok := extractNumbersFromText(parts[1]); ok {
				b = val
			}
			hoursLater = 0
			return
		}
	}

	// Fallback: "+b"
	if detail2f, err2 := ctx.RunRecognition("StockpileROIQuotaNextAddFallback", img, nil); err2 != nil {
		log.Error().
			Err(err2).
			Str("component", "Stockpile").
			Msg("failed to OCR quota next (fallback)")
	} else if text := extractOCRText(detail2f); text != "" {
		parts := strings.Split(text, "+")
		if len(parts) >= 2 {
			if val, ok := extractNumbersFromText(parts[len(parts)-1]); ok {
				b = val
			}
			hoursLater = 0
		}
	}

	return
}
