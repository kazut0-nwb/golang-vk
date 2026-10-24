package main

import "fmt"

func twoSum(nums []int, target int) []int {
	for i := 0; i < len(nums); i++ {
		for j := i + 1; j < len(nums); j++ {
			if nums[i]+nums[j] == target {
				return []int{i, j}
			}
		}
	}
	return nil
}

func main() {
	fmt.Println("Test 1:", twoSum([]int{2, 7, 11, 15}, 9))
	fmt.Println("Test 2:", twoSum([]int{3, 2, 4}, 6))
	fmt.Println("Test 3:", twoSum([]int{3, 3}, 6))
}
