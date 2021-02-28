package auth

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/google/uuid"
)

// Auth is an authentication service structure
type Auth struct {
	usersFile string
	tokens    map[string]TokenData
	tokenTTL  int
}

// TokenData contains a token data
type TokenData struct {
	Email     string
	TokenDate time.Time
}

// User contains user authentication data
type User struct {
	Login, Password string
}

// New returns new object of Auth type
func New(filePath string, ttl int) *Auth {
	var a Auth
	a.usersFile = filePath
	a.tokens = make(map[string]TokenData)
	a.tokenTTL = ttl
	return &a
}

// SignIn check the authentication data and returns token on success
func (a *Auth) SignIn(login, password string) (string, error) {
	users, err := a.loadUsers()
	if err != nil {
		return "", err
	}
	for _, u := range users {
		if login == u.Login && a.sha256(password) == u.Password {
			return a.registerToken(u), nil
		}
	}
	return "", fmt.Errorf("Wrong authentication data")
}

// ValidToken checks whether the token is valid and returns true on success
func (a *Auth) ValidToken(token string) bool {
	tok, ok := a.tokens[token]
	if !ok || time.Now().Sub(tok.TokenDate).Minutes() > float64(a.tokenTTL) {
		delete(a.tokens, token)
		return false
	}
	tok.TokenDate = time.Now()
	a.tokens[token] = tok
	return true
}

// AddUser add or update user
func (a *Auth) AddUser(login, password string) error {
	users, err := a.loadUsers()
	if err != nil {
		return err
	}
	newPassword := a.sha256(password)
	var foundID int = -1
	for i, u := range users {
		if login == u.Login {
			foundID = i
			break
		}
	}
	if foundID > -1 {
		users[foundID].Password = newPassword
	} else {
		newUser := User{
			Login:    login,
			Password: newPassword,
		}
		users = append(users, newUser)
	}
	if err := a.saveUsers(users); err != nil {
		return err
	}
	return nil
}

// loadUsers loads users from file
func (a *Auth) loadUsers() ([]User, error) {
	buf, err := ioutil.ReadFile(a.usersFile)
	if err != nil {
		return nil, err
	}
	var users []User
	if err := json.Unmarshal(buf, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// saveUsers saves users into file
func (a *Auth) saveUsers(users []User) error {
	data, err := json.MarshalIndent(users, "", "    ")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(a.usersFile, data, 0644); err != nil {
		return err
	}
	return nil
}

// registerToken generates authentication token
func (a *Auth) registerToken(user User) string {
	id := uuid.New().String()
	a.tokens[id] = TokenData{
		Email:     user.Login,
		TokenDate: time.Now(),
	}
	return id
}

// sha256 returns SHA256 encoded string
func (a *Auth) sha256(str string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(str)))
}
