package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"net/http"
	"strconv"

	"github.com/GGP1/palo/internal/response"
	"github.com/GGP1/palo/pkg/auth"
	"github.com/GGP1/palo/pkg/auth/email"
	"github.com/GGP1/palo/pkg/creating"
	"github.com/GGP1/palo/pkg/deleting"
	"github.com/GGP1/palo/pkg/listing"
	"github.com/GGP1/palo/pkg/model"
	"github.com/GGP1/palo/pkg/searching"
	"github.com/GGP1/palo/pkg/shopping"
	"github.com/GGP1/palo/pkg/updating"
	"github.com/go-chi/chi"

	"github.com/jinzhu/gorm"
)

// Users handles users routes
type Users struct {
	DB *gorm.DB
}

// Create creates a new user and saves it.
func (us *Users) Create(c creating.Service, pendingList email.Emailer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user model.User

		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			response.Error(w, r, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		token, err := auth.GenerateJWT(user)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, fmt.Errorf("could not generate the jwt token: %w", err))
			return
		}

		err = pendingList.Add(user.Email, token)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		err = email.SendValidation(user, token)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, fmt.Errorf("failed sending validation email: %w", err))
			return
		}

		err = c.CreateUser(us.DB, &user)
		if err != nil {
			response.Error(w, r, http.StatusBadRequest, err)
			return
		}

		response.HTMLText(w, r, http.StatusOK, "Your account was successfully created.\nPlease validate your email to start using Palo.")
	}
}

// Delete removes a user.
func (us *Users) Delete(d deleting.Service, s auth.Session, pendingList, validatedList email.Emailer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		uID, _ := r.Cookie("UID")

		var user model.User

		// Check if it's the same user
		userID, err := auth.ParseFixedJWT(uID.Value)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		if userID != id {
			response.Error(w, r, http.StatusUnauthorized, fmt.Errorf("not allowed to delete others user"))
			return
		}

		// Remove user from email lists
		err = pendingList.Remove(user.Email)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		err = validatedList.Remove(user.Email)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		// Delete user cart
		err = shopping.DeleteCart(us.DB, user.CartID)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		err = d.DeleteUser(us.DB, id)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		// If the user is logged in, log him out
		s.Logout(w, r, uID)

		response.HTMLText(w, r, http.StatusOK, "User deleted successfully.")
	}
}

// Get lists all the users
func (us *Users) Get(l listing.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user []model.User

		err := l.GetUsers(us.DB, &user)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, r, http.StatusOK, user)
	}
}

// GetByID lists the user with the id requested.
func (us *Users) GetByID(l listing.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var user model.User

		err := l.GetUserByID(us.DB, &user, id)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, r, http.StatusOK, user)
	}
}

// QRCode shows the user id in a qrcode format.
func (us *Users) QRCode(l listing.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var user model.User
		buffer := new(bytes.Buffer)

		err := l.GetUserByID(us.DB, &user, id)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		img, err := user.QRCode()
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		err = png.Encode(buffer, img)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, fmt.Errorf("couldn't encode the PNG image"))
			return
		}

		w.Header().Set("Content-Type", "image/png")

		w.Header().Set("Content-Length", strconv.Itoa(len(buffer.Bytes())))

		if _, err := w.Write(buffer.Bytes()); err != nil {
			fmt.Println("unable to write image.")
		}
	}
}

// Search looks for the products with the given value.
func (us *Users) Search(s searching.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		search := chi.URLParam(r, "search")

		var users []model.User

		err := s.SearchUsers(us.DB, &users, search)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, r, http.StatusOK, users)
	}
}

// Update updates the user with the given id.
func (us *Users) Update(u updating.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		uID, _ := r.Cookie("UID")

		var user model.User

		// Check if it's the same user
		userID, err := auth.ParseFixedJWT(uID.Value)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		if userID != id {
			response.Error(w, r, http.StatusUnauthorized, fmt.Errorf("not allowed to update others user"))
			return
		}

		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			response.Error(w, r, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		err = u.UpdateUser(us.DB, &user, id)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		response.HTMLText(w, r, http.StatusOK, "User updated successfully.")
	}
}
