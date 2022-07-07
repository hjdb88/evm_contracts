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
	"runtime"
	"strings"
	"time"
)

var (
	urlPattern = "https://api.etherscan.io/api?module=contract&action=getsourcecode&address=%s"

	proxyUrl = "socks5://127.0.0.1:10080"
)

func main() {
	addresses := map[string]string{
		"The Otherside":         "0x34d85c9CDeB23FA97cb08333b511ac86E1C4E258",
		"Bored Ape Yacht Club":  "0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D",
		"Mutant Ape Yacht Club": "0x60E4d786628Fea6478F785A6d7e704777c86a7c6",
		"Clone X":               "0x49cF6f5d44E70224e2E23fDcdd2C053F30aDA28B",
		"Moonbirds":             "0x23581767a106ae21c074b2276D25e5C3e136a68b",
		"WizNFT":                "0xe5E771bC685c5a89710131919C616c361ff001c6",
		"Azuki":                 "0xED5AF388653567Af2F388E6224dC7C4b3241C544",
		"Doodles":               "0x8a90CAb2b38dba80c64b7734e58Ee1dB38B8992e",
		"The Sandbox":           "0x5CC5B05a8A13E3fBDB0BB9FcCd98D38e50F90c38",
		"Admin One":             "0xD2A077Ec359D94E0A0b7E84435eaCB40A67a817c",
		"Parallel":              "0x76BE3b62873462d2142405439777e971754E8E77",
		"Moonrunners":           "0x1485297e942ce64E0870EcE60179dFda34b4C625",
		"BEANZ Official":        "0x306b1ea3ecdf94aB739F1910bbda052Ed4A9f949",
		"OnChainMonkey":         "0x86CC280D0BAC0BD4Ea38ba7d31e895Aa20Cceb4b",
		"Meebits":               "0x7Bd29408f11D2bFC23c34f18275bBf23bB716Bc7",
		"Bored":                 "0xba30E5F9Bb24caa003E9f2f0497Ad287FDF95623",
		"3Landers":              "0xb4d06d46A8285F4EC79Fd294F78a881799d8cEd9",
	}

	i := 1
	for folder, address := range addresses {
		fmt.Println()
		fmt.Printf("execute: %03d\n", i)
		downloadContractSourceCode(folder, address)
		i = i + 1
		time.Sleep(time.Duration(4) * time.Second)
	}
}

func downloadContractSourceCode(folder, address string) {
	defer func() {
		if e := recover(); nil != e {
			var buf [4096]byte
			n := runtime.Stack(buf[:], false)
			stack := string(buf[:n])
			msg := fmt.Sprintf("PANIC RECOVERED: %v\n\t%s\n", e, stack)
			fmt.Println(msg)
		}
	}()

	start := time.Now()

	uri := fmt.Sprintf(urlPattern, address)

	proxy := func(_ *http.Request) (*url.URL, error) {
		return url.Parse(proxyUrl)
	}

	client := &http.Client{Transport: &http.Transport{Proxy: proxy}}

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		fmt.Println("NewRequest error", err)
		fmt.Printf("cost: %s\n", time.Since(start))
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Do error", err)
		fmt.Printf("cost: %s\n", time.Since(start))
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll error", err)
		fmt.Printf("cost: %s\n", time.Since(start))
		return
	}

	if len(body) == 0 {
		return
	}

	var contract Contract
	json.Unmarshal(body, &contract)
	if contract.Status != "1" {
		fmt.Println("request error", string(body))
		fmt.Printf("cost: %s\n", time.Since(start))
		return
	}

	if len(contract.Result) == 0 {
		fmt.Printf("cost: %s\n", time.Since(start))
		return
	}

	for _, r := range contract.Result {
		if len(r.SourceCode) == 0 {
			continue
		}

		if !strings.HasPrefix(r.SourceCode, "{") {
			// single file
			path := GetAppPath() + string(os.PathSeparator) + "contracts" + string(os.PathSeparator) + folder + string(os.PathSeparator) + r.ContractName + ".sol"
			SaveToFile(path, r.SourceCode)
		} else {
			// multi files
			sourceCode := r.SourceCode[1 : len(r.SourceCode)-1]
			var contractSource ContractSource
			err = json.Unmarshal([]byte(sourceCode), &contractSource)
			if err != nil {
				fmt.Println("Unmarshal error", err)
				continue
			}
			for k, v := range contractSource.Sources {
				if strings.Contains(k, "/") {
					k = strings.ReplaceAll(k, "/", string(os.PathSeparator))
				}

				path := GetAppPath() + string(os.PathSeparator) + "contracts" + string(os.PathSeparator) + folder + string(os.PathSeparator) + string(os.PathSeparator) + k
				SaveToFile(path, v.Content)
			}
		}
	}
	fmt.Printf("cost: %s\n", time.Since(start))
}

func GetAppPath() string {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	index := strings.LastIndex(path, string(os.PathSeparator))

	return path[:index]
}

func SaveToFile(filePath, data string) {
	directory := filePath[:strings.LastIndex(filePath, string(os.PathSeparator))]
	_, err := os.Stat(directory)
	if err != nil && os.IsNotExist(err) {
		err := os.MkdirAll(directory, 0766)
		if err != nil {
			fmt.Println("MkdirAll error", err)
		}
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
