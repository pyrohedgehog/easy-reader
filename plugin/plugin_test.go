package plugin

import (
	"testing"

	"github.com/golangci/plugin-module-register/register"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	easyreader "github.com/pyrohedgehog/easy-reader/pkg"
)

// TestSettingsDecode guards the plugin settings contract.
// register.DecodeSettings uses encoding/json with DisallowUnknownFields, so Settings needs matching kebab-case json tags.
func TestSettingsDecode(t *testing.T) {
	raw := map[string]any{
		"max-line-width":          200,
		"max-block-lines":         3,
		"max-block-lines-private": 1,
		"end-punctuation":         ".!?",
		"exemption-markers":       []string{"comment-length-exception", "nolint:commentlength"},
		"check-generated":         true,
		"package-doc":             map[string]any{"max-block-lines": 4},
	}
	s, err := register.DecodeSettings[easyreader.Settings](raw)
	require.NoError(t, err)
	assert.Equal(t, 200, s.MaxLineWidth)
	assert.Equal(t, 3, s.MaxBlockLines)
	assert.Equal(t, 1, s.MaxBlockLinesPrivate)
	assert.Equal(t, ".!?", s.EndPunctuation)
	assert.Equal(t, []string{"comment-length-exception", "nolint:commentlength"}, s.ExemptionMarkers)
	assert.True(t, s.CheckGenerated)
	assert.Equal(t, 4, s.PackageDoc.MaxBlockLines)
}

// TestSettingsDecodeEmpty confirms an empty settings block still decodes,
// so the plugin can be enabled with no custom settings (defaults apply).
func TestSettingsDecodeEmpty(t *testing.T) {
	s, err := register.DecodeSettings[easyreader.Settings](map[string]any{})
	require.NoError(t, err)
	// DecodeSettings returns zero values; WithDefaults (called by New) fills them.
	assert.Zero(t, s.MaxLineWidth)
	d := easyreader.DefaultSettings()
	filled := s.WithDefaults()
	assert.Equal(t, d.MaxLineWidth, filled.MaxLineWidth)
}
