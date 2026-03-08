package stockpile

import "sync"

// ProductRecord stores a scanned product's price info together with its threshold and diff.
type ProductRecord struct {
	Row       int
	Col       int
	Price     int
	Threshold int
	Diff      int // threshold - price; positive means below threshold
}

var (
	stateMu sync.Mutex

	// thresholds holds the price limits for the current run.
	thresholds Thresholds
	// currentRegion tracks which region is currently being processed.
	currentRegion Region
	// records holds scanned product price data for the current region.
	records []ProductRecord
	// overflowAmount is the number of items that would overflow quota.
	overflowAmount int
	// overflowBuy controls whether to buy low-price goods on overflow.
	overflowBuy bool
	// sundayBuyAll controls whether to buy all goods on Sunday.
	sundayBuyAll bool
	// scanRow and scanCol track the current scan grid position.
	scanRow int
	scanCol int
)

func setThresholds(t Thresholds) {
	stateMu.Lock()
	defer stateMu.Unlock()
	thresholds = t
}

func getThresholds() Thresholds {
	stateMu.Lock()
	defer stateMu.Unlock()
	return thresholds
}

func setCurrentRegion(r Region) {
	stateMu.Lock()
	defer stateMu.Unlock()
	currentRegion = r
}

func getCurrentRegion() Region {
	stateMu.Lock()
	defer stateMu.Unlock()
	return currentRegion
}

func clearRecords() {
	stateMu.Lock()
	defer stateMu.Unlock()
	records = records[:0]
}

func appendRecord(r ProductRecord) {
	stateMu.Lock()
	defer stateMu.Unlock()
	records = append(records, r)
}

func getRecords() []ProductRecord {
	stateMu.Lock()
	defer stateMu.Unlock()
	out := make([]ProductRecord, len(records))
	copy(out, records)
	return out
}

func setOverflowAmount(v int) {
	stateMu.Lock()
	defer stateMu.Unlock()
	overflowAmount = v
}

func getOverflowAmount() int {
	stateMu.Lock()
	defer stateMu.Unlock()
	return overflowAmount
}

func setOverflowBuy(v bool) {
	stateMu.Lock()
	defer stateMu.Unlock()
	overflowBuy = v
}

func getOverflowBuy() bool {
	stateMu.Lock()
	defer stateMu.Unlock()
	return overflowBuy
}

func setSundayBuyAll(v bool) {
	stateMu.Lock()
	defer stateMu.Unlock()
	sundayBuyAll = v
}

func getSundayBuyAll() bool {
	stateMu.Lock()
	defer stateMu.Unlock()
	return sundayBuyAll
}

func setScanPos(row, col int) {
	stateMu.Lock()
	defer stateMu.Unlock()
	scanRow, scanCol = row, col
}

func getScanPos() (int, int) {
	stateMu.Lock()
	defer stateMu.Unlock()
	return scanRow, scanCol
}
