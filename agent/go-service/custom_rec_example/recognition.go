package customrecexample

import (
	"encoding/json"
	"image"
	"strings"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

const (
	componentName         = "CustomRecExample"
	primaryTemplateNode   = "__CustomRecExamplePrimaryTemplate"
	secondaryTemplateNode = "__CustomRecExampleSecondaryTemplate"
	ocrNode               = "__CustomRecExampleOCR"
	primaryTemplatePath   = "DailyRewards/EventRedDot.png"
	secondaryTemplatePath = "Resell/back.png"
	screenWidth           = 1280
	screenHeight          = 720
	maxOCRChecksPerRun    = 3
	defaultOCRThreshold   = 0.3
	defaultTemplateThresh = 0.8
)

type detailEntry struct {
	TemplateBox [4]int `json:"template_box"`
	OCRROI      [4]int `json:"ocr_roi"`
	OCRHit      bool   `json:"ocr_hit"`
	OCRText     string `json:"ocr_text,omitempty"`
	OCRBox      [4]int `json:"ocr_box,omitempty"`
}

type detailPayload struct {
	PrimaryTemplatePath   string        `json:"primary_template_path"`
	PrimaryTemplateHits   int           `json:"primary_template_hits"`
	SecondaryTemplatePath string        `json:"secondary_template_path"`
	SecondaryTemplateHits int           `json:"secondary_template_hits"`
	OCRNode               string        `json:"ocr_node"`
	InspectedTemplateHits int           `json:"inspected_template_hits"`
	Entries               []detailEntry `json:"entries"`
}

type multiRecognition struct{}

var _ maa.CustomRecognitionRunner = &multiRecognition{}

func (r *multiRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	if arg == nil || arg.Img == nil {
		log.Error().
			Str("component", componentName).
			Msg("custom recognition image is nil")
		return nil, false
	}

	primaryDetail, err := runTemplateMatch(ctx, arg.Img, primaryTemplateNode, primaryTemplatePath, [4]int{0, 0, screenWidth, screenHeight})
	if err != nil {
		log.Error().
			Err(err).
			Str("component", componentName).
			Str("node", primaryTemplateNode).
			Msg("failed to run primary template match")
		return nil, false
	}

	secondaryDetail, err := runTemplateMatch(ctx, arg.Img, secondaryTemplateNode, secondaryTemplatePath, [4]int{900, 80, 260, 180})
	if err != nil {
		log.Error().
			Err(err).
			Str("component", componentName).
			Str("node", secondaryTemplateNode).
			Msg("failed to run secondary template match")
		return nil, false
	}

	entries, firstBox, inspectedCount := collectTemplateOCRPairs(ctx, arg.Img, primaryDetail)
	primaryHitCount := countTemplateHits(primaryDetail)
	secondaryHitCount := countTemplateHits(secondaryDetail)

	detailJSON, err := buildResultDetail(detailPayload{
		PrimaryTemplatePath:   primaryTemplatePath,
		PrimaryTemplateHits:   primaryHitCount,
		SecondaryTemplatePath: secondaryTemplatePath,
		SecondaryTemplateHits: secondaryHitCount,
		OCRNode:               ocrNode,
		InspectedTemplateHits: inspectedCount,
		Entries:               entries,
	})
	if err != nil {
		log.Error().
			Err(err).
			Str("component", componentName).
			Msg("failed to marshal custom recognition detail")
		return nil, false
	}

	resultBox := arg.Roi
	if firstBox != nil {
		resultBox = *firstBox
	}

	hit := primaryHitCount > 0 || secondaryHitCount > 0
	log.Info().
		Str("component", componentName).
		Int("primary_hits", primaryHitCount).
		Int("secondary_hits", secondaryHitCount).
		Int("ocr_checks", inspectedCount).
		Msg("custom recognition example finished")

	return &maa.CustomRecognitionResult{
		Box:    resultBox,
		Detail: detailJSON,
	}, hit
}

func runTemplateMatch(ctx *maa.Context, img image.Image, nodeName, templatePath string, roi [4]int) (*maa.RecognitionDetail, error) {
	config := map[string]any{
		nodeName: map[string]any{
			"recognition": "TemplateMatch",
			"template":    templatePath,
			"threshold":   defaultTemplateThresh,
			"method":      5,
			"order_by":    "score",
			"roi":         roi[:],
		},
	}

	return ctx.RunRecognition(nodeName, img, config)
}

func runOCR(ctx *maa.Context, img image.Image, roi [4]int) (*maa.RecognitionDetail, error) {
	config := map[string]any{
		ocrNode: map[string]any{
			"recognition": "OCR",
			"expected":    []string{".*"},
			"threshold":   defaultOCRThreshold,
			"roi":         roi[:],
		},
	}

	return ctx.RunRecognition(ocrNode, img, config)
}

func collectTemplateOCRPairs(ctx *maa.Context, img image.Image, detail *maa.RecognitionDetail) ([]detailEntry, *maa.Rect, int) {
	results := templateResults(detail)
	entries := make([]detailEntry, 0, min(len(results), maxOCRChecksPerRun))
	var firstBox *maa.Rect
	inspectedCount := 0

	for _, result := range results {
		tm, ok := result.AsTemplateMatch()
		if !ok {
			continue
		}
		if firstBox == nil {
			box := tm.Box
			firstBox = &box
		}
		if inspectedCount >= maxOCRChecksPerRun {
			break
		}

		ocrROI := expandTemplateBoxToOCRROI(tm.Box)
		entry := detailEntry{
			TemplateBox: rectToArray(tm.Box),
			OCRROI:      ocrROI,
		}

		ocrDetail, err := runOCR(ctx, img, ocrROI)
		if err != nil {
			log.Warn().
				Err(err).
				Str("component", componentName).
				Ints("template_box", entry.TemplateBox[:]).
				Ints("ocr_roi", entry.OCRROI[:]).
				Msg("ocr call failed for template hit")
			entries = append(entries, entry)
			inspectedCount++
			continue
		}

		text, box, hit := extractOCRResult(ocrDetail)
		entry.OCRHit = hit
		entry.OCRText = text
		if hit {
			entry.OCRBox = box
		}

		entries = append(entries, entry)
		inspectedCount++
	}

	return entries, firstBox, inspectedCount
}

func countTemplateHits(detail *maa.RecognitionDetail) int {
	count := 0
	for _, result := range templateResults(detail) {
		if _, ok := result.AsTemplateMatch(); ok {
			count++
		}
	}
	return count
}

func templateResults(detail *maa.RecognitionDetail) []*maa.RecognitionResult {
	if detail == nil || detail.Results == nil {
		return nil
	}
	if len(detail.Results.Filtered) > 0 {
		return detail.Results.Filtered
	}
	if len(detail.Results.All) > 0 {
		return detail.Results.All
	}
	if detail.Results.Best != nil {
		return []*maa.RecognitionResult{detail.Results.Best}
	}
	return nil
}

func extractOCRResult(detail *maa.RecognitionDetail) (string, [4]int, bool) {
	if detail == nil || detail.Results == nil {
		return "", [4]int{}, false
	}

	for _, results := range [][]*maa.RecognitionResult{
		{detail.Results.Best},
		detail.Results.Filtered,
		detail.Results.All,
	} {
		if len(results) == 0 || results[0] == nil {
			continue
		}
		ocrResult, ok := results[0].AsOCR()
		if !ok {
			continue
		}
		text := strings.TrimSpace(ocrResult.Text)
		if text == "" {
			continue
		}
		return text, rectToArray(ocrResult.Box), true
	}

	return "", [4]int{}, false
}

func expandTemplateBoxToOCRROI(box maa.Rect) [4]int {
	return clampRectTo720P([4]int{
		box.X() - 160,
		box.Y() - 24,
		box.Width() + 220,
		box.Height() + 48,
	})
}

func clampRectTo720P(rect [4]int) [4]int {
	x := min(max(rect[0], 0), screenWidth-1)
	y := min(max(rect[1], 0), screenHeight-1)
	maxWidth := screenWidth - x
	maxHeight := screenHeight - y
	w := min(max(rect[2], 1), maxWidth)
	h := min(max(rect[3], 1), maxHeight)
	return [4]int{x, y, w, h}
}

func rectToArray(rect maa.Rect) [4]int {
	return [4]int{rect.X(), rect.Y(), rect.Width(), rect.Height()}
}

func buildResultDetail(payload detailPayload) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
