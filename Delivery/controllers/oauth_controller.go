package controllers

import (
	"net/http"

	domain "A2SV_Starter_Project_Blog/Domain"
	"github.com/gin-gonic/gin"
)

type GoogleCallbackRequest struct {
	Code string `json:"code" binding:"required"`
}

type AuthTokensResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type OAuthController struct {
	oauthUsecase domain.IOAuthUsecase
}

func NewOAuthController(usecase domain.IOAuthUsecase) *OAuthController {
	return &OAuthController{
		oauthUsecase: usecase,
	}
}

func (oc *OAuthController) HandleGoogleCallback(c *gin.Context) {
	// 1. Bind and validate the incoming JSON.
	var req GoogleCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request: 'code' is a required field"})
		return
	}

	// 2. Pass the authorization code to the usecase to handle the entire flow.
	accessToken, refreshToken, err := oc.oauthUsecase.HandleGoogleCallback(c.Request.Context(), req.Code)
	if err != nil {
		// The usecase will return specific errors (e.g., ErrEmailExists) which
		// our centralized HandleError function can map to appropriate HTTP statuses.
		HandleError(c, err)
		return
	}

	// 3. On success, return the application's own tokens to the client.
	c.JSON(http.StatusOK, AuthTokensResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}
