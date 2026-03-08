package stockpile

// FluctuationType represents the price fluctuation level of a product in the game store.
type FluctuationType int

const (
	// FluctuationModerate indicates moderate price fluctuation (适中).
	FluctuationModerate FluctuationType = iota
	// FluctuationLarge indicates large price fluctuation (较大).
	FluctuationLarge
	// FluctuationExtreme indicates extreme price fluctuation (极大).
	FluctuationExtreme
)

// Region represents a game area where stockpile operations take place.
type Region string

const (
	// RegionValleyIV represents the Valley IV (四号谷地) area.
	RegionValleyIV Region = "ValleyIV"
	// RegionWuling represents the Wuling (武陵) area.
	RegionWuling Region = "Wuling"
)

// Thresholds holds the maximum acceptable purchase price for each
// product fluctuation type per region.
type Thresholds struct {
	ValleyModerate int
	ValleyLarge    int
	ValleyExtreme  int
	WulingModerate int
}

// productFluctuationEntry maps a grid position (row, col) to its fluctuation type.
type productFluctuationEntry struct {
	Row         int
	Col         int
	Fluctuation FluctuationType
}

// valleyIVProducts maps each grid position in Valley IV to its fluctuation type.
// TODO: Update this mapping table based on actual in-game product layout data.
var valleyIVProducts = []productFluctuationEntry{
	{1, 1, FluctuationModerate}, {1, 2, FluctuationModerate},
	{1, 3, FluctuationModerate}, {1, 4, FluctuationModerate},
	{1, 5, FluctuationLarge}, {1, 6, FluctuationLarge},
	{1, 7, FluctuationLarge}, {1, 8, FluctuationLarge},
	{2, 1, FluctuationLarge}, {2, 2, FluctuationLarge},
	{2, 3, FluctuationExtreme}, {2, 4, FluctuationExtreme},
	{2, 5, FluctuationExtreme}, {2, 6, FluctuationExtreme},
	{2, 7, FluctuationExtreme}, {2, 8, FluctuationExtreme},
	{3, 1, FluctuationModerate}, {3, 2, FluctuationModerate},
	{3, 3, FluctuationLarge}, {3, 4, FluctuationLarge},
	{3, 5, FluctuationExtreme}, {3, 6, FluctuationExtreme},
	{3, 7, FluctuationExtreme}, {3, 8, FluctuationExtreme},
}

// wulingProducts maps each grid position in Wuling to its fluctuation type.
// Currently all Wuling elastic goods are moderate fluctuation.
// TODO: Update this mapping table based on actual in-game product layout data.
var wulingProducts = []productFluctuationEntry{
	{1, 1, FluctuationModerate}, {1, 2, FluctuationModerate},
	{1, 3, FluctuationModerate}, {1, 4, FluctuationModerate},
	{1, 5, FluctuationModerate}, {1, 6, FluctuationModerate},
	{1, 7, FluctuationModerate}, {1, 8, FluctuationModerate},
	{2, 1, FluctuationModerate}, {2, 2, FluctuationModerate},
	{2, 3, FluctuationModerate}, {2, 4, FluctuationModerate},
	{2, 5, FluctuationModerate}, {2, 6, FluctuationModerate},
	{2, 7, FluctuationModerate}, {2, 8, FluctuationModerate},
	{3, 1, FluctuationModerate}, {3, 2, FluctuationModerate},
	{3, 3, FluctuationModerate}, {3, 4, FluctuationModerate},
	{3, 5, FluctuationModerate}, {3, 6, FluctuationModerate},
	{3, 7, FluctuationModerate}, {3, 8, FluctuationModerate},
}

// GetThreshold returns the price threshold for a given region and fluctuation type.
func GetThreshold(t Thresholds, region Region, fluctuation FluctuationType) int {
	switch region {
	case RegionValleyIV:
		switch fluctuation {
		case FluctuationModerate:
			return t.ValleyModerate
		case FluctuationLarge:
			return t.ValleyLarge
		case FluctuationExtreme:
			return t.ValleyExtreme
		}
	case RegionWuling:
		return t.WulingModerate
	}
	return 0
}

// GetProductFluctuation returns the fluctuation type for a product at the
// given grid position in the specified region.
// Falls back to FluctuationModerate if no mapping is found.
func GetProductFluctuation(region Region, row, col int) FluctuationType {
	var products []productFluctuationEntry
	switch region {
	case RegionValleyIV:
		products = valleyIVProducts
	case RegionWuling:
		products = wulingProducts
	default:
		return FluctuationModerate
	}
	for _, p := range products {
		if p.Row == row && p.Col == col {
			return p.Fluctuation
		}
	}
	return FluctuationModerate
}
