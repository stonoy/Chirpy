package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Id            int    `json:"id"`
	Email         string `json:"email"`
	Password      string `json:"password"`
	Is_chirpy_red bool   `json:"is_chirpy_red"`
}

// type DBStructureUser struct {
// 	Users map[int]User `json:"users"`
// }

func (DB *DB) createUsers(req reqStructUser) (User, error) {

	DbStruct, err := DB.loadDB()
	if err != nil {
		fmt.Printf("error in createUser %v\n", err)
		return User{}, err
	}

	nextId := len(DbStruct.Users) + 1

	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		fmt.Printf("error in hashing password %v\n", err)
		return User{}, err
	}

	newUser := User{
		Id:            nextId,
		Email:         req.Email,
		Password:      hashedPassword,
		Is_chirpy_red: false,
	}

	DbStruct.Users[nextId] = newUser

	err = DB.writeDB(DbStruct)
	if err != nil {
		fmt.Printf("error in createUser %v\n", err)
		return User{}, err
	}

	return newUser, nil
}

func (db *DB) loginTry(req reqStructUser) (User, int, error) {

	Dbstruct, err := db.loadDB()
	if err != nil {
		fmt.Printf("error in loginUser %v\n", err)
		return User{}, 0, err
	}

	theUser, ok := checkUsers(Dbstruct.Users, req.Email)
	if !ok {
		return theUser, 18, errors.New("user not registered")
	}

	hasPasswordMatched := comparePassword(req.Password, theUser.Password)
	if !hasPasswordMatched {
		return User{}, 16, errors.New("password does not match")
	}

	return theUser, 0, nil

}

func (db *DB) updateUser(userId string, req reqStructUser) (User, error) {
	Dbstruct, err := db.loadDB()
	if err != nil {
		fmt.Printf("error in update user %v\n", err)
		return User{}, err
	}

	theUser, ok := findUserWithId(userId, Dbstruct)
	if !ok {
		return User{}, errors.New("can not found user")
	}

	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		fmt.Printf("error in hashing password %v\n", err)
		return User{}, err
	}

	updatedUser := User{
		Id:            theUser.Id,
		Email:         req.Email,
		Password:      hashedPassword,
		Is_chirpy_red: theUser.Is_chirpy_red,
	}

	Dbstruct.Users[theUser.Id] = updatedUser

	err = db.writeDB(Dbstruct)
	if err != nil {
		fmt.Printf("error in updateUser %v\n", err)
		return User{}, err
	}

	return updatedUser, nil

}

func (db *DB) addToRevokeTokens(token string) error {
	Dbstruct, err := db.loadDB()
	if err != nil {
		fmt.Printf("error in loading data %v\n", err)
		return err
	}

	theToken := RevokedToken{
		rToken: token,
		time:   time.Now(),
	}

	Dbstruct.RevokedTokens[token] = theToken

	err = db.writeDB(Dbstruct)
	if err != nil {
		fmt.Printf("error in writing r tokens in db %v\n", err)
		return err
	}

	return nil
}

func (db *DB) checkAlreadyRevoked(token, userId string) (User, bool, error) {
	Dbstruct, err := db.loadDB()
	if err != nil {
		fmt.Printf("error in loading data %v\n", err)
		return User{}, true, err
	}

	_, ok := Dbstruct.RevokedTokens[token]
	if ok {
		return User{}, true, nil
	}

	theUser, ok := findUserWithId(userId, Dbstruct)
	if !ok {
		fmt.Println("can not find user in checkAlreadyRovoked func")
		return User{}, true, nil
	}

	return theUser, false, nil
}

func (db *DB) webhookUserUpdate(req reqStructWebHooks) (bool, error) {
	Dbstruct, err := db.loadDB()
	if err != nil {
		fmt.Printf("error in loading data %v\n", err)
		return false, err
	}

	theUser, ok := findUserWithId(fmt.Sprintf("%d", req.Data.User_Id), Dbstruct)
	if !ok {
		return false, nil
	}

	updatedUser := User{
		Id:            theUser.Id,
		Email:         theUser.Email,
		Password:      theUser.Password,
		Is_chirpy_red: true,
	}

	Dbstruct.Users[theUser.Id] = updatedUser

	err = db.writeDB(Dbstruct)
	if err != nil {
		fmt.Printf("error in writing a user %v\n", err)
		return false, err
	}

	return true, nil
}

func findUserWithId(id string, DBstruct DBStructure) (User, bool) {

	intId, err := strconv.Atoi(id)
	if err != nil {
		fmt.Println(err)
		return User{}, false
	}

	for _, value := range DBstruct.Users {
		if value.Id == intId {
			return value, true
		}
	}

	return User{}, false
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func comparePassword(password, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func checkUsers(allUsers map[int]User, email string) (User, bool) {
	for _, user := range allUsers {
		if user.Email == email {
			return user, true
		}
	}
	return User{}, false
}
