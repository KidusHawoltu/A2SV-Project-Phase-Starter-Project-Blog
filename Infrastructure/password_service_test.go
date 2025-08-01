package infrastructure_test

import (
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"
	"testing"

	"github.com/stretchr/testify/suite"
)

// PasswordServiceTestSuite groups tests for PasswordService
type PasswordServiceTestSuite struct {
	suite.Suite
	passwordService infrastructure.PasswordService
}

func (s *PasswordServiceTestSuite) SetupTest() {
	s.passwordService = infrastructure.NewPasswordService()
}

func (s *PasswordServiceTestSuite) TestHashAndCompare_Success() {
	password := "secure123"
	hashed, err := s.passwordService.HashPassword(password)
	s.NoError(err)
	s.NotEmpty(hashed)
	err = s.passwordService.ComparePassword(hashed, password)
	s.NoError(err)
}

func (s *PasswordServiceTestSuite) TestComparePassword_WrongPassword() {
	password := "secure123"
	hashed, _ := s.passwordService.HashPassword(password)
	err := s.passwordService.ComparePassword(hashed, "wrong")
	s.Error(err)
}

func (s *PasswordServiceTestSuite) TestHashPassword_EmptyInput() {
	_, err := s.passwordService.HashPassword("")
	s.NoError(err) // bcrypt allows empty password, but returns a valid hash
}

func TestPasswordServiceTestSuite(t *testing.T) {
	suite.Run(t, new(PasswordServiceTestSuite))
}