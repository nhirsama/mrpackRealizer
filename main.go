package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nhirsama/mrpackRealizer/pkg"
)

var buildMode string

func unZip(packPath string, dest string) ([]string, error) {

	var filenames []string

	// 打开 zip 文件。
	r, err := zip.OpenReader(packPath)
	if err != nil {
		return filenames, err
	}
	defer func(r *zip.ReadCloser) {
		err := r.Close()
		if err != nil {

		}
	}(r)

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// 检查 ZipSlip 漏洞。
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", f.Name)
		}
		filenames = append(filenames, fpath)
		// 如果是目录，则创建并跳到下一个文件。
		if f.FileInfo().IsDir() {
			err := os.MkdirAll(fpath, os.ModePerm)
			if err != nil {
				return nil, err
			}
			continue
		}

		// 确保文件所在的父目录存在。
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		// 打开文件进行写入。
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		// 打开 zip 存档中的文件进行读取。
		rc, err := f.Open()
		if err != nil {
			err := outFile.Close()
			if err != nil {
				return nil, err
			}
			return filenames, err
		}

		// 将数据从存档复制到新文件。
		_, err = io.Copy(outFile, rc)

		// 关闭两个文件，并检查关闭时可能发生的写入错误。
		err = outFile.Close()
		if err != nil {
			return filenames, err
		}
		err = rc.Close()
		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}
func main() {
	var packPath, outPath string
	flag.StringVar(&packPath, "m", "./modpack.mrpack", "pcl整合包所在路径")
	flag.StringVar(&outPath, "o", "./", "实例化后所在路径")
	flag.StringVar(&buildMode, "debug", "release", "编译模式")
	flag.Parse()

	var tempUnzipDir string
	if filepath.Ext(packPath) == ".zip" {
		tempOutputDir, err := os.MkdirTemp("", ".unzipped_data")
		tempUnzipDir = tempOutputDir
		if err != nil {
			log.Printf("临时目录创建失败，错误信息：%s\n", err)
			tempOutputDir = "./.unzipped_data"
			err := os.Mkdir(tempOutputDir, os.ModePerm)
			if err != nil {
				log.Panicf("无法创建解压目录，错误信息：%s\n", err)
				return
			}
		}
		fileName, err := unZip(packPath, tempOutputDir)
		if err != nil {
			log.Panicf("解压失败，请检查路径、文件格式是否正确，错误信息：%s\n", err)
			return
		}
		log.Printf("已将位于 %s 的zip归档文件解压至 %s \n", packPath, tempOutputDir)
		for _, file := range fileName {
			if buildMode == "debug" {
				log.Println(file)
			}
			if filepath.Ext(file) == ".mrpack" {
				log.Printf("已将 %s 重定向为 %s\n", packPath, file)
				packPath = file
				break
			}
		}
	}
	defer func(path string) {
		if path == "" {
			return
		}
		err := os.RemoveAll(path)
		log.Printf("已将位于 %s 的临时文件删除\n", tempUnzipDir)
		if err != nil {
			log.Printf("位于%s的临时目录删除失败，请手动删除\n", path)
			log.Println(err)
		}
	}(tempUnzipDir)
	fileName, err := unZip(packPath, filepath.Join(tempUnzipDir, "unpack"))
	if err != nil {
		log.Fatalf("解包%s失败，错误信息：%s\n", packPath, err)
	}
	if buildMode == "debug" {
		log.Printf("解包后文件目录如下：")
		for _, file := range fileName {
			log.Println(file)
		}
	}

	outPath, err = pkg.Install(filepath.Join(tempUnzipDir, "unpack/modrinth.index.json"), outPath)
	if err != nil {
		log.Printf("下载 mod 清单失败，错误信息：%s\n", err)
	}
	log.Printf("正在复制overrides文件")
	err = pkg.CopyDir(filepath.Join(tempUnzipDir, "unpack/overrides"), filepath.Join(outPath))
	if err != nil {
		log.Printf("复制 %s 至 %s 失败，错误信息%s\n", filepath.Join(tempUnzipDir, "unpack/overrides"), filepath.Join(outPath))
	}
}
