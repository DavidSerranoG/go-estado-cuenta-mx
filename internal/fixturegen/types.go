package fixturegen

import (
	"context"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
)

type BrandingMode string

const (
	BrandingMixed    BrandingMode = "mixed"
	BrandingNeutral  BrandingMode = "neutral"
	BrandingPreserve BrandingMode = "preserve"
)

type OutputMode string

const (
	OutputPublic OutputMode = "public"
	OutputLocal  OutputMode = "local"
	OutputBoth   OutputMode = "both"
)

type Fidelity string

const (
	FidelityHigh   Fidelity = "visual-text-layer"
	FidelityParser Fidelity = "parser-equivalent"
)

type Config struct {
	Input            string
	Output           string
	Bank             string
	Layout           string
	File             string
	Branding         BrandingMode
	Mode             OutputMode
	Check            bool
	AllowLowFidelity bool
	ReportPath       string
	ReportFormat     string
}

type Hint struct {
	Bank   string
	Layout string
	File   string
}

type Replacement struct {
	Type           string `json:"type"`
	OriginalHash   string `json:"original_hash"`
	Replacement    string `json:"replacement"`
	Occurrences    int    `json:"occurrences"`
	ContextPreview string `json:"context_preview,omitempty"`
}

type Validation struct {
	ParseOK                bool   `json:"parse_ok"`
	BankMatched            bool   `json:"bank_matched"`
	LayoutMatched          bool   `json:"layout_matched"`
	TransactionsMatched    bool   `json:"transactions_matched"`
	KindSequenceMatched    bool   `json:"kind_sequence_matched"`
	ReferencesMatched      bool   `json:"references_matched"`
	BalancePresenceMatched bool   `json:"balance_presence_matched"`
	Error                  string `json:"error,omitempty"`
}

type Metadata struct {
	InputPath         string                 `json:"input_path"`
	OutputPath        string                 `json:"output_path,omitempty"`
	SidecarPath       string                 `json:"sidecar_path,omitempty"`
	Bank              string                 `json:"bank"`
	Layout            string                 `json:"layout"`
	Branding          BrandingMode           `json:"branding"`
	Mode              OutputMode             `json:"mode"`
	Fidelity          Fidelity               `json:"fidelity"`
	BBoxTool          string                 `json:"bbox_tool,omitempty"`
	Rasterizer        string                 `json:"rasterizer,omitempty"`
	SelectedExtractor string                 `json:"selected_extractor,omitempty"`
	ReplacementCounts map[string]int         `json:"replacement_counts"`
	Replacements      []Replacement          `json:"replacements"`
	Validation        Validation             `json:"validation"`
	Warnings          []string               `json:"warnings,omitempty"`
	Extra             map[string]interface{} `json:"extra,omitempty"`
}

type AggregateReport struct {
	Root     string     `json:"root"`
	Output   string     `json:"output"`
	Branding string     `json:"branding"`
	Mode     string     `json:"mode"`
	Files    []Metadata `json:"files"`
}

type TextDocument struct {
	Pages []TextPage
	Tool  string
}

type TextPage struct {
	Number float64
	Width  float64
	Height float64
	Lines  []TextLine
}

type TextLine struct {
	Text string
	XMin float64
	YMin float64
	XMax float64
	YMax float64
}

type RasterDocument struct {
	Pages []RasterPage
	Tool  string
}

type RasterPage struct {
	Path   string
	Width  float64
	Height float64
}

type BBoxExtractor interface {
	Extract(context.Context, []byte) (TextDocument, error)
}

type Rasterizer interface {
	Rasterize(context.Context, []byte) (RasterDocument, error)
}

type Generator struct {
	processor     *edocuenta.Processor
	bboxExtractor BBoxExtractor
	rasterizer    Rasterizer
	overridesRoot string
}
