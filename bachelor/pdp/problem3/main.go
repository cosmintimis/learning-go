package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/proabiral/puredns/v2/pkg/threadpool"
)

// used for the thread pool
type MyTask struct {
	position     int
	step         int
	matrix1      [][]float64
	matrix2      [][]float64
	resultMatrix [][]float64
}

func (m *MyTask) Run() {
	calculateElementsOfResultMatrix(m.matrix1, m.matrix2, m.resultMatrix, m.position, m.step)
}

func convertIndexToRowAndColumn(index int, sizeOfColumn int) (int, int) {
	return index / sizeOfColumn, index % sizeOfColumn
}

func computeElementOfResultMatrix(matrix1 [][]float64, matrix2 [][]float64, row int, column int) float64 {
	result := 0.0
	for i := 0; i < len(matrix1[row]); i++ {
		result += matrix1[row][i] * matrix2[i][column]
	}
	return result
}

// this function will be executing on different goroutines, there will be a total number of goroutines (K)
// every goroutine will start from a position and then will jump to the next position with a step of K
// until the end of the matrix is reached
func calculateElementsOfResultMatrix(matrix1 [][]float64, matrix2 [][]float64, resultMatrix [][]float64, position int, step int) {
	sizeOfColumnOfResultMatrix := len(matrix2[0])
	sizeOfRowOfResultMatrix := len(matrix1)
	for position < sizeOfRowOfResultMatrix*sizeOfColumnOfResultMatrix {
		currentRow, currentColumn := convertIndexToRowAndColumn(position, sizeOfColumnOfResultMatrix)
		if currentRow >= len(resultMatrix) {
			fmt.Println("Position is out of bounds.")
			return
		}
		resultMatrix[currentRow][currentColumn] = computeElementOfResultMatrix(matrix1, matrix2, currentRow, currentColumn)
		position += step
	}
}

func printMatrix(matrix [][]float64) {
	for _, row := range matrix {
		for _, element := range row {
			fmt.Printf("%.2f ", element)
		}
		fmt.Println()
	}
}

func calculateUsingThreadPool(matrix1 [][]float64, matrix2 [][]float64, resultMatrix [][]float64, numberOfTasks int, myThreadPool *threadpool.ThreadPool) {
	for i := 0; i < numberOfTasks; i++ {
		myTask := &MyTask{
			position:     i,
			step:         numberOfTasks,
			matrix1:      matrix1,
			matrix2:      matrix2,
			resultMatrix: resultMatrix,
		}
		myThreadPool.Execute(myTask)
	}
}

/*
Example 1

Matrix 1:
1 2
3 4
5 6
7 8
9 10
11 12
13 14
15 16
17 18
Matrix 2:
1 2 3 4 5 6 7 8 9
10 11 12 13 14 15 16 17 18

Result matrix:
21	24	27	30	33	36	39	42	45
43	50	57	64	71	78	85	92	99
65	76	87	98	109	120	131	142	153
87	102	117	132	147	162	177	192	207
109	128	147	166	185	204	223	242	261
131	154	177	200	223	246	269	292	315
153	180	207	234	261	288	315	342	369
175	206	237	268	299	330	361	392	423
197	232	267	302	337	372	407	442	477
*/
func main() {
	matrix1 := [][]float64{
		{1, 2},
		{3, 4},
		{5, 6},
		{7, 8},
		{9, 10},
		{11, 12},
		{13, 14},
		{15, 16},
		{17, 18},
	}
	matrix2 := [][]float64{
		{1, 2, 3, 4, 5, 6, 7, 8, 9},
		{10, 11, 12, 13, 14, 15, 16, 17, 18},
	}
	fmt.Println("Matrix 1:")
	printMatrix(matrix1)
	fmt.Println("Matrix 2:")
	printMatrix(matrix2)

	if len(matrix1[0]) != len(matrix2) {
		fmt.Println("The matrices can't be multiplied.")
		return
	}

	numberOfRowsMatrix1 := len(matrix1)
	numberOfColumnsMatrix2 := len(matrix2[0])

	resultMatrix := make([][]float64, numberOfRowsMatrix1)
	for i := range resultMatrix {
		resultMatrix[i] = make([]float64, numberOfColumnsMatrix2)
	}

	// the number of goroutines that will be used to calculate the result matrix
	numberOfGoroutines := 1
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < numberOfGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			calculateElementsOfResultMatrix(matrix1, matrix2, resultMatrix, i, numberOfGoroutines)
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)
	log.Printf("Calculation took %s", elapsed)

	fmt.Println("Result matrix:")
	printMatrix(resultMatrix)

	// now try to use a thread pool

	// myThreadPool := threadpool.NewThreadPool(4, 100)

	// // the number of tasks that will be executed by the thread pool
	// numberOfTasks := 3
	// start := time.Now()
	// calculateUsingThreadPool(matrix1, matrix2, resultMatrix, numberOfTasks, myThreadPool)
	// myThreadPool.Wait()
	// myThreadPool.Close()
	// elapsed := time.Since(start)
	// log.Printf("Calculation took %s", elapsed)
	// fmt.Println("Result matrix:")
	// printMatrix(resultMatrix)

	// 1000x800 matrix
	// bigMatrix1, err := readMatrixFromCSVFile("big_matrix1.csv")
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// //800x600 matrix
	// bigMatrix2, err := readMatrixFromCSVFile("big_matrix2.csv")
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	// bigResultMatrix := make([][]float64, 1000)
	// for i := range bigResultMatrix {
	// 	bigResultMatrix[i] = make([]float64, 600)
	// }

	// numberOfGoroutines := 4
	// var wg sync.WaitGroup
	// start := time.Now()
	// for i := 0; i < numberOfGoroutines; i++ {
	// 	wg.Add(1)
	// 	go func(i int) {
	// 		defer wg.Done()
	// 		calculateElementsOfResultMatrix(bigMatrix1, bigMatrix2, resultMatrixBig, i, numberOfGoroutines)
	// 	}(i)
	// }
	// wg.Wait()
	// elapsed := time.Since(start)
	// log.Printf("Calculation took %s", elapsed)

	// myThreadPool := threadpool.NewThreadPool(10, 100)
	// start := time.Now()
	// calculateUsingThreadPool(bigMatrix1, bigMatrix2, bigResultMatrix, 20, myThreadPool)
	// myThreadPool.Wait()
	// myThreadPool.Close()
	// elapsed := time.Since(start)
	// log.Printf("Calculation took %s", elapsed)

}

func readMatrixFromCSVFile(fileName string) ([][]float64, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read records: %w", err)
	}
	matrix := make([][]float64, len(records))

	for i, record := range records {
		matrix[i] = make([]float64, len(record))
		for j, value := range record {
			floatValue, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to convert value %s to float64: %w", value, err)
			}
			matrix[i][j] = floatValue
		}
	}

	return matrix, nil
}
