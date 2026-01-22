package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"time"
)

// --- HTML ОБРАБОТЧИКИ ---

// home отображает главную страницу
func home(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Ошибка загрузки шаблона: "+err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, nil)
}

// tutor отображает страницу со списком репетиторов и фильтрами
func tutor(w http.ResponseWriter, r *http.Request) {
	subjects, err := getSubjects()
	if err != nil {
		http.Error(w, "Ошибка загрузки предметов", http.StatusInternalServerError)
		return
	}

	grades, err := getGrades()
	if err != nil {
		http.Error(w, "Ошибка загрузки классов", http.StatusInternalServerError)
		return
	}

	tutorsBySubject, err := getAllTutorsGroupedBySubject()
	if err != nil {
		http.Error(w, "Ошибка загрузки репетиторов", http.StatusInternalServerError)
		return
	}

	data := PageData{
		Subjects: subjects,
		Grades:   grades,
		Tutors:   tutorsBySubject,
	}

	t, err := template.ParseFiles("templates/hello.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, data)
}

// adminHandler отображает панель администратора
func adminHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/admin.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, nil)
}

// --- API ОБРАБОТЧИКИ (JSON) ---

// getTutorsHandler обрабатывает AJAX-поиск репетиторов
func getTutorsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	subID, _ := strconv.Atoi(r.FormValue("subject_id"))
	gradeID, _ := strconv.Atoi(r.FormValue("grade_id"))

	tutors, err := getTutorsBySubjectAndGrade(subID, gradeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"tutors": tutors,
	})
}

// submitApplicationHandler сохраняет заявку от студента
func submitApplicationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	tutorID, _ := strconv.Atoi(r.FormValue("tutor_id"))
	name := r.FormValue("student_name")
	phone := r.FormValue("student_phone")
	email := r.FormValue("student_email")

	if name == "" || phone == "" {
		http.Error(w, "Имя и телефон обязательны", http.StatusBadRequest)
		return
	}

	err := createApplication(tutorID, name, phone, email)
	if err != nil {
		http.Error(w, "Ошибка БД", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// addTestSlotHandler — технический эндпоинт для тестов
func addTestSlotHandler(w http.ResponseWriter, r *http.Request) {
	tutorID := 3 // Пример ID
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	_, err := db.Exec(r.Context(), `
		INSERT INTO time_slots (tutor_id, date, start_time, end_time, is_available)
		VALUES ($1, $2, $3, $4, true)
	`, tutorID, tomorrow, "15:00", "16:30")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Слот добавлен"))
}
