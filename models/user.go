package models

import (
	"crypto/rand"
	"dhis2gw/db"
	"encoding/base64"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
)

// User is our user object
type User struct {
	ID          int64      `db:"id"`
	UID         string     `db:"uid" json:"uid"`
	Username    string     `db:"username"`
	Password    string     `db:"password"`
	FirstName   string     `json:"firstname" db:"firstname"`
	LastName    string     `json:"lastname" db:"lastname"`
	Email       string     `json:"email,omitempty" db:"email"`
	Phone       string     `json:"telephone,omitempty" db:"telephone"`
	IsActive    bool       `json:"is_active,omitempty" db:"is_active"`
	IsAdminUser bool       `json:"is_admin_user,omitempty" db:"is_admin_user"`
	Created     *time.Time `json:"created,omitempty" db:"created"`
	Updated     *time.Time `json:"updated,omitempty" db:"updated"`
}

type UserCreateResponse struct {
	Message string `json:"message" example:"User created successfully!"`
	UID     string `json:"uid" example:"aS1kT9rLQ9f"`
}
type UserInput struct {
	Username    string `json:"username" example:"admin"`
	Password    string `json:"password" example:"s3cretP@ss"`
	FirstName   string `json:"firstName" example:"John"`
	LastName    string `json:"lastName" example:"Doe"`
	Email       string `json:"email" example:"john@example.com"`
	Telephone   string `json:"telephone" example:"+256700000001"`
	IsActive    bool   `json:"isActive" example:"true"`
	IsAdminUser bool   `json:"isAdminUser" example:"false"`
}

type UpdateUserInput struct {
	Username  string `json:"username" example:"jdoe"`
	FirstName string `json:"firstName" example:"John"`
	LastName  string `json:"lastName" example:"Doe"`
	Email     string `json:"email" example:"john.doe@example.com"`
	Phone     string `json:"telephone" example:"+256700000000"`
}

func (u *User) DeactivateAPITokens(token string) {
	dbConn := db.GetDB()
	_, err := dbConn.NamedExec(
		`UPDATE user_apitoken SET is_active = FALSE WHERE user_id = :id`, u)
	if err != nil {
		log.WithError(err).Error("Failed to deactivate user API tokens")
	}
}

type UserToken struct {
	ID        int64     `db:"id" json:"id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	Token     string    `db:"token" json:"token"`
	IsActive  bool      `db:"is_active" json:"is_active"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	Created   time.Time `db:"created" json:"created"`
	Updated   time.Time `db:"updated" json:"updated"`
}

func (ut *UserToken) Save() {
	dbConn := db.GetDB()
	_, err := dbConn.NamedExec(`INSERT INTO user_apitoken (user_id, token)
			VALUES(:user_id, :token)`, ut)
	if err != nil {
		log.WithError(err).Error("Failed to save user API token")
	}
}

func (u *User) GetActiveToken() (string, error) {
	dbConn := db.GetDB()
	var ut UserToken
	err := dbConn.Get(&ut, "SELECT * FROM user_apitoken WHERE user_id = $1 AND is_active = TRUE LIMIT 1", u.ID)
	if err != nil {
		return "", err
	}
	return ut.Token, nil
}
func BasicAuth() gin.HandlerFunc {

	return func(c *gin.Context) {
		c.Set("dbConn", db.GetDB())
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
			// c.Writer.Header().Set("WWW-Authenticate", "Basic realm=Restricted")
			return
		}
		c.Set("currentUser", userUID)

		c.Next()
	}
}

func GetUserByUID(uid string) (*User, error) {
	userObj := User{}
	err := db.GetDB().QueryRowx(
		`SELECT
            id, uid, username, firstname, lastname , telephone, email
        FROM users
        WHERE
            uid = $1`,
		uid).StructScan(&userObj)
	if err != nil {
		return nil, err
	}
	return &userObj, nil
}

func GetUserById(id int64) (*User, error) {
	userObj := User{}
	err := db.GetDB().QueryRowx(
		`SELECT
            id, uid, username, firstname, lastname , telephone, email
        FROM users
        WHERE
            id = $1`,
		id).StructScan(&userObj)
	if err != nil {
		return nil, err
	}
	return &userObj, nil
}

func AuthenticateUser(username, password string) (bool, int64) {
	// log.Printf("Username:%s, password:%s", username, password)
	// userObj := User{}
	var user User
	err := db.GetDB().Get(&user,
		`SELECT
        	* 
        FROM users
        WHERE
            username = $1 `,
		username)
	if err != nil {
		// fmt.Printf("User:[%v]", err)
		return false, 0
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return false, 0
	}
	// fmt.Printf("User:[%v]", userObj)
	return true, user.ID
}

func AuthenticateUserToken(token string) (bool, int64) {
	userToken := UserToken{}
	err := db.GetDB().QueryRowx(
		`SELECT
            id, user_id, token, is_active
        FROM user_apitoken
        WHERE
            token = $1 AND is_active = TRUE LIMIT 1`,
		token).StructScan(&userToken)
	if err != nil {
		// fmt.Printf("User:[%v]", err)
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

func GenerateToken() (string, error) {
	// Define the length of the token in bytes
	const tokenLength = 20

	// Create a byte slice to hold the random bytes
	token := make([]byte, tokenLength)

	// Generate random bytes
	_, err := rand.Read(token)
	if err != nil {
		return "", err
	}

	// Convert the bytes to a hexadecimal string
	return hex.EncodeToString(token), nil
}
