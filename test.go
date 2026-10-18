package main

import (
	"fmt"
)

func sumOfSqua(nums []int, n int) int {
	lenth := len(nums)
	if lenth == 0 || n <= 0 {
		return 0
	}
	if n > lenth {
		n = lenth
	}

	chunkSize := (n + lenth - 1) / n
	resCh := make(chan int, n)
	worker := 0
	for i := 0; i < n; i++ {
		start := i * chunkSize
		if start > lenth {
			break
		}
		end := start + chunkSize
		if end > lenth {
			end = lenth
		}
		worker++
		go func(sub []int) {
			sum := 0
			for _, val := range sub {
				sum += val * val
			}
			resCh <- sum
		}(nums[start:end])
	}
	total := 0
	for i := 0; i < worker; i++ {
		total += <-resCh
	}
	return total
}

func main() {
	nums := []int{1, 2, 3, 4, 5}
	fmt.Println("结果:", sumOfSqua(nums, 2))
}
