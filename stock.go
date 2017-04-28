package main

import (
	"github.com/labstack/echo"
	"net/http"
	"encoding/json"
	"strconv"
	"time"
	"io/ioutil"
	"fmt"
	"io"
	"encoding/csv"
	"log"
	"os"
	"bufio"
)

const (
	version = "1.1.0"
	multiplier = 10000
)

var ids map[string]int64
var stock map[int64]Stock
var change = false

type Stock struct {
	id int64
	sku string
	qty int // store *1000 as int (quick hack for 4 digit decimal stock ala Magento)
	instock bool
}

func main() {
	ids = make(map[string]int64)
	stock = make(map[int64]Stock)
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		ret, _ := json.Marshal(map[string]interface{}{"version": version})
		return c.String(http.StatusOK, string(ret))
	})
	e.GET("/get.v"+version, getStock)
	e.GET("/set.v"+version, setStock)
	change = false

	readStockCSV("stock.csv")

	ticker := time.NewTicker(5 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <- ticker.C:
				writeStockCSV("stock.csv")
			case <- quit:
				ticker.Stop()
				return
			}
		}
	}()

	e.Logger.Fatal(e.Start(":1323"))

}

func getStock(c echo.Context) error {
	// Get team and member from the query string
	sku := c.QueryParam("sku")
	id, _ := strconv.ParseInt(c.QueryParam("id"),10,64)
	fmt.Println("getting stock for sku: %s %d",sku,id)
	if (id == 0) {
		id = ids[sku]
	}
	return c.String(http.StatusOK,strconv.FormatFloat((float64(stock[id].qty)/multiplier),'f',4,64))
}

func setStock(c echo.Context) error {
	sku := c.QueryParam("sku")
	qtyf, _ := strconv.ParseFloat(c.QueryParam("qty"),64)
	qty := int(qtyf*multiplier)
	id, _ := strconv.ParseInt(c.QueryParam("id"),10,64)
	ids[sku] = id
	fmt.Println("setting stock for sku:%s id:%d qty:%d",sku,id,qty)
	stock[id] = Stock{sku: sku, id:id, qty: qty}
	change = true
	return c.String(http.StatusOK,strconv.FormatFloat((float64(stock[id].qty)/multiplier),'f',4,64))
}

func writeStockCSV(filename string) {
	if (! change) {
		return
	}
	// setup writer
	csvOut, err := os.Create(filename)
	if err != nil {
		log.Fatal("Unable to open output")
	}
	w := csv.NewWriter(csvOut)
	defer csvOut.Close()
	for key ,item := range stock {
		record := []string{strconv.FormatInt(item.id,10), item.sku, strconv.FormatFloat((float64(item.qty)/multiplier),'f',4,64)}
		err = w.Write(record)
		if err != nil {
			fmt.Println(key,err)
			return
		}
	}
	w.Flush()
	err = csvOut.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Wrote stock change to stock CSV")
	change = false
}

func readStockCSV(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}

	// Create a new reader.
	r := csv.NewReader(bufio.NewReader(f))
	for {
		record, err := r.Read()
		// Stop at EOF.
		if err == io.EOF {
			break
		}
		id, _ := strconv.ParseInt(record[0],10,64)
		sku := record[1]
		qtyf, _ := strconv.ParseFloat(record[2],64)
		qty := int(qtyf*multiplier)
		ids[sku] = id
		stock[id] = Stock{sku: sku, id:id, qty: qty}
	}
	f.Close()
	fmt.Println("Read stock from CSV")
}