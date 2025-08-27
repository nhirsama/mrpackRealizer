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
}

// 下载文件
func downloadFile(url string, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	os.MkdirAll(filepath.Dir(path), 0755)

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// 计算 SHA1
func sha1sum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

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
	defer f.Close()

	h := sha512.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func Install(modFilePath string, outDir string) error {

	file, err := os.Open(modFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var modList ModList
	if err := json.NewDecoder(file).Decode(&modList); err != nil {
		return err
	}

	for _, f := range modList.Files {
		success := false
		newPath := outDir + "/" + f.Path
		for _, url := range f.Downloads {
			log.Println("尝试下载:", url)
			if err := downloadFile(url, newPath); err != nil {
				log.Println("下载失败:", err)
				continue
			}

			sha1Val, _ := sha1sum(newPath)
			sha512Val, _ := sha512sum(newPath)

			if sha1Val == f.Hashes.SHA1 && sha512Val == f.Hashes.SHA512 {
				log.Println("校验通过:", newPath)
				success = true
				break
			} else {
				log.Println("校验失败，尝试下一个 URL")
			}
		}
		if !success {
			log.Println("下载或校验失败:", newPath)
		}
	}
	return nil
}
