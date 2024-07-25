package main

import (
	"database/sql"
	_ "database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/tidwall/gjson"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

const (
	StatusNew = "NEW"
)

// Обработчик вебхуков
func webhookHandler(w http.ResponseWriter, r *http.Request) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	err = godotenv.Load(filepath.Join(dir, ".env"))
	if err != nil {
		log.Fatalf("Some error occured. Err: %s", err)
	}

	// Параметры подключения к базе данных
	dsn := os.Getenv("DNS_DB")

	// Подключение к базе данных
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Проверка подключения
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	// Парсинг данных формы
	err = r.ParseForm()
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	fmt.Println(r.Form)

	// Преобразование данных формы в карту
	formData := make(map[string]string)
	for key, values := range r.Form {
		formData[key] = values[0]
	}

	// Преобразование карты в JSON
	jsonData, err := json.Marshal(formData)
	if err != nil {
		http.Error(w, "Unable to marshal JSON", http.StatusInternalServerError)
		return
	}
	jsonDataString := string(jsonData)

	fmt.Println(jsonDataString)

	leadId := gjson.Get(jsonDataString, "leads[status][0][id]")

	if leadId.String() == "" {
		leadId = gjson.Get(jsonDataString, "leads[add][0][id]")
	}

	if leadId.String() != "" {
		fmt.Println(leadId)

		// Проверка существования записи
		var exists bool
		query := "SELECT EXISTS(SELECT 1 FROM amo_deals WHERE amo_deal_lead_id = ?)"
		err = db.QueryRow(query, leadId.String()).Scan(&exists)
		if err != nil {
			log.Fatal(err)
		}

		// Вставка записи, если она не существует
		if !exists {
			insertQuery := "INSERT INTO amo_deals (amo_deal_lead_id, status) VALUES (?, ?)"
			_, err := db.Exec(insertQuery, leadId.String(), StatusNew)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Record inserted successfully")
		} else {
			fmt.Println("Record already exists")
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received and processed"))
}

func main() {
	http.HandleFunc("/webhook", webhookHandler)
	log.Println("Server started on :8083")
	log.Fatal(http.ListenAndServe(":8083", nil))
}
