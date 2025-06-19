package middleware

import (
	"database/sql"
	"dhis2gw/db"
	"dhis2gw/models"
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"strings"
)

func BasicAuth(dbConn *sqlx.DB, asynqClient *asynq.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("dbConn", dbConn)
		c.Set("asynqClient", asynqClient)

		auth := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2)

		if len(auth) != 2 || (auth[0] != "Basic" && auth[0] != "Token:") {
			RespondWithError(401, "Unauthorized", c)
			return
		}
		tokenAuthenticated, userUID := AuthenticateUserToken(auth[1])
		if auth[0] == "Token:" {
			if !tokenAuthenticated {
				RespondWithError(401, "Unauthorized", c)
				return
			}
			c.Set("currentUser", userUID)
			c.Next()
			return
		}

		payload, _ := base64.StdEncoding.DecodeString(auth[1])
		pair := strings.SplitN(string(payload), ":", 2)

		basicAuthenticated, userUID := AuthenticateUser(pair[0], pair[1])

		if len(pair) != 2 || !basicAuthenticated {
			RespondWithError(401, "Unauthorized", c)
			return
		}
		c.Set("currentUser", userUID)

		c.Next()
	}
}

func AuthenticateUser(username, password string) (bool, int64) {
	var userID int64
	err := db.GetDB().Get(&userID,
		`SELECT id FROM users
         WHERE username = $1 AND password = crypt($2, password)`,
		username, password)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("No user found with username %s and given password", username)
			return false, 0
		}
		log.Printf("Error querying user: %v", err)
		return false, 0
	}
	log.Printf("User authenticated: %d", userID)
	return true, userID
}

func AuthenticateUserToken(token string) (bool, int64) {
	userToken := models.UserToken{}
	err := db.GetDB().QueryRowx(
		`SELECT
            id, user_id, token, is_active
        FROM user_apitoken
        WHERE
            token = $1 AND is_active = TRUE LIMIT 1`,
		token).StructScan(&userToken)
	if err != nil {
		return false, 0
	}
	// fmt.Printf("User:[%v]", userObj)
	return true, userToken.UserID
}

func RespondWithError(code int, message string, c *gin.Context) {
	resp := map[string]string{"error": message}

	c.JSON(code, resp)
	c.Abort()
}
