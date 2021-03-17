package stock

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/billylkc/stocklib/db"
	"github.com/billylkc/stocklib/util"
	"github.com/lib/pq"
)

// Performance from aastock
type Performance struct {
	Date     string
	Sector   string
	Industry string
	Code     string
	Close    float64
	ThreeY   float64 // 3-years
	OneY     float64 // 1-year
	SixM     float64 // 6-months
	ThreeM   float64 // 3-months
	OneM     float64 // 1-months
	OneW     float64 // 1-week
	Ytd      float64 // year to date
}

// Gets all the sectors + industry code
func GetIndustryPerformance(date string) ([]Performance, error) {
	var results []Performance

	exist := db.RecordExists("industry_performance", date)
	if exist {
		return results, fmt.Errorf("records exists in db - %s", date)
	}

	links, err := getIndustryLinks(date, 3) // check dates
	if err != nil {
		return results, err
	}

	for _, link := range links {
		rec, _ := getPerformance(date, link)
		results = append(results, rec...)
	}
	return results, nil
}

// InsertPerformance inserts to the industry_performance table
func InsertPerformance(data []Performance) error {

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

	stmt, err := txn.Prepare(pq.CopyIn("industry_performance", "date", "sector", "industry", "code", "close", "threey", "oney", "sixm", "threem", "onem", "onew", "ytd"))
	if err != nil {
		return (err)
	}

	for _, model := range data {
		_, err := stmt.Exec(model.Date, model.Sector, model.Industry, model.Code, model.Close, model.ThreeY, model.OneY, model.SixM, model.ThreeM, model.OneM, model.OneW, model.Ytd)
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

func getPerformance(date, link string) ([]Performance, error) {

	var results []Performance

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

	var (
		sector   string
		industry string
		code     string
		close    float64
		three_y  float64
		one_y    float64
		six_m    float64
		three_m  float64
		one_m    float64
		one_w    float64
		ytd      float64
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
	doc.Find("table#tbTS.tblM.s2").Each(func(i int, s *goquery.Selection) {
		s.Find("tr").Each(func(j int, tr *goquery.Selection) {
			var values []string
			tr.Find("td").Each(func(i int, td *goquery.Selection) {
				values = append(values, td.Text())
			})

			if len(values) == 12 {
				code = strings.ReplaceAll(values[0], ".HK", "") // 00301.HK -> 00301
				close, _ = util.ParseF(values[2])
				three_y, _ = util.ParseF(values[5])
				one_y, _ = util.ParseF(values[6])
				six_m, _ = util.ParseF(values[7])
				three_m, _ = util.ParseF(values[8])
				one_m, _ = util.ParseF(values[9])
				one_w, _ = util.ParseF(values[10])
				ytd, _ = util.ParseF(values[11])

				rec := Performance{
					Date:     date,
					Sector:   sector,
					Industry: industry,
					Code:     code,
					Close:    close,
					ThreeY:   three_y,
					OneY:     three_m,
					SixM:     six_m,
					ThreeM:   three_m,
					OneM:     one_m,
					OneW:     one_w,
					Ytd:      ytd,
				}
				results = append(results, rec)
			}
		})
	})
	return results, nil
}
