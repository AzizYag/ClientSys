package main

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"clientsys/internal/model"
	"clientsys/internal/store"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx   context.Context
	store *store.Store
	user  *model.User
}

type Dashboard struct {
	Clients      int                 `json:"clients"`
	OpenRequests int                 `json:"openRequests"`
	Orders       int                 `json:"orders"`
	Revenue      float64             `json:"revenue"`
	Recent       []model.Client      `json:"recentClients"`
	Requests     []model.Request     `json:"recentRequests"`
	Interactions []model.Interaction `json:"recentInteractions"`
}

func NewApp(s *store.Store) *App {
	return &App{store: s}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) requireUser() (model.User, error) {
	if a.user == nil {
		return model.User{}, errors.New("необходимо войти в систему")
	}
	return *a.user, nil
}

func (a *App) Login(login, password string) (model.User, error) {
	user, err := a.store.Authenticate(login, password)
	if err != nil {
		return model.User{}, err
	}
	a.user = &user
	return user, nil
}

func (a *App) Logout() {
	a.user = nil
}

func (a *App) Register(user model.User, password string) error {
	current, err := a.requireUser()
	if err != nil {
		return errors.New("создавать сотрудников может только администратор после входа")
	}
	if current.Role != "admin" {
		return errors.New("недостаточно прав: требуется роль администратора")
	}
	return a.store.CreateUser(user, password)
}

func (a *App) ChangePassword(current, next string) error {
	user, err := a.requireUser()
	if err != nil {
		return err
	}
	return a.store.ChangePassword(user.ID, current, next)
}

func (a *App) Dashboard() (Dashboard, error) {
	if _, err := a.requireUser(); err != nil {
		return Dashboard{}, err
	}
	report, err := a.store.Report("2000-01-01", time.Now().Format("2006-01-02"))
	if err != nil {
		return Dashboard{}, err
	}
	clients, err := a.store.Clients("", "")
	if err != nil {
		return Dashboard{}, err
	}
	requests, err := a.store.Requests("")
	if err != nil {
		return Dashboard{}, err
	}
	interactions, err := a.store.Interactions()
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{
		Clients: report.Clients, OpenRequests: report.OpenRequests, Orders: report.Orders, Revenue: report.Revenue,
		Recent: limitClients(clients, 5), Requests: limitRequests(requests, 5), Interactions: limitInteractions(interactions, 5),
	}, nil
}

func limitClients(items []model.Client, max int) []model.Client {
	if len(items) > max {
		return items[:max]
	}
	return items
}

func limitRequests(items []model.Request, max int) []model.Request {
	if len(items) > max {
		return items[:max]
	}
	return items
}

func limitInteractions(items []model.Interaction, max int) []model.Interaction {
	if len(items) > max {
		return items[:max]
	}
	return items
}

func (a *App) Users() ([]model.User, error) {
	if _, err := a.requireUser(); err != nil {
		return nil, err
	}
	return a.store.Users()
}

func (a *App) Clients(search, status string) ([]model.Client, error) {
	if _, err := a.requireUser(); err != nil {
		return nil, err
	}
	return a.store.Clients(search, status)
}

func (a *App) SaveClient(client model.Client) error {
	user, err := a.requireUser()
	if err != nil {
		return err
	}
	if client.ID == 0 {
		client.UserID = user.ID
	}
	return a.store.SaveClient(client)
}

func (a *App) DeleteClient(id int64) error {
	if _, err := a.requireUser(); err != nil {
		return err
	}
	return a.store.DeleteClient(id)
}

func (a *App) Requests(status string) ([]model.Request, error) {
	if _, err := a.requireUser(); err != nil {
		return nil, err
	}
	return a.store.Requests(status)
}

func (a *App) SaveRequest(request model.Request) error {
	if _, err := a.requireUser(); err != nil {
		return err
	}
	return a.store.SaveRequest(request)
}

func (a *App) DeleteRequest(id int64) error {
	if _, err := a.requireUser(); err != nil {
		return err
	}
	return a.store.DeleteRequest(id)
}

func (a *App) Orders(status string) ([]model.Order, error) {
	if _, err := a.requireUser(); err != nil {
		return nil, err
	}
	return a.store.Orders(status)
}

func (a *App) SaveOrder(order model.Order) error {
	if _, err := a.requireUser(); err != nil {
		return err
	}
	return a.store.SaveOrder(order)
}

func (a *App) DeleteOrder(id int64) error {
	if _, err := a.requireUser(); err != nil {
		return err
	}
	return a.store.DeleteOrder(id)
}

func (a *App) Interactions() ([]model.Interaction, error) {
	if _, err := a.requireUser(); err != nil {
		return nil, err
	}
	return a.store.Interactions()
}

func (a *App) SaveInteraction(interaction model.Interaction) error {
	user, err := a.requireUser()
	if err != nil {
		return err
	}
	if interaction.EmployeeID == 0 {
		interaction.EmployeeID = user.ID
	}
	return a.store.SaveInteraction(interaction)
}

func (a *App) DeleteInteraction(id int64) error {
	if _, err := a.requireUser(); err != nil {
		return err
	}
	return a.store.DeleteInteraction(id)
}

func (a *App) Report(from, to string) (model.Report, error) {
	if _, err := a.requireUser(); err != nil {
		return model.Report{}, err
	}
	return a.store.Report(from, to)
}

func (a *App) ExportReport(report model.Report) (string, error) {
	if _, err := a.requireUser(); err != nil {
		return "", err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Экспорт отчета",
		DefaultFilename: "clientsys_report.csv",
		Filters:         []runtime.FileFilter{{DisplayName: "CSV файл (*.csv)", Pattern: "*.csv"}},
	})
	if err != nil || path == "" {
		return path, err
	}
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.WriteString("\ufeff"); err != nil {
		return "", err
	}
	writer := csv.NewWriter(file)
	writer.Comma = ';'
	rows := [][]string{
		{"Показатель", "Значение"},
		{"Период", report.From + " - " + report.To},
		{"Новые клиенты", strconv.Itoa(report.Clients)},
		{"Заявки", strconv.Itoa(report.Requests)},
		{"Открытые заявки", strconv.Itoa(report.OpenRequests)},
		{"Заказы", strconv.Itoa(report.Orders)},
		{"Выполненные заказы", strconv.Itoa(report.CompletedOrders)},
		{"Выручка", fmt.Sprintf("%.2f", report.Revenue)},
		{"Взаимодействия", strconv.Itoa(report.Interactions)},
	}
	if err := writer.WriteAll(rows); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) BackupDatabase() (string, error) {
	if _, err := a.requireUser(); err != nil {
		return "", err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Резервная копия базы данных",
		DefaultFilename: "clientsys_backup.db",
		Filters:         []runtime.FileFilter{{DisplayName: "SQLite база (*.db)", Pattern: "*.db"}},
	})
	if err != nil || path == "" {
		return path, err
	}
	if err := a.store.Backup(path); err != nil {
		return "", err
	}
	return path, nil
}
