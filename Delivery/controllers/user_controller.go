package controllers

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userUsecase usecases.UserUsecase
}

func NewUserController(uc usecases.UserUsecase) *UserController {
	return &UserController{userUsecase: uc}
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Username string `json:"username,omitempty"`
	Password string `json:"password" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RefreshRequest struct {
	AccessToken  string `json:"access_token" binding:"required"`
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ForgetPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type UpdateProfileRequest struct {
	Bio           string `json:"bio"`
	ProfilePicURL string `json:"profilePictureUrl"`
}

type UserResponse struct {
	ID             string `json:"id"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	Bio            string `json:"bio"`
	ProfilePicture string `json:"profilepicture"`
	Role           string `json:"role"`
}

func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:             u.ID,
		Username:       u.Username,
		Email:          u.Email,
		Bio:            u.Bio,
		ProfilePicture: u.ProfilePicture,
		Role:           string(u.Role),
	}
}

func (ctrl *UserController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := &domain.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Role:     domain.RoleUser,
	}

	err := ctrl.userUsecase.Register(c.Request.Context(), user)
	if err != nil {
		switch err {
		case domain.ErrEmailExists:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case domain.ErrPasswordTooShort, domain.ErrInvalidEmailFormat:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occurred"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully", "id": user.ID})
}

func (ctrl *UserController) ActivateAccount(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "activation token is required"})
		return
	}

	err := ctrl.userUsecase.ActivateAccount(c.Request.Context(), token)
	if err != nil {
		switch err {
		case domain.ErrInvalidActivationToken:
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occurred"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account activated successfully. You may now log in."})
}

func (ctrl *UserController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	//validate that at least one identifier is provided.
	if req.Username == "" && req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email or username is required"})
		return
	}

	// prefer email if both are provided

	identifier := req.Email
	if identifier == "" {
		identifier = req.Username
	}
	accessToken, refreshToken, err := ctrl.userUsecase.Login(c.Request.Context(), identifier, req.Password)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrAuthenticationFailed.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "refresh_token": refreshToken})
}

func (ctrl *UserController) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}

	user, err := ctrl.userUsecase.GetProfile(c.Request.Context(), userID.(string))
	if err != nil {
		if err == domain.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

func (ctrl *UserController) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newAccessToken, newRefreshToken, err := ctrl.userUsecase.RefreshAccessToken(c.Request.Context(), req.AccessToken, req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": newAccessToken, "refresh_token": newRefreshToken})
}

func (ctrl *UserController) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := ctrl.userUsecase.Logout(c.Request.Context(), req.RefreshToken)
	if err != nil {
		log.Printf("Logout error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occurred during logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func (ctrl *UserController) ForgetPassword(c *gin.Context) {
	var req ForgetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := ctrl.userUsecase.ForgetPassword(c.Request.Context(), req.Email)
	if err != nil {
		log.Printf("ForgotPassword error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "a password reset link has been sent"})
}

func (ctrl *UserController) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := ctrl.userUsecase.ResetPassword(c.Request.Context(), req.Token, req.NewPassword)
	if err != nil {
		switch err {
		case domain.ErrInvalidResetToken, domain.ErrPasswordTooShort:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			log.Printf("ResetPassword error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occurred"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password has been reset successfully"})
}

func (ctrl *UserController) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}

	updatedUser, err := ctrl.userUsecase.UpdateProfile(c.Request.Context(), userID.(string), req.Bio, req.ProfilePicURL)
	if err != nil {
		if err == domain.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			log.Printf("UpdateProfile error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occurred"})
		}
		return
	}

	c.JSON(http.StatusOK, toUserResponse(updatedUser))

}
