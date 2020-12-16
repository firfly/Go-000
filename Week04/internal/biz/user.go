package biz

type User struct {
	ID   int32
	Name string
	Age  int32
}

type UserRepo interface {
	Save(*User) int32
}

func NewUserUsecase(repo UserRepo) *UserUsecase {
	return &UserUsecase{repo: repo}
}

type UserUsecase struct {
	repo UserRepo
}

func (s *UserUsecase) SaveUser(u *User) {
	id := s.repo.Save(u)
	u.ID = id
}