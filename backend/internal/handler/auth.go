package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/msqfX/CertPilot/backend/internal/config"
	"github.com/msqfX/CertPilot/backend/internal/model"
)

type AuthHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var count int64
	h.DB.Model(&model.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "邮箱已注册"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	u := model.User{Email: req.Email, PasswordHash: string(hash), Nickname: req.Nickname}
	if err := h.DB.Create(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.tokenResponse(u))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var u model.User
	if err := h.DB.Where("email = ?", req.Email).First(&u).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
		return
	}
	c.JSON(http.StatusOK, h.tokenResponse(u))
}

func (h *AuthHandler) Me(c *gin.Context) {
	uid := c.GetUint("uid")
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": u.ID, "email": u.Email, "nickname": u.Nickname})
}

func (h *AuthHandler) tokenResponse(u model.User) gin.H {
	claims := jwt.MapClaims{
		"uid": u.ID,
		"exp": time.Now().Add(time.Duration(h.Cfg.JWTExpire) * time.Hour).Unix(),
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.Cfg.JWTSecret))
	return gin.H{"token": token, "user": gin.H{"id": u.ID, "email": u.Email, "nickname": u.Nickname}}
}
