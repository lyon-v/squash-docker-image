package image

import (
	"bufio"
	"os"
)

func ReverseList(list []string) {
	// 定义两个指针，分别指向列表的起始和末尾
	left, right := 0, len(list)-1

	// 交换指针所指向位置的值，直到两个指针相遇
	for left < right {
		// 交换左右指针所指向位置的值
		list[left], list[right] = list[right], list[left]
		// 移动指针
		left++
		right--
	}
}

// 辅助函数：查找字符串在切片中的索引位置
func FindIndex(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

// ReadFileLines 读取文件的每一行并返回一个字符串数组
func ReadFileLines(filePath string) ([]string, error) {

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 如果文件不存在，返回空切片和nil错误
			return []string{}, nil
		}
		// 其他错误，返回错误信息
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}
