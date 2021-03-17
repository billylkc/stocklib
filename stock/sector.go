package stock

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/billylkc/stocklib/db"
	"github.com/billylkc/stocklib/util"
	"github.com/lib/pq"
)

// Sector for industry sector overview
type Sector struct {
	Date        string
	Sector      string
	ChangePct   float64
	PchangePct  float64
	Turnover    int
	AvgTurnover int
	AvgPE       float64
	ZoneA       int // > +2%
	ZoneB       int // 0 - +2%
	ZoneC       int // 0%
	ZoneD       int // 0 - -2%
	ZoneE       int // < -2%
	ZoneN       int // total no of stocks
}

func GetSectorOveriew(date string) ([]Sector, error) {
	var results []Sector
	link := "http://www.aastocks.com/en/stocks/market/industry/industry-performance.aspx"

	exist := db.RecordExists("sector", date)
	if exist {
		return results, fmt.Errorf("records exists in db - %s", date)
	}

	// Check if data is ready
	dataReady := util.CheckWebsiteDate(date)
	if !dataReady {
		return results, fmt.Errorf("data not ready - %s", date)
	}

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
		sector      string
		changePct   float64
		pchangePct  float64
		turnover    int
		avgTurnover int
		avgPE       float64
		zone        string // from dist string `0,2,2,9,5`
		zoneA       int    // > +2%
		zoneB       int    //  0 - +2%
		zoneC       int    // 0%
		zoneD       int    // -2% - 0
		zoneE       int    // < -2%
		zoneN       int    // No of stocks
	)

	doc.Find("table.indview_tbl").Each(func(i int, s *goquery.Selection) {
		s.Find("tr.indview_tr").Each(func(j int, tr *goquery.Selection) {
			var elements []string
			tr.Find("td").Each(func(k int, td *goquery.Selection) {
				dist, exists := td.Find("div.jsPerfDistBar").Attr("def")
				if exists {
					zone = dist
					zones := strings.Split(zone, ",")
					zoneA, _ = strconv.Atoi(zones[0])
					zoneB, _ = strconv.Atoi(zones[1])
					zoneC, _ = strconv.Atoi(zones[2])
					zoneD, _ = strconv.Atoi(zones[3])
					zoneE, _ = strconv.Atoi(zones[4])
					zoneN = zoneA + zoneB + zoneC + zoneD + zoneE
				}
				elements = append(elements, td.Text())
			})
			if len(elements) >= 6 {
				sector = strings.TrimSpace(elements[0])
				changePct, _ = util.ParseF(elements[1])
				pchangePct, _ = util.ParseF(elements[2])
				turnover, _ = util.ParseI(elements[3])
				avgTurnover, _ = util.ParseI(elements[4])
				avgPE, _ = util.ParseF(elements[5])
			}
			s := Sector{
				Date:        date,
				Sector:      sector,
				ChangePct:   changePct,
				PchangePct:  pchangePct,
				Turnover:    turnover,
				AvgTurnover: avgTurnover,
				AvgPE:       avgPE,
				ZoneA:       zoneA,
				ZoneB:       zoneB,
				ZoneC:       zoneC,
				ZoneD:       zoneD,
				ZoneE:       zoneE,
				ZoneN:       zoneN,
			}
			results = append(results, s)
		})
	})
	return results, nil
}

// InsertSector inserts to the sector table
func InsertSector(data []Sector) error {

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

	stmt, err := txn.Prepare(pq.CopyIn("sector", "date", "sector", "changepct", "pchangepct", "turnover", "avgturnover", "avgpe", "zonea", "zoneb", "zonec", "zoned", "zonee", "zonen"))
	if err != nil {
		return (err)
	}

	for _, model := range data {
		_, err := stmt.Exec(model.Date, model.Sector, model.ChangePct, model.PchangePct, model.Turnover, model.AvgTurnover, model.AvgPE, model.ZoneA, model.ZoneB, model.ZoneC, model.ZoneD, model.ZoneE, model.ZoneN)
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
