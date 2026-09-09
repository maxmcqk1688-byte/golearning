package homework01

import "sort"

// 1. 只出现一次的数字
// 给定一个非空整数数组，除了某个元素只出现一次以外，其余每个元素均出现两次。找出那个只出现了一次的元素。
func SingleNumber(nums []int) int {
	// TODO: implement
	single := 0
	for _, num := range nums {
		single ^= num
	}
	return single

}

// 2. 回文数
// 判断一个整数是否是回文数
func IsPalindrome(x int) bool {
	// TODO: implement
	// 负数不是回文数；非零末位为 0 的数也不是回文数
	if x < 0 || (x%10 == 0 && x != 0) {
		return false
	}

	revertedNumber := 0
	// 反转数字的后半部分，直到 x 小于等于 revertedNumber
	for x > revertedNumber {
		revertedNumber = revertedNumber*10 + x%10
		x /= 10
	}

	// 当数字长度为奇数时，可以通过 revertedNumber/10 去掉中间的数字
	return x == revertedNumber || x == revertedNumber/10
}

// 3. 有效的括号
// 给定一个只包括 '(', ')', '{', '}', '[', ']' 的字符串，判断字符串是否有效
func IsValid(s string) bool {
	// TODO: implement
	if len(s)%2 != 0 {
		return false
	}

	// 栈用于存放期望出现的右括号
	stack := make([]rune, 0, len(s))

	for _, ch := range s {
		switch ch {
		case '(':
			stack = append(stack, ')')
		case '[':
			stack = append(stack, ']')
		case '{':
			stack = append(stack, '}')
		default:
			// 如果是右括号，但栈为空，或者与栈顶期望的不匹配
			if len(stack) == 0 || stack[len(stack)-1] != ch {
				return false
			}
			// 匹配成功，弹出栈顶元素
			stack = stack[:len(stack)-1]
		}
	}

	// 最终栈为空则说明全部正确匹配
	return len(stack) == 0

}

// 4. 最长公共前缀
// 查找字符串数组中的最长公共前缀
func LongestCommonPrefix(strs []string) string {
	// TODO: implement
	if len(strs) == 0 {
		return ""
	}

	// 以第一个字符串 strs[0] 的字符位置逐列遍历
	for i := 0; i < len(strs[0]); i++ {
		ch := strs[0][i]
		// 检查后续每个字符串在位置 i 上的字符
		for j := 1; j < len(strs); j++ {
			// 如果达到某个字符串的长度上限，或者字符不匹配
			if i == len(strs[j]) || strs[j][i] != ch {
				return strs[0][:i]
			}
		}
	}

	// 如果全部遍历完成，说明第一个字符串就是最长公共前缀
	return strs[0]

}

// 5. 加一
// 给定一个由整数组成的非空数组所表示的非负整数，在该数的基础上加一
func PlusOne(digits []int) []int {
	// TODO: implement
	n := len(digits)

	// 从末尾向开头遍历
	for i := n - 1; i >= 0; i-- {
		if digits[i] < 9 {
			digits[i]++
			return digits // 没有产生进位，直接返回
		}
		digits[i] = 0 // 当前位是 9，加 1 后变 0，继续循环处理高位
	}

	// 如果循环正常结束，说明原本所有位数都是 9（如 [9, 9, 9] -> [0, 0, 0]）
	// 需要在最前面插入一个 1
	result := make([]int, n+1)
	result[0] = 1
	return result

}

// 6. 删除有序数组中的重复项
// 给你一个有序数组 nums ，请你原地删除重复出现的元素，使每个元素只出现一次，返回删除后数组的新长度。
// 不要使用额外的数组空间，你必须在原地修改输入数组并在使用 O(1) 额外空间的条件下完成。
func RemoveDuplicates(nums []int) int {
	// TODO: implement
	if len(nums) == 0 {
		return 0
	}

	slow := 0
	for fast := 1; fast < len(nums); fast++ {
		// 遇到不重复的新元素时，写入 slow 的下一个位置
		if nums[fast] != nums[slow] {
			slow++
			nums[slow] = nums[fast]
		}
	}

	// 新数组长度为 slow 索引 + 1
	return slow + 1

}

// 7. 合并区间
// 以数组 intervals 表示若干个区间的集合，其中单个区间为 intervals[i] = [starti, endi] 。
// 请你合并所有重叠的区间，并返回一个不重叠的区间数组，该数组需恰好覆盖输入中的所有区间。
func Merge(intervals [][]int) [][]int {
	// TODO: implement
	if len(intervals) <= 1 {
		return intervals
	}

	// 1. 按区间的左端点升序排序
	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i][0] < intervals[j][0]
	})

	res := [][]int{intervals[0]}

	// 2. 遍历后续区间进行合并
	for i := 1; i < len(intervals); i++ {
		last := res[len(res)-1]
		curr := intervals[i]

		// 判断是否有重叠
		if curr[0] <= last[1] {
			// 更新重叠区间的右端点为两者的较大值
			if curr[1] > last[1] {
				last[1] = curr[1]
			}
		} else {
			// 无重叠，将当前区间加入结果集中
			res = append(res, curr)
		}
	}

	return res

}

// 8. 两数之和
// 给定一个整数数组 nums 和一个目标值 target，请你在该数组中找出和为目标值的那两个整数
func TwoSum(nums []int, target int) []int {
	// TODO: implement
	// key: 数值, value: 对应的数组索引
	hashTable := make(map[int]int, len(nums))

	for i, num := range nums {
		complement := target - num
		// 检查补数是否存在于哈希表中
		if prevIdx, ok := hashTable[complement]; ok {
			return []int{prevIdx, i}
		}
		// 将当前数值与索引存入哈希表
		hashTable[num] = i
	}

	return nil

}
