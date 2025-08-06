package controllers

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ===========================================
// USER CONTROLLER
// ===========================================

type UserController struct {
	userUsecase usecases.UserUsecase
}

func NewUserController(uc usecases.UserUsecase) *UserController {
	return &UserController{
		userUsecase: uc,
	}
}

type UserResponse struct {
	ID    string `json:"id"`
	Username string  `json:"username"`
	Email  string `json:"email"`
	Bio string `json:"bio"`
	ProfilePicture  string `json:"profilepicture"`
	Role string `json:"role"`
}

func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID: u.ID,
		Username: u.Username,
		Email: u.Email,
		Bio: u.Bio,
		ProfilePicture: u.ProfilePicture,
		Role: string(u.Role),
	}
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
	AccessToken string `json:"access_token" binding:"required"`
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ForgetPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token     string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type UpdateProfileRequest struct {
	Bio         string `json:"bio"`
	ProfilePicURL  string `json:"profilePictureUrl"`
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
		Role:     domain.RoleUser, // Default role
	}

	err := ctrl.userUsecase.Register(c.Request.Context(), user)
	if err != nil {
		// Map domain errors to HTTP status codes
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

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

func (ctrl *UserController) ActivateAccount(c *gin.Context) {
	// The token is usually sent as a query parameter in the URL, e.g., /activate?token=...
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "activation token is required"})
		return
	}

	err := ctrl.userUsecase.ActivateAccount(c.Request.Context(), token)
	if err != nil {
		// Map domain errors to status codes
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

// GetProfile demonstrates a protected route
func (ctrl *UserController) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}

	user, err := ctrl.userUsecase.GetProfile(c.Request.Context(), userID.(string))
	if err != nil {
		if err == domain.ErrUserNotFound{
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

func (ctrl *UserController) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error":err.Error()})
		return
	}

	newAccessToken, newrefreshToken, err := ctrl.userUsecase.RefreshAccessToken(c.Request.Context(), req.AccessToken, req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired tokens"})
		return
	}

	// Return the new pair of tokens.
	c.JSON(http.StatusOK, gin.H{"access_token": newAccessToken, "refresh_token": newrefreshToken})
}

func(ctrl *UserController) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := ctrl.userUsecase.Logout(c.Request.Context(), req.RefreshToken)
	if err != nil {
		log.Printf("Internal Server Error during logout: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occured during logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func(ctrl *UserController) ForgetPassword(c *gin.Context) {
	var req ForgetPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := ctrl.userUsecase.ForgetPassword(c.Request.Context(), req.Email)
	if err != nil {
		log.Printf("Internal Server Error during ForgotPassword: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occurred"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "If an account with that email exists, a password reset link has been sent."})
}

func(ctrl *UserController) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := ctrl.userUsecase.ResetPassword(c.Request.Context(), req.Token, req.NewPassword)
	if err != nil {
		switch err {
		case domain.ErrInvalidResetToken:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case domain.ErrPasswordTooShort:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			log.Printf("Internal Server Error during ResetPassword: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occured"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Your password has been rest successfully."})
}

func(ctrl *UserController) UpdateProfile(c *gin.Context) {
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
			log.Printf("Internal Server Error during UpdateProfile: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		}
		return
	}

	c.JSON(http.StatusOK, toUserResponse(updatedUser))
}


// ===========================================
// BLOG CONTROLLER
// ===========================================
type CreateBlogRequest struct {
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
}

type UpdateBlogRequest map[string]interface{}

type BlogResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	AuthorID  string    `json:"author_id"`
	Tags      []string  `json:"tags"`
	Views     int64     `json:"views"`
	Likes     int64     `json:"likes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Pagination struct {
	Total int64 `json:"total"`
	Page  int64 `json:"page"`
	Limit int64 `json:"limit"`
}

type PaginatedBlogResponse struct {
	Data       []BlogResponse `json:"data"`
	Pagination Pagination     `json:"pagination"`
}

type BlogController struct {
	blogUsecase domain.IBlogUsecase
}

func NewBlogController(usecase domain.IBlogUsecase) *BlogController {
	return &BlogController{
		blogUsecase: usecase,
	}
}

func (bc *BlogController) Create(c *gin.Context) {
	var req CreateBlogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body: " + err.Error()})
		return
	}

	userID := c.GetString("userID")

	blog, err := bc.blogUsecase.Create(c.Request.Context(), req.Title, req.Content, userID, req.Tags)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toBlogResponse(blog))
}

func (bc *BlogController) GetByID(c *gin.Context) {
	blogID := c.Param("id")

	blog, err := bc.blogUsecase.GetByID(c.Request.Context(), blogID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toBlogResponse(blog))
}

func (bc *BlogController) Fetch(c *gin.Context) {
	page, err := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid 'page' parameter"})
		return
	}

	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid 'limit' parameter"})
		return
	}

	blogs, total, err := bc.blogUsecase.Fetch(c.Request.Context(), page, limit)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPaginatedBlogResponse(blogs, total, page, limit))
}

func (bc *BlogController) Update(c *gin.Context) {
	blogID := c.Param("id")
	userID := c.GetString("userID")
	userRole := domain.Role(c.GetString("role"))

	var updates UpdateBlogRequest
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body: " + err.Error()})
		return
	}

	updatedBlog, err := bc.blogUsecase.Update(c.Request.Context(), blogID, userID, userRole, updates)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toBlogResponse(updatedBlog))
}

func (bc *BlogController) Delete(c *gin.Context) {
	blogID := c.Param("id")
	userID := c.GetString("userID")
	userRole := domain.Role(c.GetString("role"))

	err := bc.blogUsecase.Delete(c.Request.Context(), blogID, userID, userRole)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ===========================================
// HELPERS
// ===========================================

func HandleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, usecases.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "The requested resource was not found"})
	case errors.Is(err, domain.ErrPermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"message": "You do not have permission to perform this action"})
	case errors.Is(err, usecases.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
	default:
		log.Printf("Internal Server Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "An unexpected error occurred"})
	}
}

func toBlogResponse(b *domain.Blog) BlogResponse {
	return BlogResponse{
		ID:        b.ID,
		Title:     b.Title,
		Content:   b.Content,
		AuthorID:  b.AuthorID,
		Tags:      b.Tags,
		Views:     b.Views,
		Likes:     b.Likes,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
}

func toPaginatedBlogResponse(blogs []*domain.Blog, total, page, limit int64) PaginatedBlogResponse {
	blogResponses := make([]BlogResponse, len(blogs))
	for i, b := range blogs {
		blogResponses[i] = toBlogResponse(b)
	}

	return PaginatedBlogResponse{
		Data: blogResponses,
		Pagination: Pagination{
			Total: total,
			Page:  page,
			Limit: limit,
		},
	}
}
