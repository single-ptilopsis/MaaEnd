package autostockpile

import (
	_ "embed"
	"encoding/json"

	"github.com/rs/zerolog/log"
)

//go:embed item_value_change_map.json
var itemValueChangeMapData string

// ItemValueChangeMapCache holds the loaded item tier mapping
var itemValueChangeMapCache *ItemValueChangeMap

// LoadItemValueChangeMap loads and parses the embedded item_value_change_map.json
func LoadItemValueChangeMap() (*ItemValueChangeMap, error) {
	if itemValueChangeMapCache != nil {
		return itemValueChangeMapCache, nil
	}

	var data ItemValueChangeMap
	if err := json.Unmarshal([]byte(itemValueChangeMapData), &data); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal item value change map")
		return nil, err
	}

	itemValueChangeMapCache = &data
	log.Info().Msg("item value change map loaded")
	return itemValueChangeMapCache, nil
}
