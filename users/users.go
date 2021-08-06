package users

import (
	"github.com/rudi-network/goServerAlgorithm/files"
)

type User struct {
	Uuid     string       `json:"uuid" firestore:"uuid"`
	Ip       string       `json:"ip" firestore:"ip"`
	Files    []string     `json:"files" firestore:"files"`
	FileData []files.File `json:"filedata" firestore:"filedata"`
}
