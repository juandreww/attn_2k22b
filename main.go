package main

import (
	"fmt"
	"html/template"
	"net/http"
	"database/sql"
	_ "github.com/lib/pq"
	"log"
	"strconv"
	// "os"
)

const (
	host = "ec2-52-73-155-171.compute-1.amazonaws.com"
	port = 5432
	user = "'awtpbzyctlpydm'"
	password = "a7bf40c39496f73a03e7412befbc787d29138445d7fce2a34bf31df40cf07d96"
	dbname = "d4ehughfapgq0k"
)

type currency struct {
	ID string
	Name string
}

type configconvertrate struct {
	CurrencyFrom string
	CurrencyTo string
	Rate string
}

var tpl *template.Template
var con *sql.DB

func init() {
	tpl = template.Must(template.ParseGlob("templates/*"))
}

func ConnectDB() *sql.DB {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=require",
	host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)

	if err != nil {
		panic(err)
	}

	con = db
	return db
}

func main() {
	mux := http.DefaultServeMux
	mux.HandleFunc("/", index)
	mux.HandleFunc("/savecurrency", saveCurrency)
	mux.HandleFunc("/listcurrency", listCurrency)
	mux.HandleFunc("/listcurrencyrate", listCurrencyRate)
	mux.HandleFunc("/addconversionrate", addConversionRate)
	mux.HandleFunc("/convertcurrency", convertCurrency)
	db := ConnectDB()
	defer db.Close()

	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.ListenAndServe(":8080", nil)
}

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Println("This is index api: ", r.Method)
	tpl.ExecuteTemplate(w, "index.gohtml", nil)
}

func saveCurrency(w http.ResponseWriter, r *http.Request) {
	fmt.Println("This is savecurrency api: ", r.Method)

	data := currency{
		r.FormValue("id"),
		r.FormValue("name"),
	}

	fmt.Println(data)

	cur := currency{}
	IsExist := false
	sqlStatement := `SELECT id, name FROM currency WHERE (id=$1 OR name=$2);`
	row := con.QueryRow(sqlStatement, data.ID, data.Name)
	err := row.Scan(&cur.ID, &cur.Name,)
	switch err {
	case sql.ErrNoRows:
		IsExist = false
	case nil:
		IsExist = true
	default:
	  	panic(err)
	}
	
	if IsExist == false {
		sqlStatement = `
			INSERT INTO currency (id, name)
			VALUES ($1, $2)`
		_, err := con.Exec(sqlStatement, data.ID, data.Name)
		if err != nil {
			panic(err)
		}
	} else {
		data = currency{
			"error",
			"error",
		}
	}

	tpl.ExecuteTemplate(w, "index.gohtml", data)
}

func listCurrency(w http.ResponseWriter, r *http.Request) {
	fmt.Println("This is listcurrency api: ", r.Method)
	var list []currency

	rows, err := con.Query("SELECT id, name FROM currency ORDER BY id ASC")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		cur := currency{}
		err := rows.Scan(&cur.ID, &cur.Name,)
		switch err {
		case sql.ErrNoRows:
			fmt.Println("row is not exist")
			return
		case nil:
			fmt.Println(cur.ID, cur.Name)
		default:
			panic(err)
		}

		list = append(list, cur)
	}

	tpl.ExecuteTemplate(w, "listcurrency.gohtml", list)
}

func listCurrencyRate(w http.ResponseWriter, r *http.Request) {
	fmt.Println("This is listcurrencyrate api: ", r.Method)
	var list []configconvertrate

	rows, err := con.Query("SELECT cf.name currencyfrom, ct.name currencyto, round(p.rate,2) rate FROM currencyrate p LEFT JOIN currency cf ON cf.id = p.currencyfrom LEFT JOIN currency ct ON ct.id = p.currencyto ORDER BY p.id ASC")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		cur := configconvertrate{}
		err := rows.Scan(&cur.CurrencyFrom, &cur.CurrencyTo, &cur.Rate)

		IsError := HandleErrorOfSelect(w, err)
		if (IsError == true) {
			fmt.Println("row is not exist")
			return
		}

		list = append(list, cur)
	}

	tpl.ExecuteTemplate(w, "listcurrencyrate.gohtml", list)
}

func addConversionRate(w http.ResponseWriter, r *http.Request) {
	if (r.Method == http.MethodPost) {
		fmt.Println("This is add conversion rate api: ", r.Method)
		data := configconvertrate{
			r.FormValue("currencyfrom"),
			r.FormValue("currencyto"),
			r.FormValue("rate"),
		}

		var str string
		var intval int
		check1 := currency{}

		sqlStatement := `SELECT count(id) id FROM currency WHERE (id=$1 OR id=$2);`
		row := con.QueryRow(sqlStatement, data.CurrencyFrom, data.CurrencyTo)
		err := row.Scan(&check1.ID)
		IsError := HandleErrorOfSelect(w, err)
		if (IsError == true) {
			tmp := currency{
				"error",
				"All CurrencyID is not found in database",
			}
			tpl.ExecuteTemplate(w, "addconversionrate.gohtml", tmp)
			return
		} else {
			intval, err = strconv.Atoi(check1.ID)
			if intval < 2 {
				tmp := currency{
					"error",
					"One of the CurrencyID is not found in database",
				}
				tpl.ExecuteTemplate(w, "addconversionrate.gohtml", tmp)
				return
			}
		}
		
		sqlStatement = `SELECT count(id) id FROM currencyrate 
						WHERE ((currencyfrom=$1 AND currencyto=$2) OR (currencyfrom=$3 AND currencyto=$4))`
		row = con.QueryRow(sqlStatement, data.CurrencyFrom, data.CurrencyTo, data.CurrencyTo, data.CurrencyFrom)
		err = row.Scan(&str)
		IsError = HandleErrorOfSelect(w, err)
		if (IsError == true) {
			if str != "0" {
				tmp := currency{
					"error",
					"CurrencyRate is not exist in the database",
				}
				tpl.ExecuteTemplate(w, "addconversionrate.gohtml", tmp)
				return
			}
		} else {
			intval, err = strconv.Atoi(str)
			if intval >  0 {
				tmp := currency{
					"error",
					"CurrencyRate already exist in the database",
				}
				tpl.ExecuteTemplate(w, "addconversionrate.gohtml", tmp)
				return
			}
		}

		sqlStatement = `SELECT nullif(max(id),0) id FROM currencyrate`
		row = con.QueryRow(sqlStatement)
		err = row.Scan(&str)
		IsError = HandleErrorOfSelect(w, err)
		if (IsError == true) {
			str = "0"
		}

		intval, err = strconv.Atoi(str)
		intval++
		sqlStatement = `
			INSERT INTO currencyrate (id, currencyfrom, currencyto, rate)
			VALUES ($1, $2, $3, $4)`
		_, err = con.Exec(sqlStatement, intval, data.CurrencyFrom, data.CurrencyTo, data.Rate)
		if err != nil {
			panic(err)
		}
		
		tmp := currency{
			"succeed",
			"Currency Rate added",
		}
		tpl.ExecuteTemplate(w, "addconversionrate.gohtml", tmp)
	} else {
		fmt.Println("This is add conversion rate api: ", r.Method)

		tpl.ExecuteTemplate(w, "addconversionrate.gohtml", nil)
	}
}

func convertCurrency(w http.ResponseWriter, r *http.Request) {
	if (r.Method == http.MethodPost) {
		fmt.Println("This is add convertcurrency api: ", r.Method)
		data := configconvertrate{
			r.FormValue("currencyfrom"),
			r.FormValue("currencyto"),
			r.FormValue("amount"),
		}
		fmt.Println(data)

		var str string
		var intval int
		var floatval, amount float64
		check1 := currency{}

		sqlStatement := `SELECT count(id) id FROM currency WHERE (id=$1 OR id=$2);`
		row := con.QueryRow(sqlStatement, data.CurrencyFrom, data.CurrencyTo)
		err := row.Scan(&check1.ID)
		IsError := HandleErrorOfSelect(w, err)
		if (IsError == true) {
			tmp := currency{
				"error",
				"All CurrencyID is not found in database",
			}
			tpl.ExecuteTemplate(w, "convertcurrency.gohtml", tmp)
			return
		} else {
			intval, err = strconv.Atoi(check1.ID)
			if intval < 2 {
				tmp := currency{
					"error",
					"One of the CurrencyID is not found in database",
				}
				tpl.ExecuteTemplate(w, "convertcurrency.gohtml", tmp)
				return
			}
		}
		
		sqlStatement = `SELECT count(id) id FROM currencyrate 
						WHERE ((currencyfrom=$1 AND currencyto=$2) OR (currencyfrom=$3 AND currencyto=$4))`
		row = con.QueryRow(sqlStatement, data.CurrencyFrom, data.CurrencyTo, data.CurrencyTo, data.CurrencyFrom)
		err = row.Scan(&str)
		IsError = HandleErrorOfSelect(w, err)
		if (IsError == true) {
			if str != "0" {
				tmp := currency{
					"error",
					"CurrencyRate is not exist in the database (1)",
				}
				tpl.ExecuteTemplate(w, "convertcurrency.gohtml", tmp)
				return
			}
		}

		var val1, val2 string
		sqlStatement = `SELECT currencyfrom, currencyto,rate FROM currencyrate 
					WHERE ((currencyfrom=$1 AND currencyto=$2) OR (currencyfrom=$3 AND currencyto=$4))
					LIMIT 1`
		row = con.QueryRow(sqlStatement, data.CurrencyFrom, data.CurrencyTo, data.CurrencyTo, data.CurrencyFrom)
		err = row.Scan(&val1, &val2, &str)
		IsError = HandleErrorOfSelect(w, err)
		if (IsError == true) {
			tmp := currency{
				"error",
				"CurrencyRate is not exist in the database (2)",
			}
			tpl.ExecuteTemplate(w, "convertcurrency.gohtml", tmp)
			return 
		}

		floatval, err = strconv.ParseFloat(str, 64)
		amount, err = strconv.ParseFloat(data.Rate, 64)
		if data.CurrencyFrom == val1 {
			amount = amount * floatval
		} else {
			amount = amount / floatval
		}
		
		str = fmt.Sprintf("%.2f", amount)
		
		tmp := currency{
			"succeed",
			"Congrats! You converted rate to " + str,
		}
		tpl.ExecuteTemplate(w, "convertcurrency.gohtml", tmp)
	} else {
		fmt.Println("This is add convertcurrency api: ", r.Method)

		tpl.ExecuteTemplate(w, "convertcurrency.gohtml", nil)
	}
}

func HandleErrorOfSelect(w http.ResponseWriter, err error) bool {
	data := false
	switch err {
	case sql.ErrNoRows:
		data = true
	case nil:
		data = false
	default:
		data = true
	}

	return data
}
