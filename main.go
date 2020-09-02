package main

import (
        "bufio"
        "crypto/tls"
        "flag"
        "fmt"
        "io"
        "log"
        "net/http"
        "os"
        "strings"
        "sync"
        "time"

        "github.com/PuerkitoBio/goquery"
        "golang.org/x/text/encoding/simplifiedchinese"
        "github.com/saintfish/chardet"
)

func main() {
        var targetFile string
        flag.StringVar(&targetFile, "f", "target.txt", "target file")
        var outputFile string
        flag.StringVar(&outputFile, "w", "title_output.txt", "output file")
        var timeout int
        flag.IntVar(&timeout,"t",10,"timeout(unit: second)")
        flag.Parse()
        // fmt.Println(targetFile)
        if !checkFileIsExist(targetFile) {
                fmt.Println("target file is not exist!")
                return
        }
        resultChan := make(chan string)
        var scanWg sync.WaitGroup
        defer close(resultChan)
        // 打开文件
        file, err := os.OpenFile(targetFile, os.O_RDONLY, 0755)
        if err != nil {
                log.Println(err)
        }
        defer file.Close()
        // 按行读取文件
        buf := bufio.NewReader(file)
        for {
                line, err := buf.ReadString('\n')
                line = strings.TrimSpace(line)
                if line != "" {
                        // fmt.Println(line)
                        scanWg.Add(1)
                        go getTitle(line, resultChan, &scanWg, timeout)

                }

                if err != nil {
                        if err == io.EOF {
                                break
                        } else {
                                log.Println(err)
                        }

                }

        }
        scanWg.Wait()
        writeToFile(outputFile,resultChan)
}

func getTitle(url string, resultChan chan string, scanWg *sync.WaitGroup,t int) {
        timeout := time.Duration(t) * time.Second
        tr := &http.Transport{
                TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
        }
        client := &http.Client{
                Transport: tr,
                Timeout:   timeout,
        }
        request, err := http.NewRequest("GET", url, nil)
        if err != nil {
                log.Fatal(err)
        }
        // 设置请求头
        request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.89 Safari/537.36")
        request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
        response, err := client.Do(request)
        if err != nil {
                scanWg.Done()
                return
        }
        defer response.Body.Close()
        statusCode := response.StatusCode
        doc, err := goquery.NewDocumentFromReader(response.Body)
        if err != nil {
                // log.Println(err)
                scanWg.Done()
                return
        }
        title := doc.Find("title").Text()
        var output string
        // 判断title是否为gbk编码
        detector := chardet.NewTextDetector()
        charset, _ := detector.DetectBest([]byte(title))

        if charset.Charset=="UTF-8" {
                output = fmt.Sprintf("%s\t%d\t%s", url, statusCode, title)

        } else {
                gbkTitle, _ := simplifiedchinese.GBK.NewDecoder().Bytes([]byte(title))
                output = fmt.Sprintf("%s\t%d\t%s", url, statusCode, string(gbkTitle))
        }
        // if isGBK([]byte(title)) {
        //      gbkTitle, _ := simplifiedchinese.GBK.NewDecoder().Bytes([]byte(title))
        //      output = fmt.Sprintf("%s\t%d\t%s", url, statusCode, string(gbkTitle))
        // } else {
        //      output = fmt.Sprintf("%s\t%d\t%s", url, statusCode, title)
        // }
        fmt.Println(output)
        scanWg.Done()
        resultChan <- output
}

// func isGBK(data []byte) bool {
//      length := len(data)
//      var i int = 0
//      for i < length {
//              //fmt.Printf("for %x\n", data[i])
//              if data[i] <= 0xff {
//                      //编码小于等于127,只有一个字节的编码，兼容ASCII吗
//                      i++
//                      continue
//              } else {
//                      //大于127的使用双字节编码
//                      if data[i] >= 0x81 &&
//                              data[i] <= 0xfe &&
//                              data[i+1] >= 0x40 &&
//                              data[i+1] <= 0xfe &&
//                              data[i+1] != 0xf7 {
//                              i += 2
//                              continue
//                      } else {
//                              return false
//                      }
//              }
//      }
//      return true
// }

// 写文件
func writeToFile(outputFile string, resultChan <-chan string) {
        var f *os.File
        var err error
        if checkFileIsExist(outputFile) {
                f, err = os.OpenFile(outputFile, os.O_WRONLY, 0666)
        } else {
                f, err = os.Create(outputFile)
        }
        defer f.Close()
        if err != nil {
                log.Fatal(err)
        }
        w := bufio.NewWriter(f)
        for {
                select{
                case res,ok:=<-resultChan:
                        if ok{
                                _, _ = w.WriteString(res + "\n")

                        }
                default:
                        w.Flush()
                        return
                }
        }


}

// 判断文件是否存在
func checkFileIsExist(file string) bool {
        if _, err := os.Stat(file); os.IsNotExist(err) {
                return false
        }
        return true
}
