package files

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/klauspost/compress/zstd"
)

//base File definition
type File struct {
	UserId    string   `json:"uid" firestore:"uid"`
	ID        string   `json:"id" firestore:"id"`
	Name      string   `json:"name" firestore:"name"`
	Size      int64    `json:"size" firestore:"size"`
	Extension string   `json:"extension" firestore:"extension"`
	Path      string   `json:"path" firestore:"path"`
	Shards    BuildDir `json:"shards "firestore:"shards"`
	Buffer    int32    `json:"buffer" firestore:"buffer"`
	Exists    bool     `json:"exists" firestore:"exists"`
	ShardDir  string   `json:"sharddir" firestore:"shardir"`
}

//defines Shard as map where key is secret_function(pos....) can be queried

type Ip string

type SendDir map[Ip]string

type UserDir map[string]map[int64]string

type BuildDir map[int64]string
type ByteDir map[int64][]byte

const BaseDir string = "transit/file/"

var encoder, _ = zstd.NewWriter(nil)
var decoder, _ = zstd.NewReader(nil)

func (f *File) NewFile(filepath string) {
	// Defines System for use
	var slash string
	switch runtime.GOOS {
	case "windows":
		slash = `\`
	case "darwin":
		slash = "/"
	case "Linux":
		slash = "/"
	default:
		f.Kill()
	}

	//base values don't need computation to define
	f.Path = filepath
	_, err := os.Stat(filepath)
	f.Exists = err == nil
	f.Buffer = 16536

	name := path.Base(filepath)
	//sets position of last dot in path name
	dotPos := strings.LastIndex(name, ".")

	//sets the file name to be before the dot
	if slashPos := strings.LastIndex(name, slash); slashPos > 0 {
		f.Name = name[slashPos+1 : dotPos]
	} else {
		f.Name = name[:dotPos]
	}
	//sets the extenstion to be after the dot
	f.Extension = name[dotPos+1:]

	//sets size of file if there isnt an error(if file exists)
	f.setSize()
}

//uses OS library to get size of file
func (f *File) setSize() {
	if f.Exists {
		//gets status of file
		fi, err := os.Stat(f.Path)

		if err != nil {
			f.Kill()
		}
		// get the size
		size := fi.Size()
		f.Size = size
	}
}

//sets exists attribute to false rendering it inoperable
func (f *File) Kill() {
	f.Exists = false
}

func (f *File) Shard() bool {
	if f.Exists {
		//kills file if it doesnt exist
		_, err := os.Stat(f.Path)
		if err != nil {
			f.Kill()
			return false
		}

		//creates sha256 hash name for file dir in server
		//INCLUDE USER ID
		dirName := MakeHash(f.Name)
		fileDirHash := BaseDir + dirName
		err = EnsureDir(fileDirHash)
		if err != nil {
			return false
		}
		//Changes dir to the files
		f.ShardDir = fileDirHash
		os.Chdir(fileDirHash)

		f.CreateShards(dirName)

	} else {
		return false
	}
	return true
}

func EnsureDir(dirName string) error {
	//makes sure a dir exists if not make it
	res, _ := exists(dirName)
	if !res {
		err := os.Mkdir(dirName, 0777)
		if err == nil || os.IsExist(err) {
			return nil
		} else {
			return err
		}
	}
	return nil
}

func WriteFile(filename string, data []byte) bool {
	//creates and writes file with binary data
	err := ioutil.WriteFile(filename, data, 0644)
	return err == nil
}

func (f *File) CreateShards(shardName string) bool {
	//Makes sure File is real
	file, err := os.Open(f.Path)
	if err != nil {
		f.Kill()
		return false
	}

	//Creates reader and reads all file data into byte arr
	reader := bufio.NewReader(file)
	buffer := f.Buffer
	buildDir := make(BuildDir)
	alldata := make([]byte, f.Size)
	for {
		_, err = reader.Read(alldata)
		if err != nil {
			break
		}
	}
	//gets compressed file data
	alldata = Compress(alldata)
	//Defines length and number of shards for the for lop
	shardNum := len(alldata)/int(f.Buffer) + 1
	shardSize := int32(len(alldata))

	//stores value of shardSize for later comparing compression efficency
	newSize := shardSize

	for i := int64(1); i <= int64(shardNum); i++ {
		//if the bytes left are less than the buffer make the buffer however many left
		if shardSize < buffer {
			buffer = shardSize
		}

		//creates byte arr for storing data into a new shard
		fileShard := make([]byte, 0)
		curSlice := alldata[:int64(buffer)]
		//stores a buffer sized block of bytes into shard
		fileShard = append(fileShard, curSlice...)

		//cuts off the bytes stored in new shard
		alldata = alldata[int64(buffer):]

		//creates a hash from the name of the shard and the position
		fileHash := MakeHash(fmt.Sprintf("%d>>>%v", i, shardName))
		//finalizes data as a rudi file
		finalName := fmt.Sprintf("%v.rudi", fileHash)
		buildDir[i] = finalName

		//write the new file with compressed data
		WriteFile(finalName, fileShard)
		shardSize -= f.Buffer
	}
	fmt.Printf("New Size:%d, Old Size:%d Compression Ratio:%f\n", newSize, f.Size, float64(f.Size)/float64(len(alldata)))
	//stores build dir in file struct
	f.Shards = buildDir
	file.Close()
	return true
}

func exists(path string) (bool, error) {
	// Checks if a path exists
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (f *File) Rebuild() bool {
	//changes dir to dir of the files
	os.Chdir(f.ShardDir)
	//creates a main buffer to store all data
	mainBuf := make([]byte, 0)

	// if the file has shards then execute
	if len(f.Shards) > 0 {
		//for the length of the shards
		for i := int64(1); i <= int64(len(f.Shards)); i++ {
			//checks file data to get length
			filename := f.Shards[i]
			fileInfo, _ := os.Stat(filename)

			//gets a buffer to store data of the file
			curBuf := make([]byte, fileInfo.Size())

			//opens file
			file, err := os.Open(filename)
			// if there is an error opening the file break and return the error
			if err != nil {
				return false
			}
			//read the file
			for {
				_, err := file.Read(curBuf)
				if err != nil {
					break
				}
			}
			//adds files data to main byte arr
			mainBuf = append(mainBuf, curBuf...)
			//closes and removes shard
			file.Close()
			os.Remove(filename)
		}
		//Defines file name from file struct
		newFileName := fmt.Sprintf("%v.%v", f.Name, f.Extension)
		//decompresses and checks for errors
		mainBuf, err := Decompress(mainBuf)
		if err != nil {
			mainBuf = []byte{0}
		}
		//Creates file from main byte arr
		WriteFile(newFileName, mainBuf)
		return true

	} else {
		return false
	}

}

func MakeHash(term string) string {
	//creates a sha256 hash
	Hash := sha256.New()
	Hash.Write([]byte(term))
	return fmt.Sprintf("%x", Hash.Sum(nil))
}

func Compress(src []byte) []byte {
	return encoder.EncodeAll(src, make([]byte, 0, len(src)))
}

func Decompress(src []byte) ([]byte, error) {
	return decoder.DecodeAll(src, nil)

}
