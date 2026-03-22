// version 1 of homework
package main

import "fmt"

// consumer_v1 thread that will consume the data from the channel until it is closed
func consumer_v1(dataChannel <-chan float64, resultChannel chan<- float64) {
	sum := 0.0
	for {
		value, ok := <-dataChannel
		fmt.Println("Received ", value)
		if !ok {
			fmt.Println("Channel closed")
			resultChannel <- sum
			break
		}
		sum += value
	}
}

// producer_v1 thread that will produce the data to the channel, basically multiply pairs of elements from the vectors and send them to the channel
func producer_v1(vector1 *[]float64, vector2 *[]float64, dataChannel chan<- float64) {
	defer close(dataChannel)
	for i := 0; i < len(*vector1); i++ {
		valueToSend := ((*vector1)[i]) * ((*vector2)[i])
		fmt.Println("Sending ", valueToSend)
		dataChannel <- valueToSend
	}
}

func problem2_v1(vector1 *[]float64, vector2 *[]float64) {
	if len(*vector1) != len(*vector2) {
		fmt.Println("Vectors have different sizes, exiting...")
		return
	}

	// create channel of a specific size for sending data between producer and consumer
	dataChannel := make(chan float64, 2)
	// create a channel for sending the result from the consumer to the main thread
	resultChannel := make(chan float64)

	// start consumer thread
	go consumer_v1(dataChannel, resultChannel)

	// start producer thread
	go producer_v1(vector1, vector2, dataChannel)

	// wait for the result from the consumer
	result := <-resultChannel
	fmt.Println("Result: ", result)
	close(resultChannel)
}
