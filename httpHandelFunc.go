package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

type reqStructUser struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	ExpiryTime    int    `json:"expires_in_seconds"`
	Is_chirpy_red bool   `json:"is_chirpy_red"`
}

type reqStructChirp struct {
	Body string `json:"body"`
}

// type UserId struct{
// 	User_Id int `json:"user_id"`
// }

type reqStructWebHooks struct {
	Event string `json:"event"`
	Data  struct {
		User_Id int `json:"user_id"`
	} `json:"data"`
}

func handelWebHooks(cfg *apiConfig) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		// check request has a valid apikey
		apikey, ok := getToken(r)
		if !ok {
			respWithError(w, 401, "enter a valid api key")
			return
		}

		if apikey != cfg.apiKey {
			respWithError(w, 401, "enter a valid api key")
			return
		}

		fmt.Println(apikey)

		type respStruct struct {
			Body string `json:"body"`
		}

		decoder := json.NewDecoder(r.Body)
		reqObj := reqStructWebHooks{}
		err := decoder.Decode(&reqObj)
		if err != nil {
			respWithError(w, 500, "can not decode request body")
			return
		}

		// check the event
		if reqObj.Event != "user.upgraded" {
			w.WriteHeader(200)
			return
		}

		// find the user and update in database
		ok, err = cfg.DB.webhookUserUpdate(reqObj)
		if !ok {
			w.WriteHeader(404)
			return
		}

		respWithJson(w, 200, respStruct{
			Body: "",
		})
	}
}

func postReqChirp(cfg *apiConfig) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		myDb := cfg.DB
		// Check user from r.Header -> token is valid. If valid do rest of things
		theUser, ok, err := myDb.getUserFromToken(r, cfg.jwtkey)
		if err != nil {
			respWithError(w, 500, "can not validate user")
			return
		}

		if !ok {
			respWithError(w, 401, "not authorized with refresh token")
			return
		}

		type respStruct struct {
			Response Chirp `json:"response"`
		}

		decoder := json.NewDecoder(r.Body)
		reqObj := reqStructChirp{}
		err = decoder.Decode(&reqObj)
		if err != nil {
			respWithError(w, 500, "can not decode request body")
			return
		}

		if len(reqObj.Body) > 140 {
			respWithError(w, 400, "message is more than 140 characters long!")
			return
		}

		// fmt.Println(r.Body)

		finalChripBody := removeBadWords(reqObj.Body)

		newChirp, err := myDb.createChirp(finalChripBody, theUser)

		if err != nil {
			respWithError(w, 500, fmt.Sprint("can not update the database with new chrip! ->> %v\n", err))
			return
		}

		resp := respStruct{
			Response: newChirp,
		}

		respWithJson(w, 201, resp.Response)

	}
}

func postReqUser(cfg *apiConfig) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		type respStruct struct {
			Id            int    `json:"id"`
			Email         string `json:"email"`
			Is_chirpy_red bool   `json:"is_chirpy_red"`
		}

		decoder := json.NewDecoder(r.Body)
		reqObj := reqStructUser{}
		err := decoder.Decode(&reqObj)
		if err != nil {
			respWithError(w, 500, "can not decode request body")
			return
		}

		newUser, err := cfg.DB.createUsers(reqObj)
		if err != nil {
			respWithError(w, 500, "can not create new user")
			return
		}

		respObj := respStruct{
			Id:            newUser.Id,
			Email:         newUser.Email,
			Is_chirpy_red: newUser.Is_chirpy_red,
		}

		respWithJson(w, 201, respObj)
	}
}

func loginUser(cfg *apiConfig) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		myDb := cfg.DB
		type respStruct struct {
			Id            int    `json:"id"`
			Email         string `json:"email"`
			Is_chirpy_red bool   `json:"is_chirpy_red"`
			Token         string `json:"token"`
			Refresh_Token string `json:"refresh_token"`
		}

		decoder := json.NewDecoder(r.Body)
		reqObj := reqStructUser{}
		err := decoder.Decode(&reqObj)
		if err != nil {
			respWithError(w, 500, "can not decode request body")
			return
		}

		theUser, errCode, err := myDb.loginTry(reqObj)

		if errCode == 16 {
			respWithError(w, 401, "password not matched")
			return
		}

		if errCode == 18 {
			respWithError(w, 404, "user not registered")
			return
		}
		if err != nil {
			respWithError(w, 500, "error in loginUser ")
			return
		}

		userAccessToken, err := createJwtAccessToken(cfg.jwtkey, theUser)
		if err != nil {
			respWithError(w, 500, "error in creating access token ")
			return
		}

		userRefreshToken, err := createJwtRefreshToken(cfg.jwtkey, theUser)
		if err != nil {
			respWithError(w, 500, "error in creating refresh token ")
			return
		}

		respObj := respStruct{
			Id:            theUser.Id,
			Email:         theUser.Email,
			Is_chirpy_red: theUser.Is_chirpy_red,
			Token:         userAccessToken,
			Refresh_Token: userRefreshToken,
		}

		respWithJson(w, 200, respObj)
	}
}

func putUsers(cfg *apiConfig) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get data from r.Body()
		myDb := cfg.DB
		type respStruct struct {
			Id            int    `json:"id"`
			Email         string `json:"email"`
			Is_chirpy_red bool   `json:"is_chirpy_red"`
		}

		decoder := json.NewDecoder(r.Body)
		reqObj := reqStructUser{}
		err := decoder.Decode(&reqObj)
		if err != nil {
			respWithError(w, 500, "can not decode request body")
			return
		}

		// Get token from r.Header()
		token, ok := getToken(r)
		if !ok {
			respWithError(w, 401, "No valid token")
			return
		}
		// fmt.Println(token)

		// Token -->> userId
		userId, issuer, err := checkTokenIsValid(token, cfg.jwtkey)
		if err != nil {
			respWithError(w, 401, "Not authorized")
			return
		}

		if issuer == "chirpy-refresh" {
			respWithError(w, 401, "issuer: refresh token")
			return
		}

		// Find and update user data
		updatedUser, err := myDb.updateUser(userId, reqObj)
		if err != nil {
			respWithError(w, 500, "can not create new user")
			return
		}

		respObj := respStruct{
			Id:            updatedUser.Id,
			Email:         updatedUser.Email,
			Is_chirpy_red: updatedUser.Is_chirpy_red,
		}

		respWithJson(w, 200, respObj)
	}
}

func revokeTokens(cfg *apiConfig) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		refreshToken, ok := getToken(r)
		if !ok {
			respWithError(w, 500, "can not get refresh token from header")
			return
		}

		err := cfg.DB.addToRevokeTokens(refreshToken)
		if err != nil {
			respWithError(w, 500, "can not add refresh token to database")
			return
		}

		w.WriteHeader(200)
	}
}

func refreshTokens(cfg *apiConfig) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		type respStruct struct {
			Token string `json:"token"`
		}

		refreshToken, ok := getToken(r)
		if !ok {
			respWithError(w, 500, "can not get refresh token from header")
			return
		}

		userId, issuer, err := checkTokenIsValid(refreshToken, cfg.jwtkey)
		if err != nil {
			// fmt.Println(refreshToken)
			// fmt.Printf("%v", err)
			respWithError(w, 401, "Not authorized")
			return
		}

		theUser, revoked, err := cfg.DB.checkAlreadyRevoked(refreshToken, userId)
		if err != nil {

			respWithError(w, 500, "err: finding in revokedtokens")
			return
		}

		// fmt.Println(revoked, issuer)

		if issuer == "chirpy-refresh" && revoked {
			// fmt.Println("2")
			respWithError(w, 401, "Not authorized")
			return
		}

		accessToken, err := createJwtAccessToken(cfg.jwtkey, theUser)
		if err != nil {
			respWithError(w, 500, "err: getting access token in revokedtokens")
			return
		}

		respWithJson(w, 200, respStruct{
			Token: accessToken,
		})

	}
}

func getReqChirp(myDb *DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		type respStruct struct {
			Response []Chirp
		}

		queryParams := map[string]string{}

		query_authorId := r.URL.Query().Get("author_id")
		if query_authorId != "" {
			queryParams["authorId"] = query_authorId
		}

		sort := r.URL.Query().Get("sort")
		if sort != "" {
			queryParams["sort"] = sort
		}

		if len(queryParams) > 0 {
			// Based on the queryParams we have to query the DB
			requiredChirps, errorCode := myDb.queryChirps(queryParams)
			if errorCode == 1 {
				respWithError(w, 500, "server error")
				return
			}
			if errorCode == 2 {
				respWithJson(w, 200, respStruct{Response: []Chirp{}}.Response)
				return
			}
			if errorCode == 3 {
				respWithError(w, 404, "No chirpy available by the author id")
				return
			}

			respWithJson(w, 200, respStruct{Response: requiredChirps}.Response)
			return
		}

		// if query_authorId != "" {

		// 	// get the chirps by authorid and send and return
		// 	myChirps, err := myDb.getChirpsByAuthor(query_authorId)
		// 	if err != nil {
		// 		respWithError(w, 404, "No chirpy available by the author id")
		// 		return
		// 	}
		// 	respObj := respStruct{
		// 		Response: myChirps,
		// 	}
		// 	respWithJson(w, 200, respObj.Response)
		// 	return
		// }

		allChirps, err := myDb.getChrips()
		if err != nil {
			respWithError(w, 500, "can not get all chirps!")
			return
		}

		// fmt.Println(allChirps)

		resp := respStruct{
			Response: allChirps,
		}

		respWithJson(w, 200, resp.Response)

	}
}

func getReqOneChirp(myDb *DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		type respStruct struct {
			Response Chirp
		}
		chirpsId := chi.URLParam(r, "chirpsId")

		s, err := strconv.Atoi(chirpsId)
		if err != nil {
			respWithError(w, 404, "enter a valid chirp id!")
			return
		}

		chirpy, err := myDb.getSingleChirpy(s)
		if err != nil {
			respWithError(w, 404, "No such chirp available")
			return
		}

		resp := respStruct{
			Response: chirpy,
		}

		respWithJson(w, 200, resp.Response)
	}
}

func deleteChirp(cfg *apiConfig) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get the user from tokens(authentication)
		theUser, ok, err := cfg.DB.getUserFromToken(r, cfg.jwtkey)
		if err != nil {
			respWithError(w, 500, "can not validate user")
			return
		}

		if !ok {
			respWithError(w, 401, "not authorized with refresh token")
			return
		}

		// Get the chirpy from params
		chirpsId := chi.URLParam(r, "chirpsId")

		chirpsIdInt, err := strconv.Atoi(chirpsId)
		if err != nil {
			respWithError(w, 404, "enter a valid chirp id!")
			return
		}

		chirpy, err := cfg.DB.getSingleChirpy(chirpsIdInt)
		if err != nil {
			respWithError(w, 404, "No such chirp available")
			return
		}

		// Check the author matches or not(authorization)
		if chirpy.Author_id != theUser.Id {
			respWithError(w, 403, "not authorised to delete this chirpy")
			return
		}

		err = cfg.DB.deleteChirpy(chirpsIdInt)
		if err != nil {
			respWithError(w, 500, "problem in deleting chirpy")
			return
		}

		w.WriteHeader(200)
	}
}

func checkTokenIsValid(token, key string) (string, string, error) {

	tokenInterface, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})

	if err != nil {
		return "", "", err
	}

	if claims, ok := tokenInterface.Claims.(*jwt.RegisteredClaims); ok && tokenInterface.Valid {
		userId := claims.Subject
		issuer := claims.Issuer
		return userId, issuer, nil
	} else {
		return "", "", err
	}

}

func getToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")

	// fmt.Println(header)

	headerSlice := strings.Fields(header)
	if len(headerSlice) < 2 {
		return "", false
	}
	token := headerSlice[1]
	// fmt.Printf("In getToken: %v\n", token)
	return token, true
}

func createJwtAccessToken(key string, user User) (string, error) {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "chirpy-access",
		Subject:   fmt.Sprintf("%d", user.Id),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// fmt.Println(token)
	ss, err := token.SignedString([]byte(key))
	// fmt.Println(key)
	if err != nil {
		return "", err
	}
	return ss, nil
}

func createJwtRefreshToken(key string, user User) (string, error) {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1440 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "chirpy-refresh",
		Subject:   fmt.Sprintf("%d", user.Id),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// fmt.Println(token)
	ss, err := token.SignedString([]byte(key))
	// fmt.Println(key)
	if err != nil {
		return "", err
	}
	return ss, nil
}

func contains(str string, words []string) bool {
	for _, word := range words {
		if word == str {
			return true
		}
	}
	return false
}

func removeBadWords(str string) string {
	badWordsSlice := []string{"kerfuffle", "sharbert", "fornax"}

	strSlice := strings.Split(str, " ")

	for i, s := range strSlice {
		sLower := strings.ToLower(s)
		isBadWord := contains(sLower, badWordsSlice)
		if isBadWord {
			strSlice[i] = "****"
		}
	}

	return strings.Join(strSlice, " ")

}

func respWithError(w http.ResponseWriter, code int, msg string) {
	if code > 499 {
		fmt.Printf("status code: %v\n", code)
	}
	type errorStruct struct {
		Response string `json:"response"`
	}

	respWithJson(w, code, errorStruct{
		Response: msg,
	})

}

func respWithJson(w http.ResponseWriter, code int, obj interface{}) {
	// fmt.Println(obj)
	dataByte, err := json.Marshal(obj)
	if err != nil {
		fmt.Printf("error occured in json.Marshal: %v\n", err)
		w.WriteHeader(code)
		return
	}

	// fmt.Println(dataByte)
	w.WriteHeader(code)
	w.Write(dataByte)
}
