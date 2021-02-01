// Package ddgquery provides a method to retrieve basic search results from DuckDuckGo
// ddgquery uses goquery from PuerkitoBio
package query

import (
	"fmt"
	"log"
	_ "net/url"
	_ "strings"

	"github.com/PuerkitoBio/goquery"
)

// DrudgeResault holds the returned query data
type DrudgeResault struct {
	Title string
	Ref   string
}

// DrudgeQuery calls the ddg api and puts the results into an array

func DrudgeQuery(it int) []DrudgeResault {

	qf := "https://www.drudgereport.com/"
	doc, err := goquery.NewDocument(qf)

	results := []DrudgeResault{}

	if err != nil {
		log.Fatal(err)
	}

	sel := doc.Find("center")
	test := sel.Find("a")
	fmt.Println(test.Text())
	for v, i := range test.Nodes {

		if it == len(results) {
			break
		}

		single := test.Eq(v)
		title := ""

		if single.Find("font").Text() == "" {
			title = single.Text()
		} else {
			title = single.Find("font").Text()
		}

		ref := i.Attr[0].Val
		results = append(results[:], DrudgeResault{title, ref})
	}

	// Return array of results and formated query used to get the results
	return results
}
