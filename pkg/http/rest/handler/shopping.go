package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/GGP1/palo/internal/response"
	"github.com/GGP1/palo/pkg/model"
	"github.com/GGP1/palo/pkg/shopping"
	"github.com/gorilla/mux"
)

// CartAdd appends a product to the cart
func CartAdd(cart *shopping.Cart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var product model.Product

		if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
			response.Error(w, r, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		amount := mux.Vars(r)["amount"]

		q, err := strconv.Atoi(amount)
		if err != nil {
			response.Error(w, r, http.StatusBadRequest, errors.New("Amount inserted is not valid, integers accepted only"))
		}

		quantity := float32(q)

		if quantity == 0 {
			response.Error(w, r, http.StatusBadRequest, errors.New("Please insert a valid amount"))
			return
		}

		cart.Add(&product, quantity)

		response.JSON(w, r, http.StatusOK, cart)
	}
}

// CartCheckout returns the final purchase
func CartCheckout(cart *shopping.Cart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checkout := cart.Checkout()

		response.JSON(w, r, http.StatusOK, checkout)
	}
}

// CartFilterByBrand returns the products filtered by brand
func CartFilterByBrand(cart *shopping.Cart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		brand := mux.Vars(r)["brand"]

		products, err := cart.FilterByBrand(brand)
		if err != nil {
			response.Error(w, r, http.StatusNotFound, err)
			return
		}

		response.JSON(w, r, http.StatusOK, products)
	}
}

// CartFilterByCategory returns the products filtered by category
func CartFilterByCategory(cart *shopping.Cart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		category := mux.Vars(r)["category"]

		products, err := cart.FilterByCategory(category)
		if err != nil {
			response.Error(w, r, http.StatusNotFound, err)
			return
		}

		response.JSON(w, r, http.StatusOK, products)
	}
}

// CartFilterByTotal returns the products filtered by total
func CartFilterByTotal(cart *shopping.Cart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		min := mux.Vars(r)["min"]
		max := mux.Vars(r)["max"]

		minTotal, _ := strconv.Atoi(min)
		maxTotal, _ := strconv.Atoi(max)

		products, err := cart.FilterByTotal(float32(minTotal), float32(maxTotal))
		if err != nil {
			response.Error(w, r, http.StatusNotFound, err)
			return
		}

		response.JSON(w, r, http.StatusOK, products)
	}
}

// CartFilterByType returns the products filtered by type
func CartFilterByType(cart *shopping.Cart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productType := mux.Vars(r)["type"]

		products, err := cart.FilterByType(productType)
		if err != nil {
			response.Error(w, r, http.StatusNotFound, err)
			return
		}

		response.JSON(w, r, http.StatusOK, products)
	}
}

// CartFilterByWeight returns the products filtered by total
func CartFilterByWeight(cart *shopping.Cart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		min := mux.Vars(r)["min"]
		max := mux.Vars(r)["max"]

		minWeight, _ := strconv.Atoi(min)
		maxWeight, _ := strconv.Atoi(max)

		products, err := cart.FilterByWeight(float32(minWeight), float32(maxWeight))
		if err != nil {
			response.Error(w, r, http.StatusNotFound, err)
			return
		}

		response.JSON(w, r, http.StatusOK, products)
	}
}

// CartGet returns the cart in a JSON format
func CartGet(cart *shopping.Cart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, r, http.StatusOK, cart)
	}
}

// CartItems prints cart items
func CartItems(cart *shopping.Cart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items := cart.Items()

		response.JSON(w, r, http.StatusOK, items)
	}
}

// CartRemove takes out a product from the shopping cart
func CartRemove(cart *shopping.Cart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		key, err := strconv.Atoi(id)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		err = cart.Remove(uint(key))
		if err != nil {
			response.Error(w, r, http.StatusBadRequest, err)
		}

		response.JSON(w, r, http.StatusOK, cart)
	}
}

// CartReset resets the cart to its default state
func CartReset(cart *shopping.Cart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cart.Reset()

		response.JSON(w, r, http.StatusOK, cart)
	}
}

// CartSize returns the size of the shopping cart
func CartSize(cart *shopping.Cart) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		size := cart.Size()

		response.JSON(w, r, http.StatusOK, size)
	}
}
