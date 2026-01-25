package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

// --- МЕТОДЫ ДЛЯ СПРАВОЧНИКОВ ---

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
		var s Subject
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, err
		}
		subjects = append(subjects, s)
	}
	return subjects, nil
}

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
		var g Grade
		if err := rows.Scan(&g.ID, &g.GradeName); err != nil {
			return nil, err
		}
		grades = append(grades, g)
	}
	return grades, nil
}

// --- МЕТОДЫ ДЛЯ ТЬЮТОРОВ ---

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
	return fetchTutors(query, subjectID, gradeID)
}

func getAllTutorsGroupedBySubject() (map[string][]Tutor, error) {
	subjects, err := getSubjects()
	if err != nil {
		return nil, err
	}

	tutorsBySubject := make(map[string][]Tutor)
	for _, subject := range subjects {
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
		tutors, err := fetchTutors(query, subject.ID)
		if err != nil {
			return nil, err
		}
		if len(tutors) > 0 {
			tutorsBySubject[subject.Name] = tutors
		}
	}
	return tutorsBySubject, nil
}

// --- ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ (ВНУТРЕННИЕ) ---

// fetchTutors — общая логика сканирования строк для разных запросов тьюторов
func fetchTutors(query string, args ...interface{}) ([]Tutor, error) {
	rows, err := db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tutors []Tutor
	for rows.Next() {
		var t Tutor
		err := rows.Scan(
			&t.ID, &t.FirstName, &t.LastName, &t.Email, &t.Phone,
			&t.Bio, &t.ExperienceYears, &t.HourlyRate, &t.AvgRating,
		)
		if err != nil {
			return nil, err
		}
		t.FullName = t.FirstName + " " + t.LastName

		// Обогащаем данными (Lazy Loading)
		t.Subjects, _ = getTutorSubjects(t.ID)
		t.Grades, _ = getTutorGrades(t.ID)
		t.TimeSlots, _ = getTutorTimeSlots(t.ID)

		tutors = append(tutors, t)
	}
	return tutors, nil
}

func getTutorSubjects(tutorID int) ([]Subject, error) {
	rows, err := db.Query(context.Background(), `
		SELECT s.id, s.name FROM subjects s
		JOIN tutor_subjects ts ON s.id = ts.subject_id
		WHERE ts.tutor_id = $1
	`, tutorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Subject
	for rows.Next() {
		var s Subject
		rows.Scan(&s.ID, &s.Name)
		res = append(res, s)
	}
	return res, nil
}

func getTutorGrades(tutorID int) ([]Grade, error) {
	rows, err := db.Query(context.Background(), `
		SELECT g.id, g.grade_name FROM grades g
		JOIN tutor_grades tg ON g.id = tg.grade_id
		WHERE tg.tutor_id = $1
	`, tutorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Grade
	for rows.Next() {
		var g Grade
		rows.Scan(&g.ID, &g.GradeName)
		res = append(res, g)
	}
	return res, nil
}

// --- МЕТОДЫ ЗАПИСИ ---

// Получить все уведомления о забытых паролях (для админа)
// Получить все уведомления о забытых паролях (для админа)
func getAdminNotifications() ([]map[string]interface{}, error) {
	rows, err := db.Query(context.Background(), "SELECT id, message, created_at FROM admin_notifications ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []map[string]interface{}
	for rows.Next() {
		var id int
		var msg string
		var t time.Time
		if err := rows.Scan(&id, &msg, &t); err != nil {
			return nil, err
		}
		notes = append(notes, map[string]interface{}{
			"ID":      id,
			"Message": msg,
			"Time":    t.Format("15:04 02.01.2006"),
		})
	}
	return notes, nil
}

// Получить заявки (для админа все, для репетитора — только его)
func getApplications(tutorUsername string) ([]map[string]interface{}, error) {
	var rows interface{}
	var err error

	if tutorUsername != "" {
		// Фильтр для репетитора
		rows, err = db.Query(context.Background(), `
			SELECT a.id, a.student_name, a.student_phone, t.last_name 
			FROM applications a 
			JOIN tutors t ON a.tutor_id = t.id
			WHERE t.username = $1`, tutorUsername)
	} else {
		// Для админа (все заявки)
		rows, err = db.Query(context.Background(), `
			SELECT a.id, a.student_name, a.student_phone, t.last_name 
			FROM applications a 
			JOIN tutors t ON a.tutor_id = t.id`)
	}

	if err != nil {
		return nil, err
	}

	// Приводим к типу pgx.Rows, чтобы закрыть и прочитать
	pgxRows := rows.(interface {
		Next() bool
		Scan(dest ...any) error
		Close()
	})
	defer pgxRows.Close()

	var apps []map[string]interface{}
	for pgxRows.Next() {
		var id int
		var name, phone, tutorName string
		if err := pgxRows.Scan(&id, &name, &phone, &tutorName); err != nil {
			return nil, err
		}
		apps = append(apps, map[string]interface{}{
			"ID":           id,
			"StudentName":  name,
			"StudentPhone": phone,
			"TutorName":    tutorName,
		})
	}
	return apps, nil
}

// Функция для создания нового пользователя (выдача логина админом)
// CreateUser создает нового пользователя с захешированным паролем
func CreateUser(ctx context.Context, username, plainPassword, role string) error {
	// 1. Хешируем пароль
	hashedPassword, err := HashPassword(plainPassword)
	if err != nil {
		return err
	}

	// 2. Сохраняем в базу (безопасно, через $1, $2, $3)
	query := `INSERT INTO users (username, password, role) VALUES ($1, $2, $3)`
	_, err = db.Exec(ctx, query, username, hashedPassword, role)

	return err
}

// Получить расписание конкретного ученика
// Получение расписания ученика
func getStudentLessons(username string) ([]map[string]interface{}, error) {
	query := `
        SELECT 
            l.id, 
            l.status, 
            t.first_name || ' ' || t.last_name as tutor_name, 
            ts.date, 
            ts.start_time, ts.day_of_week
        FROM lessons l
        JOIN students s ON l.student_id = s.id
        JOIN users u ON s.user_id = u.id
        JOIN tutors t ON l.tutor_id = t.id
        JOIN time_slots ts ON l.timeslot_id = ts.id
        WHERE u.username = $1
        ORDER BY ts.date ASC
    `
	// Используем log для отладки
	log.Printf("Запрос уроков для ученика: %s", username)

	rows, err := db.Query(context.Background(), query, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lessons []map[string]interface{}
	for rows.Next() {
		var id int
		var status, tutorName, startTime, dayWeek string
		var date time.Time

		if err := rows.Scan(&id, &status, &tutorName, &date, &startTime, &dayWeek); err != nil {
			log.Printf("Ошибка Scan в getStudentLessons: %v", err)
			continue
		}

		lessons = append(lessons, map[string]interface{}{
			"ID":        id,
			"TutorName": tutorName,
			"Date":      date.Format("02.01.2006"),
			"StartTime": startTime,
			"Status":    status,
			"DayWeek":   dayWeek,
		})
	}
	log.Printf("Найдено уроков для ученика %s: %d", username, len(lessons))
	return lessons, nil
}

// Обновленная функция отмены занятия

// Функция переноса занятия (уже учитывает освобождение старого и занятие нового)

// Получаем только ДОСТУПНЫЕ слоты для конкретного репетитора
// Получаем только ДОСТУПНЫЕ слоты для конкретного репетитора
func getTutorLessons(tutorUsername string) ([]map[string]interface{}, error) {
	query := `
        SELECT 
            l.id, 
            l.student_name, 
            ts.date, 
            ts.start_time, ts.day_of_week
        FROM lessons l
        JOIN tutors t ON l.tutor_id = t.id
        JOIN time_slots ts ON l.timeslot_id = ts.id
        WHERE t.username = $1 AND l.status = 'pending'
    `
	rows, err := db.Query(context.Background(), query, tutorUsername)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса: %v", err)
	}
	defer rows.Close()

	var lessons []map[string]interface{}
	for rows.Next() {
		var id int
		var name, startTime, dayWeek string
		var dateRaw interface{} // Используем interface{}, чтобы не упасть на типе DATE

		if err := rows.Scan(&id, &name, &dateRaw, &startTime, &dayWeek); err != nil {
			// Если ошибка здесь, вы увидите её в терминале
			log.Printf("!!! Ошибка сканирования строки: %v", err)
			continue
		}

		// Безопасное форматирование даты
		dateStr := ""
		if t, ok := dateRaw.(time.Time); ok {
			dateStr = t.Format("02.01.2006")
		} else {
			dateStr = fmt.Sprintf("%v", dateRaw)
		}

		lessons = append(lessons, map[string]interface{}{
			"ID":          id,
			"StudentName": name,
			"Date":        dateStr,
			"StartTime":   startTime,
			"DayWeek":     dayWeek,
		})
	}

	// ВАЖНО: Проверка на ошибки после завершения цикла
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации строк: %v", err)
	}

	log.Printf("Найдено заявок (pending) для %s: %d", tutorUsername, len(lessons))
	return lessons, nil
}
func getTutorTimeSlots(tutorID int) ([]TimeSlot, error) {
	// ОБНОВЛЕННЫЙ ЗАПРОС С ФИЛЬТРАЦИЕЙ 'pending' заявок
	rows, err := db.Query(context.Background(), `
        SELECT ts.id, ts.date, ts.start_time, ts.end_time 
        FROM time_slots ts
        LEFT JOIN lessons l ON ts.id = l.timeslot_id AND l.status IN ('scheduled', 'pending')
        WHERE ts.tutor_id = $1 
          AND ts.is_available = true 
          AND l.id IS NULL
        ORDER BY ts.date, ts.start_time
    `, tutorID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []TimeSlot
	for rows.Next() {
		var s TimeSlot
		if err := rows.Scan(&s.ID, &s.Date, &s.StartTime, &s.EndTime); err != nil {
			return nil, err
		}
		res = append(res, s)
	}
	return res, nil
}

// Бронирование занятия через поиск
func bookLesson(studentID, tutorID, slotID int) error {
	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Проверяем, не занял ли кто-то этот слот, пока мы думали
	var available bool
	err = tx.QueryRow(ctx, "SELECT is_available FROM time_slots WHERE id = $1 FOR UPDATE", slotID).Scan(&available)
	if err != nil || !available {
		return fmt.Errorf("слот уже занят")
	}

	// 2. Создаем занятие
	_, err = tx.Exec(ctx, `
		INSERT INTO lessons (student_id, tutor_id, timeslot_id, status) 
		VALUES ($1, $2, $3, 'scheduled')`, studentID, tutorID, slotID)
	if err != nil {
		return err
	}

	// 3. Помечаем слот как ЗАНЯТЫЙ
	_, err = tx.Exec(ctx, "UPDATE time_slots SET is_available = false WHERE id = $1", slotID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func createLessonRequest(tutorID, slotID int, name, phone, email string) error {
	// ВАЖНО: Мы сохраняем контактные данные прямо в таблицу lessons.
	// Убедитесь, что у вас есть эти колонки в таблице (student_name, student_phone, student_email)
	query := `
        INSERT INTO lessons (tutor_id, timeslot_id, student_name, student_phone, student_email, status) 
        VALUES ($1, $2, $3, $4, $5, 'pending')
    `
	_, err := db.Exec(context.Background(), query, tutorID, slotID, name, phone, email)
	return err
}

func acceptLesson(lessonID int) error {
	ctx := context.Background()
	// Начинаем транзакцию
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	// В случае ошибки откатываем изменения
	defer tx.Rollback(ctx)

	var slotID int
	// 1. Узнаем, какой именно временной слот привязан к этому уроку
	err = tx.QueryRow(ctx, "SELECT timeslot_id FROM lessons WHERE id = $1", lessonID).Scan(&slotID)
	if err != nil {
		return err
	}

	// 2. Устанавливаем уроку статус 'scheduled' (запланирован)
	_, err = tx.Exec(ctx, "UPDATE lessons SET status = 'scheduled' WHERE id = $1", lessonID)
	if err != nil {
		return err
	}

	// 3. ПОМЕЧАЕМ СЛОТ КАК ЗАНЯТЫЙ (теперь он исчезнет из поиска окончательно)
	_, err = tx.Exec(ctx, "UPDATE time_slots SET is_available = false WHERE id = $1", slotID)
	if err != nil {
		return err
	}

	// Подтверждаем транзакцию
	return tx.Commit(ctx)
}

func declineLesson(lessonID int) error {
	// Просто переводим в статус 'declined'
	// Слот времени (time_slots) трогать не нужно, он и так был true
	query := "UPDATE lessons SET status = 'declined' WHERE id = $1"
	_, err := db.Exec(context.Background(), query, lessonID)
	return err
}
func getConfirmedLessons(tutorUsername string) ([]map[string]interface{}, error) {
	// ПОРЯДОК: 1.id, 2.student_name, 3.student_phone, 4.date, 5.start_time
	query := `
        SELECT l.id, l.student_name, l.student_phone, ts.date, ts.start_time, ts.day_of_week
        FROM lessons l
        JOIN tutors t ON l.tutor_id = t.id
        JOIN time_slots ts ON l.timeslot_id = ts.id
        WHERE t.username = $1 AND l.status = 'scheduled'
    `
	rows, err := db.Query(context.Background(), query, tutorUsername)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lessons []map[string]interface{}
	for rows.Next() {
		var id int
		var name, phone, startTime, dayWeek string
		var date time.Time

		// СТРОГО соблюдаем порядок из SELECT выше!
		err := rows.Scan(&id, &name, &phone, &date, &startTime, &dayWeek)
		if err != nil {
			return nil, err
		}

		lessons = append(lessons, map[string]interface{}{
			"ID":           id,
			"StudentName":  name,
			"StudentPhone": phone,
			"Date":         date.Format("02.01.2006"),
			"StartTime":    startTime,
			"Status":       "scheduled",
			"DayWeek":      dayWeek,
		})
	}
	return lessons, nil
}
func getAllLessonsForAdmin() ([]map[string]interface{}, error) {
	query := `
        SELECT 
            l.id, l.student_name, l.student_phone, l.status, l.student_id,
            t.last_name as tutor_name, ts.day_of_week,
            format_lesson_date(ts.date, ts.is_recurring) as schedule_display,
            ts.start_time, ts.is_recurring,
            u.username as acc_login,
            u.password as acc_password
        FROM lessons l
        LEFT JOIN tutors t ON l.tutor_id = t.id
        LEFT JOIN time_slots ts ON l.timeslot_id = ts.id
        LEFT JOIN students s ON l.student_id = s.id
        LEFT JOIN users u ON s.user_id = u.id
        ORDER BY ts.is_recurring DESC, ts.date DESC
    `

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id int
		var sName, sPhone, status, tutorName, scheduleDisplay, startTime, dayWeek string
		var isRecurring bool
		var studentID *int
		var accLogin, accPassword *string

		err := rows.Scan(
			&id, &sName, &sPhone, &status, &studentID,
			&tutorName, &dayWeek, &scheduleDisplay, &startTime, &isRecurring,
			&accLogin, &accPassword,
		)
		if err != nil {
			return nil, err
		}

		row := map[string]interface{}{
			"ID":              id,
			"StudentName":     sName,
			"StudentPhone":    sPhone,
			"Status":          status,
			"TutorName":       tutorName,
			"ScheduleDisplay": scheduleDisplay, // Теперь здесь строка из SQL функции
			"Day":             dayWeek,
			"StartTime":       startTime,
			"IsRecurring":     isRecurring,
		}

		if studentID != nil {
			row["StudentID"] = *studentID
		}
		if accLogin != nil {
			row["AccLogin"] = *accLogin
		}
		if accPassword != nil {
			row["AccPassword"] = *accPassword
		}

		results = append(results, row)
	}
	return results, nil
}

func cancelLesson(lessonID int) error {
	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var slotID int
	tx.QueryRow(ctx, "SELECT timeslot_id FROM lessons WHERE id = $1", lessonID).Scan(&slotID)

	// 1. Статус урока — отменен
	tx.Exec(ctx, "UPDATE lessons SET status = 'cancelled' WHERE id = $1", lessonID)
	// 2. Слот времени снова СВОБОДЕН для записи
	tx.Exec(ctx, "UPDATE time_slots SET is_available = true WHERE id = $1", slotID)

	return tx.Commit(ctx)
}

func getAvailableSlotsForTutor(tutorUsername string) ([]map[string]interface{}, error) {
	// Выбираем только свободные слоты конкретного репетитора
	query := `
        SELECT ts.id, ts.date, ts.start_time
        FROM time_slots ts
        JOIN tutors t ON ts.tutor_id = t.id
        WHERE t.username = $1 AND ts.is_available = true
        ORDER BY ts.date ASC, ts.start_time ASC
    `

	rows, err := db.Query(context.Background(), query, tutorUsername)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения свободных слотов: %v", err)
	}
	defer rows.Close()

	var slots []map[string]interface{}
	for rows.Next() {
		var id int
		var startTime string
		var date time.Time

		if err := rows.Scan(&id, &date, &startTime); err != nil {
			return nil, err
		}

		slots = append(slots, map[string]interface{}{
			"ID":        id,
			"Date":      date.Format("02.01.2006"),
			"StartTime": startTime,
		})
	}

	return slots, nil
}
func updateLessonSlot(lessonID int, newSlotID int) error {
	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Находим текущий слот занятия, чтобы его освободить
	var oldSlotID int
	err = tx.QueryRow(ctx, "SELECT timeslot_id FROM lessons WHERE id = $1", lessonID).Scan(&oldSlotID)
	if err != nil {
		return err
	}

	// 2. Делаем старый слот СВОБОДНЫМ
	_, err = tx.Exec(ctx, "UPDATE time_slots SET is_available = true WHERE id = $1", oldSlotID)
	if err != nil {
		return err
	}

	// 3. Привязываем урок к новому слоту и меняем статус (если нужно)
	_, err = tx.Exec(ctx, "UPDATE lessons SET timeslot_id = $1, status = 'scheduled' WHERE id = $2", newSlotID, lessonID)
	if err != nil {
		return err
	}

	// 4. Делаем новый слот ЗАНЯТЫМ
	_, err = tx.Exec(ctx, "UPDATE time_slots SET is_available = false WHERE id = $1", newSlotID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
func createAndAssignNewSlot(lessonID int, tutorUsername string, date string, timeStr string) error {
	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Получаем tutor_id по username
	var tutorID int
	err = tx.QueryRow(ctx, "SELECT id FROM tutors WHERE username = $1", tutorUsername).Scan(&tutorID)
	if err != nil {
		return err
	}

	// 2. Создаем НОВЫЙ слот и получаем его ID
	var newSlotID int

	queryInsert := `
        INSERT INTO time_slots (tutor_id, date, start_time, end_time, is_available) 
        VALUES ($1, $2, $3, ($3::time + interval '1 hour'), false) 
        RETURNING id`

	err = tx.QueryRow(ctx, queryInsert, tutorID, date, timeStr).Scan(&newSlotID)
	if err != nil {
		return fmt.Errorf("ошибка вставки слота: %v", err)
	}
	// 3. Находим СТАРЫЙ слот урока, чтобы его освободить
	var oldSlotID int
	err = tx.QueryRow(ctx, "SELECT timeslot_id FROM lessons WHERE id = $1", lessonID).Scan(&oldSlotID)
	if err != nil {
		return err
	}

	// 4. Делаем старый слот СВОБОДНЫМ
	_, err = tx.Exec(ctx, "UPDATE time_slots SET is_available = true WHERE id = $1", oldSlotID)
	if err != nil {
		return err
	}

	// 5. Привязываем урок к НОВОМУ слоту
	_, err = tx.Exec(ctx, "UPDATE lessons SET timeslot_id = $1, status = 'scheduled' WHERE id = $2", newSlotID, lessonID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func CreateStudent(ctx context.Context, lessonID int, login, password string) error {
	// 1. Хешируем пароль
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 2. Достаем данные ученика, которые уже были в уроке
	var name, phone string
	err = tx.QueryRow(ctx, "SELECT student_name, student_phone FROM lessons WHERE id = $1", lessonID).Scan(&name, &phone)
	if err != nil {
		return fmt.Errorf("урок не найден: %v", err)
	}

	// 3. Создаем запись в users
	var userID int
	err = tx.QueryRow(ctx,
		"INSERT INTO users (username, password, role) VALUES ($1, $2, 'student') RETURNING id",
		login, hashedPassword).Scan(&userID)
	if err != nil {
		return fmt.Errorf("ошибка создания аккаунта (возможно, логин занят): %v", err)
	}

	// 4. Создаем профиль в students
	var studentID int
	err = tx.QueryRow(ctx,
		"INSERT INTO students (user_id, full_name, phone) VALUES ($1, $2, $3) RETURNING id",
		userID, name, phone).Scan(&studentID)
	if err != nil {
		return fmt.Errorf("ошибка создания профиля студента: %v", err)
	}

	// 5. Привязываем созданного студента к уроку
	_, err = tx.Exec(ctx, "UPDATE lessons SET student_id = $1 WHERE id = $2", studentID, lessonID)
	if err != nil {
		return fmt.Errorf("ошибка привязки урока: %v", err)
	}

	return tx.Commit(ctx)
}
