package stock

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Company struct {
	Code string
	Name string
}

// GetCompanyName looks up company name from HKEX
func getcompanyname(c int) (Company, error) {
	var result Company

	// Handle input, e.g. code = 00005, date 2021-02-01
	targetCode := fmt.Sprintf("%05d", c) // zfill to 5 digit
	currentTime := time.Now()
	d := currentTime.Format("2006-01-02")
	d = strings.ReplaceAll(d, "-", "") // date in string format

	url := fmt.Sprintf("https://www.hkexnews.hk/sdw/search/stocklist_c.aspx?sortby=stockcode&shareholdingdate=%s", d)
	res, err := http.Get(url)
	if err != nil {
		return result, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return result, err
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return result, err
	}

	// Find the review items
	doc.Find("table.table > tbody > tr").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title

		content := s.Find("td").Text()
		regex := *regexp.MustCompile(`\s*(\d{5})\s*(.*)`)
		matched := regex.FindAllStringSubmatch(content, -1)
		for i := range matched {
			codeStr := matched[i][1]
			companyStr := matched[i][2]

			if codeStr == targetCode {
				result = Company{
					Code: targetCode,
					Name: companyStr,
				}
				break // find then break
			}
		}
	})
	return result, nil
}

// GetCompanyList looks up all the companies' code on HKEX
func GetCompanyList() ([]int, error) {
	var result []int

	currentTime := time.Now()
	d := currentTime.Format("2006-01-02")
	d = strings.ReplaceAll(d, "-", "") // date in string format

	url := fmt.Sprintf("https://www.hkexnews.hk/sdw/search/stocklist_c.aspx?sortby=stockcode&shareholdingdate=%s", d)
	res, err := http.Get(url)
	if err != nil {
		return result, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return result, err
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return result, err
	}

	// Find the review items
	doc.Find("table.table > tbody > tr").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title

		content := s.Find("td").Text()
		regex := *regexp.MustCompile(`\s*(\d{5})\s*(.*)`)
		matched := regex.FindAllStringSubmatch(content, -1)
		for i := range matched {
			codeF := matched[i][1]
			code, err := strconv.Atoi(codeF)
			if err != nil { // ignore
				fmt.Println(err.Error())
			}
			if code < 10000 {
				if code <= 8000 || code >= 9000 {
					result = append(result, code)
				}
			}
		}
	})
	if len(result) == 0 {
		return result, errors.New("something wrong with the hkex company list")
	}
	return result, nil
}
