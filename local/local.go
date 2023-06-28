package local

import (
	"fmt"
	"strings"
	"time"

	"github.com/billylkc/stocklib/db"
	"github.com/billylkc/stocklib/util"
)

type StockPrice struct {
	Code     string
	DateRaw  time.Time // real date format
	Date     string    // date in string format, DD/MM
	Close    float64
	Changes  float64 // Percentage changes in float
	ChangesF string  // Percentage changes on Close. Formatted with +/- sign
}

// GetStockPrice gets the historical stock price of a certain code
func GetStockPrice(code ...int) ([]StockPrice, error) {
	var result []StockPrice
	database, err := db.GetConnection()
	if err != nil {
		return result, err
	}

	var codeList []string
	for _, c := range code {
		codeList = append(codeList, fmt.Sprintf("'%05d'", c))
	}
	// Query data

	queryF := `
    SELECT
       code, date, close
    FROM
       stock
    WHERE
       code IN (%s)
    ORDER BY
       date desc
    LIMIT 50;
    `
	query := fmt.Sprintf(queryF, strings.Join(codeList, ","))
	fmt.Println(query)
	rows, err := database.Query(query)
	defer rows.Close()
	if err != nil {
		fmt.Println(err.Error())
	}

	for rows.Next() {
		var sp StockPrice
		_ = rows.Scan(&sp.Code, &sp.DateRaw, &sp.Close)
		sp.Date = sp.DateRaw.Format("02/01")
		result = append(result, sp)
	}

	// Derive % changes on Close
	for i, _ := range result {
		var changes float64
		if i < len(result)-1 {
			changes = util.PercentChange(result[i].Close, result[i+1].Close)
		}
		result[i].Changes = changes
		result[i].ChangesF = util.PercentFormat(changes)
	}

	return result, nil
}
