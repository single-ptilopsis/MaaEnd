package stockpile

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/MaaXYZ/MaaEnd/agent/go-service/pkg/maafocus"
	"github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

// StockpileDecideAction analyses all scanned product records and decides
// which product to purchase based on the diff-maximisation strategy,
// overflow handling, and Sunday buy-all logic.
var _ maa.CustomActionRunner = &StockpileDecideAction{}

// StockpileDecideAction picks the best purchase candidate from the scan results.
type StockpileDecideAction struct{}

func (a *StockpileDecideAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	records := getRecords()
	overflow := getOverflowAmount()
	buyOverflow := getOverflowBuy()
	buyAll := getSundayBuyAll() && IsSunday()

	if len(records) == 0 {
		log.Info().
			Str("component", "Stockpile").
			Msg("no products scanned, switching region")
		maafocus.NodeActionStarting(ctx, "⚠️ 无可购买商品，切换地区")
		ctx.OverrideNext(arg.CurrentTaskName, []maa.NextItem{{Name: "StockpileChangeRegionPrepare"}})
		return true
	}

	// Sort records by diff descending (largest bargain first).
	sorted := make([]ProductRecord, len(records))
	copy(sorted, records)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Diff > sorted[j].Diff
	})

	best := sorted[0]

	// --- Sunday buy-all mode ---
	if buyAll {
		log.Info().
			Str("component", "Stockpile").
			Int("row", best.Row).
			Int("col", best.Col).
			Int("price", best.Price).
			Msg("sunday mode: buying best bargain")
		maafocus.NodeActionStarting(ctx,
			fmt.Sprintf("🗓️ 周日模式：购买第%d行第%d列 (价格: %d)", displayRow(best.Row), best.Col, best.Price))
		overrideNextToBuy(ctx, arg.CurrentTaskName, best.Row, best.Col)
		return true
	}

	// --- Normal mode: buy only when diff > 0 ---
	if best.Diff > 0 {
		log.Info().
			Str("component", "Stockpile").
			Int("row", best.Row).
			Int("col", best.Col).
			Int("diff", best.Diff).
			Msg("best bargain found, purchasing")
		maafocus.NodeActionStarting(ctx,
			fmt.Sprintf("💰 购买第%d行第%d列 (差值: %d, 价格: %d)",
				displayRow(best.Row), best.Col, best.Diff, best.Price))
		overrideNextToBuy(ctx, arg.CurrentTaskName, best.Row, best.Col)
		return true
	}

	// --- Overflow handling ---
	if overflow > 0 && buyOverflow {
		// Buy the cheapest product to consume quota before overflow.
		cheapest := sorted[len(sorted)-1]
		for _, r := range sorted {
			if r.Price < cheapest.Price {
				cheapest = r
			}
		}
		log.Info().
			Str("component", "Stockpile").
			Int("overflow", overflow).
			Int("row", cheapest.Row).
			Int("col", cheapest.Col).
			Int("price", cheapest.Price).
			Msg("overflow detected, buying cheapest")
		maafocus.NodeActionStarting(ctx,
			fmt.Sprintf("⚠️ 配额溢出 %d 件，购买最低价商品：第%d行第%d列 (价格: %d)",
				overflow, displayRow(cheapest.Row), cheapest.Col, cheapest.Price))
		overrideNextToBuy(ctx, arg.CurrentTaskName, cheapest.Row, cheapest.Col)
		return true
	}

	// --- No purchase ---
	msg := fmt.Sprintf("💡 无满足阈值的商品\n最佳: 第%d行第%d列 (价格: %d, 差值: %d)",
		displayRow(best.Row), best.Col, best.Price, best.Diff)
	if overflow > 0 {
		msg = fmt.Sprintf("⚠️ 配额溢出 %d 件，但未启用溢出购买\n最佳: 第%d行第%d列 (价格: %d, 差值: %d)",
			overflow, displayRow(best.Row), best.Col, best.Price, best.Diff)
	}
	maafocus.NodeActionStarting(ctx, msg)
	log.Info().
		Str("component", "Stockpile").
		Int("bestDiff", best.Diff).
		Msg("no product meets threshold, switching region")
	ctx.OverrideNext(arg.CurrentTaskName, []maa.NextItem{{Name: "StockpileChangeRegionPrepare"}})
	return true
}

// StockpileFinishAction marks the end of the stockpile flow.
var _ maa.CustomActionRunner = &StockpileFinishAction{}

// StockpileFinishAction is executed when all regions have been processed.
type StockpileFinishAction struct{}

func (a *StockpileFinishAction) Run(_ *maa.Context, _ *maa.CustomActionArg) bool {
	log.Info().
		Str("component", "Stockpile").
		Msg("stockpile task finished")
	return true
}

// --- quota recognition ---

// quotaRecoResult is the JSON payload exchanged between the custom
// recognition and the custom action for quota data.
type quotaRecoResult struct {
	X int `json:"x"` // current quota used
	Y int `json:"y"` // max quota
	B int `json:"b"` // next increment amount
}

// StockpileCheckQuotaRecognition performs OCR on the quota region of the
// screenshot and returns the parsed values via Detail.
var _ maa.CustomRecognitionRunner = &StockpileCheckQuotaRecognition{}

// StockpileCheckQuotaRecognition reads quota info from the game screen.
type StockpileCheckQuotaRecognition struct{}

func (r *StockpileCheckQuotaRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	log.Info().
		Str("component", "Stockpile").
		Msg("checking quota status")

	if arg.Img == nil {
		log.Error().
			Str("component", "Stockpile").
			Msg("screenshot is nil")
		return &maa.CustomRecognitionResult{
			Box:    arg.Roi,
			Detail: `{"x":-1,"y":-1,"b":-1}`,
		}, true
	}

	currentQuota, maxQuota, _, nextIncrement := ocrAndParseQuota(ctx, arg.Img)
	if currentQuota < 0 || maxQuota <= 0 || nextIncrement < 0 {
		log.Info().
			Str("component", "Stockpile").
			Msg("could not parse quota, continuing normally")
	}
	result := quotaRecoResult{X: currentQuota, Y: maxQuota, B: nextIncrement}
	detailJSON, _ := json.Marshal(result)
	return &maa.CustomRecognitionResult{
		Box:    arg.Roi,
		Detail: string(detailJSON),
	}, true
}

// StockpileCheckQuotaAction processes the quota recognition result
// and calculates the overflow amount.
var _ maa.CustomActionRunner = &StockpileCheckQuotaAction{}

// StockpileCheckQuotaAction parses quota results and resets scan position.
type StockpileCheckQuotaAction struct{}

func (a *StockpileCheckQuotaAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	overflowAmt := 0
	detailJSON := extractRecoDetailJson(arg.RecognitionDetail)
	if detailJSON != "" {
		var reco quotaRecoResult
		if err := json.Unmarshal([]byte(detailJSON), &reco); err != nil {
			log.Warn().
				Err(err).
				Str("component", "Stockpile").
				Msg("failed to parse quota result")
		} else if reco.X >= 0 && reco.Y > 0 && reco.B >= 0 {
			overflowAmt = reco.X + reco.B - reco.Y
			log.Info().
				Str("component", "Stockpile").
				Int("overflow", overflowAmt).
				Int("current", reco.X).
				Int("max", reco.Y).
				Int("nextAdd", reco.B).
				Msg("quota overflow calculated")
		}
	}

	setOverflowAmount(overflowAmt)
	clearRecords()

	// Reset scan position for the new region.
	_ = ctx.OverridePipeline(map[string]any{
		"StockpileScan": map[string]any{
			"custom_action_param": map[string]any{
				"row": 1,
				"col": 1,
			},
		},
	})
	return true
}

// --- helpers ---

// extractRecoDetailJson retrieves the detail JSON from a RecognitionDetail,
// handling the wrapped best.detail format from custom recognitions.
func extractRecoDetailJson(rd *maa.RecognitionDetail) string {
	if rd == nil || rd.DetailJson == "" {
		return ""
	}
	var wrapped struct {
		Best struct {
			Detail json.RawMessage `json:"detail"`
		} `json:"best"`
	}
	if err := json.Unmarshal([]byte(rd.DetailJson), &wrapped); err == nil && len(wrapped.Best.Detail) > 0 {
		return string(wrapped.Best.Detail)
	}
	return rd.DetailJson
}

// overrideNextToBuy sets the pipeline's next node to the product selection
// node for the given row and column.
func overrideNextToBuy(ctx *maa.Context, currentTask string, row, col int) {
	taskName := fmt.Sprintf("StockpileSelectProductRow%dCol%d", row, col)
	ctx.OverrideNext(currentTask, []maa.NextItem{{Name: taskName}})
}

// displayRow adjusts the internal row index for user-facing display.
// Row 2 maps to visual row 1, row 3 maps to visual row 2.
func displayRow(row int) int {
	if row >= 2 {
		return row - 1
	}
	return row
}
