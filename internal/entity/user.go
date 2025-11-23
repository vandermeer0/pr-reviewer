// Package entity содержит основные сущности бизнес-логики
package entity

import "fmt"

// User - участник команды
type User struct {
	ID       string
	Username string
	TeamName string
	IsActive bool
}

// Validate проверяет минимальные требования к данным пользователя
func (u *User) Validate() error {
	if u.ID == "" {
		return fmt.Errorf("user id is empty")
	}
	if u.Username == "" {
		return fmt.Errorf("username is empty")
	}
	return nil
}
