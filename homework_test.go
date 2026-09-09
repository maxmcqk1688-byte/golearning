package homework01

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"sync"
	"testing"
)

var (
	failedQuestions []string
	totalQuestions  int
	mu              sync.Mutex
)

func recordResult(t *testing.T, name string) {
	mu.Lock()
	defer mu.Unlock()
	totalQuestions++
	if t.Failed() {
		failedQuestions = append(failedQuestions, name)
	}
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Print summary
	if totalQuestions > 0 {
		fmt.Println("\n---------------------------------------------------")
		fmt.Printf("Total Questions: %d\n", totalQuestions)
		fmt.Printf("Passed: %d\n", totalQuestions-len(failedQuestions))
		fmt.Printf("Failed: %d\n", len(failedQuestions))

		score := float64(totalQuestions-len(failedQuestions)) / float64(totalQuestions) * 100
		fmt.Printf("Score: %.2f%%\n", score)

		if len(failedQuestions) > 0 {
			fmt.Println("Failed Questions:")
			for _, q := range failedQuestions {
				fmt.Printf("- %s\n", q)
			}
		}
		fmt.Println("---------------------------------------------------")
	}

	os.Exit(code)
}

func TestSingleNumber(t *testing.T) {
	defer recordResult(t, "SingleNumber")
	tests := []struct {
		name string
		nums []int
		want int
	}{
		{"Example 1", []int{2, 2, 1}, 1},
		{"Example 2", []int{4, 1, 2, 1, 2}, 4},
		{"Example 3", []int{1}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SingleNumber(tt.nums); got != tt.want {
				t.Errorf("SingleNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPalindrome(t *testing.T) {
	defer recordResult(t, "IsPalindrome")
	tests := []struct {
		name string
		x    int
		want bool
	}{
		{"Example 1", 121, true},
		{"Example 2", -121, false},
		{"Example 3", 10, false},
		{"Example 4", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPalindrome(tt.x); got != tt.want {
				t.Errorf("IsPalindrome() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValid(t *testing.T) {
	defer recordResult(t, "IsValid")
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"Example 1", "()", true},
		{"Example 2", "()[]{}", true},
		{"Example 3", "(]", false},
		{"Example 4", "([)]", false},
		{"Example 5", "{[]}", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValid(tt.s); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLongestCommonPrefix(t *testing.T) {
	defer recordResult(t, "LongestCommonPrefix")
	tests := []struct {
		name string
		strs []string
		want string
	}{
		{"Example 1", []string{"flower", "flow", "flight"}, "fl"},
		{"Example 2", []string{"dog", "racecar", "car"}, ""},
		{"Example 3", []string{"a"}, "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LongestCommonPrefix(tt.strs); got != tt.want {
				t.Errorf("LongestCommonPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlusOne(t *testing.T) {
	defer recordResult(t, "PlusOne")
	tests := []struct {
		name   string
		digits []int
		want   []int
	}{
		{"Example 1", []int{1, 2, 3}, []int{1, 2, 4}},
		{"Example 2", []int{4, 3, 2, 1}, []int{4, 3, 2, 2}},
		{"Example 3", []int{0}, []int{1}},
		{"Example 4", []int{9}, []int{1, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PlusOne(tt.digits); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PlusOne() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveDuplicates(t *testing.T) {
	defer recordResult(t, "RemoveDuplicates")
	tests := []struct {
		name         string
		nums         []int
		want         int
		expectedNums []int
	}{
		{"Example 1", []int{1, 1, 2}, 2, []int{1, 2}},
		{"Example 2", []int{0, 0, 1, 1, 1, 2, 2, 3, 3, 4}, 5, []int{0, 1, 2, 3, 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy since modification is in-place
			numsCopy := make([]int, len(tt.nums))
			copy(numsCopy, tt.nums)
			got := RemoveDuplicates(numsCopy)
			if got != tt.want {
				t.Errorf("RemoveDuplicates() = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(numsCopy[:got], tt.expectedNums) {
				t.Errorf("RemoveDuplicates() array content = %v, want %v", numsCopy[:got], tt.expectedNums)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	defer recordResult(t, "Merge")
	tests := []struct {
		name      string
		intervals [][]int
		want      [][]int
	}{
		{"Example 1", [][]int{{1, 3}, {2, 6}, {8, 10}, {15, 18}}, [][]int{{1, 6}, {8, 10}, {15, 18}}},
		{"Example 2", [][]int{{1, 4}, {4, 5}}, [][]int{{1, 5}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Merge(tt.intervals)
			// Sort result to ensure order doesn't affect correctness if logic varies,
			// though standard merge intervals usually returns sorted.
			// Let's assume standard behavior: sorted by start time.
			sort.Slice(got, func(i, j int) bool {
				return got[i][0] < got[j][0]
			})
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Merge() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTwoSum(t *testing.T) {
	defer recordResult(t, "TwoSum")
	tests := []struct {
		name   string
		nums   []int
		target int
		want   []int
	}{
		{"Example 1", []int{2, 7, 11, 15}, 9, []int{0, 1}},
		{"Example 2", []int{3, 2, 4}, 6, []int{1, 2}},
		{"Example 3", []int{3, 3}, 6, []int{0, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TwoSum(tt.nums, tt.target)
			sort.Ints(got) // Index order doesn't strictly matter for validity unless specified, but TwoSum usually implies indices.
			// Standard TwoSum can return in any order, so we sort to compare.
			sort.Ints(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TwoSum() = %v, want %v", got, tt.want)
			}
		})
	}
}
