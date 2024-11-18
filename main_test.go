// main_test.go
package main

import (
	"testing"
)

func TestInitialize(t *testing.T) {
	game := &Game{}
	game.Initialise()

	// Test counts
	fishCount := 0
	sharkCount := 0
	emptyCount := 0

	// Count entities
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			cell := game.grid[y][x]
			switch cell.Type {
			case Fish:
				fishCount++
				if cell.BreedTime != fishBreedTime {
					t.Errorf("Fish at (%d,%d) has incorrect breed time: got %d, want %d",
						x, y, cell.BreedTime, fishBreedTime)
				}
			case Shark:
				sharkCount++
				if cell.BreedTime != sharkBreedTime {
					t.Errorf("Shark at (%d,%d) has incorrect breed time: got %d, want %d",
						x, y, cell.BreedTime, sharkBreedTime)
				}
				if cell.StarveTime != sharkStarveTime {
					t.Errorf("Shark at (%d,%d) has incorrect starve time: got %d, want %d",
						x, y, cell.StarveTime, sharkStarveTime)
				}
			case Empty:
				emptyCount++
			}
		}
	}

	totalCells := gridSize * gridSize
	fishPercentageActual := (float64(fishCount) / float64(totalCells)) * 100
	sharkPercentageActual := (float64(sharkCount) / float64(totalCells)) * 100

	// Allow for some random variation (±5%)
	tolerance := 5.0
	if abs(fishPercentageActual-float64(fishPercentage)) > tolerance {
		t.Errorf("Fish percentage outside acceptable range: got %.2f%%, want %d%% ±%.1f%%",
			fishPercentageActual, fishPercentage, tolerance)
	}
	if abs(sharkPercentageActual-float64(sharkPercentage)) > tolerance {
		t.Errorf("Shark percentage outside acceptable range: got %.2f%%, want %d%% ±%.1f%%",
			sharkPercentageActual, sharkPercentage, tolerance)
	}
}

func TestGetAdjacent(t *testing.T) {
	game := &Game{}

	testCases := []struct {
		name     string
		x, y     int
		expected int
	}{
		{"Center", gridSize / 2, gridSize / 2, 4},
		{"Top Left", 0, 0, 4},
		{"Top Right", gridSize - 1, 0, 4},
		{"Bottom Left", 0, gridSize - 1, 4},
		{"Bottom Right", gridSize - 1, gridSize - 1, 4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adjacent := game.getAdjacent(tc.x, tc.y)

			if len(adjacent) != tc.expected {
				t.Errorf("got %d adjacent cells, want %d", len(adjacent), tc.expected)
			}

			// Check for duplicates
			seen := make(map[[2]int]bool)
			for _, pos := range adjacent {
				if seen[pos] {
					t.Errorf("found duplicate position %v", pos)
				}
				seen[pos] = true

				// Verify coordinates are within grid bounds
				if pos[0] < 0 || pos[0] >= gridSize || pos[1] < 0 || pos[1] >= gridSize {
					t.Errorf("position %v is out of bounds", pos)
				}
			}
		})
	}
}

// Helper function for calculating absolute value of float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
