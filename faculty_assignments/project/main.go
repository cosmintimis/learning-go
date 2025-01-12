package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	NumWorkers = 4
	MaxBound   = 80
)

func main() {
	initialState, err := ReadFromFile("input.in")
	if err != nil {
		log.Fatalf("Failed to read input: %v", err)
	}

	start := time.Now()

	minimumBound := initialState.manhattan
	for {
		tasks := make(chan *Task, NumWorkers)
		results := make(chan *Result, NumWorkers)

		var workerWg sync.WaitGroup

		for i := 0; i < NumWorkers; i++ {
			workerWg.Add(1)
			go worker(tasks, results, &workerWg)
		}

		// TODO: Create task for every posible state from the initial state
		go func() {
			tasks <- &Task{
				Current:    initialState,
				NumSteps:   0,
				Bound:      minimumBound,
				NumWorkers: NumWorkers,
			}
			close(tasks)
		}()

		// Wait for workers to finish and then close the results channel
		go func() {
			workerWg.Wait()
			close(results)
		}()

		// Collect results
		newMinimum := int(^uint(0) >> 1) // Set to max int
		for result := range results {
			if result.Solved {
				fmt.Printf("Solution found in %d steps - %v\n", result.Steps, time.Since(start))
				fmt.Println(result.FinalMatrix.ToString())
				fmt.Printf("Execution time: %v\n", time.Since(start))
				return
			}
			if result.Minimum < newMinimum {
				newMinimum = result.Minimum
			}
		}

		if newMinimum == int(^uint(0)>>1) { // No further solutions possible
			fmt.Println("No solution found")
			return
		}

		fmt.Printf("%d steps - %v\n", newMinimum, time.Since(start))
		minimumBound = newMinimum
	}
}

type Task struct {
	Current    *Matrix
	NumSteps   int
	Bound      int
	NumWorkers int
}

type Result struct {
	Minimum     int
	FinalMatrix *Matrix
	Solved      bool
	Steps       int
}

func worker(tasks <-chan *Task, results chan<- *Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for task := range tasks {
		minimum, finalMatrix, solved, steps := search(task.Current, task.NumSteps, task.Bound, task.NumWorkers)
		results <- &Result{
			Minimum:     minimum,
			FinalMatrix: finalMatrix,
			Solved:      solved,
			Steps:       steps,
		}
	}
}

func search(current *Matrix, numSteps, bound, numWorkers int) (int, *Matrix, bool, int) {
	estimation := numSteps + current.manhattan
	if estimation > bound || estimation > MaxBound {
		return estimation, nil, false, 0
	}
	if current.manhattan == 0 {
		return -1, current, true, numSteps
	}

	var minimum = int(^uint(0) >> 1) // Max int
	var solution *Matrix
	solved := false

	moves := current.GenerateMoves()
	if numWorkers > 1 {
		var wg sync.WaitGroup
		moveResults := make(chan *Result, len(moves))
		for _, next := range moves {
			wg.Add(1)
			go func(nextState *Matrix) {
				defer wg.Done()
				min, final, sol, steps := search(nextState, numSteps+1, bound, numWorkers/len(moves))
				moveResults <- &Result{Minimum: min, FinalMatrix: final, Solved: sol, Steps: steps}
			}(next)
		}
		wg.Wait()
		close(moveResults)

		for result := range moveResults {
			if result.Solved {
				return -1, result.FinalMatrix, true, result.Steps
			}
			if result.Minimum < minimum {
				minimum = result.Minimum
				solution = result.FinalMatrix
			}
		}
	} else {
		for _, next := range moves {
			min, final, sol, steps := search(next, numSteps+1, bound, 1)
			if sol {
				return -1, final, true, steps
			}
			if min < minimum {
				minimum = min
				solution = final
			}
		}
	}

	return minimum, solution, solved, numSteps
}
