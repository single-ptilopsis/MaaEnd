package autostockpile

import (
	"encoding/json"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func getSelectionConfigFromNode(ctx *maa.Context, nodeName string) (SelectionConfig, error) {
	raw, err := ctx.GetNodeJSON(nodeName)
	if err != nil {
		log.Error().Err(err).Str("component", "autostockpile").Str("node", nodeName).Msg("failed to get node json")
		return SelectionConfig{}, err
	}

	return parseSelectionConfigFromNodeJSON(raw)
}

func parseSelectionConfigFromNodeJSON(raw string) (SelectionConfig, error) {
	cfg := SelectionConfig{FallbackThreshold: defaultFallbackBuyThreshold}
	if raw == "" {
		return cfg, nil
	}

	var wrapper struct {
		Attach SelectionConfig `json:"attach"`
	}

	if err := json.Unmarshal([]byte(raw), &wrapper); err != nil {
		return SelectionConfig{}, err
	}
	if wrapper.Attach.FallbackThreshold <= 0 {
		wrapper.Attach.FallbackThreshold = defaultFallbackBuyThreshold
	}

	attachJSON, err := json.Marshal(wrapper.Attach)
	if err != nil {
		log.Warn().Err(err).Str("component", "autostockpile").Msg("failed to marshal attach config")
	} else {
		log.Info().Str("component", "autostockpile").Str("attach", string(attachJSON)).Msg("attach config loaded")
	}

	return wrapper.Attach, nil
}
