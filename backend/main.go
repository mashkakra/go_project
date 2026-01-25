package main

import (
	"log"
	"net/http"
)

func main() {
	// 1. Инициализируем базу данных (из config.go)
	initDB()
	defer db.Close()

	// 2. Регистрируем маршруты (обработчики из handlers.go)
	http.HandleFunc("/", home)
	http.HandleFunc("/fortutor/", tutor)
	http.HandleFunc("/admin/", adminHandler)
	http.HandleFunc("/login", loginPage)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/admin/dashboard", adminDashboard)
	http.HandleFunc("/tutor/dashboard", tutorDashboard)
	http.HandleFunc("/student/dashboard", studentDashboardHandler)
	http.HandleFunc("/tutor/reschedule", reschedulePageHandler) // Сама страница со списком

	// API маршруты
	http.HandleFunc("/api/tutors", getTutorsHandler)
	http.HandleFunc("/api/application", submitApplicationHandler)
	http.HandleFunc("/api/add-slot", addTestSlotHandler)
	http.HandleFunc("/api/login", loginHandler)
	http.HandleFunc("/api/admin/create-user", createUserHandler)
	http.HandleFunc("/api/admin/create-student", adminCreateStudentHandler)
	http.HandleFunc("/api/tutor/lesson-action", tutorActionHandler)
	http.HandleFunc("/api/forgot-password", forgotPasswordHandler)
	http.HandleFunc("/api/student/lesson-action", lessonActionHandler)
	http.HandleFunc("/api/tutor/confirm-reschedule", confirmRescheduleHandler)
	http.HandleFunc("/api/tutor/create-and-reschedule", createAndRescheduleHandler)

	// 3. Запуск сервера
	port := ":8080"
	log.Printf("Server started on http://localhost%s\n", port)
	log.Printf("Booking a tutor: http://localhost%s/fortutor/\n", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("Error starting server: ", err)
	}

}
