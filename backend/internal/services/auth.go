package services

import (
	"errors"
	"strings"

	"github.com/kelvins-io/stock-matching-system/internal/middleware"
	"github.com/kelvins-io/stock-matching-system/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db  *gorm.DB
	jwt *middleware.JWTAuth
}

func NewAuthService(db *gorm.DB, jwt *middleware.JWTAuth) *AuthService {
	return &AuthService{db: db, jwt: jwt}
}

type RegisterReq struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6,max=64"`
}

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResp struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

func (s *AuthService) Register(req RegisterReq) (*AuthResp, error) {
	username := strings.TrimSpace(req.Username)
	var exists int64
	s.db.Model(&models.User{}).Where("username = ?", username).Count(&exists)
	if exists > 0 {
		return nil, errors.New("用户名已存在")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := models.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         models.RoleUser,
		Balance:      1000000, // 新用户初始资金 100 万
	}
	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}
	token, err := s.jwt.GenerateToken(&user)
	if err != nil {
		return nil, err
	}
	return &AuthResp{Token: token, User: &user}, nil
}

func (s *AuthService) Login(req LoginReq) (*AuthResp, error) {
	var user models.User
	if err := s.db.Where("username = ?", strings.TrimSpace(req.Username)).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户名或密码错误")
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("用户名或密码错误")
	}
	token, err := s.jwt.GenerateToken(&user)
	if err != nil {
		return nil, err
	}
	return &AuthResp{Token: token, User: &user}, nil
}

func (s *AuthService) GetUser(id interface{}) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
