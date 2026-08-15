package facade

import (
	"crypto/md5"
	"fmt"
	"time"

	JWT "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

const (
	ConfigNameCrypt   = "crypt"
	DefaultJwtExpire  = "15 * 24 * 60 * 60" // 默认登录会话有效期：15 天
	DefaultJwtIssuer  = "inis"
	DefaultJwtSubject = "inis"
)

// CryptToml - Crypt配置文件
var CryptToml *utils.ViperResponse

func init() {
	initCryptToml()
	initCrypt()

	WatchConfigChange(CryptToml, initCrypt)
}

// initCryptToml - 初始化Crypt配置文件
func initCryptToml() {
	key := fmt.Sprintf("%s-%v", uuid.New().String(), time.Now().Unix())
	secret := fmt.Sprintf("INIS-%x", md5.Sum([]byte(key)))

	item := utils.Viper(utils.ViperModel{
		Path: ConfigPath,
		Mode: ModeToml,
		Name: ConfigNameCrypt,
		Content: utils.Replace(TempCrypt, map[string]any{
			"${jwt.key}":     secret,
			"${jwt.expire}":  DefaultJwtExpire,
			"${jwt.issuer}":  DefaultJwtIssuer,
			"${jwt.subject}": DefaultJwtSubject,
		}),
	}).Read()

	if item.Error != nil {
		Log.Error(map[string]any{
			"error":     item.Error,
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line}, "Crypt配置初始化错误")
	}

	CryptToml = &item
}

func initCrypt() {

}

// JWT相关结构体和接口
type JwtStruct struct {
	request  JwtRequest
	response JwtResponse
}

type JwtRequest struct {
	Expire  int64  `json:"expire"`
	Issuer  string `json:"issuer"`
	Subject string `json:"subject"`
	Key     string `json:"key"`
}

type JwtResponse struct {
	Text  string         `json:"text"`
	Data  map[string]any `json:"data"`
	Error error          `json:"error"`
	Valid int64          `json:"valid"`
}

type JwtClaims struct {
	Data map[string]any `json:"data"`
	JWT.RegisteredClaims
}

// Jwt - JWT入口函数
func Jwt(request ...JwtRequest) *JwtStruct {
	req := JwtRequest{}

	if len(request) > 0 {
		req = request[0]
	}

	// 设置默认值
	if req.Expire == 0 {
		req.Expire = cast.ToInt64(utils.Calc(CryptToml.Get("jwt.expire", DefaultJwtExpire)))
	}
	if utils.Is.Empty(req.Issuer) {
		req.Issuer = cast.ToString(CryptToml.Get("jwt.issuer", DefaultJwtIssuer))
	}
	if utils.Is.Empty(req.Subject) {
		req.Subject = cast.ToString(CryptToml.Get("jwt.subject", DefaultJwtSubject))
	}
	if utils.Is.Empty(req.Key) {
		req.Key = cast.ToString(CryptToml.Get("jwt.key", ""))
	}

	return &JwtStruct{
		request: req,
		response: JwtResponse{
			Data: make(map[string]any),
		},
	}
}

// Create - 创建JWT
func (this *JwtStruct) Create(data map[string]any) JwtResponse {

	// 密钥为空时禁止签发，避免使用空密钥导致的安全问题
	if utils.Is.Empty(this.request.Key) {
		this.response.Error = fmt.Errorf("JWT密钥未配置，无法签发令牌！")
		Log.Error(map[string]any{
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line,
		}, "JWT密钥为空")
		return this.response
	}

	now := time.Now()
	claims := JwtClaims{
		Data: data,
		RegisteredClaims: JWT.RegisteredClaims{
			ID:        uuid.New().String(), // jti - 唯一标识，可用于令牌撤销
			IssuedAt:  JWT.NewNumericDate(now),
			NotBefore: JWT.NewNumericDate(now.Add(-time.Minute)), // 允许 1 分钟以内的时钟偏移
			ExpiresAt: JWT.NewNumericDate(now.Add(time.Duration(this.request.Expire) * time.Second)),
			Issuer:    this.request.Issuer,
			Subject:   this.request.Subject,
		},
	}

	token, err := JWT.NewWithClaims(JWT.SigningMethodHS256, claims).SignedString([]byte(this.request.Key))
	if err != nil {
		this.response.Error = err
		return this.response
	}

	this.response.Text = token
	this.response.Valid = this.request.Expire // 有效时长（秒），供前端同步 cookie/缓存过期时间
	return this.response
}

// Parse - 解析JWT
func (this *JwtStruct) Parse(token any) JwtResponse {

	claims := &JwtClaims{}
	jwtToken, err := JWT.ParseWithClaims(cast.ToString(token), claims, func(token *JWT.Token) (any, error) {
		// 校验签名算法，防止算法混淆攻击（alg=none）
		if _, ok := token.Method.(*JWT.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非法的签名算法：%v", token.Header["alg"])
		}
		return []byte(this.request.Key), nil
	},
		// 严格校验签发者与主题
		JWT.WithIssuer(this.request.Issuer),
		JWT.WithSubject(this.request.Subject),
		// 允许 1 分钟的时钟偏移
		JWT.WithLeeway(time.Minute),
	)

	if err != nil {
		Log.Error(map[string]any{
			"error":     err,
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line}, "JWT解析错误")
		this.response.Error = err
		return this.response
	}

	if jwtToken.Valid {
		this.response.Data = claims.Data
		this.response.Valid = claims.ExpiresAt.Time.Unix() - time.Now().Unix()
	}

	return this.response
}
