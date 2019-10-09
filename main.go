package main

import (
	"bufio"
	"flag"
	"os"
	"os/user"
	"path/filepath"

	"github.com/vqhuy/kindle-voz/voz"
)

var inputPtr = flag.String("urls", "urls.txt", "URLs")
var namePtr = flag.String("name", "From F17 with Love", "Thread's Subject")

func readUrls(input string) ([]string, error) {
	file, err := os.Open(input)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var urls []string
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}

	return urls, nil
}

var configDir string

func init() {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	configDir = filepath.Join(usr.HomeDir, ".config", "kindle-voz")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()

	v, err := voz.New(*namePtr)
	defer v.Remove()
	if err != nil {
		panic(err)
	}

	urls, _ := readUrls(*inputPtr)
	mail, err := restoreMailSettings()
	if err != nil {
		panic(err)
	}

	out, err := v.Run(urls)
	if err != nil {
		panic(err)
	}
	if err = sendToKindle(mail, out); err != nil {
		panic(err)
	}
}
