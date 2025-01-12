package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Matrix represents the state of the puzzle.
type Matrix struct {
	values        [4][4]byte
	numberOfSteps int
	freeI         int
	freeJ         int
	previousState *Matrix
	manhattan     int
	move          string
}

var (
	moveVertically   = []int{0, -1, 0, 1}
	moveHorizontally = []int{-1, 0, 1, 0}
	moves            = []string{"left", "up", "right", "down"}
)

func NewMatrix(values [4][4]byte, freeI, freeJ, numberOfSteps int, previousState *Matrix, move string) *Matrix {
	matrix := &Matrix{
		values:        values,
		numberOfSteps: numberOfSteps,
		freeI:         freeI,
		freeJ:         freeJ,
		previousState: previousState,
		move:          move,
	}
	matrix.manhattan = matrix.manhattanDistance()
	return matrix
}

func ReadFromFile(filePath string) (*Matrix, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var values [4][4]byte
	var freeI, freeJ int = -1, -1
	scanner := bufio.NewScanner(file)

	for i := 0; scanner.Scan() && i < 4; i++ {
		line := strings.Fields(scanner.Text())
		for j, str := range line {
			val, err := strconv.Atoi(str)
			if err != nil {
				return nil, err
			}
			values[i][j] = byte(val)
			if val == 0 {
				freeI, freeJ = i, j
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return NewMatrix(values, freeI, freeJ, 0, nil, ""), nil
}

func (m *Matrix) manhattanDistance() int {
	sum := 0
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if m.values[i][j] != 0 {
				targetI := int((m.values[i][j] - 1) / 4)
				targetJ := int((m.values[i][j] - 1) % 4)
				sum += abs(i-targetI) + abs(j-targetJ)
			}
		}
	}
	return sum
}

func (m *Matrix) GenerateMoves() []*Matrix {
	var possibleFutureStates []*Matrix

	for k := 0; k < 4; k++ {
		newFreeI := m.freeI + moveVertically[k]
		newFreeJ := m.freeJ + moveHorizontally[k]
		if newFreeI >= 0 && newFreeI < 4 && newFreeJ >= 0 && newFreeJ < 4 {
			// Prevent reversing to the previous state
			if m.previousState != nil && newFreeI == m.previousState.freeI && newFreeJ == m.previousState.freeJ {
				continue
			}

			// Clone the current values
			var newValues [4][4]byte
			for i := range m.values {
				copy(newValues[i][:], m.values[i][:])
			}

			// Perform the move
			newValues[m.freeI][m.freeJ] = newValues[newFreeI][newFreeJ]
			newValues[newFreeI][newFreeJ] = 0

			possibleFutureStates = append(possibleFutureStates, NewMatrix(newValues, newFreeI, newFreeJ, m.numberOfSteps+1, m, moves[k]))
		}
	}

	return possibleFutureStates
}

func (m *Matrix) ToString() string {
	var steps []string
	current := m

	for current != nil {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("\n%s\n", current.move))
		for _, row := range current.values {
			sb.WriteString(fmt.Sprintf("%v\n", row))
		}
		steps = append(steps, sb.String())
		current = current.previousState
	}

	// Reverse the steps to start with the initial state
	for i, j := 0, len(steps)-1; i < j; i, j = i+1, j-1 {
		steps[i], steps[j] = steps[j], steps[i]
	}

	return fmt.Sprintf("Moves\n%s\n%d steps", strings.Join(steps, ""), m.numberOfSteps)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
