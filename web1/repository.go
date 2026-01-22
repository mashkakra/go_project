package main

import (
	"context"
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

func getTutorTimeSlots(tutorID int) ([]TimeSlot, error) {
	rows, err := db.Query(context.Background(), `
		SELECT id, date, start_time, end_time 
		FROM time_slots 
		WHERE tutor_id = $1 AND is_available = true 
		ORDER BY date, start_time
	`, tutorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []TimeSlot
	for rows.Next() {
		var s TimeSlot
		rows.Scan(&s.ID, &s.Date, &s.StartTime, &s.EndTime)
		res = append(res, s)
	}
	return res, nil
}

// --- МЕТОДЫ ЗАПИСИ ---

func createApplication(tutorID int, name, phone, email string) error {
	_, err := db.Exec(context.Background(), `
		INSERT INTO applications (tutor_id, student_name, student_phone, student_email, created_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
	`, tutorID, name, phone, email)
	return err
}
