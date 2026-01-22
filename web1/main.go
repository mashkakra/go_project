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
	http.HandleFunc("/admin/dashboard", adminDashboard)
	http.HandleFunc("/tutor/dashboard", tutorDashboard)
	http.HandleFunc("/student/dashboard", studentDashboardHandler)

	// API маршруты
	http.HandleFunc("/api/tutors", getTutorsHandler)
	http.HandleFunc("/api/application", submitApplicationHandler)
	http.HandleFunc("/api/add-slot", addTestSlotHandler)
	http.HandleFunc("/api/login", loginHandler)
	http.HandleFunc("/api/admin/create-user", createUserHandler)
	http.HandleFunc("/api/forgot-password", forgotPasswordHandler)
	http.HandleFunc("/api/student/lesson-action", lessonActionHandler)

	// 3. Запуск сервера
	port := ":8080"
	log.Printf("🚀 Сервер запущен на http://localhost%s\n", port)
	log.Printf("🎓 Запись к репетиторам: http://localhost%s/fortutor/\n", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("Ошибка запуска сервера: ", err)
	}

}
