package model

type User struct {
	ID           int64  `json:"id"`
	Login        string `json:"login"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	FullName     string `json:"fullName"`
	Role         string `json:"role"`
	CreatedAt    string `json:"createdAt"`
}

type Client struct {
	ID               int64  `json:"id"`
	UserID           int64  `json:"userId"`
	LastName         string `json:"lastName"`
	FirstName        string `json:"firstName"`
	Patronymic       string `json:"patronymic"`
	Phone            string `json:"phone"`
	Email            string `json:"email"`
	Address          string `json:"address"`
	RegistrationDate string `json:"registrationDate"`
	Status           string `json:"status"`
	Comment          string `json:"comment"`
}

func (c Client) FullName() string {
	name := c.LastName + " " + c.FirstName
	if c.Patronymic != "" {
		name += " " + c.Patronymic
	}
	return name
}

type Request struct {
	ID              int64  `json:"id"`
	ClientID        int64  `json:"clientId"`
	ClientName      string `json:"clientName"`
	Theme           string `json:"theme"`
	Description     string `json:"description"`
	CreateDate      string `json:"createDate"`
	Status          string `json:"status"`
	EmployeeComment string `json:"employeeComment"`
}

type Order struct {
	ID          int64   `json:"id"`
	ClientID    int64   `json:"clientId"`
	ClientName  string  `json:"clientName"`
	ServiceName string  `json:"serviceName"`
	OrderDate   string  `json:"orderDate"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"`
}

type Interaction struct {
	ID          int64  `json:"id"`
	ClientID    int64  `json:"clientId"`
	ClientName  string `json:"clientName"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Date        string `json:"date"`
	EmployeeID  int64  `json:"employeeId"`
	Employee    string `json:"employee"`
}

type Report struct {
	From            string  `json:"from"`
	To              string  `json:"to"`
	Clients         int     `json:"clients"`
	Requests        int     `json:"requests"`
	OpenRequests    int     `json:"openRequests"`
	Orders          int     `json:"orders"`
	CompletedOrders int     `json:"completedOrders"`
	Revenue         float64 `json:"revenue"`
	Interactions    int     `json:"interactions"`
}
