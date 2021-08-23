package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/gorilla/mux"
)

type page struct {
	parent int
	index  int
	words  []string
}

type query struct {
	index int
	words []string
}
type pair struct {
	index int
	val   float64
}

func parsePageQuery(lines string) ([]page, []query) {
	var pages []page
	var queries []query
	lineSlice := strings.Split(lines, "\n")
	var p page
	var q query
	var pIndex, qIndex, subpageIndex int
	for _, v := range lineSlice {
		line := strings.Split(v, " ")
		wt := len(line) - 1
		if maxWt < wt {
			maxWt = wt
		}
		if line[0] == "P" {
			p.words = line[1:]
			pIndex++
			subpageIndex = 0
			p.index = pIndex
			p.parent = 0
			pages = append(pages, p)

		} else if line[0] == "PP" {
			p.words = line[1:]
			p.parent = pIndex
			subpageIndex++
			p.index = subpageIndex
			pages = append(pages, p)

		} else if line[0] == "Q" {
			q.words = line[1:]
			qIndex++
			q.index = qIndex
			queries = append(queries, q)

		} else {
			fmt.Println("invalid line present")
		}
	}
	return pages, queries
}

func getMainPageCount(pages []page) int {
	var c int
	for _, v := range pages {
		if v.parent == 0 {
			c++
		}
	}
	return c
}

func prepareOutput(pageRank []pair, qNum int, outfile []string) {
	var max pair
	// fmt.Printf("Q%d :=", q_num)
	outfile[qNum-1] = fmt.Sprintf("Q%d :=", qNum)
	for i := 0; i < 5 && i < len(pageRank); i++ {
		max.val = 0
		for ind, v := range pageRank {
			if max.val < v.val {
				max.val = v.val
				max.index = ind
			}
		}
		pageRank[max.index].val = 0
		if max.val > 0 {
			// fmt.Printf(" P%d", max.index+1)
			outfile[qNum-1] = outfile[qNum-1] + fmt.Sprintf(" P%d", pageRank[max.index].index)
		} else {
			break
		}
	}
	// fmt.Println("")

}

func calcEachQuery(qNum int, query query, pCount int, pages []page, outfile []string, wg *sync.WaitGroup) {
	var tempWt float64
	pageRank := make([]pair, pCount)
	var pageNum = -1
	for _, page := range pages {
		if page.parent == 0 {
			tempWt = 0
			pageNum++
		}
		for i, queryWord := range query.words {
			for j, pageWord := range page.words {
				if queryWord == pageWord {
					if page.parent == 0 {
						tempWt += (float64(maxWt) - float64(i)) * (float64(maxWt) - float64(j))
						break
					} else {
						tempWt += (float64(maxWt) - float64(i)) * (float64(maxWt) - float64(j)) * .1
					}
				}
			}
		}
		pageRank[pageNum] = pair{pageNum + 1, tempWt}
	}
	// fmt.Println(page_rank)
	// fmt.Println(max_wt)
	prepareOutput(pageRank, qNum+1, outfile)
	//fmt.Println("from brute: ", page_rank)
	wg.Done()
}

func calcStrength(outfile []string, pages []page, queries []query) {

	// fmt.Println(max_wt)
	var wg sync.WaitGroup
	pCount := getMainPageCount(pages)
	for qNum, query := range queries {
		wg.Add(1)
		go calcEachQuery(qNum, query, pCount, pages, outfile, &wg)

	}

	wg.Wait()
}

//////////////////////////////////////////////////////////////////

func ceateRefTable(pages []page) (map[string][]pair, map[string][]pair) {
	mainRefTable := make(map[string][]pair)
	subRefTable := make(map[string][]pair)
	for _, v := range pages {
		for j, word := range v.words {
			if v.parent == 0 {
				res := mainRefTable[word]
				res = append(res, pair{v.index, float64(maxWt) - float64(j)})
				mainRefTable[word] = res
			} else {
				res := subRefTable[word]
				res = append(res, pair{v.parent, float64(maxWt) - float64(j)})
				subRefTable[word] = res
			}

		}
	}
	return mainRefTable, subRefTable
}

func getPairSlice(pageRank map[int]float64) []pair {
	keys := make([]int, 0)
	for k := range pageRank {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var p []pair
	for _, v := range keys {
		p = append(p, pair{v, pageRank[v]})
	}
	return p
}

func calPwtRefTable(mainRefTable map[string][]pair, subRefTable map[string][]pair, qNum int, query query, outFile []string, wg1 *sync.WaitGroup) {
	pageRank := make(map[int]float64)
	for _, word := range query.words {
		pageArray := mainRefTable[word]
		for _, val := range pageArray {
			pageRank[val.index] += float64(maxWt) * val.val
		}
		subPageArray := subRefTable[word]
		for _, val := range subPageArray {
			pageRank[val.index] += 0.1 * (float64(maxWt) * val.val)
		}

	}
	// fmt.Println("Query num:= ", q_num+1)
	//fmt.Println(page_rank)
	forOut := getPairSlice(pageRank)
	fmt.Println(qNum+1, forOut)
	prepareOutput(forOut, qNum+1, outFile)
	wg1.Done()
}
func calcStrengthRefTable(mainRefTable map[string][]pair, subRefTable map[string][]pair, queries []query, outFile []string) {

	var wg1 sync.WaitGroup
	for qNum, query := range queries {
		wg1.Add(1)
		go calPwtRefTable(mainRefTable, subRefTable, qNum, query, outFile, &wg1)
	}
	wg1.Wait()
}

////////////////////////////////////////////////////

var maxWt int

func startProcess(w http.ResponseWriter, r *http.Request) {
	var pages []page
	var queries []query

	content, err := ioutil.ReadAll(r.Body)
	if err == nil {
		lines := string(content)
		pages, queries = parsePageQuery(lines)
	}

	var outfile = make([]string, len(queries))
	var outfile1 = make([]string, len(queries))

	//page weight calculation using bruteforce
	calcStrength(outfile, pages, queries)

	//creating map of keywords and associated pages
	main, sub := ceateRefTable(pages)
	calcStrengthRefTable(main, sub, queries, outfile1)

	for _, v := range outfile {
		fmt.Fprintf(w, v)
		fmt.Println(v)
		fmt.Fprintf(w, "\n")
	}
	fmt.Fprintf(w, "\n\n")
	fmt.Println("reftable")
	for _, v := range outfile1 {
		fmt.Fprintf(w, v)
		fmt.Println(v)
		fmt.Fprintf(w, "\n")
	}

}

func main() {
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/page", startProcess).Methods("POST")
	log.Fatal(http.ListenAndServe(":8000", r))

}
