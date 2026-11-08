package easyreader

import "strings"

// csvFlag adapts ExemptionMarkers to flag.Value for analyzer flags.
type csvFlag []string

func (c *csvFlag) String() string { return strings.Join(*c, ",") }

func (c *csvFlag) Set(v string) error {
	var out []string
	for m := range strings.SplitSeq(v, ",") {
		if m = strings.TrimSpace(m); m != "" {
			out = append(out, m)
		}
	}
	*c = out
	return nil
}

// Settings configures the easyreader analyzer; keys map to .golangci.yml settings (see README).
// Keys carry mapstructure and json tags: DecodeSettings uses encoding/json with DisallowUnknownFields.
type Settings struct {
	// MaxLineWidth is the maximum allowed comment width in characters.
	MaxLineWidth int `mapstructure:"max-line-width" json:"max-line-width"`

	// MaxBlockLines is the maximum allowed block of consecutive inline comment lines on exported declarations.
	// A negative value disables the check.
	MaxBlockLines int `mapstructure:"max-block-lines" json:"max-block-lines"`

	// MaxBlockLinesPrivate caps inline comment blocks on unexported declarations, including their bodies.
	// Zero inherits MaxBlockLines; a negative value disables the check.
	MaxBlockLinesPrivate int `mapstructure:"max-block-lines-private" json:"max-block-lines-private"`

	// EndPunctuation lists the allowed terminal characters for a standalone comment line.
	// The auto-fix appends the first character, so list the preferred terminator first.
	EndPunctuation string `mapstructure:"end-punctuation" json:"end-punctuation"`

	// ExemptionMarkers disables block-length for blocks containing one of these substrings (case-insensitive).
	ExemptionMarkers []string `mapstructure:"exemption-markers" json:"exemption-markers"`

	// CheckGenerated lints files carrying the standard generated marker.
	// Default false skips them, matching golangci-lint's exclusion.
	CheckGenerated bool `mapstructure:"check-generated" json:"check-generated"`

	// Per-kind block-length overrides; zero fields inherit the globals above.
	PackageDoc BlockLimits `mapstructure:"package-doc" json:"package-doc"`
	FuncDoc    BlockLimits `mapstructure:"func-doc" json:"func-doc"`
	TypeDoc    BlockLimits `mapstructure:"type-doc" json:"type-doc"`
	FieldDoc   BlockLimits `mapstructure:"field-doc" json:"field-doc"`
	ValueDoc   BlockLimits `mapstructure:"value-doc" json:"value-doc"`
	ImportDoc  BlockLimits `mapstructure:"import-doc" json:"import-doc"`
}

// BlockLimits overrides the block-length limits for one comment kind.
// Zero fields inherit (private falls back to public, then globals); negative disables.
type BlockLimits struct {
	MaxBlockLines        int `mapstructure:"max-block-lines" json:"max-block-lines"`
	MaxBlockLinesPrivate int `mapstructure:"max-block-lines-private" json:"max-block-lines-private"`
}

// DefaultSettings returns the original validator's Tier 2 defaults.
func DefaultSettings() Settings {
	return Settings{
		MaxLineWidth:     150,
		MaxBlockLines:    2,
		EndPunctuation:   ".,!?:",
		ExemptionMarkers: []string{"comment-length-exception", "nolint:commentlength", "nolint:cle", "lint:ignore"},
	}
}

// WithDefaults fills unset fields with DefaultSettings values.
// Zero-value knobs stay "off" so mapstructure decode merges cleanly.
func (s Settings) WithDefaults() Settings {
	d := DefaultSettings()
	if s.MaxLineWidth == 0 {
		s.MaxLineWidth = d.MaxLineWidth
	}
	if s.MaxBlockLines == 0 {
		s.MaxBlockLines = d.MaxBlockLines
	}
	if s.MaxBlockLinesPrivate == 0 {
		s.MaxBlockLinesPrivate = s.MaxBlockLines
	}
	if s.EndPunctuation == "" {
		s.EndPunctuation = d.EndPunctuation
	}
	if s.ExemptionMarkers == nil {
		s.ExemptionMarkers = d.ExemptionMarkers
	}
	return s
}

// blockLimit resolves the block-length limit for a block with metadata m.
// Cascade for unexported kinds: kind-private, kind-public, global-private, global-public.
func (s Settings) blockLimit(m commentMeta) int {
	k := s.kindLimits(m.doc)
	if m.exported {
		return orDefault(k.MaxBlockLines, s.MaxBlockLines)
	}
	return orDefault(k.MaxBlockLinesPrivate, k.MaxBlockLines, s.MaxBlockLinesPrivate, s.MaxBlockLines)
}

// kindLimits returns the per-kind overrides for k; docNone has none.
func (s Settings) kindLimits(k docKind) BlockLimits {
	switch k {
	case docPackage:
		return s.PackageDoc
	case docFunc:
		return s.FuncDoc
	case docType:
		return s.TypeDoc
	case docField:
		return s.FieldDoc
	case docValue:
		return s.ValueDoc
	case docImport:
		return s.ImportDoc
	}
	return BlockLimits{}
}

// orDefault returns the first non-zero value.
func orDefault(vals ...int) int {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}
