package quandl

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/billylkc/stocklib/db"
	"github.com/billylkc/stocklib/stock"
	"github.com/gocarina/gocsv"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// HistoricalPrice as the struct for the API result
type HistoricalPrice struct {
	Code     int     `csv:"-"`
	CodeF    string  `csv:"-"` // code in string format
	Date     string  `csv:"Date"`
	Ask      float64 `csv:"Ask"`
	Bid      float64 `csv:"Bid"`
	Open     float64 `csv:"Previous Close"` // open is missing in quandl, using prev close
	High     float64 `csv:"High"`
	Low      float64 `csv:"Low"`
	Close    float64 `csv:"Nominal Price"`
	Volume   int     `csv:"Share Volume (000)"`
	Turnover int     `csv:"Turnover (000)"`
}

// Quandl
type Quandl struct {
	logger *logrus.Logger
	limit  int
	start  string // not using start date right now
	end    string
	order  string
}

type option func(*Quandl)

// New as Quandl constructor
func New(logger *logrus.Logger) Quandl {
	today := time.Now().Format("2006-01-02")

	return Quandl{
		logger: logger,
		limit:  10,
		end:    today,
		order:  "desc",
	}
}

// Dev for development
func Dev() {

	logger := logrus.New()
	logger.Out = io.Writer(os.Stdout)

	q := New(logger)
	res, _ := q.GetStock(5, "2021-02-21")
	fmt.Println(res)
}

// GetStockByCode is a wrapper to get all the historical dat a for a single stock
func (q *Quandl) GetStockByCode(code int) ([]HistoricalPrice, error) {
	return q.GetStock(code, "")
}

// GetStockByDate
func (q *Quandl) GetStockByDate(date string) ([]HistoricalPrice, error) {
	var result []HistoricalPrice

	companies, err := stock.GetCompanyList()
	if err != nil {
		return result, err
	}

	fmt.Printf("Getting date - %s - %d \n\n", date, len(companies))

	var counter int
	for _, code := range companies {

		// Check for consecutive failures
		if counter >= 20 {
			return []HistoricalPrice{}, fmt.Errorf("Data not ready - %s", date)
		}

		fmt.Printf("(%s) Getting stock - %d", date, code)
		data, err := q.GetStock(code, date)
		if err != nil {
			counter += 1
			// fmt.Printf("Error - %v \n", err)
		} else {
			counter = 0 // reset
			result = append(result, data...)
		}
		fmt.Printf(" - %d records \n", len(data))
	}
	return result, err
}

// GetStock is the underlying function to get the stock by different code and date settings
func (q *Quandl) GetStock(code int, date string) ([]HistoricalPrice, error) {
	var data []HistoricalPrice

	// Derive input
	if date == "" {
		today := time.Now().Format("2006-01-02")
		q.option(setEndDate(today))
		q.option(setLimit(10000))
	} else {
		q.option(setEndDate(date))
		q.option(setLimit(10))
	}

	codeF := fmt.Sprintf("%05d", code)
	endpoint, _ := q.getEndpoint(code)

	response, err := http.Get(endpoint)
	if err != nil {
		return data, errors.Wrap(err, "something is wrong with the request")
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)

	if err := gocsv.UnmarshalBytes(body, &data); err != nil {
		q.logger.Error("unable to unmarshal the response")
		return data, errors.New("unable to unmarshal the response")
	}

	for i, _ := range data {
		data[i].Code = code
		data[i].CodeF = codeF
		data[i].Volume = data[i].Volume * 1000
		data[i].Turnover = data[i].Turnover * 1000
	}

	// Handle date logic
	var matched bool
	var result []HistoricalPrice
	if date == "" {
		matched = true
		result = data
	} else {
		for _, d := range data {
			if d.Date == date {
				matched = true
				result = []HistoricalPrice{d}
			}
		}
	}
	if !matched {
		return []HistoricalPrice{}, errors.New("not found")
	}
	return result, nil
}

func (q *Quandl) Insert(data []HistoricalPrice) error {

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

	stmt, err := txn.Prepare(pq.CopyIn("stock", "date", "ask", "bid", "open", "high", "low", "close", "volume", "turnover", "code"))
	if err != nil {
		return (err)
	}

	for _, model := range data {
		_, err := stmt.Exec(model.Date, model.Ask, model.Bid, model.Open, model.High, model.Low, model.Close, model.Volume, model.Turnover, model.CodeF)
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

// Option sets the options specified.
func (q *Quandl) option(opts ...option) {
	for _, opt := range opts {
		opt(q)
	}
}

//getEndpoint gets the endpoint for the quandl api
func (q *Quandl) getEndpoint(code int) (string, error) {
	token, err := getToken()
	if err != nil {
		return "", err
	}
	codeF := fmt.Sprintf("%05d", code)
	endpoint := fmt.Sprintf("https://www.quandl.com/api/v3/datasets/HKEX/%s/data.csv?limit=%d&end_date=%s&order=%s&api_key=%s", codeF, q.limit, q.end, q.order, token)
	return endpoint, nil
}

// getToken returns the quandl api token
func getToken() (string, error) {
	token := os.Getenv("QUANDL_TOKEN")
	if token == "" {
		return "", errors.New("please check you env variable QUANDL_TOKEN")
	}
	return token, nil
}

func setLimit(n int) option {
	return func(q *Quandl) {
		q.limit = n
	}
}
func setOrder(settings string) option {
	return func(q *Quandl) {
		q.order = settings
	}
}
func setStartDate(settings string) option {
	return func(q *Quandl) {
		q.start = settings
	}
}
func setEndDate(settings string) option {
	return func(q *Quandl) {
		q.end = settings
	}
}
