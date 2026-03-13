package autostockpile

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
)

var (
	// Interface implementation assertions for compile-time verification
	_ maa.CustomActionRunner      = &SelectItemAction{}
	_ maa.CustomRecognitionRunner = &ItemValueChangeRecognition{}
)

// ItemValueChangeMap represents the tier mapping for items
// Structure: locale -> item_name -> tier_id
type ItemValueChangeMap struct {
	ZhCn map[string]string `json:"zh_cn"`
}

// SelectItemAction handles item selection based on OCR results
type SelectItemAction struct{}

// ItemValueChangeRecognition handles recognition of item value changes
type ItemValueChangeRecognition struct{}

// Run executes the item value change recognition
func (r *ItemValueChangeRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	return &maa.CustomRecognitionResult{}, true
}

// OCRToken 表示 OCR 识别出的最小文本单元。
type OCRToken struct {
	Text  string
	X     int
	Y     int
	W     int
	H     int
	Score float64
}

// CenterX 返回 token 的中心 X 坐标。
func (t OCRToken) CenterX() int {
	return t.X + t.W/2
}

// CenterY 返回 token 的中心 Y 坐标。
func (t OCRToken) CenterY() int {
	return t.Y + t.H/2
}

// OCRCandidateRegion 表示一个候选商品区域（行列分组后）。
type OCRCandidateRegion struct {
	Row    int
	Column int
	Bounds [4]int
	Tokens []OCRToken
}

// OCRParseResult 表示自动囤货页 OCR 解析结果。
type OCRParseResult struct {
	AllTokens               []OCRToken
	ProductTokens           []OCRToken
	CandidateRegions        []OCRCandidateRegion
	OverflowMarkerTokens    []OCRToken
	EffectiveProductDivider int
}

// SelectionResult 表示自动囤货商品选择结果。
type SelectionResult struct {
	Selected      bool
	ProductName   string
	CanonicalName string
	Threshold     int
	CurrentPrice  int
	Score         int
	ClickX        int
	ClickY        int
	Reason        string
}

// SelectionConfig 表示 AutoStockpile 选择策略配置。
type SelectionConfig struct {
	Strategy          string           `json:"strategy"`
	OverflowMode      bool             `json:"overflow_mode"`
	SundayMode        bool             `json:"sunday_mode"`
	FallbackThreshold int              `json:"fallback_threshold"`
	PriceLimits       PriceLimitConfig `json:"price_limits"`
}
