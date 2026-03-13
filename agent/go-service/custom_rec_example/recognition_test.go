package customrecexample

import (
	"encoding/json"
	"testing"
)

func TestBuildResultDetail(t *testing.T) {
	payload := detailPayload{
		PrimaryTemplatePath:   primaryTemplatePath,
		PrimaryTemplateHits:   2,
		SecondaryTemplatePath: secondaryTemplatePath,
		SecondaryTemplateHits: 1,
		OCRNode:               ocrNode,
		InspectedTemplateHits: 2,
		Entries: []detailEntry{
			{
				TemplateBox: [4]int{1, 2, 3, 4},
				OCRROI:      [4]int{5, 6, 7, 8},
				OCRHit:      true,
				OCRText:     "活动中心",
				OCRBox:      [4]int{9, 10, 11, 12},
			},
		},
	}

	raw, err := buildResultDetail(payload)
	if err != nil {
		t.Fatalf("buildResultDetail returned error: %v", err)
	}

	var decoded detailPayload
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("failed to unmarshal detail json: %v", err)
	}

	if decoded.PrimaryTemplateHits != payload.PrimaryTemplateHits {
		t.Fatalf("unexpected primary template hits: got %d want %d", decoded.PrimaryTemplateHits, payload.PrimaryTemplateHits)
	}
	if decoded.SecondaryTemplateHits != payload.SecondaryTemplateHits {
		t.Fatalf("unexpected secondary template hits: got %d want %d", decoded.SecondaryTemplateHits, payload.SecondaryTemplateHits)
	}
	if len(decoded.Entries) != 1 {
		t.Fatalf("unexpected entry count: got %d want 1", len(decoded.Entries))
	}
	if decoded.Entries[0].OCRText != payload.Entries[0].OCRText {
		t.Fatalf("unexpected OCR text: got %q want %q", decoded.Entries[0].OCRText, payload.Entries[0].OCRText)
	}
}

func TestClampRectTo720P(t *testing.T) {
	rect := clampRectTo720P([4]int{-20, 700, 200, 80})
	want := [4]int{0, 700, 200, 20}
	if rect != want {
		t.Fatalf("unexpected clamped rect: got %v want %v", rect, want)
	}
}

func TestClampRectTo720PForOffscreenOrigin(t *testing.T) {
	rect := clampRectTo720P([4]int{1400, 900, 200, 80})
	want := [4]int{1279, 719, 1, 1}
	if rect != want {
		t.Fatalf("unexpected clamped rect for offscreen origin: got %v want %v", rect, want)
	}
}
