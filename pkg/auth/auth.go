// Package auth provides authentication and authorization support.
package auth

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/GGP1/palo/internal/token"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// Session provides auth operations.
type Session interface {
	AlreadyLoggedIn(w http.ResponseWriter, r *http.Request) bool
	Clean()
	Login(ctx context.Context, w http.ResponseWriter, email, password string) error
	Logout(w http.ResponseWriter, r *http.Request, c *http.Cookie)
}

type userData struct {
	email    string
	lastSeen time.Time
}

type session struct {
	sync.RWMutex

	DB *sqlx.DB
	// user session
	store map[string]userData
	// last time the user logged in
	cleaned time.Time
	// session length
	length int
	// number of tries to log in
	tries map[string][]struct{}
	// time to wait after failing x times (increments every fail)
	delay map[string]time.Time
}

// NewSession creates a new session with the necessary dependencies.
func NewSession(db *sqlx.DB) Session {
	return &session{
		DB:      db,
		store:   make(map[string]userData),
		cleaned: time.Now(),
		length:  0,
		tries:   make(map[string][]struct{}),
		delay:   make(map[string]time.Time),
	}
}

// AlreadyLoggedIn checks if the user has an active session or not.
func (s *session) AlreadyLoggedIn(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie("SID")
	if err != nil {
		return false
	}

	s.Lock()
	defer s.Unlock()

	user, ok := s.store[cookie.Value]
	if ok {
		user.lastSeen = time.Now()
		s.store[cookie.Value] = user
	}

	cookie.MaxAge = s.length

	return ok
}

// Clean deletes all the sessions that have expired.
func (s *session) Clean() {
	for key, value := range s.store {
		if time.Now().Sub(value.lastSeen) > (time.Hour * 120) {
			delete(s.store, key)
		}
	}
	s.cleaned = time.Now()
}

// Login authenticates users.
func (s *session) Login(ctx context.Context, w http.ResponseWriter, email, password string) error {
	if !s.delay[email].IsZero() && s.delay[email].Sub(time.Now()) > 0 {
		return fmt.Errorf("please wait %v before trying again", s.delay[email].Sub(time.Now()))
	}

	var user User
	q := `SELECT id, cart_id, username, email, password, email_verified_at FROM users WHERE email=$1`

	// Check if the email exists and if it is verified
	if err := s.DB.GetContext(ctx, &user, q, email); err != nil {
		s.loginDelay(email)
		return errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		s.loginDelay(email)
		return errors.New("invalid email or password")
	}

	for _, admin := range AdminList {
		if admin == user.Email {
			admID := token.GenerateRunes(8)
			setCookie(w, "AID", admID, "/", s.length)
		}
	}

	// -SID- used to add the user to the session map
	sID := token.GenerateRunes(27)
	setCookie(w, "SID", sID, "/", s.length)

	s.Lock()
	s.store[sID] = userData{user.Email, time.Now()}
	delete(s.tries, email)
	delete(s.delay, email)
	s.Unlock()

	// -UID- used to deny users from making requests to other accounts
	userID, err := token.GenerateFixedJWT(user.ID)
	if err != nil {
		return errors.Wrap(err, "failed generating a jwt token")
	}
	setCookie(w, "UID", userID, "/", s.length)

	// -CID- used to identify which cart belongs to each user
	setCookie(w, "CID", user.CartID, "/", s.length)

	return nil
}

// Logout removes the user session and its cookies.
func (s *session) Logout(w http.ResponseWriter, r *http.Request, c *http.Cookie) {
	admin, _ := r.Cookie("AID")
	if admin != nil {
		deleteCookie(w, "AID")
	}

	deleteCookie(w, "SID")
	deleteCookie(w, "UID")
	deleteCookie(w, "CID")

	s.Lock()
	delete(s.store, c.Value)
	s.Unlock()

	if time.Now().Sub(s.cleaned) > (time.Second * 30) {
		go s.Clean()
	}
}

// loginDelay increments the time that the user will have to wait after failing.
func (s *session) loginDelay(email string) {
	s.Lock()
	defer s.Unlock()

	s.tries[email] = append(s.tries[email], struct{}{})

	d := (len(s.tries[email]) * 2)
	s.delay[email] = time.Now().Add(time.Second * time.Duration(d))
}

func deleteCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "0",
		Expires:  time.Unix(1414414788, 1414414788000),
		Path:     "/",
		Domain:   "127.0.0.1",
		Secure:   false,
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func setCookie(w http.ResponseWriter, name, value, path string, length int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   "127.0.0.1",
		Secure:   false,
		HttpOnly: true,
		SameSite: 3,
		MaxAge:   length,
	})
}
