package util

import (
	"errors"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// JWT和密码相关的哨兵错误
var (
	ErrPasswordEmpty       = errors.New("password is empty")
	ErrPasswordTooShort    = errors.New("password is too short")
	ErrPasswordTooWeak     = errors.New("password is too weak")
	ErrPasswordTooLong     = errors.New("password is too long")
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenExpired        = errors.New("token is expired")
	ErrTokenEmpty          = errors.New("token is empty")
	ErrInvalidSignMethod   = errors.New("invalid signing method")
	ErrUserIDEmpty         = errors.New("user id is empty")
	ErrTokenNotRefreshable = errors.New("token is not refreshable")
)

// 密码加密
// 使用 bcrypt 生成密码哈希
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrPasswordEmpty
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ValidatePasswordStrength 验证密码强度
// 密码要求：长度8-16，包含大小写字母、数字、特殊字符中的至少3种 常规要求
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	if len(password) > 16 {
		return ErrPasswordTooLong
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	// 至少包含3种类型
	count := 0
	if hasUpper {
		count++
	}
	if hasLower {
		count++
	}
	if hasDigit {
		count++
	}
	if hasSpecial {
		count++
	}

	if count < 3 {
		return ErrPasswordTooWeak
	}
	return nil
}

// ComparePassword 校验明文密码与哈希是否匹配
func ComparePassword(hash string, plain string) (bool, error) {
	if hash == "" || plain == "" {
		return false, ErrPasswordEmpty
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

//

// jwt  后续登录给到token
// JWT工具类
type JWTUtil struct {
	secretKey   []byte //密钥
	expireHours int    //过期时间
}

// JWT声明
type Claims struct {
	UserID string `json:"user_id"` //用户唯一标识  解析token识别用户
	jwt.RegisteredClaims
}

// 创建JWT工具实例
func NewJWTUtil(secretKey string, expireHours int) *JWTUtil {
	// 如果过期时间为0或负数，设置默认值为24小时
	if expireHours <= 0 {
		expireHours = 24
	}
	return &JWTUtil{
		secretKey:   []byte(secretKey),
		expireHours: expireHours,
	}
}

// GenerateToken 生成jwt令牌
func (j *JWTUtil) GenerateToken(userID string) (string, error) {
	if userID == "" {
		return "", ErrUserIDEmpty
	}

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(j.expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// ValidateToken 验证JWT令牌
func (j *JWTUtil) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrTokenEmpty
	}

	// 解析令牌
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSignMethod
		}
		return j.secretKey, nil
	})

	if err != nil {
		// jwt库返回的过期错误特殊处理
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, err
	}

	// 获取声明
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// RefreshToken 刷新令牌
func (j *JWTUtil) RefreshToken(tokenString string) (string, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// 检查令牌是否即将过期（剩余时间少于1小时）
	remainingTime := time.Until(claims.ExpiresAt.Time)
	if remainingTime < 0 {
		// 已经过期，返回错误
		return "", ErrTokenExpired
	}
	if remainingTime < time.Hour {
		// 小于1小时，重新生成
		return j.GenerateToken(claims.UserID)
	}

	// 还有超过1小时才过期，不需要刷新
	return tokenString, nil
}
