package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

func extractPayloadBin(filename string) string {
	zipReader, err := zip.OpenReader(filename)
	if err != nil {
		log.Fatalf("不是有效的zip压缩包: %s\n", filename)
	}
	defer zipReader.Close()

	for _, file := range zipReader.Reader.File {
		if file.Name == "payload.bin" && file.UncompressedSize64 > 0 {
			zippedFile, err := file.Open()
			if err != nil {
				log.Fatalf("未能读取压缩文件: %s\n", file.Name)
			}

			tempfile, err := os.CreateTemp(os.TempDir(), "payload_*.bin")
			if err != nil {
				log.Fatalf("未能于: %s 创建临时文件\n", tempfile.Name())
			}
			defer tempfile.Close()

			_, err = io.Copy(tempfile, zippedFile)
			if err != nil {
				log.Fatal(err)
			}

			return tempfile.Name()
		}
	}

	return ""
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var (
		list            bool
		partitions      string
		outputDirectory string
		concurrency     int
	)

	flag.IntVar(&concurrency, "c", 4, "提取线程数量 (缩写)")
	flag.IntVar(&concurrency, "concurrency", 4, "提取线程数量")
	flag.BoolVar(&list, "l", false, "显示在 payload.bin 中的分区的列表 (缩写)")
	flag.BoolVar(&list, "list", false, "显示在 payload.bin 中的分区的列表")
	flag.StringVar(&outputDirectory, "o", "", "设置输出目录 (缩写)")
	flag.StringVar(&outputDirectory, "output", "", "设置输出目录")
	flag.StringVar(&partitions, "p", "", "只提取选择的分区 (用逗号分隔) (缩写)")
	flag.StringVar(&partitions, "partitions", "", "只提取选择的分区 (用逗号分隔)")
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
	}
	filename := flag.Arg(0)

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		log.Fatalf("文件不存在!: %s\n", filename)
	}

	payloadBin := filename
	if strings.HasSuffix(filename, ".zip") {
		fmt.Println("请稍候, 正在从压缩文件中提取 payload.bin")
		payloadBin = extractPayloadBin(filename)
		if payloadBin == "" {
			log.Fatal("从压缩文件中提取 payload.bin 失败")
		} else {
			defer os.Remove(payloadBin)
		}
	}
	fmt.Printf("payload.bin: %s\n", payloadBin)

	payload := NewPayload(payloadBin)
	if err := payload.Open(); err != nil {
		log.Fatal(err)
	}
	payload.Init()

	if list {
		return
	}

	now := time.Now()

	targetDirectory := outputDirectory
	if targetDirectory == "" {
		targetDirectory = fmt.Sprintf("已提取的分区_%d%02d%02d_%02d%02d%02d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	}
	if _, err := os.Stat(targetDirectory); os.IsNotExist(err) {
		if err := os.Mkdir(targetDirectory, 0o755); err != nil {
			log.Fatal("未能创建目标目录")
		}
	}

	payload.SetConcurrency(concurrency)
	fmt.Printf("提取线程数量: %d\n", payload.GetConcurrency())

	if partitions != "" {
		if err := payload.ExtractSelected(targetDirectory, strings.Split(partitions, ",")); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := payload.ExtractAll(targetDirectory); err != nil {
			log.Fatal(err)
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "使用: %s [选项] [输入文件]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}
