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

func parse_page_query(lines string) ([]page, []query) {
	var pages []page
	var queries []query
	line_slice := strings.Split(lines, "\n")
	var p page
	var q query
	var p_index, q_index, subpage_index int
	for _, v := range line_slice {
		line := strings.Split(v, " ")
		wt := len(line) - 1
		if max_wt < wt {
			max_wt = wt
		}
		if line[0] == "P" {
			p.words = line[1:]
			p_index++
			subpage_index = 0
			p.index = p_index
			p.parent = 0
			pages = append(pages, p)

		} else if line[0] == "PP" {
			p.words = line[1:]
			p.parent = p_index
			subpage_index++
			p.index = subpage_index
			pages = append(pages, p)

		} else if line[0] == "Q" {
			q.words = line[1:]
			q_index++
			q.index = q_index
			queries = append(queries, q)

		} else {
			fmt.Println("invalid line present")
		}
	}
	return pages, queries
}

func get_main_page_count(pages []page) int {
	var c int
	for _, v := range pages {
		if v.parent == 0 {
			c++
		}
	}
	return c
}

func prepare_output(page_rank []pair, q_num int, outfile []string) {
	var max pair
	// fmt.Printf("Q%d :=", q_num)
	outfile[q_num-1] = fmt.Sprintf("Q%d :=", q_num)
	for i := 0; i < 5 && i < len(page_rank); i++ {
		max.val = 0
		for ind, v := range page_rank {
			if max.val < v.val {
				max.val = v.val
				max.index = ind
			}
		}
		page_rank[max.index].val = 0
		if max.val > 0 {
			// fmt.Printf(" P%d", max.index+1)
			outfile[q_num-1] = outfile[q_num-1] + fmt.Sprintf(" P%d", page_rank[max.index].index)
		} else {
			break
		}
	}
	// fmt.Println("")

}

func calc_each_query(q_num int, query query, p_count int, pages []page, outfile []string, wg *sync.WaitGroup) {
	var temp_wt float64
	page_rank := make([]pair, p_count)
	var page_num int = -1
	for _, page := range pages {
		if page.parent == 0 {
			temp_wt = 0
			page_num++
		}
		for i, query_word := range query.words {
			for j, page_word := range page.words {
				if query_word == page_word {
					if page.parent == 0 {
						temp_wt += (float64(max_wt) - float64(i)) * (float64(max_wt) - float64(j))
						break
					} else {
						temp_wt += (float64(max_wt) - float64(i)) * (float64(max_wt) - float64(j)) * .1
					}
				}
			}
		}
		page_rank[page_num] = pair{page_num + 1, temp_wt}
	}
	// fmt.Println(page_rank)
	// fmt.Println(max_wt)
	prepare_output(page_rank, q_num+1, outfile)
	//fmt.Println("from brute: ", page_rank)
	wg.Done()
}

func calc_strength(outfile []string, pages []page, queries []query) {

	// fmt.Println(max_wt)
	var wg sync.WaitGroup
	p_count := get_main_page_count(pages)
	for q_num, query := range queries {
		wg.Add(1)
		go calc_each_query(q_num, query, p_count, pages, outfile, &wg)

	}

	wg.Wait()
}

//////////////////////////////////////////////////////////////////

func ceate_ref_table(pages []page) (map[string][]pair, map[string][]pair) {
	main_ref_table := make(map[string][]pair)
	sub_ref_table := make(map[string][]pair)
	for _, v := range pages {
		for j, word := range v.words {
			if v.parent == 0 {
				res := main_ref_table[word]
				res = append(res, pair{v.index, float64(max_wt) - float64(j)})
				main_ref_table[word] = res
			} else {
				res := sub_ref_table[word]
				res = append(res, pair{v.parent, float64(max_wt) - float64(j)})
				sub_ref_table[word] = res
			}

		}
	}
	return main_ref_table, sub_ref_table
}

func get_pair_slice(page_rank map[int]float64) []pair {
	keys := make([]int, 0)
	for k, _ := range page_rank {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var p []pair
	for _, v := range keys {
		p = append(p, pair{v, page_rank[v]})
	}
	return p
}

func cal_pwt_ref_table(main_ref_table map[string][]pair, sub_ref_table map[string][]pair, q_num int, query query, out_file []string, wg1 *sync.WaitGroup) {
	page_rank := make(map[int]float64)
	for _, word := range query.words {
		page_array := main_ref_table[word]
		for _, val := range page_array {
			page_rank[val.index] += float64(max_wt) * val.val
		}
		sub_page_array := sub_ref_table[word]
		for _, val := range sub_page_array {
			page_rank[val.index] += (0.1 * (float64(max_wt) * val.val))
		}

	}
	// fmt.Println("Query num:= ", q_num+1)
	//fmt.Println(page_rank)
	for_out := get_pair_slice(page_rank)
	fmt.Println(q_num+1, for_out)
	prepare_output(for_out, q_num+1, out_file)
	wg1.Done()
}
func calc_strength_ref_table(main_ref_table map[string][]pair, sub_ref_table map[string][]pair, queries []query, out_file []string) {

	var wg1 sync.WaitGroup
	for q_num, query := range queries {
		wg1.Add(1)
		go cal_pwt_ref_table(main_ref_table, sub_ref_table, q_num, query, out_file, &wg1)
	}
	wg1.Wait()
}

////////////////////////////////////////////////////

var max_wt int

func start_process(w http.ResponseWriter, r *http.Request) {
	var pages []page
	var queries []query

	content, err := ioutil.ReadAll(r.Body)
	if err == nil {
		lines := string(content)
		pages, queries = parse_page_query(lines)
	}

	var outfile = make([]string, len(queries))
	var outfile1 = make([]string, len(queries))

	//pageweight calculaton using bruteforce
	calc_strength(outfile, pages, queries)

	//creating map of keywords and associated pages
	main, sub := ceate_ref_table(pages)
	calc_strength_ref_table(main, sub, queries, outfile1)

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
	r.HandleFunc("/page", start_process).Methods("POST")
	log.Fatal(http.ListenAndServe(":8000", r))

}
