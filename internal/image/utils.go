package image

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// Helper function to Copy a file from src to dest
func CopyFile(src, dest string, records map[string]int) (int, error) {

	if _, ok := records[src]; ok {
		return 1, nil
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	destination, err := CreateFileWithDirs(dest)
	if err != nil {
		return 0, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return 0, fmt.Errorf("failed to Copy file content: %w", err)
	}

	records[src] += 1
	return 1, nil
}

// Helper function to create a file and its parent directories
func CreateFileWithDirs(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}
	return os.Create(path)
}

func PathExists(path string) bool {
	_, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func CreateSymlink(src, dest string) error {

	if PathExists(dest) {
		os.RemoveAll(dest)
	}

	// Ensure the destination directory exists
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create the symbolic link
	cmd := exec.Command("ln", "-s", src, dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create symbolic link: %w, output: %s", err, output)
	}

	return nil
}

func HasFiles(path string) (bool, error) {
	// Check if the path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, fmt.Errorf("directory does not exist: %s", path)
	}

	// Read the directory
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return false, err
	}

	// Check if directory is empty
	if len(files) == 0 {
		return false, nil
	}

	return true, nil
}

func NormalizePath(p string) string {

	// 规范化路径
	cleanPath := filepath.Clean(p)

	// 去除前缀 ./
	if strings.HasPrefix(cleanPath, "./") {
		cleanPath = strings.TrimPrefix(cleanPath, "./")
	}

	return cleanPath
}

// GetWhiteoutAndRegularFiles 遍历目录并分类 whiteout 和普通文件
func GetWhiteoutAndRegularFiles(root string) (whfiles, refiles []FileWithPath, err error) {
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasPrefix(info.Name(), ".wh.") {
			if strings.Contains(path, "home") {
				fmt.Println()
			}
			whfiles = append(whfiles, FileWithPath{Path: path, Info: info})
		} else {
			refiles = append(refiles, FileWithPath{Path: path, Info: info})
		}
		return nil
	})
	return whfiles, refiles, err
}

func ExtractTar(src, dest string) ([]byte, error) {
	cmd := exec.Command("tar", "--same-owner", "--xattrs", "--overwrite",
		"--preserve-permissions", "-xf", src, "-C", dest)
	return cmd.CombinedOutput()
}

// CreateTar 打包目录为 tar 文件
func CreateTar(srcDir, tarFile string) error {
	cmd := exec.Command("tar", "-cf", tarFile, "-C", srcDir, ".")
	return cmd.Run()
}
