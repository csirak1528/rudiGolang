package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	//"strings"

	"cloud.google.com/go/firestore"
	"github.com/rudi-network/goServerAlgorithm/files"
	"github.com/rudi-network/goServerAlgorithm/firebase"
	u "github.com/rudi-network/goServerAlgorithm/users"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type FirebasePostFiles struct{
	Files []files.File `json:"files"`
}

func GetFiles(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	vars :=mux.Vars(r)
	uid:=vars["uid"]
	docRef,err := db.Collection("users").Doc(uid).Get(context.Background())
	if err!=nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	userdata:=docRef.Data()
	userfiles:=fmt.Sprintf("%v",userdata["rudi"])
	fmt.Printf("%T",userfiles)
	files,err:=JSONtoStruct(userfiles)
	if err!=nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func SendFile(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	//vars := mux.Vars(r)
	//uid := vars["uid"]

	// Maximum upload of 10 MB files
	r.ParseMultipartForm(10 << 20)

	// Get handler for filename, size and headers
	file, handler, err := r.FormFile("myFile")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}

	defer file.Close()

	fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)
	os.Chdir("transit")
	dst, err := os.Create(handler.Filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer dst.Close()
	defer os.Chdir("")

	// Copy the uploaded file to the created file on the filesystem
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Successfully Uploaded File\n")
	var f files.File
	f.ID = uuid.New().String()
	//userdata, err := db.Collection("users").Doc(uid).Get(context.Background())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// rudi := userdata.Data()["rudi"]
	// var UserFiles files.File
	// err = FirebaseToFile(rudi, &UserFiles)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// }
	//fmt.Println(UserFiles)

	f.NewFile(handler.Filename)
	userFiles = append(userFiles, f)
	fmt.Println(userFiles)

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
func GetUser(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.Header().Set("Content-Type", "application/json")
	fmt.Println(r.RemoteAddr)
	json.NewEncoder(w).Encode(users)
}

//when user creates an account first time creates a space in the data base with their user information
//called one time on account creation
func CreateUser(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	vars := mux.Vars(r)
	uid := vars["uid"]
	//uuid := uuid.New().String()
	//ip := strings.Split(r.RemoteAddr, ":")[0]
	filesFirstore := StructToJSON(userFiles)
	_, err := db.Collection("users").Doc(uid).Update(context.Background(), []firestore.Update{{Path: "rudi", Value:filesFirstore }})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func AuthUser(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	vars := mux.Vars(r)
	uid := vars["uid"]

	fmt.Println(uid, r.RemoteAddr)
	//firebase auth check

	resp, err := db.Collection("users").Doc(uid).Get(context.Background())
	if err != nil {
		http.Error(w, "User was not found", http.StatusUnauthorized)
	}
	user := resp.Data()
	fmt.Println(user)

	w.WriteHeader(200)
}

func StructToJSON(f []files.File)string {
	filesStruct:=FirebasePostFiles{Files:f}
	jsonStruct,_:=json.Marshal(filesStruct)
	return string(jsonStruct)
}

func JSONtoStruct(fbJson string)([]files.File,error) {
	var userdata []files.File
	fbBytes :=[]byte(fbJson)
	err := json.Unmarshal(fbBytes,&userdata)
	if err != nil{
		return userdata,err
	}
	return userdata,nil
}



func UpdateUserFiles(uid,f files.File){
	
}


func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

var userFiles []files.File
var users []u.User
var db = firebase.InitDb()

func Server() {
	defer db.Close()

	//Init router
	r := mux.NewRouter()

	//Mock Data
	var f files.File
	f.NewFile("go.sum")
	userFiles = append(userFiles,f)
	r.HandleFunc("/api/files/{uid}", GetFiles).Methods("GET")
	r.HandleFunc("/api/users", GetUser).Methods("GET")

	r.HandleFunc("/api/files/{uid}", SendFile).Methods("POST")
	r.HandleFunc("/api/users/create/{uid}", CreateUser).Methods("POST")
	r.HandleFunc("/api/users/auth/{uid}", AuthUser).Methods("POST")

	// r.HandleFunc("/api/files", deleteFile).Methods("DELETE")
	// r.HandleFunc("/api/files", updateFile).Methods("PUT")

	go http.ListenAndServe(":8000", r)
	go http.ListenAndServe(":8001", r)
	http.ListenAndServe(":8002", r)
}
