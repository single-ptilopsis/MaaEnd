package autostockpile

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
)

func TestParseAllOCRTokens(t *testing.T) {
	detail := &maa.RecognitionDetail{
		Results: &maa.RecognitionResults{
			All: []*maa.RecognitionResult{
				newOCRRecognitionResult("市场", [4]int{65, 402, 60, 25}, 0.99),
				newOCRRecognitionResult("武侠电影货组", [4]int{96, 610, 89, 21}, 0.95),
				newOCRRecognitionResult("3127", [4]int{129, 571, 75, 24}, 0.99),
			},
			Best: newOCRRecognitionResult("仅Best文本", [4]int{1, 1, 10, 10}, 1.0),
		},
	}

	parsed, err := ParseOCRCandidates(nil, detail, "")
	if err != nil {
		t.Fatalf("ParseOCRCandidates returned error: %v", err)
	}

	if len(parsed.AllTokens) != 3 {
		t.Fatalf("expected parser to consume all OCR tokens, got %d", len(parsed.AllTokens))
	}

	allTexts := make([]string, 0, len(parsed.AllTokens))
	for _, token := range parsed.AllTokens {
		allTexts = append(allTexts, token.Text)
	}

	if containsText(allTexts, "仅Best文本") {
		t.Fatalf("parser should not rely on Best token only: %v", allTexts)
	}

	if !containsText(allTexts, "武侠电影货组") {
		t.Fatalf("expected product name token from Results.All, got %v", allTexts)
	}
}

func TestFilterOwnedSectionAndMarkers(t *testing.T) {
	fixturePath := filepath.Join("WulingOCR.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	detail := &maa.RecognitionDetail{
		DetailJson: `{"all":` + string(raw) + `,"best":null,"filtered":[]}`,
	}

	parsed, err := ParseOCRCandidates(nil, detail, "")
	if err != nil {
		t.Fatalf("ParseOCRCandidates returned error: %v", err)
	}

	if len(parsed.AllTokens) == 0 {
		t.Fatal("expected tokens from fallback DetailJson path")
	}

	if len(parsed.CandidateRegions) == 0 {
		t.Fatal("expected product candidate regions after filtering")
	}

	if len(parsed.OverflowMarkerTokens) == 0 {
		t.Fatal("expected overflow marker detection in noisy OCR")
	}

	for _, marker := range parsed.OverflowMarkerTokens {
		norm := normalizeMarkerText(marker.Text)
		if !strings.Contains(norm, "溢出") && !strings.Contains(norm, "即出") {
			t.Fatalf("unexpected overflow marker token: %q", marker.Text)
		}
	}

	for _, token := range parsed.ProductTokens {
		norm := normalizeMarkerText(token.Text)
		if strings.Contains(norm, "市场") || strings.Contains(norm, "拥有") {
			t.Fatalf("product tokens should exclude divider and owned sections, got %q", token.Text)
		}
		if token.CenterY() <= parsed.EffectiveProductDivider {
			t.Fatalf("product token should be below divider: token=%q y=%d divider=%d", token.Text, token.CenterY(), parsed.EffectiveProductDivider)
		}
	}
}

func TestParseEmptyOCRResult(t *testing.T) {
	detail := &maa.RecognitionDetail{
		Results: &maa.RecognitionResults{
			All: []*maa.RecognitionResult{},
		},
	}

	parsed, err := ParseOCRCandidates(nil, detail, "")
	if err != nil {
		t.Fatalf("ParseOCRCandidates returned error: %v", err)
	}

	if len(parsed.AllTokens) != 0 {
		t.Fatalf("expected 0 tokens for empty OCR result, got %d", len(parsed.AllTokens))
	}
	if len(parsed.CandidateRegions) != 0 {
		t.Fatalf("expected 0 candidate regions, got %d", len(parsed.CandidateRegions))
	}
}

func TestParseNilRecognitionDetail(t *testing.T) {
	parsed, err := ParseOCRCandidates(nil, nil, "")
	if err != nil {
		t.Fatalf("ParseOCRCandidates returned error: %v", err)
	}

	if len(parsed.AllTokens) != 0 {
		t.Fatalf("expected 0 tokens for nil detail, got %d", len(parsed.AllTokens))
	}
	if parsed == nil {
		t.Fatal("expected non-nil OCRParseResult, got nil")
	}
}

func TestParseOCRWithoutDivider(t *testing.T) {
	detail := &maa.RecognitionDetail{
		Results: &maa.RecognitionResults{
			All: []*maa.RecognitionResult{
				newOCRRecognitionResult("天使罐头货组", [4]int{100, 300, 90, 21}, 0.95),
				newOCRRecognitionResult("1200", [4]int{200, 300, 60, 21}, 0.99),
				newOCRRecognitionResult("19.6%", [4]int{280, 300, 50, 21}, 0.99),
			},
		},
	}

	parsed, err := ParseOCRCandidates(nil, detail, "")
	if err != nil {
		t.Fatalf("ParseOCRCandidates returned error: %v", err)
	}

	if len(parsed.AllTokens) == 0 {
		t.Fatal("expected tokens in parse result")
	}
	if parsed.EffectiveProductDivider != 0 {
		t.Fatalf("expected divider=0 when no market marker present, got %d", parsed.EffectiveProductDivider)
	}
	if len(parsed.CandidateRegions) == 0 {
		t.Fatal("expected candidate regions (all tokens should be products when no divider)")
	}
}

func TestParseOCRWithOnlyNoise(t *testing.T) {
	detail := &maa.RecognitionDetail{
		Results: &maa.RecognitionResults{
			All: []*maa.RecognitionResult{
				newOCRRecognitionResult("市场", [4]int{65, 402, 60, 25}, 0.99),
				newOCRRecognitionResult("拥有5个", [4]int{150, 450, 60, 21}, 0.95),
			},
		},
	}

	parsed, err := ParseOCRCandidates(nil, detail, "")
	if err != nil {
		t.Fatalf("ParseOCRCandidates returned error: %v", err)
	}

	if len(parsed.AllTokens) == 0 {
		t.Fatal("expected tokens to be extracted")
	}
	if len(parsed.ProductTokens) != 0 {
		t.Fatalf("expected 0 product tokens when only divider and owned section present, got %d", len(parsed.ProductTokens))
	}
	if len(parsed.CandidateRegions) != 0 {
		t.Fatalf("expected 0 candidate regions from pure noise, got %d", len(parsed.CandidateRegions))
	}
}

func containsText(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func newOCRRecognitionResult(text string, box [4]int, score float64) *maa.RecognitionResult {
	result := &maa.RecognitionResult{}
	setUnexportedField(result, "tp", maa.RecognitionTypeOCR)
	setUnexportedField(result, "val", &maa.OCRResult{Text: text, Box: maa.Rect(box), Score: score})
	return result
}

func setUnexportedField(target any, fieldName string, value any) {
	v := reflect.ValueOf(target).Elem().FieldByName(fieldName)
	reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}
