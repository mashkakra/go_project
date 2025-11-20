package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Структуры для данных репетиторов
type Subject struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Grade struct {
	ID        int    `json:"id"`
	GradeName string `json:"grade_name"`
}

type TimeSlot struct {
	ID        int       `json:"id"`
	Date      time.Time `json:"date"`
	StartTime string    `json:"start_time"`
	EndTime   string    `json:"end_time"`
}

type Tutor struct {
	ID              int        `json:"id"`
	FirstName       string     `json:"first_name"`
	LastName        string     `json:"last_name"`
	FullName        string     `json:"full_name"`
	Email           string     `json:"email"`
	Phone           string     `json:"phone"`
	Bio             string     `json:"bio"`
	ExperienceYears int        `json:"experience_years"`
	HourlyRate      float64    `json:"hourly_rate"`
	Subjects        []Subject  `json:"subjects"`
	Grades          []Grade    `json:"grades"`
	TimeSlots       []TimeSlot `json:"time_slots"`
	AvgRating       float64    `json:"avg_rating"`
}

type PageData struct {
	Subjects []Subject
	Tutors   map[string][]Tutor // ключ - название предмета
	Grades   []Grade
}

// Глобальная переменная для пула подключений
var db *pgxpool.Pool

// Конфигурация базы данных
type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

func LoadConfig() Config {
	return Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "00000000"),
		DBName:     getEnv("DB_NAME", "postgres"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c Config) GetConnectionString() string {
	return "postgres://" + c.DBUser + ":" + c.DBPassword + "@" + c.DBHost + ":" + c.DBPort + "/" + c.DBName + "?sslmode=disable"
}

// Инициализация подключения к базе данных
func initDB() {
	var err error
	config := LoadConfig()

	connStr := config.GetConnectionString()

	// Создаем пул подключений
	db, err = pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatal("Ошибка создания пула подключений:", err)
	}

	// Проверяем подключение
	err = db.Ping(context.Background())
	if err != nil {
		log.Fatal("Не удалось подключиться к БД:", err)
	}

	log.Println("✅ Успешное подключение к PostgreSQL с использованием pgx!")
}

// Функция для получения всех предметов
func getSubjects() ([]Subject, error) {
	rows, err := db.Query(context.Background(), `
		SELECT id, name FROM subjects ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjects []Subject
	for rows.Next() {
		var subject Subject
		err := rows.Scan(&subject.ID, &subject.Name)
		if err != nil {
			return nil, err
		}
		subjects = append(subjects, subject)
	}

	return subjects, nil
}

// Функция для получения всех классов
func getGrades() ([]Grade, error) {
	rows, err := db.Query(context.Background(), `
		SELECT id, grade_name FROM grades ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grades []Grade
	for rows.Next() {
		var grade Grade
		err := rows.Scan(&grade.ID, &grade.GradeName)
		if err != nil {
			return nil, err
		}
		grades = append(grades, grade)
	}

	return grades, nil
}

// Функция для получения репетиторов по предмету и классу
func getTutorsBySubjectAndGrade(subjectID, gradeID int) ([]Tutor, error) {
	query := `
		SELECT DISTINCT 
			t.id, t.first_name, t.last_name, t.email, t.phone, 
			t.bio, t.experience_years, t.hourly_rate,
			COALESCE(AVG(r.rating), 0) as avg_rating
		FROM tutors t
		JOIN tutor_subjects ts ON t.id = ts.tutor_id
		JOIN tutor_grades tg ON t.id = tg.tutor_id
		LEFT JOIN reviews r ON t.id = r.tutor_id
		WHERE ts.subject_id = $1 AND tg.grade_id = $2 AND t.is_active = true
		GROUP BY t.id
		ORDER BY avg_rating DESC
	`

	rows, err := db.Query(context.Background(), query, subjectID, gradeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tutors []Tutor
	for rows.Next() {
		var tutor Tutor
		err := rows.Scan(
			&tutor.ID, &tutor.FirstName, &tutor.LastName, &tutor.Email, &tutor.Phone,
			&tutor.Bio, &tutor.ExperienceYears, &tutor.HourlyRate, &tutor.AvgRating,
		)
		if err != nil {
			return nil, err
		}
		tutor.FullName = tutor.FirstName + " " + tutor.LastName

		// Загружаем предметы репетитора
		subjects, err := getTutorSubjects(tutor.ID)
		if err != nil {
			return nil, err
		}
		tutor.Subjects = subjects

		// Загружаем классы репетитора
		grades, err := getTutorGrades(tutor.ID)
		if err != nil {
			return nil, err
		}
		tutor.Grades = grades

		// Загружаем свободные окна
		timeSlots, err := getTutorTimeSlots(tutor.ID)
		if err != nil {
			return nil, err
		}
		tutor.TimeSlots = timeSlots

		tutors = append(tutors, tutor)
	}

	return tutors, nil
}

// Функция для получения всех репетиторов сгруппированных по предметам
func getAllTutorsGroupedBySubject() (map[string][]Tutor, error) {
	subjects, err := getSubjects()
	if err != nil {
		return nil, err
	}

	tutorsBySubject := make(map[string][]Tutor)

	for _, subject := range subjects {
		// Для каждого предмета получаем репетиторов (без фильтра по классу)
		query := `
			SELECT DISTINCT 
				t.id, t.first_name, t.last_name, t.email, t.phone, 
				t.bio, t.experience_years, t.hourly_rate,
				COALESCE(AVG(r.rating), 0) as avg_rating
			FROM tutors t
			JOIN tutor_subjects ts ON t.id = ts.tutor_id
			LEFT JOIN reviews r ON t.id = r.tutor_id
			WHERE ts.subject_id = $1 AND t.is_active = true
			GROUP BY t.id
			ORDER BY avg_rating DESC
		`

		rows, err := db.Query(context.Background(), query, subject.ID)
		if err != nil {
			return nil, err
		}

		var tutors []Tutor
		for rows.Next() {
			var tutor Tutor
			err := rows.Scan(
				&tutor.ID, &tutor.FirstName, &tutor.LastName, &tutor.Email, &tutor.Phone,
				&tutor.Bio, &tutor.ExperienceYears, &tutor.HourlyRate, &tutor.AvgRating,
			)
			if err != nil {
				rows.Close()
				return nil, err
			}
			tutor.FullName = tutor.FirstName + " " + tutor.LastName

			// Загружаем предметы репетитора
			subjects, err := getTutorSubjects(tutor.ID)
			if err != nil {
				rows.Close()
				return nil, err
			}
			tutor.Subjects = subjects

			// Загружаем классы репетитора
			grades, err := getTutorGrades(tutor.ID)
			if err != nil {
				rows.Close()
				return nil, err
			}
			tutor.Grades = grades

			// Загружаем свободные окна
			timeSlots, err := getTutorTimeSlots(tutor.ID)
			if err != nil {
				rows.Close()
				return nil, err
			}
			tutor.TimeSlots = timeSlots

			tutors = append(tutors, tutor)
		}
		rows.Close()

		if len(tutors) > 0 {
			tutorsBySubject[subject.Name] = tutors
		}
	}

	return tutorsBySubject, nil
}

// Вспомогательные функции для загрузки связанных данных
func getTutorSubjects(tutorID int) ([]Subject, error) {
	rows, err := db.Query(context.Background(), `
		SELECT s.id, s.name 
		FROM subjects s
		JOIN tutor_subjects ts ON s.id = ts.subject_id
		WHERE ts.tutor_id = $1
	`, tutorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjects []Subject
	for rows.Next() {
		var subject Subject
		err := rows.Scan(&subject.ID, &subject.Name)
		if err != nil {
			return nil, err
		}
		subjects = append(subjects, subject)
	}

	return subjects, nil
}

func getTutorGrades(tutorID int) ([]Grade, error) {
	rows, err := db.Query(context.Background(), `
		SELECT g.id, g.grade_name 
		FROM grades g
		JOIN tutor_grades tg ON g.id = tg.grade_id
		WHERE tg.tutor_id = $1
	`, tutorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grades []Grade
	for rows.Next() {
		var grade Grade
		err := rows.Scan(&grade.ID, &grade.GradeName)
		if err != nil {
			return nil, err
		}
		grades = append(grades, grade)
	}

	return grades, nil
}

func getTutorTimeSlots(tutorID int) ([]TimeSlot, error) {
	rows, err := db.Query(context.Background(), `
		SELECT id, date, start_time, end_time 
		FROM time_slots 
		WHERE tutor_id = $1 AND is_available = true AND date >= CURRENT_DATE
		ORDER BY date, start_time
	`, tutorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var timeSlots []TimeSlot
	for rows.Next() {
		var slot TimeSlot
		err := rows.Scan(&slot.ID, &slot.Date, &slot.StartTime, &slot.EndTime)
		if err != nil {
			return nil, err
		}
		timeSlots = append(timeSlots, slot)
	}

	return timeSlots, nil
}

// Функция для создания заявки на занятие
func createApplication(tutorID int, studentName, studentPhone, studentEmail string) error {
	_, err := db.Exec(context.Background(), `
		INSERT INTO applications (tutor_id, student_name, student_phone, student_email, created_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
	`, tutorID, studentName, studentPhone, studentEmail)
	return err
}

// Обработчики HTTP
func home(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("static/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, nil)
}

func tutor(w http.ResponseWriter, r *http.Request) {
	// Загружаем все данные для страницы записи
	subjects, err := getSubjects()
	if err != nil {
		http.Error(w, "Ошибка загрузки предметов: "+err.Error(), http.StatusInternalServerError)
		return
	}

	grades, err := getGrades()
	if err != nil {
		http.Error(w, "Ошибка загрузки классов: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tutorsBySubject, err := getAllTutorsGroupedBySubject()
	if err != nil {
		http.Error(w, "Ошибка загрузки репетиторов: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Subjects: subjects,
		Grades:   grades,
		Tutors:   tutorsBySubject,
	}

	t, err := template.ParseFiles("hello.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Обработчик для AJAX запроса получения репетиторов
func getTutorsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	subjectID := r.FormValue("subject_id")
	gradeID := r.FormValue("grade_id")

	if subjectID == "" || gradeID == "" {
		http.Error(w, "Не указаны subject_id или grade_id", http.StatusBadRequest)
		return
	}

	// Здесь можно реализовать логику для возврата JSON с репетиторами
	// Пока возвращаем простой ответ
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "success", "message": "Данные получены"}`))
}

// Обработчик для отправки заявки
func submitApplicationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	tutorID := r.FormValue("tutor_id")
	studentName := r.FormValue("student_name")
	studentPhone := r.FormValue("student_phone")
	studentEmail := r.FormValue("student_email")

	if tutorID == "" || studentName == "" || studentPhone == "" || studentEmail == "" {
		http.Error(w, "Все поля обязательны для заполнения", http.StatusBadRequest)
		return
	}

	// Здесь можно добавить создание заявки в БД
	// err := createApplication(tutorID, studentName, studentPhone, studentEmail)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "success", "message": "Заявка отправлена успешно!"}`))
}

func getRequest() {
	// Инициализируем подключение к БД
	initDB()
	defer db.Close()

	// Настраиваем обработчики
	http.HandleFunc("/", home)
	http.HandleFunc("/fortutor/", tutor)
	http.HandleFunc("/api/tutors", getTutorsHandler)
	http.HandleFunc("/api/application", submitApplicationHandler)

	log.Println("🚀 Сервер запущен на http://localhost:8080")
	log.Println("🎓 Запись к репетиторам доступна по адресу: http://localhost:8080/fortutor/")
	http.ListenAndServe(":8080", nil)
}

func main() {
	getRequest()
}
