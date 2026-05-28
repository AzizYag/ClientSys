package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"clientsys/internal/model"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

const dateLayout = "2006-01-02"

type Store struct {
	db   *sql.DB
	path string
}

func DefaultPath() (string, error) {
	if path := os.Getenv("CLIENTSYS_DB"); path != "" {
		return path, nil
	}
	if runtime.GOOS == "windows" {
		executable, err := os.Executable()
		if err == nil && executable != "" {
			return filepath.Join(filepath.Dir(executable), "clientsys.db"), nil
		}
	}
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "ClientSys")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "clientsys.db"), nil
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db, path: path}
	if err := s.init(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Path() string { return s.path }

func (s *Store) Backup(destination string) error {
	if filepath.Clean(destination) == filepath.Clean(s.path) {
		return errors.New("выберите другой файл для резервной копии")
	}
	if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	escaped := strings.ReplaceAll(destination, "'", "''")
	_, err := s.db.Exec(`VACUUM INTO '` + escaped + `'`)
	return err
}

func (s *Store) init() error {
	statements := []string{
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS users (
			user_id INTEGER PRIMARY KEY AUTOINCREMENT,
			login TEXT NOT NULL UNIQUE COLLATE NOCASE,
			email TEXT NOT NULL UNIQUE COLLATE NOCASE,
			password_hash TEXT NOT NULL,
			full_name TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'manager',
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS clients (
			client_id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(user_id),
			last_name TEXT NOT NULL,
			first_name TEXT NOT NULL,
			patronymic TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL,
			email TEXT NOT NULL DEFAULT '',
			address TEXT NOT NULL DEFAULT '',
			registration_date TEXT NOT NULL,
			status TEXT NOT NULL,
			comment TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS requests (
			request_id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id INTEGER NOT NULL REFERENCES clients(client_id) ON DELETE CASCADE,
			theme TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			create_date TEXT NOT NULL,
			status TEXT NOT NULL,
			employee_comment TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS orders (
			order_id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id INTEGER NOT NULL REFERENCES clients(client_id) ON DELETE CASCADE,
			service_name TEXT NOT NULL,
			order_date TEXT NOT NULL,
			amount REAL NOT NULL CHECK (amount >= 0),
			status TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS interactions (
			interaction_id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id INTEGER NOT NULL REFERENCES clients(client_id) ON DELETE CASCADE,
			interaction_type TEXT NOT NULL,
			description TEXT NOT NULL,
			interaction_date TEXT NOT NULL,
			employee_id INTEGER NOT NULL REFERENCES users(user_id)
		);`,
		`CREATE TABLE IF NOT EXISTS app_metadata (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS clients_search ON clients(last_name, phone, email, status);`,
		`CREATE INDEX IF NOT EXISTS requests_client ON requests(client_id, create_date);`,
		`CREATE INDEX IF NOT EXISTS orders_client ON orders(client_id, order_date);`,
		`CREATE INDEX IF NOT EXISTS interactions_client ON interactions(client_id, interaction_date);`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("prepare database: %w", err)
		}
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		_, err = s.db.Exec(`INSERT INTO users(login, email, password_hash, full_name, role, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`, "admin", "admin@clientsys.local", string(hash),
			"Администратор", "admin", time.Now().Format(dateLayout))
		if err != nil {
			return err
		}
	}
	return s.seedDemoData()
}

func (s *Store) seedDemoData() error {
	var clientCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM clients`).Scan(&clientCount); err != nil {
		return err
	}
	if clientCount > 0 {
		_, err := s.db.Exec(`INSERT OR IGNORE INTO app_metadata(key, value) VALUES ('demo_seeded', 'existing-data')`)
		return err
	}
	var employeeID int64
	if err := s.db.QueryRow(`SELECT user_id FROM users ORDER BY CASE WHEN role='admin' THEN 0 ELSE 1 END, user_id LIMIT 1`).Scan(&employeeID); err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	type demoClient struct {
		lastName, firstName, patronymic, phone, email, address, status, comment string
		daysAgo                                                                 int
	}
	demoClients := []demoClient{
		{"Абдуллина", "Алина", "Маратовна", "+7 (917) 210-14-55", "alina.a@mail.ru", "г. Уфа, ул. Менделеева, 138", "Активный", "Постоянный клиент. Предпочитает связь по телефону.", 3},
		{"Васильев", "Максим", "Олегович", "+7 (927) 355-21-08", "vasilev.m@example.ru", "г. Уфа, ул. Комсомольская, 27", "Новый", "Первичная консультация по услуге.", 1},
		{"Гарипова", "Лилия", "Ильдаровна", "+7 (987) 602-09-32", "liliya.g@example.ru", "г. Уфа, пр. Октября, 72", "Активный", "Заключен повторный заказ.", 9},
		{"Иванов", "Сергей", "Петрович", "+7 (905) 440-77-12", "ivanov.s@example.ru", "г. Уфа, ул. Российская, 41", "Неактивный", "Обращение завершено, ожидает повторного контакта.", 22},
		{"Каримова", "Диана", "Рустемовна", "+7 (917) 880-11-48", "d.karimova@example.ru", "г. Уфа, ул. Ленина, 54", "Активный", "Приоритетный клиент.", 14},
		{"Мусин", "Артур", "Ринатович", "+7 (937) 101-88-90", "artur.musin@example.ru", "г. Уфа, ул. Первомайская, 19", "Новый", "Запросил расчет стоимости.", 5},
		{"Сафина", "Эльвира", "Азатовна", "+7 (965) 501-24-60", "safina.e@example.ru", "г. Уфа, ул. Айская, 11", "Активный", "Заказ выполнен, получена положительная обратная связь.", 28},
		{"Хабиров", "Тимур", "Ильшатович", "+7 (919) 312-45-72", "t.khabirov@example.ru", "г. Уфа, ул. Революционная, 96", "Неактивный", "Отложил решение до следующего месяца.", 18},
	}
	clientIDs := make([]int64, 0, len(demoClients))
	for _, c := range demoClients {
		result, err := tx.Exec(`INSERT INTO clients(user_id, last_name, first_name, patronymic, phone, email,
			address, registration_date, status, comment) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			employeeID, c.lastName, c.firstName, c.patronymic, c.phone, c.email, c.address,
			demoDate(c.daysAgo), c.status, c.comment)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		clientIDs = append(clientIDs, id)
	}

	requests := []struct {
		client                              int
		theme, description, status, comment string
		daysAgo                             int
	}{
		{0, "Продление обслуживания", "Клиент интересуется новым пакетом сопровождения.", "В работе", "Подготовить коммерческое предложение.", 2},
		{1, "Первичная консультация", "Требуется консультация по условиям подключения.", "Новая", "Назначить звонок на сегодня.", 1},
		{2, "Изменение заказа", "Необходимо добавить дополнительную услугу.", "Выполнена", "Изменения согласованы.", 7},
		{4, "Запрос документов", "Клиент запросил копии договора и счета.", "Закрыта", "Документы отправлены по e-mail.", 11},
		{5, "Расчет стоимости", "Запрос расчета для расширенного тарифа.", "В работе", "Ожидается согласование состава услуг.", 4},
		{7, "Повторный контакт", "Уточнить актуальность ранее предложенной услуги.", "Новая", "", 3},
	}
	for _, r := range requests {
		if _, err := tx.Exec(`INSERT INTO requests(client_id, theme, description, create_date, status, employee_comment)
			VALUES (?, ?, ?, ?, ?, ?)`, clientIDs[r.client], r.theme, r.description, demoDate(r.daysAgo), r.status, r.comment); err != nil {
			return err
		}
	}

	orders := []struct {
		client          int
		service, status string
		amount          float64
		daysAgo         int
	}{
		{0, "Годовое сопровождение", "В работе", 42000, 2},
		{2, "Настройка клиентского кабинета", "Выполнен", 18500, 8},
		{4, "Комплексное обслуживание", "Выполнен", 56000, 13},
		{5, "Аудит потребностей", "Новый", 9500, 4},
		{6, "Консультационное сопровождение", "Выполнен", 24000, 25},
		{3, "Техническая консультация", "Отменен", 5000, 20},
	}
	for _, o := range orders {
		if _, err := tx.Exec(`INSERT INTO orders(client_id, service_name, order_date, amount, status)
			VALUES (?, ?, ?, ?, ?)`, clientIDs[o.client], o.service, demoDate(o.daysAgo), o.amount, o.status); err != nil {
			return err
		}
	}

	interactions := []struct {
		client            int
		kind, description string
		daysAgo           int
	}{
		{1, "Звонок", "Проведена первичная консультация, клиент уточняет сроки.", 1},
		{0, "Письмо", "Направлена презентация вариантов сопровождения.", 2},
		{5, "Звонок", "Получены данные для подготовки расчета стоимости.", 3},
		{2, "Встреча", "Согласован перечень дополнительных услуг.", 7},
		{4, "Письмо", "Переданы закрывающие документы и счет.", 10},
		{6, "Звонок", "Получена положительная оценка оказанной услуги.", 24},
		{7, "Комментарий", "Клиент попросил вернуться к обсуждению позже.", 17},
		{3, "Консультация", "Обращение завершено без оформления нового заказа.", 19},
	}
	for _, i := range interactions {
		if _, err := tx.Exec(`INSERT INTO interactions(client_id, interaction_type, description, interaction_date, employee_id)
			VALUES (?, ?, ?, ?, ?)`, clientIDs[i.client], i.kind, i.description, demoDate(i.daysAgo), employeeID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`INSERT OR REPLACE INTO app_metadata(key, value) VALUES ('demo_seeded', 'created')`); err != nil {
		return err
	}
	return tx.Commit()
}

func demoDate(daysAgo int) string {
	return time.Now().AddDate(0, 0, -daysAgo).Format(dateLayout)
}

func (s *Store) Authenticate(login, password string) (model.User, error) {
	var u model.User
	err := s.db.QueryRow(`SELECT user_id, login, email, password_hash, full_name, role, created_at
		FROM users WHERE login = ?`, strings.TrimSpace(login)).
		Scan(&u.ID, &u.Login, &u.Email, &u.PasswordHash, &u.FullName, &u.Role, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return u, errors.New("неверный логин или пароль")
	}
	if err != nil {
		return u, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return model.User{}, errors.New("неверный логин или пароль")
	}
	return u, nil
}

func (s *Store) CreateUser(u model.User, password string) error {
	if strings.TrimSpace(u.Login) == "" || strings.TrimSpace(u.FullName) == "" || strings.TrimSpace(u.Email) == "" {
		return errors.New("заполните логин, ФИО и электронную почту")
	}
	if len(password) < 6 {
		return errors.New("пароль должен содержать не менее 6 символов")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if u.Role == "" {
		u.Role = "manager"
	}
	_, err = s.db.Exec(`INSERT INTO users(login, email, password_hash, full_name, role, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`, strings.TrimSpace(u.Login), strings.TrimSpace(u.Email),
		string(hash), strings.TrimSpace(u.FullName), u.Role, time.Now().Format(dateLayout))
	if err != nil {
		return errors.New("логин или электронная почта уже используются")
	}
	return nil
}

func (s *Store) ChangePassword(userID int64, current, next string) error {
	if len(next) < 6 {
		return errors.New("новый пароль должен содержать не менее 6 символов")
	}
	var hash string
	if err := s.db.QueryRow(`SELECT password_hash FROM users WHERE user_id=?`, userID).Scan(&hash); err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(current)) != nil {
		return errors.New("текущий пароль указан неверно")
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE users SET password_hash=? WHERE user_id=?`, string(newHash), userID)
	return err
}

func (s *Store) Users() ([]model.User, error) {
	rows, err := s.db.Query(`SELECT user_id, login, email, full_name, role, created_at
		FROM users ORDER BY full_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Login, &u.Email, &u.FullName, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		values = append(values, u)
	}
	return values, rows.Err()
}

func (s *Store) Clients(search, status string) ([]model.Client, error) {
	search = strings.ToLower(strings.TrimSpace(search))
	rows, err := s.db.Query(`SELECT client_id, user_id, last_name, first_name, patronymic, phone,
		email, address, registration_date, status, comment FROM clients
		WHERE (? = '' OR status = ?) ORDER BY last_name, first_name`, status, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []model.Client
	for rows.Next() {
		var c model.Client
		if err := rows.Scan(&c.ID, &c.UserID, &c.LastName, &c.FirstName, &c.Patronymic, &c.Phone,
			&c.Email, &c.Address, &c.RegistrationDate, &c.Status, &c.Comment); err != nil {
			return nil, err
		}
		haystack := strings.ToLower(c.FullName() + " " + c.Phone + " " + c.Email)
		if search != "" && !strings.Contains(haystack, search) {
			continue
		}
		values = append(values, c)
	}
	return values, rows.Err()
}

func (s *Store) SaveClient(c model.Client) error {
	if strings.TrimSpace(c.LastName) == "" || strings.TrimSpace(c.FirstName) == "" || strings.TrimSpace(c.Phone) == "" {
		return errors.New("фамилия, имя и телефон обязательны")
	}
	if c.Status == "" {
		c.Status = "Новый"
	}
	if c.RegistrationDate == "" {
		c.RegistrationDate = time.Now().Format(dateLayout)
	}
	if _, err := time.Parse(dateLayout, c.RegistrationDate); err != nil {
		return errors.New("дата должна быть в формате ГГГГ-ММ-ДД")
	}
	if c.ID == 0 {
		_, err := s.db.Exec(`INSERT INTO clients(user_id, last_name, first_name, patronymic, phone, email,
			address, registration_date, status, comment) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			c.UserID, strings.TrimSpace(c.LastName), strings.TrimSpace(c.FirstName), strings.TrimSpace(c.Patronymic),
			strings.TrimSpace(c.Phone), strings.TrimSpace(c.Email), strings.TrimSpace(c.Address),
			c.RegistrationDate, c.Status, strings.TrimSpace(c.Comment))
		return err
	}
	_, err := s.db.Exec(`UPDATE clients SET last_name=?, first_name=?, patronymic=?, phone=?, email=?,
		address=?, registration_date=?, status=?, comment=? WHERE client_id=?`,
		strings.TrimSpace(c.LastName), strings.TrimSpace(c.FirstName), strings.TrimSpace(c.Patronymic),
		strings.TrimSpace(c.Phone), strings.TrimSpace(c.Email), strings.TrimSpace(c.Address),
		c.RegistrationDate, c.Status, strings.TrimSpace(c.Comment), c.ID)
	return err
}

func (s *Store) DeleteClient(id int64) error {
	_, err := s.db.Exec(`DELETE FROM clients WHERE client_id=?`, id)
	return err
}

func (s *Store) Requests(status string) ([]model.Request, error) {
	rows, err := s.db.Query(`SELECT r.request_id, r.client_id, c.last_name || ' ' || c.first_name,
		r.theme, r.description, r.create_date, r.status, r.employee_comment
		FROM requests r JOIN clients c ON c.client_id=r.client_id
		WHERE (?='' OR r.status=?) ORDER BY r.create_date DESC, r.request_id DESC`, status, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []model.Request
	for rows.Next() {
		var r model.Request
		if err := rows.Scan(&r.ID, &r.ClientID, &r.ClientName, &r.Theme, &r.Description,
			&r.CreateDate, &r.Status, &r.EmployeeComment); err != nil {
			return nil, err
		}
		values = append(values, r)
	}
	return values, rows.Err()
}

func (s *Store) SaveRequest(r model.Request) error {
	if r.ClientID == 0 || strings.TrimSpace(r.Theme) == "" {
		return errors.New("выберите клиента и укажите тему заявки")
	}
	if r.CreateDate == "" {
		r.CreateDate = time.Now().Format(dateLayout)
	}
	if r.Status == "" {
		r.Status = "Новая"
	}
	if _, err := time.Parse(dateLayout, r.CreateDate); err != nil {
		return errors.New("дата должна быть в формате ГГГГ-ММ-ДД")
	}
	if r.ID == 0 {
		_, err := s.db.Exec(`INSERT INTO requests(client_id, theme, description, create_date, status, employee_comment)
			VALUES (?, ?, ?, ?, ?, ?)`, r.ClientID, strings.TrimSpace(r.Theme), strings.TrimSpace(r.Description),
			r.CreateDate, r.Status, strings.TrimSpace(r.EmployeeComment))
		return err
	}
	_, err := s.db.Exec(`UPDATE requests SET client_id=?, theme=?, description=?, create_date=?, status=?,
		employee_comment=? WHERE request_id=?`, r.ClientID, strings.TrimSpace(r.Theme),
		strings.TrimSpace(r.Description), r.CreateDate, r.Status, strings.TrimSpace(r.EmployeeComment), r.ID)
	return err
}

func (s *Store) DeleteRequest(id int64) error {
	_, err := s.db.Exec(`DELETE FROM requests WHERE request_id=?`, id)
	return err
}

func (s *Store) Orders(status string) ([]model.Order, error) {
	rows, err := s.db.Query(`SELECT o.order_id, o.client_id, c.last_name || ' ' || c.first_name,
		o.service_name, o.order_date, o.amount, o.status
		FROM orders o JOIN clients c ON c.client_id=o.client_id
		WHERE (?='' OR o.status=?) ORDER BY o.order_date DESC, o.order_id DESC`, status, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []model.Order
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.ClientID, &o.ClientName, &o.ServiceName, &o.OrderDate, &o.Amount, &o.Status); err != nil {
			return nil, err
		}
		values = append(values, o)
	}
	return values, rows.Err()
}

func (s *Store) SaveOrder(o model.Order) error {
	if o.ClientID == 0 || strings.TrimSpace(o.ServiceName) == "" || o.Amount < 0 {
		return errors.New("выберите клиента, укажите услугу и корректную стоимость")
	}
	if o.OrderDate == "" {
		o.OrderDate = time.Now().Format(dateLayout)
	}
	if o.Status == "" {
		o.Status = "Новый"
	}
	if _, err := time.Parse(dateLayout, o.OrderDate); err != nil {
		return errors.New("дата должна быть в формате ГГГГ-ММ-ДД")
	}
	if o.ID == 0 {
		_, err := s.db.Exec(`INSERT INTO orders(client_id, service_name, order_date, amount, status)
			VALUES (?, ?, ?, ?, ?)`, o.ClientID, strings.TrimSpace(o.ServiceName), o.OrderDate, o.Amount, o.Status)
		return err
	}
	_, err := s.db.Exec(`UPDATE orders SET client_id=?, service_name=?, order_date=?, amount=?, status=?
		WHERE order_id=?`, o.ClientID, strings.TrimSpace(o.ServiceName), o.OrderDate, o.Amount, o.Status, o.ID)
	return err
}

func (s *Store) DeleteOrder(id int64) error {
	_, err := s.db.Exec(`DELETE FROM orders WHERE order_id=?`, id)
	return err
}

func (s *Store) Interactions() ([]model.Interaction, error) {
	rows, err := s.db.Query(`SELECT i.interaction_id, i.client_id, c.last_name || ' ' || c.first_name,
		i.interaction_type, i.description, i.interaction_date, i.employee_id, u.full_name
		FROM interactions i JOIN clients c ON c.client_id=i.client_id
		JOIN users u ON u.user_id=i.employee_id ORDER BY i.interaction_date DESC, i.interaction_id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []model.Interaction
	for rows.Next() {
		var i model.Interaction
		if err := rows.Scan(&i.ID, &i.ClientID, &i.ClientName, &i.Type, &i.Description,
			&i.Date, &i.EmployeeID, &i.Employee); err != nil {
			return nil, err
		}
		values = append(values, i)
	}
	return values, rows.Err()
}

func (s *Store) SaveInteraction(i model.Interaction) error {
	if i.ClientID == 0 || i.EmployeeID == 0 || strings.TrimSpace(i.Description) == "" {
		return errors.New("выберите клиента и укажите результат взаимодействия")
	}
	if i.Type == "" {
		i.Type = "Комментарий"
	}
	if i.Date == "" {
		i.Date = time.Now().Format(dateLayout)
	}
	if _, err := time.Parse(dateLayout, i.Date); err != nil {
		return errors.New("дата должна быть в формате ГГГГ-ММ-ДД")
	}
	if i.ID == 0 {
		_, err := s.db.Exec(`INSERT INTO interactions(client_id, interaction_type, description, interaction_date, employee_id)
			VALUES (?, ?, ?, ?, ?)`, i.ClientID, i.Type, strings.TrimSpace(i.Description), i.Date, i.EmployeeID)
		return err
	}
	_, err := s.db.Exec(`UPDATE interactions SET client_id=?, interaction_type=?, description=?, interaction_date=?
		WHERE interaction_id=?`, i.ClientID, i.Type, strings.TrimSpace(i.Description), i.Date, i.ID)
	return err
}

func (s *Store) DeleteInteraction(id int64) error {
	_, err := s.db.Exec(`DELETE FROM interactions WHERE interaction_id=?`, id)
	return err
}

func (s *Store) Report(from, to string) (model.Report, error) {
	r := model.Report{From: from, To: to}
	if _, err := time.Parse(dateLayout, from); err != nil {
		return r, errors.New("неверная начальная дата")
	}
	if _, err := time.Parse(dateLayout, to); err != nil {
		return r, errors.New("неверная конечная дата")
	}
	queries := []struct {
		sql  string
		args []any
		dest any
	}{
		{`SELECT COUNT(*) FROM clients WHERE registration_date BETWEEN ? AND ?`, []any{from, to}, &r.Clients},
		{`SELECT COUNT(*) FROM requests WHERE create_date BETWEEN ? AND ?`, []any{from, to}, &r.Requests},
		{`SELECT COUNT(*) FROM requests WHERE create_date BETWEEN ? AND ? AND status NOT IN ('Закрыта','Выполнена')`, []any{from, to}, &r.OpenRequests},
		{`SELECT COUNT(*) FROM orders WHERE order_date BETWEEN ? AND ?`, []any{from, to}, &r.Orders},
		{`SELECT COUNT(*) FROM orders WHERE order_date BETWEEN ? AND ? AND status='Выполнен'`, []any{from, to}, &r.CompletedOrders},
		{`SELECT COALESCE(SUM(amount), 0) FROM orders WHERE order_date BETWEEN ? AND ? AND status='Выполнен'`, []any{from, to}, &r.Revenue},
		{`SELECT COUNT(*) FROM interactions WHERE interaction_date BETWEEN ? AND ?`, []any{from, to}, &r.Interactions},
	}
	for _, query := range queries {
		if err := s.db.QueryRow(query.sql, query.args...).Scan(query.dest); err != nil {
			return r, err
		}
	}
	return r, nil
}
