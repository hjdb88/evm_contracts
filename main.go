package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	urlPattern = "https://api.etherscan.io/api?module=contract&action=getsourcecode&address=%s"
)

type Contract struct {
	Status  string   `json:"status"`
	Message string   `json:"message"`
	Result  []Result `json:"result"`
}

type Result struct {
	SourceCode           string `json:"SourceCode"`
	ABI                  string `json:"ABI"`
	ContractName         string `json:"ContractName"`
	CompilerVersion      string `json:"CompilerVersion"`
	OptimizationUsed     string `json:"OptimizationUsed"`
	Runs                 string `json:"Runs"`
	ConstructorArguments string `json:"ConstructorArguments"`
	EVMVersion           string `json:"EVMVersion"`
	Library              string `json:"Library"`
	LicenseType          string `json:"LicenseType"`
	Proxy                string `json:"Proxy"`
	Implementation       string `json:"Implementation"`
	SwarmSource          string `json:"SwarmSource"`
}

type ContractSource struct {
	Language string            `json:"language"`
	Sources  map[string]Source `json:"sources"`
}

type Source struct {
	Content string `json:"content"`
}

func main() {
	start := time.Now()
	defer fmt.Printf("花费时间：%s", time.Since(start))

	address := "0xed5af388653567af2f388e6224dc7c4b3241c544"
	uri := fmt.Sprintf(urlPattern, address)

	proxy := func(_ *http.Request) (*url.URL, error) {
		return url.Parse("socks5://127.0.0.1:1080")
	}

	client := &http.Client{Transport: &http.Transport{Proxy: proxy}}

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		fmt.Println("NewRequest error", err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Do error", err)
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll error", err)
		return
	}

	if len(body) == 0 {
		return
	}

	var contract Contract
	json.Unmarshal(body, &contract)
	if contract.Status != "1" {
		fmt.Println("request error", string(body))
		return
	}

	if len(contract.Result) == 0 {
		return
	}

	for _, r := range contract.Result {
		if len(r.SourceCode) == 0 {
			continue
		}

		sourceCode := r.SourceCode[1 : len(r.SourceCode)-1]

		var contractSource ContractSource
		err = json.Unmarshal([]byte(sourceCode), &contractSource)
		if err != nil {
			fmt.Println("Unmarshal error", err)
			continue
		}
		for k, v := range contractSource.Sources {
			fmt.Println(k)

			if strings.Contains(k, "/") {
				k = strings.ReplaceAll(k, "/", string(os.PathSeparator))
			}

			path := GetAppPath() + string(os.PathSeparator) + r.ContractName + string(os.PathSeparator) + k
			SaveToFile(path, v.Content)
		}
	}
}

func GetAppPath() string {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	index := strings.LastIndex(path, string(os.PathSeparator))

	return path[:index]
}

func SaveToFile(filePath, data string) {
	directory := filePath[:strings.LastIndex(filePath, string(os.PathSeparator))]
	err := os.MkdirAll(directory, 0766)
	if err != nil {
		fmt.Println("MkdirAll error", err)
		return
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Create file error: %v\n", err)
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	fmt.Fprintln(w, data)
	w.Flush()
}
