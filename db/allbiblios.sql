drop table if exists admin_notifications;
drop table if exists students cascade;
drop table if exists users cascade;
drop table if exists lessons cascade;
drop table if exists tutors cascade;
drop table if exists subjects cascade;
drop table if exists tutor_subjects cascade;
drop table if exists applications cascade;
drop table if exists grades cascade;
drop table if exists reviews cascade;
drop table if exists tutor_grades cascade;
drop table if exists time_slots cascade;
drop table if exists slot_exceptions cascade;





-- Таблица для уведомлений админу

CREATE TABLE admin_notifications (
    id SERIAL PRIMARY KEY,
    message TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица предметов

CREATE TABLE subjects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT

);



-- Таблица репетиторов

CREATE TABLE tutors (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    email VARCHAR(100) UNIQUE,
    phone VARCHAR(20),
    bio TEXT,
    experience_years INTEGER DEFAULT 0,
    hourly_rate DECIMAL(10,2),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	username varchar(12) UNIQUE NOT NULL
);



-- Связующая таблица репетитор-предмет (многие ко многим)

CREATE TABLE tutor_subjects (
    id SERIAL PRIMARY KEY,
    tutor_id INTEGER REFERENCES tutors(id) ON DELETE CASCADE,
    subject_id INTEGER REFERENCES subjects(id) ON DELETE CASCADE,
    UNIQUE(tutor_id, subject_id)

);



-- Таблица классов (для каких классов преподает репетитор)

CREATE TABLE grades (
    id SERIAL PRIMARY KEY,
    grade_name VARCHAR(20) NOT NULL UNIQUE, -- '1-4', '5-9', '10-11', 'студенты'
    description TEXT

);



-- Связующая таблица репетитор-классы

CREATE TABLE tutor_grades (
    id SERIAL PRIMARY KEY,
    tutor_id INTEGER REFERENCES tutors(id) ON DELETE CASCADE,
    grade_id INTEGER REFERENCES grades(id) ON DELETE CASCADE,
    UNIQUE(tutor_id, grade_id)

);



-- Таблица свободных окон (расписание)

CREATE TABLE time_slots (
    id SERIAL PRIMARY KEY,
    tutor_id INTEGER REFERENCES tutors(id) ON DELETE CASCADE,
	day_of_week CHAR(2) NOT NULL,
	date Date,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    is_available BOOLEAN DEFAULT true,
	is_template BOOLEAN DEFAULT true,
    is_recurring BOOLEAN DEFAULT false, -- повторяющееся окно (например, каждую неделю)
    recurring_pattern VARCHAR(50), -- 'weekly', 'bi-weekly'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP

);

-- Форматирование даты для вывода расписания
CREATE OR REPLACE FUNCTION format_lesson_date(p_date DATE, p_is_recurring BOOLEAN)
RETURNS TEXT
LANGUAGE plpgsql
AS $$
BEGIN
    IF p_date IS NULL THEN
        RETURN '';
    END IF;

    IF p_is_recurring THEN
        RETURN to_char(p_date, 'DD.MM.YYYY') || ' (еженедельно)';
    END IF;

    RETURN to_char(p_date, 'DD.MM.YYYY');
END;
$$;

CREATE TABLE slot_exceptions (
    id SERIAL PRIMARY KEY,
    timeslot_id INTEGER REFERENCES time_slots(id) ON DELETE CASCADE,
    exception_date DATE NOT NULL, -- Конкретная дата (например, 2026-02-01)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(timeslot_id, exception_date) -- Нельзя создать два исключения на один день
);

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    role TEXT NOT NULL -- 'admin', 'tutor', 'student'
);

CREATE TABLE students (
	id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    full_name VARCHAR(100) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	UNIQUE (id, user_id)
);



CREATE TABLE IF NOT EXISTS lessons (
    id SERIAL PRIMARY KEY,
    tutor_id INTEGER REFERENCES tutors(id) ON DELETE CASCADE,
    timeslot_id INTEGER REFERENCES time_slots(id) ON DELETE CASCADE,
    student_name VARCHAR(100) NOT NULL,
    student_phone VARCHAR(20) NOT NULL,
    student_email VARCHAR(100),
    -- Статусы: 'pending' (ожидает), 'scheduled' (подтвержден), 'declined' (отклонен)
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	student_id INT REFERENCES students(id) ON DELETE SET NULL
);
ALTER TABLE lessons ADD COLUMN IF NOT EXISTS comment TEXT;


-- 1. Создаем таблицу профилей учеников







-- Таблица для хранения заявок

CREATE TABLE applications (

    id SERIAL PRIMARY KEY,
    tutor_id INTEGER REFERENCES tutors(id),
    student_name VARCHAR(100) NOT NULL,
    student_phone VARCHAR(20) NOT NULL,
    student_email VARCHAR(100) NOT NULL,
    status VARCHAR(20) DEFAULT 'new',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP

);

CREATE TABLE reviews (

    id SERIAL PRIMARY KEY,

    tutor_id INTEGER REFERENCES tutors(id) ON DELETE CASCADE,

    student_name VARCHAR(100),

    rating INTEGER CHECK (rating >= 1 AND rating <= 5),

    comment TEXT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP

);


INSERT INTO subjects (name, description) VALUES 

('Математика', 'Математика для школьников и студентов'),

('Физика', 'Физика от основ до олимпиадных задач'),

('Химия', 'Химия для школьников и подготовка к ЕГЭ'),

('Русский язык', 'Русский язык и литература'),

('Английский язык', 'Английский язык для всех уровней'),

('Информатика', 'Программирование и компьютерные науки'),

('История', 'История России и всемирная история'),

('Биология', 'Биология и подготовка к экзаменам');



-- Вставляем классы

INSERT INTO grades (grade_name, description) VALUES 

('1-4', 'Начальная школа'),

('5-9', 'Средняя школа'),

('10-11', 'Старшая школа, подготовка к ЕГЭ'),

('Студенты', 'Университетская программа');



-- Вставляем репетиторов

INSERT INTO tutors (first_name, last_name, email, phone, bio, experience_years, hourly_rate,username) VALUES 

('Анна', 'Иванова', 'anna.ivanova@email.com', '+7-911-123-4567', 

 'Опытный репетитор по математике с 10-летним стажем. Специализируюсь на подготовке к ЕГЭ и ОГЭ. Мои ученики consistently показывают высокие результаты.',

 10, 1500.00,'tutor_ann'),

('Петр', 'Сидоров', 'petr.sidorov@email.com', '+7-911-234-5678',

   'Преподаватель физики с ученой степенью. Работаю со школьниками 7-11 классов. Объясняю сложные concepts простым языком.',

   8, 1800.00,'tutor_petr'),



('Мария', 'Петрова', 'maria.petrova@email.com', '+7-911-345-6789',

 'Репетитор по английскому языку, сертификат CPE. Опыт работы за рубежом. Готовлю к IELTS, TOEFL, ЕГЭ.',

 6, 2000.00,'tutor_mary'),



('Алексей', 'Кузнецов', 'alexey.kuznetsov@email.com', '+7-911-456-7890',

 'Учитель информатики и программирования. Преподаю Python, Java, основы алгоритмов. Готовлю к олимпиадам.',

 5, 1700.00,'tutor_alex'),



('Ольга', 'Смирнова', 'olga.smirnova@email.com', '+7-911-567-8901',

 'Репетитор по химии и биологии. Кандидат химических наук. Помогаю с школьной программой и подготовкой к поступлению.',

 12, 1600.00,'tutor_olga');

-- Связываем репетиторов с предметами

INSERT INTO tutor_subjects (tutor_id, subject_id) VALUES 

(1, 1), -- Анна - Математика

(2, 2), -- Петр - Физика

(3, 5), -- Мария - Английский
(4, 6), -- Алексей - Информатика
(4, 1), -- Алексей - Математика
(5, 3), -- Ольга - Химия
(5, 8); -- Ольга - Биология


-- Связываем репетиторов с классами
INSERT INTO tutor_grades (tutor_id, grade_id) VALUES 
(1, 3), (1, 4), -- Анна: 10-11, студенты
(2, 2), (2, 3), -- Петр: 5-9, 10-11
(3, 1), (3, 2), (3, 3), (3, 4), -- Мария: все классы
(4, 2), (4, 3), (4, 4), -- Алексей: 5-9, 10-11, студенты
(5, 2), (5, 3), (5, 4); -- Ольга: 5-9, 10-11, студенты

-- Добавляем свободные окна
INSERT INTO time_slots (tutor_id, day_of_week,date, start_time, end_time, is_recurring, recurring_pattern) VALUES 
(1, 'ЧТ','2026-01-15', '15:00', '16:30', true, 'weekly'),
(1, 'ЧТ','2026-01-15', '17:00', '18:30', true, 'weekly'),
(2,'ПТ','2026-01-16', '14:00', '15:30', true, 'weekly'),
(2, 'ПТ','2026-01-16', '16:00', '17:30', true, 'weekly'),
(3, 'СБ','2026-01-17', '10:00', '11:30', true, 'weekly'),
(3, 'СБ','2026-01-17', '13:00', '14:30', true, 'weekly'),
(4, 'ВС','2026-01-18', '15:00', '16:30', true, 'weekly'),
(5, 'ПН','2026-01-19', '11:00', '12:30', true, 'weekly');


-- Добавляем отзывы
INSERT INTO reviews (tutor_id, student_name, rating, comment) VALUES 
(1, 'Иван Петров', 5, 'Анна прекрасный преподаватель! Благодаря ей сдал ЕГЭ на 94 балла!'),
(1, 'Елена Сидорова', 5, 'Очень понятно объясняет, дочь наконец-то полюбила математику'),
(2, 'Дмитрий Иванов', 4, 'Хороший преподаватель, физика стала понятнее'),
(3, 'Анна Ковалева', 5, 'Занимаюсь с Марией уже год, значительно улучшила английский'),
(4, 'Сергей Попов', 5, 'Алексей крутой! Научил программировать на Python с нуля');
select * from time_slots;

select * from students;
select * from lessons;
select * from time_slots;
SELECT l.id, l.student_name, t.last_name, ts.date
FROM lessons l
INNER JOIN tutors t ON l.tutor_id = t.id
INNER JOIN time_slots ts ON l.timeslot_id = ts.id;
select * from users;
INSERT INTO users (username, password, role) 

VALUES ('admin_boss', '$2a$10$iiZ0ZeVmLHl5TaC58VzFCOWFKuipwEA5nkxrZ4VoiKxnVTSA5GouC', 'admin');

SELECT id, student_name, student_id FROM lessons WHERE student_id IS NULL;
