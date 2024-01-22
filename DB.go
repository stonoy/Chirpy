package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

type DB struct {
	path string
	mux  *sync.RWMutex
}

type RevokedToken struct {
	rToken string
	time   time.Time
}

type DBStructure struct {
	Chirps        map[int]Chirp           `json:"chirps"`
	Users         map[int]User            `json:"users"`
	RevokedTokens map[string]RevokedToken `json:"revokedTokens"`
}

func NewDb(path string) (*DB, error) {
	// first create the DB struct(just body info) syntax
	myDB := &DB{
		path: path,
		mux:  &sync.RWMutex{},
	}

	// creating/ checking the myDB on the path
	err := myDB.ensureDB()
	if err != nil {
		return myDB, err
	}

	return myDB, nil

}

func (db *DB) ensureDB() error {
	// check(ensure) if the database already exists on the path
	_, err := os.ReadFile(db.path)
	if errors.Is(err, os.ErrNotExist) {
		//create the database on the path
		err := db.createDB()
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) createDB() error {
	DBStructure := DBStructure{
		Chirps:        map[int]Chirp{},
		Users:         map[int]User{},
		RevokedTokens: map[string]RevokedToken{},
	}
	err := db.writeDB(DBStructure)
	if err != nil {
		fmt.Printf("createDB problem %v\n", err)
		return err
	}
	return nil
}

func (db *DB) loadDB() (DBStructure, error) {

	db.mux.RLock()
	defer db.mux.RUnlock()

	// reads the data from certain path and gives its []byte representation
	dbByte, err := os.ReadFile(db.path)
	if err != nil {
		fmt.Printf("read err: %v\n", err)
		return DBStructure{}, err
	}

	// create a place to hold the unmarshal value of dbByte
	DBStruct := DBStructure{
		Chirps:        map[int]Chirp{},
		Users:         map[int]User{},
		RevokedTokens: map[string]RevokedToken{},
	}

	// convert and stores those []byte data inti DBStruct
	err = json.Unmarshal(dbByte, &DBStruct)
	if err != nil {
		fmt.Printf("unmarshal err: %v\n", err)
		return DBStructure{}, err
	}

	return DBStruct, nil

}

func (db *DB) writeDB(dbStructure DBStructure) error {

	db.mux.Lock()
	defer db.mux.Unlock()

	// convert the dbStructure into []byte
	dataByte, err := json.Marshal(dbStructure)
	if err != nil {
		fmt.Printf("marshal err: %v\n", err)
		return err
	}

	// write those converted []byte into db path
	err = os.WriteFile(db.path, dataByte, 0666)
	if err != nil {
		fmt.Printf("writefile err: %v\n", err)
		return err
	}
	return nil
}

// err := os.WriteFile(path, []byte{}, 0666)
// 	if err != nil {
// 		return nil, err
// 	}
// 	newDb := &DB{
// 		path: path,
// 		mux:  &sync.RWMutex{},
// 	}
// 	return newDb, nil
