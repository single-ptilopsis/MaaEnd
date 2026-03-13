package autostockpile

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/MaaXYZ/MaaEnd/agent/go-service/pkg/maafocus"
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

type candidateGoods struct {
	goods     GoodsItem
	threshold int
	score     int
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

	detailJSON := extractRecoDetailJson(arg.RecognitionDetail)
	if detailJSON == "" {
		log.Error().
			Str("component", "autostockpile").
			Msg("recognition detail json is empty")
		return false
	}

	var result RecognitionResult
	if err := json.Unmarshal([]byte(detailJSON), &result); err != nil {
		log.Error().
			Err(err).
			Str("component", "autostockpile").
			Msg("failed to parse recognition result")
		return false
	}

	log.Info().
		Str("component", "autostockpile").
		Bool("overflow", result.Overflow).
		Bool("sunday", result.Sunday).
		Int("goods_count", len(result.Goods)).
		Msg("recognition result parsed")

	if result.Overflow && !cfg.OverflowMode {
		log.Info().
			Str("component", "autostockpile").
			Msg("overflow detected and overflow mode disabled")
		maafocus.NodeActionStarting(ctx, "库存将溢出，跳过购买")
		return true
	}

	allowAll := (result.Overflow && cfg.OverflowMode) || (result.Sunday && cfg.SundayMode)
	if allowAll {
		log.Info().
			Str("component", "autostockpile").
			Bool("overflow_allow", result.Overflow && cfg.OverflowMode).
			Bool("sunday_allow", result.Sunday && cfg.SundayMode).
			Msg("allow all goods mode enabled")
	}

	selection := SelectBestProduct(result, cfg, allowAll)
	if !selection.Selected {
		log.Info().
			Str("component", "autostockpile").
			Str("reason", selection.Reason).
			Msg("no qualifying product selected")
		maafocus.NodeActionStarting(ctx, fmt.Sprintf("未找到符合条件的物资 (原因: %s)", selection.Reason))
		return true
	}

	_ = ctx.OverridePipeline(map[string]any{
		"AutoStockpileSelectedProductClick": map[string]any{
			"enabled":  true,
			"template": []string{selection.ProductName},
		},
	})

	log.Info().
		Str("component", "autostockpile").
		Str("template", selection.ProductName).
		Str("tier", selection.CanonicalName).
		Int("threshold", selection.Threshold).
		Int("price", selection.CurrentPrice).
		Int("score", selection.Score).
		Msg("product selected and pipeline overridden")
	maafocus.NodeActionStarting(ctx, fmt.Sprintf("已选择物资: %s (价格 %d < 阈值 %d, 利润 %d)", selection.ProductName, selection.CurrentPrice, selection.Threshold, selection.Score))

	return true
}

func SelectBestProduct(result RecognitionResult, cfg SelectionConfig, allowAll bool) SelectionResult {
	if len(result.Goods) == 0 {
		return SelectionResult{Selected: false, Reason: "no_goods"}
	}

	candidates := make([]candidateGoods, 0, len(result.Goods))
	for _, goods := range result.Goods {
		threshold := resolveTierThreshold(goods.Tier, cfg)
		score := threshold - goods.Price

		log.Debug().
			Str("component", "autostockpile").
			Str("name", goods.Name).
			Str("tier", goods.Tier).
			Int("price", goods.Price).
			Int("threshold", threshold).
			Int("score", score).
			Bool("allow_all", allowAll).
			Msg("evaluating goods")

		if !allowAll && score <= 0 {
			continue
		}

		candidates = append(candidates, candidateGoods{
			goods:     goods,
			threshold: threshold,
			score:     score,
		})
	}

	if len(candidates) == 0 {
		return SelectionResult{Selected: false, Reason: "no_qualifying_products"}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].goods.Price != candidates[j].goods.Price {
			return candidates[i].goods.Price < candidates[j].goods.Price
		}
		if candidates[i].goods.Tier != candidates[j].goods.Tier {
			return candidates[i].goods.Tier < candidates[j].goods.Tier
		}
		return candidates[i].goods.Name < candidates[j].goods.Name
	})

	best := candidates[0]
	return SelectionResult{
		Selected:      true,
		ProductName:   best.goods.Name,
		CanonicalName: best.goods.Tier,
		Threshold:     best.threshold,
		CurrentPrice:  best.goods.Price,
		Score:         best.score,
	}
}

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

func resolveTierThreshold(tierID string, cfg SelectionConfig) int {
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

func resolveFallbackThreshold(raw int) int {
	if raw > 0 {
		return raw
	}
	return defaultFallbackBuyThreshold
}
