package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	str string
)

type SymbolValue struct {
	Price      float64 `json:"price_24h"`
	Volume     float64 `json:"volume_24h"`
	Last_Trade float64 `json:"last_trade_price"`
}

type SymbolInfo struct {
	Symbol string `json:"symbol"`
	SymbolValue
}

func home(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(str)) //выводим результат
}

// Принимаем данные прайса
func ResponceData(url string) []SymbolInfo {
	res, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}
	si := []SymbolInfo{}
	err = json.Unmarshal(body, &si)
	if err != nil {
		log.Fatal(err)
	}
	return si
}

//Загружаем данные в Mysql
func SaveToBd(si []SymbolInfo) error {
	//Обновляем данные в бд каждые 30 сек
	go func() {
		for {
			fmt.Println("подключение к SQL")
			db, err := sql.Open("mysql", "root:1352@tcp(appsDB)/golang")
			if err != nil {
				panic(err)
			}

			defer db.Close()
			for _, vol := range si {
				//загружаем данные в базу данных

				insert, err := db.Query(fmt.Sprintf("INSERT INTO tickers (symbol, price, volume, last_trade) VALUES('%s', '%f', '%f', '%f')", vol.Symbol, vol.Price, vol.Volume, vol.Last_Trade))
				if err != nil {
					log.Fatal(err)
				}
				defer insert.Close()
			}
			fmt.Println("Обновлено")
			RequestMysql()
			time.Sleep(30 * time.Second)
		}
	}()
	return nil

}

//Получаем данные из Mysql
func RequestMysql() error {
	db, err := sql.Open("mysql", "root:1352@tcp(appsDB)/golang")
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()
	fmt.Println("подключение к SQL")

	//Берем данные из базы данных (все)
	res, err := db.Query("SELECT * FROM tickers")
	if err != nil {
		log.Fatal(err)
	}
	var si = SymbolInfo{}
	dto := make(map[string](SymbolValue))

	for res.Next() {
		err = res.Scan(&si.Symbol, &si.Price, &si.Volume, &si.Last_Trade)
		if err != nil {
			panic(err)
		}
		dto[si.Symbol] = si.SymbolValue
		b, err := json.Marshal(dto)
		if err != nil {
			log.Fatal(err)
		}
		str = string(b) //возвращаем результат в глоб. перемен.
	}

	return nil
}

func main() {
	url := ResponceData("https://api.blockchain.com/v3/exchange/tickers") //получаем адрес
	SaveToBd(url)

	time.Sleep(2 * time.Second)

	addr := flag.String("addr", ":4000", "Сетевой адрес веб-сервера")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", home)
	log.Printf("Запуск сервера на %s", *addr)
	err := http.ListenAndServe(*addr, mux)
	log.Fatal(err)

}
