package user

// UserRepo is the data access layer
type UserRepo struct {
	users map[string]string
}

// NewUserRepo creates a new user repository
func NewUserRepo() *UserRepo {
	return &UserRepo{users: map[string]string{
		"1": "Alice",
		"2": "Bob",
		"3": "Charlie",
	}}
}

// Get retrieves a user by ID
func (r *UserRepo) Get(id string) string {
	return r.users[id]
}

// List returns all users
func (r *UserRepo) List() map[string]string {
	return r.users
}