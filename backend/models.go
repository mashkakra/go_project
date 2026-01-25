package main

import (
	"time"
)

// Subject описывает учебную дисциплину (например, Математика)
type Subject struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Grade описывает класс обучения (например, 9 класс)
type Grade struct {
	ID        int    `json:"id"`
	GradeName string `json:"grade_name"`
}

// TimeSlot описывает доступное временное окно для записи
type TimeSlot struct {
	ID          int       `json:"id"`
	TutorID     int       `json:"tutor_id"`
	Date        time.Time `json:"date"`
	DayOfWeek   int       `json:"day_of_week"` // 1-7
	DayName     string    `json:"day_name"`    // "Пн", "Вт" и т.д.
	StartTime   string    `json:"start_time"`
	EndTime     string    `json:"end_time"`
	IsAvailable bool      `json:"is_available"`
}

// Tutor — основная модель репетитора со всеми связанными данными
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
	AvgRating       float64    `json:"avg_rating"`
	Subjects        []Subject  `json:"subjects"`
	Grades          []Grade    `json:"grades"`
	TimeSlots       []TimeSlot `json:"time_slots"`
}

// PageData используется для передачи данных в HTML-шаблоны
type PageData struct {
	Subjects []Subject
	Grades   []Grade
	Tutors   map[string][]Tutor // Ключ — название предмета
}

// Config хранит параметры подключения к базе данных
type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

// LoginRequest данные из формы входа
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// User — общая структура пользователя (для сессий)
type User struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"` // "student", "tutor", "admin"
}
