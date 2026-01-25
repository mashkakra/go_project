package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
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

	login := r.FormValue("login")
	password := r.FormValue("password")

	// 1. Достаем из базы хешированный пароль и роль пользователя
	var storedHash string
	var role string

	// Ищем пользователя только по username
	err := db.QueryRow(r.Context(),
		"SELECT password, role FROM users WHERE username=$1",
		login).Scan(&storedHash, &role)
	storedHash = strings.TrimSpace(storedHash)
	if err != nil {
		// Если пользователь не найден (или ошибка БД)
		http.Redirect(w, r, "/login?error=auth", http.StatusSeeOther)
		return
	}

	// 2. Сравниваем введенный пароль с хешем из базы
	// Используем ту самую функцию CheckPasswordHash из auth.go
	log.Printf("Сверяем пароли: Введено [%s], В базе [%s]", password, storedHash)
	if !CheckPasswordHash(password, storedHash) {
		// Если пароль не подошел
		http.Redirect(w, r, "/login?error=auth", http.StatusSeeOther)
		log.Printf("no")
		return
	}

	// 3. ПАРОЛЬ ВЕРЕН. Теперь создаем сессию
	http.SetCookie(w, &http.Cookie{
		Name:     "session_user",
		Value:    login,
		Path:     "/",
		HttpOnly: true, // Защита от XSS (JS не сможет прочитать куку)
		MaxAge:   3600, // 1 час
	})

	// 4. Редирект в зависимости от роли
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

// Страница кабинета ученика
func studentDashboardHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_user")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lessons, err := getStudentLessons(cookie.Value)
	if err != nil {
		log.Printf("Ошибка получения уроков ученика: %v", err)
	}

	data := map[string]interface{}{
		"Username":   cookie.Value,
		"AllLessons": lessons,
	}

	renderTemplate(w, "student_dashboard.html", data)
}

func lessonActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Println("Ошибка: Метод не POST")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	lessonID, _ := strconv.Atoi(r.FormValue("lesson_id"))
	action := r.FormValue("action") // Проверьте, что тут именно "action"

	log.Printf("Запрос: ID=%d, Action=%s", lessonID, action)

	if action == "cancel" {
		err := declineLesson(lessonID)
		if err != nil {
			log.Printf("Ошибка в БД: %v", err)
			http.Error(w, "Ошибка БД", 500)
			return
		}
		log.Println("Статус успешно обновлен")
		http.Redirect(w, r, "/student/dashboard", http.StatusSeeOther)
		return
	}

	// Если мы попали сюда, значит action не равен "cancel"
	log.Printf("Action '%s' не обработан, редирект на главную", action)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func tutorActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}

	lessonID, _ := strconv.Atoi(r.FormValue("lesson_id"))
	action := r.FormValue("action")

	var err error
	switch action {
	case "accept":
		err = acceptLesson(lessonID)
	case "decline":
		err = declineLesson(lessonID)
	case "cancel":
		err = cancelLesson(lessonID)
	case "reschedule":
		// Просто перенаправляем на страницу выбора времени
		http.Redirect(w, r, fmt.Sprintf("/tutor/reschedule?lesson_id=%d", lessonID), http.StatusSeeOther)
		return
	}

	if err != nil {
		log.Printf("Ошибка действия %s: %v", action, err)
	}

	http.Redirect(w, r, "/tutor/dashboard", http.StatusSeeOther)
}

func reschedulePageHandler(w http.ResponseWriter, r *http.Request) {
	lessonID := r.URL.Query().Get("lesson_id")
	cookie, _ := r.Cookie("session_user")

	// Получаем свободные слоты
	slots, err := getAvailableSlotsForTutor(cookie.Value)
	if err != nil { /* обработка ошибки */
	}

	data := map[string]interface{}{
		"LessonID":       lessonID,
		"AvailableSlots": slots,
	}
	renderTemplate(w, "reschedule.html", data)
}

func confirmRescheduleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/tutor/dashboard", http.StatusSeeOther)
		return
	}

	lessonID, _ := strconv.Atoi(r.FormValue("lesson_id"))
	newSlotID, _ := strconv.Atoi(r.FormValue("new_slot_id"))

	err := updateLessonSlot(lessonID, newSlotID)
	if err != nil {
		log.Printf("Ошибка переноса: %v", err)
		http.Error(w, "Не удалось перенести занятие", http.StatusInternalServerError)
		return
	}

	// После успешного переноса возвращаем репетитора в кабинет
	http.Redirect(w, r, "/tutor/dashboard", http.StatusSeeOther)
}
func createAndRescheduleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/tutor/dashboard", http.StatusSeeOther)
		return
	}

	// Извлекаем данные из формы
	lessonID, _ := strconv.Atoi(r.FormValue("lesson_id"))
	newDate := r.FormValue("date")
	newTime := r.FormValue("time")

	// Получаем username репетитора из куки
	cookie, _ := r.Cookie("session_user")
	tutorUsername := cookie.Value

	// Вызываем функцию базы данных
	err := createAndAssignNewSlot(lessonID, tutorUsername, newDate, newTime)
	if err != nil {
		log.Printf("Ошибка при создании и переносе: %v", err)
		http.Error(w, "Ошибка при создании нового времени", 500)
		return
	}

	// Возвращаемся в кабинет
	http.Redirect(w, r, "/tutor/dashboard", http.StatusSeeOther)
}

func adminCreateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		return
	}

	// Читаем данные из формы (имена должны совпадать с name="..." в HTML)
	login := r.FormValue("new_login")       // Было name="new_login"
	password := r.FormValue("new_password") // Было name="new_password"
	role := r.FormValue("role")             // Было name="role"
	// Логируем для проверки в терминале
	log.Printf("Получены данные из формы: login=%s, role=%s", login, role)

	if login == "" || password == "" {
		http.Error(w, "Логин и пароль обязательны", http.StatusBadRequest)
		return
	}

	// Вызываем функцию создания пользователя (с хешированием внутри)
	err := CreateUser(r.Context(), login, password, role)
	if err != nil {
		log.Printf("Ошибка при создании пользователя в БД: %v", err)
		http.Error(w, "Ошибка: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Пользователь %s успешно создан!", login)

	// Возвращаемся обратно в админку
	http.Redirect(w, r, "/admin/dashboard?success=user_created", http.StatusSeeOther)
}

func adminCreateStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		return
	}

	// Получаем данные из вашей HTML-формы
	lessonIDStr := r.FormValue("lesson_id")
	login := r.FormValue("new_login")
	password := r.FormValue("new_password")

	lessonID, err := strconv.Atoi(lessonIDStr)
	if err != nil || login == "" || password == "" {
		http.Error(w, "Некорректные данные формы", http.StatusBadRequest)
		return
	}

	// Вызываем нашу функцию
	err = CreateStudent(r.Context(), lessonID, login, password)
	if err != nil {
		log.Printf("Ошибка выдачи доступа: %v", err)
		http.Error(w, "Ошибка: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Доступ выдан! Ученик: %s, Урок ID: %d", login, lessonID)

	// Возвращаемся в админку
	http.Redirect(w, r, "/admin/dashboard?success=access_granted", http.StatusSeeOther)
}
