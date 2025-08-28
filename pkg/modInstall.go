package pkg

import (
	"crypto/sha1"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type FileEntry struct {
	Path   string `json:"path"`
	Hashes struct {
		SHA1   string `json:"sha1"`
		SHA512 string `json:"sha512"`
	} `json:"hashes"`
	Downloads []string `json:"downloads"`
}

type ModList struct {
	Files []FileEntry `json:"files"`
	Name  string      `json:"name"`
}

// 下载文件
func downloadFile(url string, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	err = os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			log.Println(err)
		}
	}(out)

	_, err = io.Copy(out, resp.Body)
	return err
}

// 计算 SHA1
func sha1sum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Println(err)
		}
	}(f)

	h := sha1.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// 计算 SHA512
func sha512sum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Println(err)
		}
	}(f)

	h := sha512.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// worker 是一个工作协程，负责处理单个文件的下载和校验任务
func worker(id int, tasks <-chan FileEntry, wg *sync.WaitGroup, outDir string) {
	defer wg.Done()
	for f := range tasks {
		success := false
		newPath := outDir + "/" + f.Path

		if _, err := os.Stat(newPath); err == nil {
			log.Printf("发现位于%s的文件存在", newPath)
			sha1Val, _ := sha1sum(newPath)
			sha512Val, _ := sha512sum(newPath)
			if sha1Val == f.Hashes.SHA1 && sha512Val == f.Hashes.SHA512 {
				log.Printf("Worker %d: 校验通过: %s\n", id, newPath)
				continue
			}
		}

		for _, url := range f.Downloads {
			log.Printf("Worker %d: 尝试下载: %s\n", id, url)
			if err := downloadFile(url, newPath); err != nil {
				log.Printf("Worker %d: 下载失败: %v\n", id, err)
				continue
			}

			sha1Val, _ := sha1sum(newPath)
			sha512Val, _ := sha512sum(newPath)

			if sha1Val == f.Hashes.SHA1 && sha512Val == f.Hashes.SHA512 {
				log.Printf("Worker %d: 校验通过: %s\n", id, newPath)
				success = true
				break
			} else {
				log.Printf("Worker %d: 校验失败，尝试下一个 URL\n", id)
			}
		}
		if !success {
			log.Printf("Worker %d: 下载或校验失败: %s\n", id, newPath)
		}
	}
}

func Install(modFilePath string, outDir string) (string, error) {
	file, err := os.Open(modFilePath)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Println(err)
		}
	}(file)

	var modList ModList
	if err := json.NewDecoder(file).Decode(&modList); err != nil {
		return "", err
	}
	outDir = outDir + "/" + modList.Name
	// 确定并发数
	numWorkers := 8
	tasks := make(chan FileEntry, len(modList.Files))
	var wg sync.WaitGroup

	// 启动工作协程
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, tasks, &wg, outDir)
	}

	// 将所有任务发送到通道
	for _, f := range modList.Files {
		tasks <- f
	}

	// 关闭通道，告诉工作协程没有更多任务了
	close(tasks)

	// 等待所有任务完成
	wg.Wait()

	return outDir, nil
}
