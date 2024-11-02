package main

import (
	"fmt"
)

func main() {
	vector1 := []float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	vector2 := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// if vectors dont have same size, padding with 0
	if len(vector1) != len(vector2) {
		if len(vector1) > len(vector2) {
			for i := len(vector2); i < len(vector1); i++ {
				vector2 = append(vector2, 0)
			}
		} else {
			for i := len(vector1); i < len(vector2); i++ {
				vector1 = append(vector1, 0)
			}
		}
	}
	// print our vectors
	fmt.Println("Vector 1: ", vector1)
	fmt.Println("Vector 2: ", vector2)

	// use the lab2_v1 function to calculate the result
	//lab2_v1(&vector1, &vector2)

	// use the lab2_v2 function to calculate the result
	problem2_v2(&vector1, &vector2)
}
