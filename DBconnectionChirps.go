package main

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
)

type Chirp struct {
	Body      string `json:"body"`
	Id        int    `json:"id"`
	Author_id int    `json:"author_id"`
}

func (db *DB) getUserFromToken(r *http.Request, key string) (User, bool, error) {
	Dbstruct, err := db.loadDB()
	if err != nil {
		fmt.Printf("error in validating user %v\n", err)
		return User{}, false, err
	}

	token, ok := getToken(r)
	if !ok {
		return User{}, false, errors.New("can not get token")
	}

	userId, issuer, err := checkTokenIsValid(token, key)
	if err != nil {
		return User{}, false, err
	}
	if issuer == "chirpy-refresh" {
		return User{}, false, nil
	}

	theUser, ok := findUserWithId(userId, Dbstruct)
	if !ok {
		return User{}, false, errors.New("error in finding the user")
	}

	return theUser, true, nil

}

func (DB *DB) createChirp(body string, theUser User) (Chirp, error) {

	DbStruct, err := DB.loadDB()
	if err != nil {
		fmt.Printf("error in createChirp %v\n", err)
		return Chirp{}, err
	}

	nextId := len(DbStruct.Chirps) + 1

	newChirp := Chirp{
		Id:        nextId,
		Body:      body,
		Author_id: theUser.Id,
	}

	DbStruct.Chirps[nextId] = newChirp

	err = DB.writeDB(DbStruct)

	if err != nil {
		fmt.Printf("error in createChirp %v\n", err)
		return Chirp{}, err
	}

	return newChirp, nil

}

func (DB *DB) getChrips() ([]Chirp, error) {
	chirpsArr := []Chirp{}

	DbStruct, err := DB.loadDB()
	if err != nil {
		fmt.Printf("error in getChirps %v\n", err)
		return chirpsArr, err
	}

	if len(DbStruct.Chirps) == 0 {
		fmt.Println("No chirps")
		return chirpsArr, nil
	}

	for _, value := range DbStruct.Chirps {

		chirpsArr = append(chirpsArr, value)
	}

	// fmt.Println(chirpsArr)

	return chirpsArr, nil
}

func (DB *DB) getSingleChirpy(id int) (Chirp, error) {
	DbStruct, err := DB.loadDB()
	if err != nil {
		fmt.Printf("error in getSingleChirpy %v\n", err)
		return Chirp{}, err
	}

	if len(DbStruct.Chirps) == 0 {
		fmt.Println("No chirps")
		return Chirp{}, nil
	}

	myChirp, ok := DbStruct.Chirps[id]
	if !ok {
		fmt.Printf("no chirp with id: %v\n", id)
		return Chirp{}, errors.New(fmt.Sprintf("no chirp with id: %v\n", id))
	}

	return myChirp, nil
}

func (db *DB) queryChirps(queryParams map[string]string) ([]Chirp, int) {
	DbStruct, err := db.loadDB()
	if err != nil {
		fmt.Printf("error in getSingleChirpy %v\n", err)
		return []Chirp{}, 1
	}

	if len(DbStruct.Chirps) == 0 {
		fmt.Println("No chirps")
		return []Chirp{}, 2
	}

	availableChirps := mapToSlice(DbStruct.Chirps)

	for param, value := range queryParams {
		// fmt.Println(param)
		if param == "authorId" {
			myChirps, ok := getChirpByAuthorId(value, availableChirps)
			if !ok {
				return []Chirp{}, 3
			}

			availableChirps = myChirps
		}

		if param == "sort" {
			if value == "asc" {
				sortAsendingChirps(availableChirps)
			}
			if value == "desc" {
				sortDesendingChirps(availableChirps)
			}

		}
	}

	return availableChirps, 0

}

func (DB *DB) deleteChirpy(id int) error {
	DbStruct, err := DB.loadDB()
	if err != nil {
		fmt.Printf("error in delete chirps %v\n", err)
		return err
	}

	if len(DbStruct.Chirps) == 0 {
		fmt.Println("No chirps")
		return errors.New("no chirps available")
	}

	if _, ok := DbStruct.Chirps[id]; ok {
		delete(DbStruct.Chirps, id)
		return nil
	}

	return errors.New("no chrip matches with the id")
}

func getChirpByAuthorId(id string, presentChirps []Chirp) ([]Chirp, bool) {
	// convert id: string -> int
	idInt, err := strconv.Atoi(id)
	if err != nil {
		// respWithError(w, 404, "enter a valid chirp id!")
		return []Chirp{}, false
	}

	myChirps := []Chirp{}
	anyChirpsFound := false

	for _, value := range presentChirps {
		if value.Author_id == idInt {
			anyChirpsFound = true
			myChirps = append(myChirps, value)
		}
	}
	return myChirps, anyChirpsFound
}

func sortAsendingChirps(presentChirps []Chirp) {
	sort.Slice(presentChirps, func(i, j int) bool { return presentChirps[i].Id < presentChirps[j].Id })
	// fmt.Println(presentChirps)
	// return presentChirps
}

func sortDesendingChirps(presentChirps []Chirp) {
	sort.Slice(presentChirps, func(i, j int) bool { return presentChirps[i].Id > presentChirps[j].Id })
	// fmt.Println(presentChirps)
	// return presentChirps
}

func mapToSlice(anyDS map[int]Chirp) []Chirp {
	DSlice := []Chirp{}
	for _, value := range anyDS {
		DSlice = append(DSlice, value)
	}
	return DSlice
}

// func getChirpsByAuthor(id string, presentChirps []Chirp) ([]Chirp, error) {

// 	// convert id: string -> int
// 	idInt, err := strconv.Atoi(id)
// 	if err != nil {
// 		// respWithError(w, 404, "enter a valid chirp id!")
// 		return []Chirp{}, err
// 	}

// 	myChirp, ok := getChirpByAuthorId(idInt, presentChirps)
// 	if !ok {
// 		return []Chirp{}, errors.New("no chirp available by the author id")
// 	}

// 	return myChirp, nil
// }
