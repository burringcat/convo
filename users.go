package main

import (
	"context"
	"errors"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"net/http"
	"strings"
	"time"
)

type ConvoUser struct {
	ID int64 `json:"id"`
	Username string `json:"username"`
}
func (cu *ConvoUser) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
type ConvoUserRequest struct {
	*ConvoUser
	Password string `json:"password, omitempty"`
}
func (cur *ConvoUserRequest) Bind(r *http.Request) error{
	if cur.Username == "" || cur.Password == "" {
		return errors.New("invalid user information")
	}
	return nil
}

type ConvoUserResponse struct {
	*ConvoUser
	Token string `json:"token, omitempty"`
}
func (cup *ConvoUserResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
type ConvoUserClaims struct {
	UserID int64 `json:"user_id"`
	Signed int64 `json:"signed, omitempty"`
	jwt.StandardClaims
}
func UserGenToken(w http.ResponseWriter, r *http.Request) {
	cur := &ConvoUserRequest{}
	if err := render.Bind(r, cur); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	result, err := sq.Select("id", "username", "passhash").
		From("users").
		Where(sq.Eq{"username": cur.Username}).
		RunWith(DB.db).Query()
	defer result.Close()
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	var (
		userID int64
		username string
		passhash string
	)
	for result.Next() {
		result.Scan(&userID, &username, &passhash)
	}
	passwordCorrect := checkPassword(cur.Password, passhash)
	user := &ConvoUser{
		ID: userID,
		Username: username,
	}
	if passwordCorrect {
		secretKey := []byte(Config.SecretKey)
		timeNow := time.Now().UnixNano()
		claims := ConvoUserClaims{
			userID,
			timeNow,
			jwt.StandardClaims{
				ExpiresAt: 0,
				Issuer: "convo",
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, err := token.SignedString(secretKey)
		if err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}
		resp := &ConvoUserResponse{
			ConvoUser: user,
			Token: tokenStr,
		}
		render.Render(w, r, resp)
		return
	} else {
		var err error
		if username == "" {
			err = errors.New("no such user")
		} else {
			err = errors.New("wrong password")
		}
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
}
func UserCreate(w http.ResponseWriter, r *http.Request) {
	cur := &ConvoUserRequest{}
	if err := render.Bind(r, cur); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	var usernameFromDb string
	result, err := sq.Select("username").
		From("users").
		Where(sq.Eq{"username": cur.Username}).RunWith(DB.db).Query()
	defer result.Close()
	for result.Next() {
		err = result.Scan(&usernameFromDb)
	}
	usernameExists := err == nil && usernameFromDb == cur.Username
	if usernameExists {
		err := errors.New("username exists")
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	passhash, err := GenPasswordHash(cur.Password)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	insertResult, err := sq.Insert("users").
		Columns("username", "passhash").
		Values(cur.Username, passhash).
		RunWith(DB.db).Exec()
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	userID, _ := insertResult.LastInsertId()
	newUser := &ConvoUser{ID:userID, Username: cur.Username}
	render.Render(w, r, newUser)
}
func ConvoUserTokenCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		splitToken := strings.Split(authHeader, "Bearer ")
		if len(splitToken) < 2 {
			err := errors.New("bad authorization header")
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}
		tokenStr := splitToken[1]
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
			return []byte(Config.SecretKey), nil
		})
		if err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}
		var (
			username string
			userID int64
		)
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userID = int64(claims["user_id"].(float64))
			result, err := sq.Select("id", "username").From("users").
				Where(sq.Eq{"id": userID}).RunWith(DB.db).Query()
			if err != nil {
				render.Render(w, r, ErrInvalidRequest(err))
				return
			}
			for result.Next() {
				result.Scan(&userID, &username)
			}
			if username == "" {
				err := errors.New("no such user(bad user id)")
				render.Render(w, r, ErrInvalidRequest(err))
				return
			}
		} else {
			err := errors.New("invalid token")
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}
		user := &ConvoUser{
			userID,
			username,
		}
		ctx := context.WithValue(r.Context(), "convouser", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func ConvoUserTokenRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(ConvoUserTokenCtx)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value("convouser").(*ConvoUser)
		render.Render(w, r, user)
		return
	})
	return r
}