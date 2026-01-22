package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
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

	// Считываем ID репетитора и ID слота
	tutorID, _ := strconv.Atoi(r.FormValue("tutor_id"))
	slotID, _ := strconv.Atoi(r.FormValue("slot_id")) // ОБЯЗАТЕЛЬНО получаем slot_id

	name := r.FormValue("student_name")
	phone := r.FormValue("student_phone")
	email := r.FormValue("student_email")

	if name == "" || phone == "" || slotID == 0 {
		http.Error(w, "Имя, телефон и время занятия обязательны", http.StatusBadRequest)
		return
	}

	// Вызываем обновленную функцию (мы поправим её ниже в Repository)
	// Передаем все данные, чтобы они не потерялись
	err := createLessonRequest(tutorID, slotID, name, phone, email)
	if err != nil {
		fmt.Println("Ошибка БД:", err) // Для отладки в консоли сервера
		http.Error(w, "Ошибка сохранения заявки", http.StatusInternalServerError)
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

// loginPage — отображение страницы входа
func loginPage(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("templates/login.html")
	t.Execute(w, nil)
}

// loginHandler — обработка входа (упрощенная версия для курсовой)

// forgotPasswordHandler — уведомление админу
func forgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Теперь получаем 'login' вместо 'email'
	login := r.FormValue("login")

	if login == "" {
		http.Error(w, "Логин не может быть пустым", http.StatusBadRequest)
		return
	}

	// Формируем текст сообщения для админа
	message := "Запрос на восстановление пароля. Пользователь (логин): " + login +
		". Напоминание: пароль должен состоять из 8 символов."

	// Записываем уведомление в базу
	_, err := db.Exec(r.Context(),
		"INSERT INTO admin_notifications (message, created_at) VALUES ($1, CURRENT_TIMESTAMP)",
		message)

	if err != nil {
		log.Printf("Ошибка записи уведомления: %v", err)
		http.Error(w, "Ошибка при отправке запроса", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("Запрос отправлен администратору. Он свяжется с вами для сброса пароля до 8 символов."))
}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	login := r.FormValue("login")       // Получаем логин
	password := r.FormValue("password") // Получаем пароль

	// Валидация: пароль должен быть ровно 8 символов
	if len(password) != 8 {
		http.Error(w, "Пароль должен содержать ровно 8 символов", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session_user", // Имя должно быть ОДИНАКОВЫМ везде
		Value:    login,
		Path:     "/",  // Доступно во всем приложении
		HttpOnly: true, // Защита от кражи скриптами
		MaxAge:   3600, // Живет 1 час
	})
	var role string
	// Ищем пользователя по логину и паролю
	err := db.QueryRow(r.Context(),
		"SELECT role FROM users WHERE username=$1 AND password=$2",
		login, password).Scan(&role)

	if err != nil {
		// Если не нашли — возвращаем на страницу входа с ошибкой
		http.Redirect(w, r, "/login?error=auth", http.StatusSeeOther)
		return
	}

	// Редирект в зависимости от роли
	switch role {
	case "admin":
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
	case "tutor":
		http.Redirect(w, r, "/tutor/dashboard", http.StatusSeeOther)
	case "student":
		http.Redirect(w, r, "/student/dashboard", http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// Кабинет администратора
func adminDashboard(w http.ResponseWriter, r *http.Request) {
	// 1. Получаем уведомления о паролях
	notes, _ := getAdminNotifications()
	// 2. Получаем все заявки на обучение
	lessons, _ := getAllLessonsForAdmin()
	apps, _ := getApplications("") // пустая строка = все

	data := map[string]interface{}{
		"Notifications": notes,
		"Applications":  apps,
		"AllLessons":    lessons, // Ключ "AllLessons"
	}

	t, _ := template.ParseFiles("templates/admin_dashboard.html")
	t.Execute(w, data)
}
func renderTemplate(w http.ResponseWriter, tmplName string, data interface{}) {
	// Укажите путь к папке с вашими HTML файлами
	lp := filepath.Join("templates", tmplName)

	tmpl, err := template.ParseFiles(lp)
	if err != nil {
		http.Error(w, "Ошибка загрузки шаблона: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Ошибка отрисовки шаблона: "+err.Error(), http.StatusInternalServerError)
	}
}

// Кабинет репетитора
func tutorDashboard(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_user")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	username := cookie.Value

	// 1. Получаем новые заявки (pending)
	pending, err := getTutorLessons(username)
	if err != nil {
		log.Printf("Ошибка получения новых заявок: %v", err)
	}

	// 2. Получаем подтвержденные уроки (scheduled)
	confirmed, err := getConfirmedLessons(username)
	if err != nil {
		log.Printf("Ошибка получения подтвержденных уроков: %v", err)
	}

	// 3. Подготовка данных для шаблона
	data := map[string]interface{}{
		"Username":       username,
		"AllLessons":     confirmed, // Заполняем таблицу "Мои ученики" через этот ключ
		"PendingLessons": pending,   // Заполняем карточки "Новые запросы"
		"Applications":   confirmed, // Для списка внизу (дублируем, чтобы работало везде)
	}

	renderTemplate(w, "tutor_dashboard.html", data)
}
func createUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	newLogin := r.FormValue("new_login")
	newPass := r.FormValue("new_password")
	role := r.FormValue("role")

	// Валидация пароля (ровно 8 символов)
	if len(newPass) != 8 {
		http.Error(w, "Пароль должен быть строго 8 символов", http.StatusBadRequest)
		return
	}

	err := createUser(newLogin, newPass, role)
	if err != nil {
		http.Error(w, "Ошибка создания: возможно логин занят", http.StatusInternalServerError)
		return
	}

	// Возвращаемся обратно в админку
	http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
}

// Страница кабинета ученика
func studentDashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Для теста берем ID = 1 (admin_boss). В реальности берется из сессии после логина.
	studentID := 3

	lessons, err := getStudentLessons(studentID)
	if err != nil {
		http.Error(w, "Ошибка базы данных", 500)
		return
	}

	t, _ := template.ParseFiles("templates/student_dashboard.html")
	t.Execute(w, map[string]interface{}{
		"Lessons": lessons,
	})
}

func lessonActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}

	lessonID, _ := strconv.Atoi(r.FormValue("lesson_id"))
	action := r.FormValue("action")

	if action == "cancel" {
		// Вызываем функцию с транзакцией, которая освободит слот
		err := cancelLesson(lessonID)
		if err != nil {
			log.Printf("Ошибка отмены: %v", err)
			http.Error(w, "Ошибка при отмене", 500)
			return
		}
		http.Redirect(w, r, "/student/dashboard", http.StatusSeeOther)
	} else if action == "reschedule" {
		// Перенаправляем на страницу выбора нового времени
		http.Redirect(w, r, "/student/reschedule?lesson_id="+strconv.Itoa(lessonID), http.StatusSeeOther)
	}
}

func bookLessonHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}

	// В идеале берем из сессии, для теста ID = 1 (ученик)
	studentID := 1
	slotID, _ := strconv.Atoi(r.FormValue("slot_id"))
	tutorID, _ := strconv.Atoi(r.FormValue("tutor_id"))

	err := bookLesson(studentID, tutorID, slotID)
	if err != nil {
		http.Error(w, "Не удалось забронировать: "+err.Error(), http.StatusConflict)
		return
	}

	// После успешной записи отправляем в личный кабинет ученика
	http.Redirect(w, r, "/student/dashboard", http.StatusSeeOther)
}

func tutorActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	lessonID, _ := strconv.Atoi(r.FormValue("lesson_id"))
	action := r.FormValue("action")

	var err error
	if action == "accept" {
		err = acceptLesson(lessonID) // Меняет статус на 'scheduled' и is_available = false
	} else if action == "decline" {
		err = declineLesson(lessonID) // Меняет статус на 'declined', слот остается свободным
	}

	if err != nil {
		http.Error(w, "Ошибка при обработке: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем репетитора обратно в кабинет
	http.Redirect(w, r, "/tutor/dashboard", http.StatusSeeOther)
}
