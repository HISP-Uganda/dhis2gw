package controllers

import (
	"dhis2gw/db"
	"dhis2gw/models"
	"dhis2gw/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

type UserController struct{}

// CreateUser - Admin only
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
			lastname, email, telephone, is_active, is_system_user, created, updated)
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

// UpdateUser updates user details using uid
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

// CreateUserToken generates and saves an API token for the currently authenticated user
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Set token expiration (e.g., 30 days from now)
	expirationTime := time.Now().Add(30 * 24 * time.Hour) // 30-day validity

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
		INSERT INTO user_apitoken (user_id, token, is_active, expires_at, created_at, updated_at)
		VALUES (:user_id, :token, :is_active, :expires_at, :created, :updated)`, userToken)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Token created successfully",
		"token":   token,
		"expires": expirationTime,
	})
}

// RefreshUserToken allows the currently authenticated user to refresh their own API token
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

	// Set new expiration (e.g., 30 days from now)
	newExpiration := time.Now().Add(30 * 24 * time.Hour)

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
		INSERT INTO user_apitoken (user_id, token, is_active, expires_at, created_at, updated_at)
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

// ListUsers lists existing users
func (uc *UserController) DeleteUserToken(c *gin.Context) {

}
