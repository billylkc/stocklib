package stock

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/billylkc/stocklib/db"
	"github.com/billylkc/stocklib/util"
	"github.com/lib/pq"
)

// Industry Overview from aastock
type Industry struct {
	Date      string
	Sector    string
	Industry  string
	CodeF     string
	Close     float64
	Change    float64
	ChangePct float64
	Volume    int
	Turnover  int
	PE        float64 // Price per Earnings
	PB        float64 // Price to Book
	YieldPct  float64
	MarketCap int
}

// Gets all the sectors + industry code
func GetIndustryOverview(date string) ([]Industry, error) {
	var results []Industry

	exist := db.RecordExists("industry", date)
	if exist {
		return results, fmt.Errorf("records exists in db - %s", date)
	}

	links, err := getIndustryLinks(date, 1) // check dates
	if err != nil {
		return results, err
	}

	for _, link := range links {
		industry, _ := getIndustryOverview(date, link)
		results = append(results, industry...)
	}
	return results, nil
}

// InsertIndustry inserts to the industry table
func InsertIndustry(data []Industry) error {

	if len(data) == 0 {
		return errors.New("no records to be inserted")
	}
	fmt.Printf("Start inserting records - %d\n", len(data))

	db, err := db.GetConnection()
	if err != nil {
		return err
	}

	txn, err := db.Begin()
	if err != nil {
		return err
	}
	defer txn.Commit()

	stmt, err := txn.Prepare(pq.CopyIn("industry", "date", "sector", "industry", "code", "close", "change", "changepct", "volume", "turnover", "pe", "pb", "yieldpct", "marketcap"))
	if err != nil {
		return (err)
	}

	for _, model := range data {
		_, err := stmt.Exec(model.Date, model.Sector, model.Industry, model.CodeF, model.Close, model.Change, model.ChangePct, model.Volume, model.Turnover, model.PE, model.PB, model.YieldPct, model.MarketCap)
		if err != nil {
			txn.Rollback()
			return err
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	err = stmt.Close()
	if err != nil {
		return err
	}
	fmt.Println("Done")
	return nil
}

// getIndustryOverview gets a single industry overview from aastock
func getIndustryOverview(date, link string) ([]Industry, error) {

	var result []Industry

	res, err := http.Get(link)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Title
	var (
		sector    string // Industry sectors, e.g. Materials
		industry  string // Industry, e.g. Chemical Products
		code      string
		close     float64
		change    float64
		changePct float64
		volume    int
		turnover  int
		pe        float64
		pb        float64
		yield     float64
		marketCap int
	)
	doc.Find("h1").Each(func(i int, s *goquery.Selection) {
		text := s.Text() // e.g. Industry Details - Materials - Chemical Products
		texts := strings.Split(text, "-")
		if len(texts) == 3 {
			sector = strings.TrimSpace(texts[1])   // e.g. Materials
			industry = strings.TrimSpace(texts[2]) // e.g. Chemical Products
			fmt.Printf("Getting [%s] - [%s]\n", sector, industry)
		}
	})

	// For each code inside a sector, gets the details
	doc.Find("span.float_l").Each(func(i int, s *goquery.Selection) {
		code = strings.TrimSpace(s.Text()) // e.g. 00301.HK
		if strings.Contains(code, "0") {   // Check starts with 0
			code = strings.ReplaceAll(code, ".HK", "") // 00301.HK -> 00301
			ss := s.ParentsUntil("tbody")
			var values []string

			ss.Each(func(j int, tb *goquery.Selection) {
				tb.Find("td.cls.txt_r.pad3").Each(func(i int, td *goquery.Selection) {
					// fmt.Println(td.Text())
					values = append(values, td.Text())
				})
			})

			if len(values) == 10 {
				_ = values[0] // Some empty string
				close, _ = util.ParseF(values[1])
				change, _ = util.ParseF(values[2])
				changePct, _ = util.ParseF(values[3])
				volume, _ = util.ParseI(values[4])
				turnover, _ = util.ParseI(values[5])
				pe, _ = util.ParseF(values[6])
				pb, _ = util.ParseF(values[7])
				yield, _ = util.ParseF(values[8])
				marketCap, _ = util.ParseI(values[9])

				rec := Industry{
					Date:      date,
					Sector:    sector,
					Industry:  industry,
					CodeF:     code,
					Close:     close,
					Change:    change,
					ChangePct: changePct,
					Volume:    volume,
					Turnover:  turnover,
					PE:        pe,
					PB:        pb,
					YieldPct:  yield,
					MarketCap: marketCap,
				}
				result = append(result, rec)
			}
		}
	})
	return result, nil
}

// getIndustryLinks gets all the individual sector/industires links
func getIndustryLinks(date string, tab int) ([]string, error) {
	// tab reference
	// 1 - Overview
	// 2 - Range
	// 3 - Performance
	// 4 - Financial Ratio
	// 5 - Banking Ratio (Blank)
	// 6 - Earnings

	var links []string

	// Check if data is ready
	dataReady := util.CheckWebsiteDate(date)
	if !dataReady {
		return links, fmt.Errorf("data not ready - %s", date)
	}

	res, err := http.Get("http://www.aastocks.com/en/stocks/market/industry/sector-industry-details.aspx")
	if err != nil {
		return links, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return links, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}
	body, err := ioutil.ReadAll(res.Body)

	r := regexp.MustCompile("gotoindustry\\(\\'(\\d{4})\\'\\)")
	matches := r.FindAllStringSubmatch(string(body), -1)
	for _, match := range matches {
		if len(match) >= 2 {
			industryCode := match[1]
			link := fmt.Sprintf("http://www.aastocks.com/en/stocks/market/industry/sector-industry-details.aspx?industrysymbol=%s&t=%d&s=&o=&p=", industryCode, tab)

			links = append(links, link)
		}
	}
	return links, nil
}
