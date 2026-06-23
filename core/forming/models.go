package main

import jwt "github.com/golang-jwt/jwt/v5"

type jwtClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}
