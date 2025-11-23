package entity

// Team — группа пользователей
type Team struct {
	Name    string
	Members []*User
}

// HasMember проверяет является ли userID участником этой команды
func (t *Team) HasMember(userID string) bool {
	for _, m := range t.Members {
		if m.ID == userID {
			return true
		}
	}
	return false
}
