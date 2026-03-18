package userservice

import (
	"fmt"
	"seekF-backend/internal/configs"
	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/auth"
	"seekF-backend/internal/pkg/jwt"
	"seekF-backend/internal/pkg/util"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Nickname  string `json:"nickname" binding:"required"`
	Telephone string `json:"telephone" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Telephone string `json:"telephone" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

type LoginRespond struct {
	User  UserInfoResponse `json:"user"`
	Token string           `json:"token"`
}

type UserInfoResponse struct {
	Uuid      string `json:"uuid"`
	Telephone string `json:"telephone"`
	Nickname  string `json:"nickname"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
	Gender    int    `json:"gender"`
	Birthday  string `json:"birthday"`
	Signature string `json:"signature"`
	IsAdmin   int    `json:"isAdmin"`
	Status    int    `json:"status"`
}

type AuthService interface {
	Register(req *RegisterRequest) error
	Login(req *LoginRequest) (*LoginRespond, error)
	Logout(tokenString string) error
}

type AuthServiceImpl struct {
	userInfoDAO userdao.UserInfoDAO
}

func NewAuthService(userInfoDAO userdao.UserInfoDAO) AuthService {
	return &AuthServiceImpl{
		userInfoDAO: userInfoDAO,
	}
}

// Register 用户注册
func (s *AuthServiceImpl) Register(req *RegisterRequest) error {
	// 检查手机号是否已存在
	existingUser, err := s.userInfoDAO.FindUserByTelephone(req.Telephone)
	if err != nil {
		return err
	}
	if existingUser != nil {
		return fmt.Errorf("该手机号已被注册")
	}

	// 创建新用户
	password, err := encryptPassword(req.Password)
	if err != nil {
		return err
	}

	// 生成UUID
	userUUID := "U" + util.GetNowAndLenRandomString(11)

	user := &models.UserInfo{
		Uuid:      userUUID,
		Nickname:  req.Nickname,
		Telephone: req.Telephone,
		Password:  password,
	}

	err = s.userInfoDAO.CreateUser(user)
	if err != nil {
		return err
	}

	return nil
}

// Login 用户登录
func (s *AuthServiceImpl) Login(req *LoginRequest) (*LoginRespond, error) {
	// 根据手机号查找用户
	user, err := s.userInfoDAO.FindUserByTelephone(req.Telephone)
	if err != nil {
		return nil, fmt.Errorf("登录失败：%v", err)
	}

	if user == nil {
		return nil, fmt.Errorf("该用户不存在")
	}

	// 验证密码
	err = verifyPassword(user.Password, req.Password)
	if err != nil {
		return nil, fmt.Errorf("密码错误")
	}

	cfg := configs.GetConfig()
	mode := strings.ToLower(strings.TrimSpace(cfg.AuthConfig.Mode))

	var token string
	if mode == "jwt" {
		// JWT 方案：纯 JWT（无服务端状态）
		token, err = jwt.GenerateToken(uint64(user.Id), user.Uuid, user.Telephone, user.Nickname)
		if err != nil {
			return nil, fmt.Errorf("生成令牌失败：%v", err)
		}
	} else {
		// 默认方案：不透明 token + Redis 会话
		token, err = auth.GenerateToken()
		if err != nil {
			return nil, fmt.Errorf("生成令牌失败：%v", err)
		}
		if err := auth.SetSession(token, auth.Session{
			Id:       uint64(user.Id),
			UUID:     user.Uuid,
			Phone:    user.Telephone,
			Nickname: user.Nickname,
		}); err != nil {
			return nil, fmt.Errorf("生成令牌失败：%v", err)
		}
	}

	// 构造登录响应
	loginRsp := &LoginRespond{
		User: UserInfoResponse{
			Uuid:      user.Uuid,
			Telephone: user.Telephone,
			Nickname:  user.Nickname,
			Email:     user.Email,
			Avatar:    user.Avatar,
			Gender:    int(user.Gender),
			Birthday:  user.Birthday,
			Signature: user.Signature,
			IsAdmin:   int(user.IsAdmin),
			Status:    int(user.Status),
		},
		Token: token,
	}

	return loginRsp, nil
}

// Logout 用户登出
func (s *AuthServiceImpl) Logout(tokenString string) error {
	cfg := configs.GetConfig()
	mode := strings.ToLower(strings.TrimSpace(cfg.AuthConfig.Mode))

	if mode == "jwt" {
		// 纯 JWT 无服务端会话可删；客户端自行丢弃 token
		_ = tokenString
		return nil
	}

	// token+redis：删除会话
	if err := auth.DelSession(tokenString); err != nil {
		return fmt.Errorf("删除token失败: %v", err)
	}
	return nil
}

// 加密密码
func encryptPassword(password string) (string, error) {
	// 使用bcrypt加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// 验证密码
func verifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
