package main

import (
	"context"
	"fmt"
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
func createUser(username, password, role string) error {
	_, err := db.Exec(context.Background(),
		"INSERT INTO users (username, password, role) VALUES ($1, $2, $3)",
		username, password, role)
	return err
}

// Получить расписание конкретного ученика
// Получение расписания ученика
func getStudentLessons(studentID int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			l.id, 
			t.first_name, t.last_name, 
			ts.date, ts.start_time, 
			l.status
		FROM lessons l
		JOIN tutors t ON l.tutor_id = t.id
		JOIN time_slots ts ON l.timeslot_id = ts.id
		WHERE l.student_id = $1
		ORDER BY ts.date DESC, ts.start_time DESC
	`
	rows, err := db.Query(context.Background(), query, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lessons []map[string]interface{}
	for rows.Next() {
		var id int
		var fName, lName, startTime, status string
		var date time.Time
		if err := rows.Scan(&id, &fName, &lName, &date, &startTime, &status); err != nil {
			return nil, err
		}

		lessons = append(lessons, map[string]interface{}{
			"ID":        id,
			"TutorName": fName + " " + lName,
			"Date":      date.Format("02.01.2006"),
			"Time":      startTime,
			"Status":    status,
		})
	}
	return lessons, nil
}

// Обновленная функция отмены занятия
func cancelLesson(lessonID int) error {
	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Получаем ID слота, который сейчас занят этим уроком
	var slotID int
	err = tx.QueryRow(ctx, "SELECT timeslot_id FROM lessons WHERE id = $1", lessonID).Scan(&slotID)
	if err != nil {
		return err
	}

	// 2. Помечаем слот как СВОБОДНЫЙ (теперь его увидят другие)
	_, err = tx.Exec(ctx, "UPDATE time_slots SET is_available = true WHERE id = $1", slotID)
	if err != nil {
		return err
	}

	// 3. Обновляем статус самого урока
	_, err = tx.Exec(ctx, "UPDATE lessons SET status = 'cancelled' WHERE id = $1", lessonID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Функция переноса занятия (уже учитывает освобождение старого и занятие нового)

// Получаем только ДОСТУПНЫЕ слоты для конкретного репетитора
// Получаем только ДОСТУПНЫЕ слоты для конкретного репетитора
func getTutorLessons(tutorUsername string) ([]map[string]interface{}, error) {
	// ВАЖНО: Проверьте имена колонок: student_name, date, start_time
	query := `
        SELECT 
            l.id, 
            l.student_name, 
            ts.date, 
            ts.start_time
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
		var name, startTime string
		var date time.Time
		// Проверьте, что Scan получает именно 4 аргумента, как в SELECT выше
		if err := rows.Scan(&id, &name, &date, &startTime); err != nil {
			return nil, fmt.Errorf("ошибка сканирования: %v", err)
		}

		lessons = append(lessons, map[string]interface{}{
			"ID":          id,
			"StudentName": name,
			"Date":        date.Format("02.01.2006"),
			"StartTime":   startTime,
		})
	}
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
