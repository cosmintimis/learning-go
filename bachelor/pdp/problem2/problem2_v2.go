package main

import (
	"fmt"
	"math"
	"sync"
)

type DataChannel struct {
	data     []float64
	capacity int
}

type SharedData struct {
	channel DataChannel
	condVar *sync.Cond
}

// consumer_v2
func consumer_v2(sharedData *SharedData, resultChannel chan<- float64) {
	sum := 0.0
	finish := false
	for {
		// wait for the producer to signal that the data is ready
		sharedData.condVar.L.Lock()
		for len(sharedData.channel.data) == 0 {
			sharedData.condVar.Wait()
		}
		fmt.Println("Producer signaled that data is ready")

		// Process data
		for _, value := range sharedData.channel.data {
			if math.IsNaN(value) {
				resultChannel <- sum
				sharedData.condVar.L.Unlock()
				finish = true
				break
			}
			sum += value
		}
		// clear the data from the channel
		sharedData.channel.data = []float64{}

		// if the producer has finished, break the loop
		if finish {
			break
		}
		// signal the producer that the data was read
		sharedData.condVar.Signal()
		sharedData.condVar.L.Unlock()
	}
}

// producer_v2
func producer_v2(vector1 *[]float64, vector2 *[]float64, sharedData *SharedData) {
	for i := 0; i < len(*vector1); i++ {
		valueToSend := ((*vector1)[i]) * ((*vector2)[i])
		fmt.Println("Sending ", valueToSend)
		// acquire the lock and write the data, then signal the consumer that the data is ready
		sharedData.condVar.L.Lock()
		// if the channel is full, signal the consumer to read the data and wait for the consumer to signal that the data was read
		for len(sharedData.channel.data) == sharedData.channel.capacity {
			sharedData.condVar.Wait()
		}

		sharedData.channel.data = append(sharedData.channel.data, valueToSend)
		fmt.Println("Sent:", valueToSend)
		sharedData.condVar.Signal()
		sharedData.condVar.L.Unlock()
	}

	// signal the consumer that the producer has finished by sending math.NaN()
	sharedData.condVar.L.Lock()
	// check again if the channel is full
	for len(sharedData.channel.data) == sharedData.channel.capacity {
		sharedData.condVar.Wait()
	}
	sharedData.channel.data = append(sharedData.channel.data, math.NaN())
	sharedData.condVar.Signal()
	sharedData.condVar.L.Unlock()
}

func problem2_v2(vector1 *[]float64, vector2 *[]float64) {
	if len(*vector1) != len(*vector2) {
		fmt.Println("Vectors have different sizes, exiting...")
		return
	}

	// create a shared data structure
	sharedData := SharedData{
		channel: DataChannel{
			data:     []float64{},
			capacity: 2,
		},
		condVar: sync.NewCond(&sync.Mutex{}),
	}
	// create a channel for sending the result from the consumer to the main thread
	resultChannel := make(chan float64)

	// start consumer thread
	go consumer_v2(&sharedData, resultChannel)

	// start producer thread
	go producer_v2(vector1, vector2, &sharedData)

	// wait for the result from the consumer
	result := <-resultChannel
	fmt.Println("Result: ", result)
	close(resultChannel)
}
