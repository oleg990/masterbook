package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	DB        *pgxpool.Pool
	JWTSecret string
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid JSON",
		})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "name is required",
		})
		return
	}

	if req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "email is required",
		})
		return
	}

	if len(req.Password) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "password must be at least 8 characters",
		})
		return
	}

	if len([]byte(req.Password)) > 72 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "password is too long",
		})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to hash password",
		})
		return
	}

	var response RegisterResponse

	err = h.DB.QueryRow(
		r.Context(),
		`
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, role
		`,
		req.Name,
		req.Email,
		string(passwordHash),
	).Scan(
		&response.ID,
		&response.Name,
		&response.Email,
		&response.Role,
	)

	if err != nil {
		if isUniqueViolation(err) {
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "email already exists",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to create user",
		})
		return
	}

	writeJSON(w, http.StatusCreated, response)
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "duplicate key")
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid JSON",
		})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "email and password are required",
		})
		return
	}

	var (
		userID       int64
		passwordHash string
		role         string
	)

	err := h.DB.QueryRow(
		r.Context(),
		`
		SELECT id, password_hash, role
		FROM users
		WHERE email = $1
		`,
		req.Email,
	).Scan(
		&userID,
		&passwordHash,
		&role,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "invalid email or password",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to find user",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(req.Password),
	); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "invalid email or password",
		})
		return
	}

	token, err := CreateToken(
		userID,
		req.Email,
		role,
		h.JWTSecret,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to create token",
		})
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)

	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "user is not authenticated",
		})
		return
	}

	var response RegisterResponse

	err := h.DB.QueryRow(
		r.Context(),
		`
		SELECT id, name, email, role
		FROM users
		WHERE id = $1
		`,
		userID,
	).Scan(
		&response.ID,
		&response.Name,
		&response.Email,
		&response.Role,
	)

	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "user not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, response)
}
