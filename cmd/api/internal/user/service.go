package user

// UserService handles business logic for users
type UserService struct {
	repo *UserRepo
}

// NewUserService creates a new user service
func NewUserService(repo *UserRepo) *UserService {
	return &UserService{repo: repo}
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(id string) string {
	return s.repo.Get(id)
}

// List returns all users
func (s *UserService) List() map[string]string {
	return s.repo.List()
}