package ollamarunner

import (
	"image"
	"testing"
	"time"

	"github.com/ollama/ollama/model/input"
)

func TestCountCommon(t *testing.T) {
	imgA := image.NewRGBA(image.Rect(0, 0, 100, 100))
	imgB := image.NewRGBA(image.Rect(0, 0, 50, 50))
	imgC := image.NewRGBA(image.Rect(50, 50, 100, 100))

	tests := []struct {
		name     string
		t1       []input.Input
		t2       []input.Input
		expected int32
	}{
		{
			name:     "Equal",
			t1:       []input.Input{{Token: 1}, {Token: 2}, {Token: 3}},
			t2:       []input.Input{{Token: 1}, {Token: 2}, {Token: 3}},
			expected: 3,
		},
		{
			name:     "Prefix",
			t1:       []input.Input{{Token: 1}},
			t2:       []input.Input{{Token: 1}, {Token: 2}, {Token: 3}},
			expected: 1,
		},
		{
			name:     "Image Prefix",
			t1:       []input.Input{{Multimodal: imgA, MultimodalHash: 1}},
			t2:       []input.Input{{Multimodal: imgA, MultimodalHash: 1}, {Multimodal: imgB, MultimodalHash: 2}, {Multimodal: imgC, MultimodalHash: 3}},
			expected: 1,
		},
		{
			name:     "Mixed",
			t1:       []input.Input{{Token: 1}, {Multimodal: imgA, MultimodalHash: 1}},
			t2:       []input.Input{{Token: 1}, {Multimodal: imgA, MultimodalHash: 1}, {Token: 5}},
			expected: 2,
		},
		{
			name:     "Mixed, Same Length",
			t1:       []input.Input{{Token: 1}, {Multimodal: imgA, MultimodalHash: 1}},
			t2:       []input.Input{{Token: 1}, {Multimodal: imgB, MultimodalHash: 2}},
			expected: 1,
		},
		{
			name:     "Empty",
			t1:       []input.Input{},
			t2:       []input.Input{{Token: 1}, {Token: 2}, {Token: 3}},
			expected: 0,
		},
		{
			name:     "Both Empty",
			t1:       []input.Input{},
			t2:       []input.Input{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countCommonPrefix(tt.t1, tt.t2)
			if result != tt.expected {
				t.Errorf("countCommonPrefix(%v, %v): have %v; want %v", tt.t1, tt.t2, result, tt.expected)
			}
		})
	}
}

func TestFindCacheSlot(t *testing.T) {
	type expected struct {
		result int
		len    int32
	}

	tests := []struct {
		name    string
		cache   InputCache
		prompt  []input.Input
		longest expected
		best    expected
	}{
		{
			name: "Empty",
			cache: InputCache{slots: []InputCacheSlot{
				{
					Id:       0,
					Inputs:   []input.Input{},
					InUse:    false,
					lastUsed: time.Time{},
				},
				{
					Id:       1,
					Inputs:   []input.Input{},
					InUse:    false,
					lastUsed: time.Time{},
				},
			}},
			prompt:  []input.Input{{Token: 1}},
			longest: expected{result: 0, len: 0},
			best:    expected{result: 0, len: 0},
		},
		{
			name: "Extend",
			cache: InputCache{slots: []InputCacheSlot{
				{
					Id:       0,
					Inputs:   []input.Input{{Token: 1}},
					InUse:    false,
					lastUsed: time.Now().Add(-time.Second),
				},
				{
					Id:       1,
					Inputs:   []input.Input{{Token: 1}, {Token: 2}},
					InUse:    false,
					lastUsed: time.Now().Add(-2 * time.Second),
				},
			}},
			prompt:  []input.Input{{Token: 1}, {Token: 2}},
			longest: expected{result: 1, len: 2},
			best:    expected{result: 1, len: 2},
		},
		{
			name: "New",
			cache: InputCache{slots: []InputCacheSlot{
				{
					Id:       0,
					Inputs:   []input.Input{{Token: 1}, {Token: 2}},
					InUse:    false,
					lastUsed: time.Now().Add(-time.Second),
				},
				{
					Id:       1,
					Inputs:   []input.Input{},
					InUse:    false,
					lastUsed: time.Time{},
				},
			}},
			prompt:  []input.Input{{Token: 2}},
			longest: expected{result: 0, len: 0},
			best:    expected{result: 1, len: 0},
		},
		{
			name: "Fork",
			cache: InputCache{
				slots: []InputCacheSlot{
					{
						Id:       0,
						Inputs:   []input.Input{{Token: 1}, {Token: 2}},
						InUse:    false,
						lastUsed: time.Now().Add(-time.Second),
					},
					{
						Id:       1,
						Inputs:   []input.Input{},
						InUse:    false,
						lastUsed: time.Time{},
					},
				},
			},
			prompt:  []input.Input{{Token: 1}},
			longest: expected{result: 0, len: 1},
			best:    expected{result: 1, len: 1},
		},
		{
			name: "Evict",
			cache: InputCache{slots: []InputCacheSlot{
				{
					Id:       0,
					Inputs:   []input.Input{{Token: 1}},
					InUse:    false,
					lastUsed: time.Now().Add(-time.Second),
				},
				{
					Id:       1,
					Inputs:   []input.Input{{Token: 1}, {Token: 2}},
					InUse:    false,
					lastUsed: time.Now().Add(-2 * time.Second),
				},
			}},
			prompt:  []input.Input{{Token: 2}, {Token: 3}},
			longest: expected{result: 0, len: 0},
			best:    expected{result: 1, len: 0},
		},
		{
			name: "In use",
			cache: InputCache{slots: []InputCacheSlot{
				{
					Id:       0,
					Inputs:   []input.Input{{Token: 1}, {Token: 2}},
					InUse:    true,
					lastUsed: time.Now().Add(-time.Second),
				},
				{
					Id:       1,
					Inputs:   []input.Input{{Token: 1}},
					InUse:    false,
					lastUsed: time.Now().Add(-2 * time.Second),
				},
			}},
			prompt:  []input.Input{{Token: 1}, {Token: 2}},
			longest: expected{result: 1, len: 1},
			best:    expected{result: 1, len: 2},
		},
	}

	for _, tt := range tests {
		t.Run("Longest-"+tt.name, func(t *testing.T) {
			result, resultLen, err := tt.cache.findLongestCacheSlot(tt.prompt)
			if err != nil {
				t.Errorf("findLongestCacheSlot: err %v", err)
			} else if result.Id != tt.longest.result || resultLen != tt.longest.len {
				t.Errorf("findLongestCacheSlot: slot have %v, want %v len have %v, want %v",
					result.Id, tt.longest.result, resultLen, tt.longest.len)
			}
		})
	}

	for _, tt := range tests {
		t.Run("Best-"+tt.name, func(t *testing.T) {
			result, resultLen, err := tt.cache.findBestCacheSlot(tt.prompt)
			if err != nil {
				t.Errorf("findBestCacheSlot: err %v", err)
			} else if result.Id != tt.best.result || resultLen != tt.best.len {
				t.Errorf("findBestCacheSlot: slot have %v, want %v len have %v, want %v",
					result.Id, tt.best.result, resultLen, tt.best.len)
			}
		})
	}
}

func TestShiftDiscard(t *testing.T) {
	tests := []struct {
		name     string
		numCtx   int32
		numKeep  int32
		inputLen int32
		expected int32
	}{
		{
			name:     "Shift",
			numCtx:   2048,
			numKeep:  5,
			inputLen: 2048,
			expected: 1021,
		},
		{
			name:     "Max Keep",
			numCtx:   2048,
			numKeep:  2047,
			inputLen: 2048,
			expected: 1,
		},
		{
			name:     "No Keep",
			numCtx:   2048,
			numKeep:  0,
			inputLen: 2048,
			expected: 1024,
		},
		{
			name:     "Truncate",
			numCtx:   2048,
			numKeep:  5,
			inputLen: 5000,
			expected: 3973,
		},
		{
			name:     "Truncate Keep",
			numCtx:   2048,
			numKeep:  2047,
			inputLen: 5000,
			expected: 2953,
		},
		{
			name:     "No Op",
			numCtx:   2048,
			numKeep:  5,
			inputLen: 512,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := InputCache{numCtx: tt.numCtx}
			result := c.ShiftDiscard(tt.inputLen, tt.numKeep)
			if result != tt.expected {
				t.Errorf("shiftDiscard(ctx: %v, keep: %v input: %v): have %v; want %v", tt.numCtx, tt.numKeep, tt.inputLen, result, tt.expected)
			}
		})
	}
}
