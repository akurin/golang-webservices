package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	prevJobOut := make(chan interface{})
	for _, j := range jobs {
		out := make(chan interface{})
		go runAndCloseChan(j, prevJobOut, out)
		prevJobOut = out
	}
	_ = <-prevJobOut
}

func runAndCloseChan(j job, in, out chan interface{}) {
	j(in, out)
	close(out)
}

func SingleHash(in, out chan interface{}) {
	wg := sync.WaitGroup{}
	md5Mutex := sync.Mutex{}
	for data := range in {
		wg.Add(1)
		go func(data2 interface{}) {
			dataInt := data2.(int)
			dataString := strconv.Itoa(dataInt)

			crc32Chan := make(chan string)
			go func() {
				crc32Chan <- DataSignerCrc32(dataString)
			}()

			crc32md5Chan := make(chan string)
			go func() {
				md5Mutex.Lock()
				md5 := DataSignerMd5(dataString)
				md5Mutex.Unlock()

				crc32md5Chan <- DataSignerCrc32(md5)
			}()

			out <- <-crc32Chan + "~" + <-crc32md5Chan
			wg.Done()
		}(data)
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	wg := sync.WaitGroup{}
	for data := range in {
		wg.Add(1)
		go func(data2 interface{}) {
			dataString := data2.(string)

			var resultChans []chan string
			const hashCount = 6

			for th := 0; th < hashCount; th++ {
				crc32Chan := make(chan string)
				go func(th int) {
					crc32Chan <- DataSignerCrc32(strconv.Itoa(th) + dataString)
				}(th)
				resultChans = append(resultChans, crc32Chan)
			}

			var result string
			for th := 0; th < hashCount; th++ {
				result = result + <-resultChans[th]
			}
			out <- result
			wg.Done()
		}(data)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	var results []string
	for result := range in {
		resultString := result.(string)
		results = append(results, resultString)
	}
	sort.Strings(results)
	joined := strings.Join(results, "_")
	out <- joined
}

func main() {
	println("run as\n\ngo test -v -race")
}
