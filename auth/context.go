package auth

import "github.com/gin-gonic/gin"

type contextKey string

const UserContextKey contextKey = "auth.user"

func GetUser(c *gin.Context) *User {
	u, ok := c.Get(string(UserContextKey))
	if !ok {
		return nil
	}
	return u.(*User)
}
