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

// setThresholds updates the global price thresholds under mutex protection.
func setThresholds(t Thresholds) {
	stateMu.Lock()
	defer stateMu.Unlock()
	thresholds = t
}

// getThresholds returns a copy of the current price thresholds.
func getThresholds() Thresholds {
	stateMu.Lock()
	defer stateMu.Unlock()
	return thresholds
}

// setCurrentRegion stores the region currently being processed.
func setCurrentRegion(r Region) {
	stateMu.Lock()
	defer stateMu.Unlock()
	currentRegion = r
}

// getCurrentRegion returns the region currently being processed.
func getCurrentRegion() Region {
	stateMu.Lock()
	defer stateMu.Unlock()
	return currentRegion
}

// clearRecords empties the scanned product records for a new scan cycle.
func clearRecords() {
	stateMu.Lock()
	defer stateMu.Unlock()
	records = records[:0]
}

// appendRecord adds a product scan result to the records slice.
func appendRecord(r ProductRecord) {
	stateMu.Lock()
	defer stateMu.Unlock()
	records = append(records, r)
}

// getRecords returns a snapshot copy of the current scanned records.
func getRecords() []ProductRecord {
	stateMu.Lock()
	defer stateMu.Unlock()
	out := make([]ProductRecord, len(records))
	copy(out, records)
	return out
}

// setOverflowAmount stores the calculated quota overflow count.
func setOverflowAmount(v int) {
	stateMu.Lock()
	defer stateMu.Unlock()
	overflowAmount = v
}

// getOverflowAmount returns the current quota overflow count.
func getOverflowAmount() int {
	stateMu.Lock()
	defer stateMu.Unlock()
	return overflowAmount
}

// setOverflowBuy sets whether low-price buying on overflow is enabled.
func setOverflowBuy(v bool) {
	stateMu.Lock()
	defer stateMu.Unlock()
	overflowBuy = v
}

// getOverflowBuy returns whether low-price buying on overflow is enabled.
func getOverflowBuy() bool {
	stateMu.Lock()
	defer stateMu.Unlock()
	return overflowBuy
}

// setSundayBuyAll sets whether all-buy mode on Sunday is enabled.
func setSundayBuyAll(v bool) {
	stateMu.Lock()
	defer stateMu.Unlock()
	sundayBuyAll = v
}

// getSundayBuyAll returns whether all-buy mode on Sunday is enabled.
func getSundayBuyAll() bool {
	stateMu.Lock()
	defer stateMu.Unlock()
	return sundayBuyAll
}

// setScanPos updates the current scan grid position (row, col).
func setScanPos(row, col int) {
	stateMu.Lock()
	defer stateMu.Unlock()
	scanRow, scanCol = row, col
}

// getScanPos returns the current scan grid position (row, col).
func getScanPos() (int, int) {
	stateMu.Lock()
	defer stateMu.Unlock()
	return scanRow, scanCol
}
