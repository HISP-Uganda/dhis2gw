package controllers

import (
	"dhis2gw/db"
	"dhis2gw/models"
	"dhis2gw/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"math"
	"net/http"
	"strconv"
	"time"
)

type UserController struct{}

// CreateUser godoc
// @Summary Create a new user
// @Description Registers a new user with username, password, and profile info
// @Tags users
// @Accept json
// @Produce json
// @Param request body models.UserInput true "User registration input"
// @Success 201 {object} models.UserCreateResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /users [post]
// @Security BasicAuth
// @Security TokenAuth
func (uc *UserController) CreateUser(c *gin.Context) {
	var input struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Email       string `json:"email"`
		Telephone   string `json:"telephone"`
		IsActive    bool   `json:"isActive"`
		IsAdminUser bool   `json:"isAdminUser,omitempty"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Generate UID
	uid := utils.GenerateUID()
	created := time.Now()
	user := models.User{
		UID:         uid,
		Username:    input.Username,
		IsActive:    input.IsActive,
		IsAdminUser: input.IsAdminUser,
		Created:     &created,
		Updated:     &created,
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	user.Password = string(hash)

	// Save user
	_, err = db.GetDB().NamedExec(`INSERT INTO users (uid, username, password, firstname, 
			lastname, email, telephone, is_active, is_admin_user, created, updated)
		VALUES (:uid, :username, :password, :firstname, :lastname, :email, :telephone, 
			:is_active, :is_system_user, :created, :updated) RETURNING id`, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully!", "uid": uid})
}

// GetUserByUID ...
func (uc *UserController) GetUserByUID(c *gin.Context) {
	uid := c.Param("uid")

	user, err := models.GetUserByUID(uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser godoc
// @Summary Update user details
// @Description Updates an existing user's profile using their UID. Requires authentication.
// @Tags users
// @Accept json
// @Produce json
// @Param uid path string true "User UID"
// @Param user body models.UpdateUserInput true "Updated user information"
// @Success 200 {object} models.SuccessResponse "User updated successfully"
// @Failure 400 {object} models.ErrorResponse "Invalid request body"
// @Failure 500 {object} models.ErrorResponse "Server/internal error"
// @Security BasicAuth
// @Security TokenAuth
// @Router /users/{uid} [put]
func (uc *UserController) UpdateUser(c *gin.Context) {
	uid := c.Param("uid")

	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	updated := time.Now()
	user.Updated = &updated

	_, err := db.GetDB().NamedExec(`
		UPDATE users 
		SET username=:username, firstname=:firstname, lastname=:lastname, email=:email, telephone=:telephone, updated=:updated 
		WHERE uid=:uid`, map[string]interface{}{
		"uid":       uid,
		"username":  user.Username,
		"firstname": user.FirstName,
		"lastname":  user.LastName,
		"email":     user.Email,
		"telephone": user.Phone,
		"updated":   user.Updated,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// DeleteUser (Soft delete)
func (uc *UserController) DeleteUser(c *gin.Context) {
	uid := c.Param("uid")

	_, err := db.GetDB().Exec(`UPDATE users SET is_active = FALSE WHERE uid = $1`, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deactivated"})
}

// CreateUserToken godoc
// @Summary Generate or return a user token
// @Description If the user has an active token, it is returned; otherwise a new one is created. Requires authentication.
// @Tags users
// @Produce json
// @Security BasicAuth
// @Security TokenAuth
// @Success 200 {object} models.UserTokenResponse "Returned if an active token already exists"
// @Success 201 {object} models.UserTokenResponse "Returned if a new token was generated"
// @Failure 401 {object} models.ErrorResponse "Unauthorized or missing user context"
// @Failure 404 {object} models.ErrorResponse "User not found"
// @Failure 500 {object} models.ErrorResponse "Server/internal error"
// @Router /users/getToken [post] generates and saves an API token for the currently authenticated user
func (uc *UserController) CreateUserToken(c *gin.Context) {
	// Extract the authenticated user's UID from the request context
	authUserUID, exists := c.Get("currentUser")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Ensure UID is a valid string
	userID, ok := authUserUID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Invalid user identifier: %v", authUserUID)})
		return
	}

	// Fetch the full user details using UID
	user, err := models.GetUserById(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	dbConn := db.GetDB()
	// Invalidate all existing active tokens for the user
	_, err = dbConn.Exec(`UPDATE user_apitoken SET is_active = FALSE WHERE user_id = $1 AND is_active = TRUE`, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to invalidate existing tokens"})
		return
	}

	var existingToken models.UserToken

	// Check if an active, non-expired token already exists
	err = dbConn.Get(&existingToken, `
		SELECT * FROM user_apitoken 
		WHERE user_id = $1 AND is_active = TRUE AND expires_at > NOW() 
		LIMIT 1`, user.ID)

	if err == nil {
		// If an active token exists, return it instead of creating a new one
		c.JSON(http.StatusOK, gin.H{
			"message": "An active token already exists",
			"token":   existingToken.Token,
			"expires": existingToken.ExpiresAt,
		})
		return
	}

	// Generate a new token
	token, err := models.GenerateToken()
	if err != nil {
		log.Errorf("Failed to generate token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Set token expiration (e.g., 365 days from now)
	expirationTime := time.Now().Add(365 * 24 * time.Hour) // 365-day validity

	// Create a new UserToken object
	userToken := models.UserToken{
		UserID:    user.ID,
		Token:     token,
		IsActive:  true,
		ExpiresAt: expirationTime,
		Created:   time.Now(),
		Updated:   time.Now(),
	}

	// Save token in the database
	_, err = dbConn.NamedExec(`
		INSERT INTO user_apitoken (user_id, token, is_active, expires_at, created, updated)
		VALUES (:user_id, :token, :is_active, :expires_at, :created, :updated)`, userToken)

	if err != nil {
		log.Errorf("Failed to generate token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Token created successfully",
		"token":   token,
		"expires": expirationTime,
	})
}

// RefreshUserToken godoc
// @Summary Refresh user's API token
// @Description Deactivates the current active token (if any) and generates a new one. Requires authentication.
// @Tags users
// @Produce json
// @Security BasicAuth
// @Security TokenAuth
// @Success 200 {object} models.UserTokenResponse "New token generated"
// @Failure 401 {object} models.ErrorResponse "Unauthorized or missing user context"
// @Failure 404 {object} models.ErrorResponse "No active token found"
// @Failure 500 {object} models.ErrorResponse "Server/internal error"
// @Router /users/refreshToken [post]
func (uc *UserController) RefreshUserToken(c *gin.Context) {
	// Extract the authenticated user's UID from context
	authUserUID, exists := c.Get("currentUser")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Convert UID to string
	userID, ok := authUserUID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user identifier"})
		return
	}

	// Fetch the full user details using ID
	user, err := models.GetUserById(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	dbConn := db.GetDB()
	var existingToken models.UserToken

	// Check for an active token belonging to the authenticated user
	err = dbConn.Get(&existingToken, `
		SELECT * FROM user_apitoken 
		WHERE user_id = $1 AND is_active = TRUE AND expires_at > NOW() 
		LIMIT 1`, user.ID)

	if err != nil {
		log.Infof("No token found for user: %s", err.Error())
		c.JSON(http.StatusNotFound, gin.H{"error": "No active token found to refresh"})
		return
	}

	// Deactivate the old token
	_, err = dbConn.Exec(`UPDATE user_apitoken SET is_active = FALSE WHERE id = $1`, existingToken.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate old token"})
		return
	}

	// Generate a new token
	newToken, err := models.GenerateToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate new token"})
		return
	}

	// Set new expiration (e.g., 365 days from now)
	newExpiration := time.Now().Add(365 * 24 * time.Hour)

	// Create a new UserToken object
	newUserToken := models.UserToken{
		UserID:    user.ID,
		Token:     newToken,
		IsActive:  true,
		ExpiresAt: newExpiration,
		Created:   time.Now(),
		Updated:   time.Now(),
	}

	// Save the new token
	_, err = dbConn.NamedExec(`
		INSERT INTO user_apitoken (user_id, token, is_active, expires_at, created, updated)
		VALUES (:user_id, :token, :is_active, :expires_at, :created_at, :updated_at)`, newUserToken)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save new token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"token":   newToken,
		"expires": newExpiration,
	})
}

type PaginatedUserResponse models.PaginatedResponse[models.User]

// GetUsersHandler godoc
// @Summary      Get users
// @Description  Returns a paginated list of users, with optional filters.
// @Tags         users
// @Produce      json
// @Param        uid        query     string  false  "Filter by UID"
// @Param        username   query     string  false  "Filter by username"
// @Param        email      query     string  false  "Filter by email"
// @Param        is_active  query     bool    false  "Filter by active status"
// @Param        is_admin   query     bool    false  "Filter by admin user status"
// @Param        page       query     int     false  "Page number (default 1)"
// @Param        page_size  query     int     false  "Items per page (default 20)"
// @Success      200        {object}  PaginatedUserResponse
// @Failure      500        {object}  models.ErrorResponse "Server-side error"
// @Router       /users [get]
func (uc *UserController) GetUsersHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var filter models.UserFilter

		if uid := c.Query("uid"); uid != "" {
			filter.UID = &uid
		}
		if username := c.Query("username"); username != "" {
			filter.Username = &username
		}
		if email := c.Query("email"); email != "" {
			filter.Email = &email
		}
		if isActive := c.Query("is_active"); isActive != "" {
			if b, err := strconv.ParseBool(isActive); err == nil {
				filter.IsActive = &b
			}
		}
		if isAdmin := c.Query("is_admin"); isAdmin != "" {
			if b, err := strconv.ParseBool(isAdmin); err == nil {
				filter.IsAdmin = &b
			}
		}
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
		filter.Page = page
		filter.PageSize = pageSize

		users, total, err := models.GetUsers(db, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		totalPages := int(math.Ceil(float64(total) / float64(filter.PageSize)))

		response := PaginatedUserResponse{
			Items:      users,
			Total:      int64(total),
			Page:       filter.Page,
			TotalPages: totalPages,
			PageSize:   filter.PageSize,
		}
		c.JSON(http.StatusOK, response)
	}
}
