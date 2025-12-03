package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLudusaviExecutor_ParseOutput_Success(t *testing.T) {
	executor := NewLudusaviExecutor()

	output := []byte(`{
		"overall": {
			"totalGames": 178,
			"totalBytes": 3353481924,
			"processedGames": 174,
			"processedBytes": 3330568438,
			"changedGames": {
				"new": 2,
				"different": 5,
				"same": 167
			}
		},
		"games": {}
	}`)

	stats, err := executor.parseOutput(output)
	require.NoError(t, err)

	assert.Equal(t, 178, stats.TotalGames)
	assert.Equal(t, int64(3353481924), stats.TotalBytes)
	assert.Equal(t, 174, stats.ProcessedGames)
	assert.Equal(t, int64(3330568438), stats.ProcessedBytes)
	assert.Equal(t, 2, stats.NewGames)
	assert.Equal(t, 5, stats.ChangedGames)
	assert.Equal(t, 167, stats.SameGames)
}

func TestLudusaviExecutor_ParseOutput_Empty(t *testing.T) {
	executor := NewLudusaviExecutor()

	// Empty output (e.g., cloud upload with nothing to sync)
	stats, err := executor.parseOutput([]byte(`{}`))
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.TotalGames)
}

func TestLudusaviExecutor_ParseOutput_WhitespaceOnly(t *testing.T) {
	executor := NewLudusaviExecutor()

	stats, err := executor.parseOutput([]byte("   \n  "))
	require.NoError(t, err)
	assert.NotNil(t, stats)
}

func TestLudusaviExecutor_ParseOutput_InvalidJSON(t *testing.T) {
	executor := NewLudusaviExecutor()

	_, err := executor.parseOutput([]byte("not json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON")
}

func TestLudusaviExecutor_ParseOutput_CloudUpload(t *testing.T) {
	executor := NewLudusaviExecutor()

	// Cloud upload response format
	output := []byte(`{
		"overall": {
			"totalGames": 50,
			"totalBytes": 1000000000,
			"processedGames": 50,
			"processedBytes": 1000000000,
			"changedGames": {
				"new": 0,
				"different": 3,
				"same": 47
			}
		},
		"games": {}
	}`)

	stats, err := executor.parseOutput(output)
	require.NoError(t, err)

	assert.Equal(t, 50, stats.TotalGames)
	assert.Equal(t, 3, stats.ChangedGames)
}

func TestLudusaviExecutor_GetCommonPaths(t *testing.T) {
	executor := NewLudusaviExecutor()
	paths := executor.getCommonPaths()

	assert.NotEmpty(t, paths)
	// Should have at least one candidate path
	assert.GreaterOrEqual(t, len(paths), 1)
}

func TestNewLudusaviExecutor_WithBinaryPath(t *testing.T) {
	executor := NewLudusaviExecutor(WithBinaryPath("/custom/path/ludusavi"))

	assert.Equal(t, "/custom/path/ludusavi", executor.binaryPath)
}
