package userservice

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"seekF-backend/internal/configs"
	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/auth"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/email"
	"seekF-backend/internal/pkg/jwt"
	"seekF-backend/internal/pkg/oauth"
	"seekF-backend/internal/pkg/redis"
	"seekF-backend/internal/pkg/sms"
	"seekF-backend/internal/pkg/util"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Nickname  string `json:"nickname" binding:"required"`
	Telephone string `json:"telephone"`
	Email     string `json:"email"`
	Code      string `json:"code" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Telephone string `json:"telephone"`
	Email     string `json:"email"`
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
	Register(ctx context.Context, req *RegisterRequest) error
	Login(ctx context.Context, req *LoginRequest) (*LoginRespond, error)
	Logout(tokenString string) error
	SendVerifyCode(telephone, email string) error
	LoginByCode(ctx context.Context, telephone, email, code string) (*LoginRespond, error)
	// 生成GitHub授权地址及需要写入浏览器的临时状态。
	GithubAuthURL(ctx context.Context) (authURL, state string, err error)
	// 验证浏览器绑定并完成GitHub登录。
	LoginByGithub(ctx context.Context, code, state, browserState string) (*LoginRespond, error)
	// 生成Gitee授权地址及需要写入浏览器的临时状态。
	GiteeAuthURL(ctx context.Context) (authURL, state string, err error)
	// 验证浏览器绑定并完成Gitee登录。
	LoginByGitee(ctx context.Context, code, state, browserState string) (*LoginRespond, error)
	// 验证授权取消回调并消费对应的一次性状态。
	CancelOAuthLogin(ctx context.Context, provider, state, browserState string) error
}

type AuthServiceImpl struct {
	userInfoDAO userdao.UserInfoDAO
}

func NewAuthService(userInfoDAO userdao.UserInfoDAO) AuthService {
	return &AuthServiceImpl{
		userInfoDAO: userInfoDAO,
	}
}

// Register 用户注册（支持手机号和邮箱）
func (s *AuthServiceImpl) Register(ctx context.Context, req *RegisterRequest) error {
	// 校验：手机号和邮箱至少填一个
	if strings.TrimSpace(req.Telephone) == "" && strings.TrimSpace(req.Email) == "" {
		return fmt.Errorf("手机号和邮箱不能同时为空")
	}

	// 验证验证码
	var verifyKey string
	if isEmail(req.Email) {
		verifyKey = fmt.Sprintf("verify_code:%s", req.Email)
	} else {
		verifyKey = fmt.Sprintf("verify_code:%s", req.Telephone)
	}

	storedCode, err := redis.GetKey(verifyKey)
	if err != nil {
		return fmt.Errorf("获取验证码失败：%v", err)
	}
	if storedCode == "" {
		return fmt.Errorf("%w：验证码不存在或已过期", auth.ErrInvalidCredentials)
	}
	if storedCode != req.Code {
		return fmt.Errorf("%w：验证码错误", auth.ErrInvalidCredentials)
	}

	// 验证成功，删除验证码
	if err := redis.DelKeyIfExists(verifyKey); err != nil {
		return fmt.Errorf("删除验证码失败：%v", err)
	}

	// 检查用户名是否已存在
	existingUser, err := s.userInfoDAO.FindUserByNickname(ctx, req.Nickname)
	if err != nil {
		return err
	}
	if existingUser != nil {
		return fmt.Errorf("该用户名已被使用")
	}

	// 创建新用户
	password, err := encryptPassword(req.Password)
	if err != nil {
		return err
	}

	// 生成UUID
	userUUID := "U" + util.GetNowAndLenRandomString(11)

	user := &models.UserInfo{
		Uuid:     userUUID,
		Nickname: req.Nickname,
		Password: password,
	}

	if isEmail(req.Email) {
		// 邮箱注册：检查邮箱是否已存在
		existingUser, err := s.userInfoDAO.FindUserByEmail(ctx, req.Email)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return fmt.Errorf("该邮箱已被注册")
		}
		user.Email = req.Email
		user.Telephone = ""
	} else {
		// 手机号注册：检查手机号是否已存在
		existingUser, err := s.userInfoDAO.FindUserByTelephone(ctx, req.Telephone)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return fmt.Errorf("该手机号已被注册")
		}
		user.Telephone = req.Telephone
	}

	err = s.userInfoDAO.CreateUser(ctx, user)
	if err != nil {
		return err
	}

	return nil
}

// Login 用户登录（支持手机号和邮箱）
func (s *AuthServiceImpl) Login(ctx context.Context, req *LoginRequest) (*LoginRespond, error) {
	if strings.TrimSpace(req.Telephone) == "" && strings.TrimSpace(req.Email) == "" {
		return nil, fmt.Errorf("手机号和邮箱不能同时为空")
	}

	// 根据手机号或邮箱查找用户
	var user *models.UserInfo
	var err error

	if isEmail(req.Email) {
		user, err = s.userInfoDAO.FindUserByEmail(ctx, req.Email)
	} else {
		user, err = s.userInfoDAO.FindUserByTelephone(ctx, req.Telephone)
	}
	if err != nil {
		return nil, fmt.Errorf("登录失败：%v", err)
	}

	if user == nil {
		return nil, fmt.Errorf("%w：该用户不存在", auth.ErrInvalidCredentials)
	}

	// 验证密码
	err = verifyPassword(user.Password, req.Password)
	if err != nil {
		return nil, fmt.Errorf("%w：密码错误", auth.ErrInvalidCredentials)
	}

	cfg := configs.GetConfig()
	mode := strings.ToLower(strings.TrimSpace(cfg.AuthConfig.Mode))

	token, err := s.createLoginToken(user, mode)
	if err != nil {
		return nil, err
	}

	// 构造登录响应
	loginRsp := s.buildLoginRespond(user, token)

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

// SendVerifyCode 发送验证码（支持手机号和邮箱）
func (s *AuthServiceImpl) SendVerifyCode(telephone, emailAddr string) error {
	// 校验：手机号和邮箱至少填一个
	if strings.TrimSpace(telephone) == "" && strings.TrimSpace(emailAddr) == "" {
		return fmt.Errorf("手机号和邮箱不能同时为空")
	}

	// 生成6位数字验证码
	code := generateVerifyCode()

	// 验证码有效期5分钟
	expiresAt := 5 * time.Minute

	// 判断是邮箱还是手机号
	if isEmail(emailAddr) {
		// 邮箱验证码
		key := fmt.Sprintf("verify_code:%s", emailAddr)
		err := redis.SetKeyEx(key, code, expiresAt)
		if err != nil {
			return fmt.Errorf("存储验证码失败：%v", err)
		}

		// 发送邮件验证码
		err = email.SendVerifyCodeEmail(emailAddr, code)
		if err != nil {
			return fmt.Errorf("发送验证码失败：%v", err)
		}
	} else {
		// 手机短信验证码
		key := fmt.Sprintf("verify_code:%s", telephone)
		err := redis.SetKeyEx(key, code, expiresAt)
		if err != nil {
			return fmt.Errorf("存储验证码失败：%v", err)
		}

		// 发送短信验证码
		err = sms.SendCode(telephone, code)
		if err != nil {
			return fmt.Errorf("发送验证码失败：%v", err)
		}
	}

	return nil
}

// LoginByCode 验证码登录（支持手机号和邮箱）
func (s *AuthServiceImpl) LoginByCode(ctx context.Context, telephone, emailAddr, code string) (*LoginRespond, error) {
	// 判断是邮箱还是手机号
	var key string
	var isEmailLogin bool

	if isEmail(emailAddr) {
		key = fmt.Sprintf("verify_code:%s", emailAddr)
		isEmailLogin = true
	} else if strings.TrimSpace(telephone) != "" {
		key = fmt.Sprintf("verify_code:%s", telephone)
		isEmailLogin = false
	} else {
		return nil, fmt.Errorf("手机号和邮箱不能同时为空")
	}

	// 从Redis获取验证码
	storedCode, err := redis.GetKey(key)
	if err != nil {
		return nil, fmt.Errorf("获取验证码失败：%v", err)
	}

	if storedCode == "" {
		return nil, fmt.Errorf("%w：验证码不存在或已过期", auth.ErrInvalidCredentials)
	}

	if storedCode != code {
		return nil, fmt.Errorf("%w：验证码错误", auth.ErrInvalidCredentials)
	}

	// 验证码验证成功，删除验证码
	err = redis.DelKeyIfExists(key)
	if err != nil {
		return nil, fmt.Errorf("删除验证码失败：%v", err)
	}

	// 根据手机号或邮箱查找用户
	var user *models.UserInfo
	if isEmailLogin {
		user, err = s.userInfoDAO.FindUserByEmail(ctx, emailAddr)
	} else {
		user, err = s.userInfoDAO.FindUserByTelephone(ctx, telephone)
	}
	if err != nil {
		return nil, fmt.Errorf("登录失败：%v", err)
	}

	if user == nil {
		// 如果用户不存在，自动注册
		userUUID := "U" + util.GetNowAndLenRandomString(11)
		user = &models.UserInfo{
			Uuid:     userUUID,
			Password: "", // 验证码登录不需要密码
		}

		if isEmailLogin {
			// 邮箱注册：昵称取邮箱前缀
			parts := strings.Split(emailAddr, "@")
			nickname := "用户"
			if len(parts) > 0 && len(parts[0]) > 0 {
				nickname = parts[0]
			}
			user.Nickname = nickname
			user.Email = emailAddr
			user.Telephone = ""
		} else {
			// 手机号注册
			user.Nickname = "用户" + telephone[len(telephone)-4:]
			user.Telephone = telephone
		}

		err = s.userInfoDAO.CreateUser(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("自动注册失败：%v", err)
		}
	}

	// 生成token
	cfg := configs.GetConfig()
	mode := strings.ToLower(strings.TrimSpace(cfg.AuthConfig.Mode))

	token, err := s.createLoginToken(user, mode)
	if err != nil {
		return nil, err
	}

	// 构造登录响应
	loginRsp := s.buildLoginRespond(user, token)

	return loginRsp, nil
}

// GithubAuthURL 生成 GitHub OAuth 授权地址
func (s *AuthServiceImpl) GithubAuthURL(ctx context.Context) (authURL, state string, err error) {
	cfg := configs.GetConfig()
	if strings.TrimSpace(cfg.GithubOAuthConfig.ClientID) == "" {
		return "", "", fmt.Errorf("GitHub OAuth 未配置")
	}

	state, err = createOAuthState(ctx, constants.OAuthProviderGithub)
	if err != nil {
		return "", "", err
	}
	return oauth.GetAuthCodeURL(state), state, nil
}

// LoginByGithub GitHub OAuth 登录
func (s *AuthServiceImpl) LoginByGithub(ctx context.Context, code, state, browserState string) (*LoginRespond, error) {
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("授权码不能为空")
	}
	if err := consumeOAuthState(ctx, constants.OAuthProviderGithub, state, browserState); err != nil {
		return nil, err
	}

	githubUser, err := oauth.ExchangeAndGetUser(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("GitHub 登录失败：%v", err)
	}

	user, err := s.userInfoDAO.FindUserByGithubId(ctx, githubUser.ID)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败：%v", err)
	}

	if user == nil {
		nickname := strings.TrimSpace(githubUser.Name)
		if nickname == "" {
			nickname = githubUser.Login
		}
		if nickname == "" {
			nickname = "GitHub用户"
		}

		userUUID := "U" + util.GetNowAndLenRandomString(11)
		user = &models.UserInfo{
			Uuid:     userUUID,
			Nickname: nickname,
			Avatar:   githubUser.AvatarURL,
			GithubId: sql.NullInt64{Int64: githubUser.ID, Valid: true},
			Password: "",
			Birthday: sql.NullString{Valid: false},
		}

		user, err = createOAuthUser(ctx, s.userInfoDAO, user, func() (*models.UserInfo, error) {
			return s.userInfoDAO.FindUserByGithubId(ctx, githubUser.ID)
		})
		if err != nil {
			return nil, fmt.Errorf("创建 GitHub 用户失败：%v", err)
		}
	} else {
		updated := false
		if githubUser.AvatarURL != "" && user.Avatar != githubUser.AvatarURL {
			user.Avatar = githubUser.AvatarURL
			updated = true
		}
		if updated {
			if err := s.userInfoDAO.UpdateUserInfo(ctx, user); err != nil {
				return nil, fmt.Errorf("更新 GitHub 用户信息失败：%v", err)
			}
		}
	}

	cfg := configs.GetConfig()
	mode := strings.ToLower(strings.TrimSpace(cfg.AuthConfig.Mode))

	token, err := s.createLoginToken(user, mode)
	if err != nil {
		return nil, err
	}

	return s.buildLoginRespond(user, token), nil
}

// GiteeAuthURL 生成 Gitee OAuth 授权地址
func (s *AuthServiceImpl) GiteeAuthURL(ctx context.Context) (authURL, state string, err error) {
	cfg := configs.GetConfig()
	if strings.TrimSpace(cfg.GiteeOAuthConfig.ClientID) == "" {
		return "", "", fmt.Errorf("Gitee OAuth 未配置")
	}

	state, err = createOAuthState(ctx, constants.OAuthProviderGitee)
	if err != nil {
		return "", "", err
	}
	return oauth.GetGiteeAuthCodeURL(state), state, nil
}

// LoginByGitee Gitee OAuth 登录
func (s *AuthServiceImpl) LoginByGitee(ctx context.Context, code, state, browserState string) (*LoginRespond, error) {
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("授权码不能为空")
	}
	if err := consumeOAuthState(ctx, constants.OAuthProviderGitee, state, browserState); err != nil {
		return nil, err
	}

	giteeUser, err := oauth.ExchangeAndGetGiteeUser(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("Gitee 登录失败：%v", err)
	}

	user, err := s.userInfoDAO.FindUserByGiteeId(ctx, giteeUser.ID)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败：%v", err)
	}

	if user == nil {
		nickname := strings.TrimSpace(giteeUser.Name)
		if nickname == "" {
			nickname = giteeUser.Login
		}
		if nickname == "" {
			nickname = "Gitee用户"
		}

		userUUID := "U" + util.GetNowAndLenRandomString(11)
		user = &models.UserInfo{
			Uuid:     userUUID,
			Nickname: nickname,
			Avatar:   giteeUser.AvatarURL,
			GiteeId:  sql.NullInt64{Int64: giteeUser.ID, Valid: true},
			Password: "",
			Birthday: sql.NullString{Valid: false},
		}

		user, err = createOAuthUser(ctx, s.userInfoDAO, user, func() (*models.UserInfo, error) {
			return s.userInfoDAO.FindUserByGiteeId(ctx, giteeUser.ID)
		})
		if err != nil {
			return nil, fmt.Errorf("创建 Gitee 用户失败：%v", err)
		}
	} else {
		updated := false
		if giteeUser.AvatarURL != "" && user.Avatar != giteeUser.AvatarURL {
			user.Avatar = giteeUser.AvatarURL
			updated = true
		}
		if updated {
			if err := s.userInfoDAO.UpdateUserInfo(ctx, user); err != nil {
				return nil, fmt.Errorf("更新 Gitee 用户信息失败：%v", err)
			}
		}
	}

	cfg := configs.GetConfig()
	mode := strings.ToLower(strings.TrimSpace(cfg.AuthConfig.Mode))

	token, err := s.createLoginToken(user, mode)
	if err != nil {
		return nil, err
	}

	return s.buildLoginRespond(user, token), nil
}

func (s *AuthServiceImpl) createLoginToken(user *models.UserInfo, mode string) (string, error) {
	if mode == "jwt" {
		token, err := jwt.GenerateToken(uint64(user.Id), user.Uuid, user.Telephone, user.Nickname)
		if err != nil {
			return "", fmt.Errorf("生成令牌失败：%v", err)
		}
		return token, nil
	}

	token, err := auth.GenerateToken()
	if err != nil {
		return "", fmt.Errorf("生成令牌失败：%v", err)
	}
	if err := auth.SetSession(token, auth.Session{
		Id:       uint64(user.Id),
		UUID:     user.Uuid,
		Phone:    user.Telephone,
		Nickname: user.Nickname,
	}); err != nil {
		return "", fmt.Errorf("生成令牌失败：%v", err)
	}
	return token, nil
}

func (s *AuthServiceImpl) buildLoginRespond(user *models.UserInfo, token string) *LoginRespond {
	birthday := ""
	if user.Birthday.Valid {
		birthday = user.Birthday.String
	}
	return &LoginRespond{
		User: UserInfoResponse{
			Uuid:      user.Uuid,
			Telephone: user.Telephone,
			Nickname:  user.Nickname,
			Email:     user.Email,
			Avatar:    user.Avatar,
			Gender:    int(user.Gender),
			Birthday:  birthday,
			Signature: user.Signature,
			IsAdmin:   int(user.IsAdmin),
			Status:    int(user.Status),
		},
		Token: token,
	}
}

// CancelOAuthLogin 验证取消授权的请求属于当前浏览器，并消费一次性状态。
func (s *AuthServiceImpl) CancelOAuthLogin(ctx context.Context, provider, state, browserState string) error {
	return consumeOAuthState(ctx, provider, state, browserState)
}

// createOAuthState 创建有时限的平台独立授权记录，等待回调与浏览器Cookie核对。
func createOAuthState(ctx context.Context, provider string) (string, error) {
	state, err := generateOAuthState()
	if err != nil {
		return "", fmt.Errorf("生成 OAuth 状态码失败: %w", err)
	}
	key := constants.OAuthStateKeyPrefix + provider + ":" + state
	if err := redis.GetClient().Set(ctx, key, "1", constants.OAuthStateTTL).Err(); err != nil {
		return "", fmt.Errorf("保存 OAuth 状态码失败: %w", err)
	}
	return state, nil
}

// consumeOAuthState 先验证浏览器与回调匹配，再原子消费记录，拒绝跨浏览器和重复回调。
func consumeOAuthState(ctx context.Context, provider, state, browserState string) error {
	if provider != constants.OAuthProviderGithub && provider != constants.OAuthProviderGitee {
		return fmt.Errorf("OAuth 授权平台无效")
	}
	if len(state) != constants.OAuthStateLength || len(browserState) != constants.OAuthStateLength || subtle.ConstantTimeCompare([]byte(state), []byte(browserState)) != 1 {
		return fmt.Errorf("OAuth 状态与当前浏览器不匹配，请重新发起登录")
	}
	if _, err := hex.DecodeString(state); err != nil {
		return fmt.Errorf("OAuth 状态码格式无效")
	}
	key := constants.OAuthStateKeyPrefix + provider + ":" + state
	consumed, err := redis.CompareAndDeleteString(ctx, key, "1")
	if err != nil {
		return fmt.Errorf("校验 OAuth 状态码失败: %w", err)
	}
	if !consumed {
		return fmt.Errorf("OAuth 状态码无效、已过期或已使用")
	}
	return nil
}

// generateOAuthState 使用密码学随机数生成一次授权的状态标识。
func generateOAuthState() (string, error) {
	buf := make([]byte, constants.OAuthStateLength/2)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// generateVerifyCode 生成6位数字验证码（使用密码学安全的随机数）
func generateVerifyCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", n.Int64()+100000)
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

// isEmail 判断字符串是否为邮箱格式
func isEmail(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	// 简单的邮箱格式校验
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(s)
}
