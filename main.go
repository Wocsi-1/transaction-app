package main //здесь мы запускаем ядро GO

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
) // Тут мы импортируем нужные библеотека

var db *sql.DB

type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        time.Time
} //Обьявляем структуру Transaction И айди, суммой, категорией, описанием и датой

type Analyzer interface {
	Name() string
	Analyze(t []Transaction) string
} // Здесь мы обьявляем интерфейс Analyzer с данными Name названия и самого анализа

type TopSpenderAnalyzer struct{} //Анализ самой дорогой покупки, без данных просто название самой для методов

func (t TopSpenderAnalyzer) Name() string {
	return "Самая дорогая покупка"
} // Делаем метод который даёт интерфейсу название операции

func (t TopSpenderAnalyzer) Analyze(transactions []Transaction) string { // обьявляем метод анализа самой большой покупки
	var maxTran Transaction             // переменная maxTran типа структуры Transaction
	for _, tran := range transactions { // запускаем цикл где берём срез transactions и испозьзуем только значения, без ключа структуры Transaction
		if tran.Amount > maxTran.Amount { // обьявляем условие что если сумма в нашем срезе transactions больше чем в срезе maxTran, то мы присваевыем ему функцию
			maxTran = tran
		}
	}
	time.Sleep(1 * time.Second)                                                                         //делаем задержку в 1 секунду
	return fmt.Sprintf("💸 %s (%s) — %.2f BYN\n", maxTran.Description, maxTran.Category, maxTran.Amount) //Выводим нашу страку через Sprintf, чтобы это было возможно с нужными данными
}

type CategoryAnalyzer struct { //Анализ расходов по категориям, без данных просто название самой для методов
}

func (c CategoryAnalyzer) Name() string { //выводим имя
	return "Расходы по категориям"
}

func (c CategoryAnalyzer) Analyze(t []Transaction) string { // обявляем метод анализа расхода по категориям
	categories := make(map[string]float64) //обьявляем карту categories

	for _, trans := range t { //обьявляем цикл по срезу t, где берём просто значения
		categories[trans.Category] += trans.Amount //слаживаем суммы категорий
	}

	result := ""                       // резкльтат это строка
	for cat, sum := range categories { //цикл где мы берём из карты categories и категорию и значения
		result += fmt.Sprintf("%s: %.2fBYN\n", cat, sum) //приплючаваяем к результату категории с суммами и их категорями
	}
	time.Sleep(1 * time.Second) // ждём 1 секунду
	return result               // отдаём результат
}

func addTransaction(transactions *[]Transaction, amount float64, category string, description string) { //Добавляем транзакции

	newID := len(*transactions) + 1 //вписываем айди так, чтобы каждый раз она была на 1 больше

	newT := Transaction{ //обьявляем новую транзакцию
		ID:          newID,
		Amount:      amount,
		Category:    category,
		Description: description,
		Date:        time.Now(),
	}
	*transactions = append(*transactions, newT) //изменяя срез transactions добавляем новую транзакцию
}

func readString(promt string) string { //обявляем функцию чтения написаного
	fmt.Print(promt) //читаем промт

	reader := bufio.NewReader(os.Stdin) //в переменную reader даем читателя, обьясни поподробнее кстати как это работает

	text, _ := reader.ReadString('\n') // берём только текст читаем строку

	return strings.TrimSpace(text)
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite", "transactions.db")
	if err != nil {
		panic("Ошибка подключения к БД: " + err.Error())
	}
	createTableSQl := `CREATE TABLE IF NOT EXISTS expenses (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	amount REAL,
	category TEXT,
	description TEXT,
	date TEXT
	);`
	_, err = db.Exec(createTableSQl)
	if err != nil {
		panic("Ошибка создания таблицы: " + err.Error())
	}
	fmt.Println("✅ База данных подключена и готова к работе!")
}

func addTransactionToDB(amount float64, category string, description string) {
	insertSQL := `INSERT INTO expenses(amount, category, description, date) VALUES (?, ?, ?, ?)`

	_, err := db.Exec(insertSQL, amount, category, description, time.Now().Format("2006-01-02 15:04:05"))

	if err != nil {
		fmt.Println("⚠️ Ошибка при сохранении:", err)
	} else {
		fmt.Println("✅ Транзакция сохранена в базу данных!")
	}
}

func getTransactionFromDB() []Transaction {
	row, err := db.Query("SELECT id, amount, category, description, date FROM expenses ORDER BY id DESC")
	if err != nil {
		fmt.Println("⚠️ Ошибка при чтении БД:", err)
		return nil
	}
	defer row.Close()

	var transactions []Transaction
	for row.Next() {
		var t Transaction
		var dateStr string

		err := row.Scan(&t.ID, &t.Amount, &t.Category, &t.Description, &dateStr)
		if err != nil {
			fmt.Println("⚠️ Ошибка разбора строки:", err)
			continue
		}

		t.Date, _ = time.Parse("2006-01-02 15:04:05", dateStr)

		transactions = append(transactions, t)
	}

	return transactions

}

func handleTransactions(w http.ResponseWriter, r *http.Request) {

	transactions := getTransactionFromDB()

	jsonData, err := json.Marshal(transactions)
	if err != nil {
		http.Error(w, "Ошибка при создании JSON", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonData)
}

func handleTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", 405)
		return
	}

	var input struct {
		Amount      float64 `json:"amount"`
		Category    string  `json:"category"`
		Description string  `json:"description"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Неверный JSON", 400)
		return
	}

	addTransactionToDB(input.Amount, input.Category, input.Description)

	w.Write([]byte("OK"))
}

type CategoryStat struct {
	Category string  `json:"category"`
	Sum      float64 `json:"sum"`
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	transactions := getTransactionFromDB()

	categories := make(map[string]float64)
	for _, t := range transactions {
		categories[t.Category] += t.Amount
	}

	var stats []CategoryStat
	for cat, sum := range categories {
		stats = append(stats, CategoryStat{Category: cat, Sum: sum})
	}

	jsonData, err := json.Marshal(stats)
	if err != nil {
		http.Error(w, "Ошибка", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func main() {
	var wg sync.WaitGroup
	initDB()
	var mu sync.Mutex

	http.HandleFunc("/api/add", handleTransaction)

	http.HandleFunc("/api/stats", handleStats)

	http.HandleFunc("/api/transactions", handleTransactions)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	fmt.Println("🚀 Сервер запущен! Открой в браузере: http://localhost:8080")
	http.ListenAndServe(":8080", nil)

	for {
		fmt.Println("\n--- МЕНЮ ---")
		fmt.Println("1. Добавить транзакцию")
		fmt.Println("2. Показать анализ")
		fmt.Println("3. Выход")

		choice := readString("Ваш выбор (1/2/3): ")

		switch choice {
		case "1":
			amountStr := readString("Введите сумму: ")
			amount, err := strconv.ParseFloat(amountStr, 64)
			if err != nil {
				fmt.Println("⚠️ Ошибка: это не число!")
				continue
			}

			category := readString("Введите категорию: ")
			description := readString("Введите описание: ")

			addTransactionToDB(amount, category, description)
			fmt.Println("✅ Успешно добавлено!")

		case "2":

			transactions := getTransactionFromDB()
			if len(transactions) == 0 {
				fmt.Println("📭 Список пуст! Добавьте что-нибудь через пункт 1.")
				continue
			}

			fmt.Println("\n=== Все транзакции ===")
			for _, t := range transactions {
				fmt.Printf("%d. %s (%s): %.2f BYN | %s\n",
					t.ID, t.Description, t.Category, t.Amount, t.Date.Format("02.01.2006"))
			}

			analyzrers := []Analyzer{
				CategoryAnalyzer{},
				TopSpenderAnalyzer{},
			}

			fmt.Println("\n🚀 Запускаем анализ параллельно...")

			var result []string

			for _, analyzrer := range analyzrers {
				wg.Add(1)
				go func(a Analyzer) {
					defer wg.Done()
					report := a.Analyze(transactions)
					mu.Lock()
					result = append(result, fmt.Sprintf("📊 %s:\n%s", a.Name(), report))
					mu.Unlock()

				}(analyzrer)
			}

			wg.Wait()

			fmt.Println("\n=== ИТОГОВЫЕ ОТЧЕТЫ ===")
			for _, res := range result {
				fmt.Println(res)
			}
		case "3":
			fmt.Println("👋 До встречи!")
			return // Выход из функции main (можно и break)

		default:
			fmt.Println("⚠️ Неверный выбор, попробуй снова.")
		}
	}

}
