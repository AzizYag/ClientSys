package store

import (
	"os"
	"path/filepath"
	"testing"

	"clientsys/internal/model"
)

func TestClientAccountingWorkflow(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	admin, err := s.Authenticate("admin", "admin123")
	if err != nil {
		t.Fatalf("authenticate seeded administrator: %v", err)
	}
	seedClients, err := s.Clients("", "")
	if err != nil || len(seedClients) != 8 {
		t.Fatalf("seed demonstration clients: got %d, err=%v", len(seedClients), err)
	}
	seedRequests, _ := s.Requests("")
	seedOrders, _ := s.Orders("")
	seedInteractions, _ := s.Interactions()
	baseline, err := s.Report("2000-01-01", demoDate(0))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ChangePassword(admin.ID, "admin123", "admin456"); err != nil {
		t.Fatalf("change administrator password: %v", err)
	}
	if _, err := s.Authenticate("admin", "admin456"); err != nil {
		t.Fatalf("authenticate with new password: %v", err)
	}
	if err := s.CreateUser(model.User{Login: "manager", Email: "manager@example.test", FullName: "Иванов Иван", Role: "manager"}, "secret123"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := s.SaveClient(model.Client{
		UserID:           admin.ID,
		LastName:         "Петров",
		FirstName:        "Петр",
		Phone:            "+7 900 000-00-00",
		Email:            "client@example.test",
		RegistrationDate: demoDate(0),
		Status:           "Активный",
	}); err != nil {
		t.Fatalf("save client: %v", err)
	}
	clients, err := s.Clients("Петров", "Активный")
	if err != nil || len(clients) != 1 {
		t.Fatalf("search clients: got %d, err=%v", len(clients), err)
	}
	clientID := clients[0].ID

	if err := s.SaveRequest(model.Request{ClientID: clientID, Theme: "Консультация", CreateDate: demoDate(0), Status: "В работе"}); err != nil {
		t.Fatalf("save request: %v", err)
	}
	if err := s.SaveOrder(model.Order{ClientID: clientID, ServiceName: "Настройка", OrderDate: demoDate(0), Amount: 1500, Status: "Выполнен"}); err != nil {
		t.Fatalf("save order: %v", err)
	}
	if err := s.SaveInteraction(model.Interaction{ClientID: clientID, EmployeeID: admin.ID, Type: "Звонок", Description: "Услуга согласована", Date: demoDate(0)}); err != nil {
		t.Fatalf("save interaction: %v", err)
	}

	report, err := s.Report("2000-01-01", demoDate(0))
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	if report.Clients != baseline.Clients+1 || report.Requests != baseline.Requests+1 ||
		report.Orders != baseline.Orders+1 || report.Interactions != baseline.Interactions+1 ||
		report.Revenue != baseline.Revenue+1500 {
		t.Fatalf("workflow missing from report: %+v", report)
	}
	backup := filepath.Join(t.TempDir(), "backup.db")
	if err := s.Backup(backup); err != nil {
		t.Fatalf("backup database: %v", err)
	}
	if info, err := os.Stat(backup); err != nil || info.Size() == 0 {
		t.Fatalf("backup file not created: info=%v err=%v", info, err)
	}

	if err := s.DeleteClient(clientID); err != nil {
		t.Fatalf("delete client: %v", err)
	}
	requests, _ := s.Requests("")
	orders, _ := s.Orders("")
	interactions, _ := s.Interactions()
	if len(requests) != len(seedRequests) || len(orders) != len(seedOrders) || len(interactions) != len(seedInteractions) {
		t.Fatal("dependent records were not deleted with client")
	}
}

func TestDemoDataIsCreatedWhenDatabaseIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "once.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	clients, _ := s.Clients("", "")
	for _, client := range clients {
		if err := s.DeleteClient(client.ID); err != nil {
			t.Fatal(err)
		}
	}
	s.Close()

	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	clients, err = s.Clients("", "")
	if err != nil || len(clients) != 8 {
		t.Fatalf("demo data was not restored for an empty database: got %d, err=%v", len(clients), err)
	}
}
