package controllers

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

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

type SetRoleRequest struct {
	NewRole domain.Role `json:"newRole" binding:"required"`
}

type UserResponse struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Bio            string    `json:"bio,omitempty"`
	ProfilePicture string    `json:"profile_picture,omitempty"`
	Role           string    `json:"role"`
	IsActive       bool      `json:"is_active"`
	Provider       string    `json:"provider"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// PaginatedUserResponse defines the structure for a paginated list of users.
type PaginatedUserResponse struct {
	Data       []UserResponse `json:"data"`
	Pagination Pagination     `json:"pagination"`
}

func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:             u.ID,
		Username:       u.Username,
		Email:          u.Email,
		Bio:            u.Bio,
		ProfilePicture: u.ProfilePicture,
		Role:           string(u.Role),
		IsActive:       u.IsActive,
		Provider:       string(u.Provider),
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

func (ctrl *UserController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	password := req.Password
	user := &domain.User{
		Username: req.Username,
		Email:    req.Email,
		Password: &password,
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
	// 1. Get the logged-in user's ID from the context (set by middleware).
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}

	// 2. Parse the multipart form data from the request.
	// 10 << 20 specifies a max memory of 10 MB for the form parts.
	err := c.Request.ParseMultipartForm(10 << 20)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data"})
		return
	}

	// 3. Get the 'bio' value from the form. This replaces ShouldBindJSON.
	bio := c.Request.FormValue("bio")

	// 4. Get the 'profilePicture' file from the form.
	file, header, err := c.Request.FormFile("profilePicture")
	// It's not an error if the file is missing; this is an optional field.
	if err != nil && err != http.ErrMissingFile {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file upload"})
		return
	}

	// Ensure the file is closed if it was opened.
	if file != nil {
		defer file.Close()
	}

	// 5. Call the usecase with the parsed data.
	updatedUser, err := ctrl.userUsecase.UpdateProfile(c.Request.Context(), userID.(string), bio, file, header)
	if err != nil {
		// Use the centralized error handler for cleaner code
		HandleError(c, err)
		return
	}

	// 6. On success, return the updated user object.
	c.JSON(http.StatusOK, toUserResponse(updatedUser))
}

// SearchAndFilter handles requests for searching and filtering users.
// This is intended for admin use.
func (ctrl *UserController) SearchAndFilter(c *gin.Context) {
	var options domain.UserSearchFilterOptions

	// 1. Parse all optional query parameters from the request.
	// This code is very similar to your blog search controller, demonstrating pattern reuse.

	// Pagination
	page, err := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'page' parameter"})
		return
	}
	options.Page = page

	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil || limit < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'limit' parameter"})
		return
	}
	options.Limit = limit

	// Search and Filter criteria (using pointers for optional fields)
	if username := c.Query("username"); username != "" {
		options.Username = &username
	}
	if email := c.Query("email"); email != "" {
		options.Email = &email
	}
	if roleStr := c.Query("role"); roleStr != "" {
		role := domain.Role(roleStr)
		if !role.IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'role' parameter. Must be 'user' or 'admin'."})
			return
		}
		options.Role = &role
	}
	if isActiveStr := c.Query("isActive"); isActiveStr != "" {
		isActive, err := strconv.ParseBool(isActiveStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'isActive' parameter. Must be 'true' or 'false'."})
			return
		}
		options.IsActive = &isActive
	}
	if providerStr := c.Query("provider"); providerStr != "" {
		provider := domain.AuthProvider(providerStr)
		options.Provider = &provider
	}

	// Global Logic
	if strings.ToUpper(c.Query("logic")) == string(domain.GlobalLogicOR) {
		options.GlobalLogic = domain.GlobalLogicOR
	} else {
		options.GlobalLogic = domain.GlobalLogicAND // Default to AND
	}

	// Date range filtering (can be copied from your blog controller)
	if startDateStr := c.Query("startDate"); startDateStr != "" {
		if t, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			options.StartDate = &t
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'startDate' format. Use RFC3339 (e.g., 2023-10-27T10:00:00Z)"})
			return
		}
	}
	if endDateStr := c.Query("endDate"); endDateStr != "" {
		if t, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			options.EndDate = &t
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'endDate' format. Use RFC3339 (e.g., 2023-10-27T10:00:00Z)"})
			return
		}
	}

	// Sorting
	options.SortBy = c.Query("sortBy") // e.g., "username", "email", "createdAt"
	if strings.ToUpper(c.Query("sortOrder")) == string(domain.SortOrderASC) {
		options.SortOrder = domain.SortOrderASC
	} else {
		options.SortOrder = domain.SortOrderDESC // Default to DESC
	}

	// 2. Call the usecase with the populated options struct.
	users, total, err := ctrl.userUsecase.SearchAndFilter(c.Request.Context(), options)
	if err != nil {
		// Using a generic error handler is good practice
		log.Printf("Error searching users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occurred while searching for users."})
		return
	}

	// 3. Format and return the paginated response.
	c.JSON(http.StatusOK, toPaginatedUserResponse(users, total, options.Page, options.Limit))
}

// toPaginatedUserResponse is a helper to format the paginated response.
func toPaginatedUserResponse(users []*domain.User, total, page, limit int64) PaginatedUserResponse {
	userResponses := make([]UserResponse, len(users))
	for i, u := range users {
		userResponses[i] = toUserResponse(u)
	}

	return PaginatedUserResponse{
		Data: userResponses,
		Pagination: Pagination{
			Total: total,
			Page:  page,
			Limit: limit,
		},
	}
}

// SetUserRole handles requests to promote or demote a user.
func (ctrl *UserController) SetUserRole(c *gin.Context) {
	// 1. Get the target user's ID from the URL path parameter.
	targetUserID := c.Param("userID")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target user ID is required in the URL path."})
		return
	}

	// 2. Get the logged-in admin's details from the context (set by middleware).
	actorUserID, exists := c.Get("userID")
	if !exists {
		// This should theoretically be caught by the middleware, but it's good practice to check.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication details not found."})
		return
	}
	actorRole, exists := c.Get("userRole")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication role not found."})
		return
	}

	// 3. Bind and validate the JSON request body.
	var req SetRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// 4. Call the usecase with all the necessary information.
	updatedUser, err := ctrl.userUsecase.SetUserRole(
		c.Request.Context(),
		actorUserID.(string),
		actorRole.(domain.Role),
		targetUserID,
		req.NewRole,
	)

	// 5. Handle any errors returned from the usecase.
	if err != nil {
		HandleError(c, err)
		return
	}

	// 6. On success, return the updated user object.
	c.JSON(http.StatusOK, toUserResponse(updatedUser))
}
