package stockpile

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

// StockpileInitAction parses configuration parameters from Pipeline and
// initialises module state for the current run.
var _ maa.CustomActionRunner = &StockpileInitAction{}

// StockpileInitAction resets internal state and stores the thresholds /
// flags received from the Pipeline's custom_action_param.
type StockpileInitAction struct{}

func (a *StockpileInitAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	log.Info().
		Str("component", "Stockpile").
		Msg("initialising stockpile flow")

	var params struct {
		Strategy       string      `json:"Strategy"`
		ValleyModerate interface{} `json:"ValleyModerate"`
		ValleyLarge    interface{} `json:"ValleyLarge"`
		ValleyExtreme  interface{} `json:"ValleyExtreme"`
		WulingModerate interface{} `json:"WulingModerate"`
		OverflowBuy    interface{} `json:"OverflowBuy"`
		SundayBuyAll   interface{} `json:"SundayBuyAll"`
	}
	if err := json.Unmarshal([]byte(arg.CustomActionParam), &params); err != nil {
		log.Error().
			Err(err).
			Str("component", "Stockpile").
			Msg("failed to parse custom_action_param")
		return false
	}

	t := Thresholds{
		ValleyModerate: parseIntField(params.ValleyModerate, 1000),
		ValleyLarge:    parseIntField(params.ValleyLarge, 1400),
		ValleyExtreme:  parseIntField(params.ValleyExtreme, 1700),
		WulingModerate: parseIntField(params.WulingModerate, 1700),
	}

	setThresholds(t)
	setOverflowBuy(parseBoolField(params.OverflowBuy))
	setSundayBuyAll(parseBoolField(params.SundayBuyAll))
	clearRecords()
	setOverflowAmount(0)

	log.Info().
		Str("component", "Stockpile").
		Str("strategy", params.Strategy).
		Int("valleyModerate", t.ValleyModerate).
		Int("valleyLarge", t.ValleyLarge).
		Int("valleyExtreme", t.ValleyExtreme).
		Int("wulingModerate", t.WulingModerate).
		Bool("overflowBuy", getOverflowBuy()).
		Bool("sundayBuyAll", getSundayBuyAll()).
		Msg("parameters loaded")

	return true
}

// parseIntField converts an interface{} (float64 or string from JSON) to int.
func parseIntField(v interface{}, defaultVal int) int {
	if v == nil {
		return defaultVal
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case string:
		n, err := strconv.Atoi(val)
		if err != nil {
			return defaultVal
		}
		return n
	default:
		return defaultVal
	}
}

// parseBoolField converts an interface{} to bool (supports bool and string "true").
func parseBoolField(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		b, _ := strconv.ParseBool(val)
		return b
	case float64:
		return val != 0
	default:
		return false
	}
}

// IsSunday returns true when the current system time falls within the
// game's "Sunday" window: Sunday 04:00 – Monday 03:59 (local time).
func IsSunday() bool {
	now := time.Now()
	weekday := now.Weekday()
	hour := now.Hour()

	// Sunday 04:00 ~ Sunday 23:59
	if weekday == time.Sunday && hour >= 4 {
		return true
	}
	// Monday 00:00 ~ Monday 03:59
	if weekday == time.Monday && hour < 4 {
		return true
	}
	return false
}
