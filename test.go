package mehs

import (
	"fmt"
	"log"
	"os"

	"github.com/garaekz/fileman/internal/util"
)

func main() {
	files := []string{"*.go", "test.txt"}
	expandedFiles := util.ExpandGlobs(files)
	fmt.Println("Expanded files:", expandedFiles)

	data := []byte("Hello, World!")
	hash := util.SHA1Hex(data)
	fmt.Println("SHA1 Hash:", hash)

	fileHash, err := util.SHA1FileHex("test.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("File SHA1 Hash:", fileHash)

	beforeInfo, _ := os.Stat("test.txt")
	afterInfo, _ := os.Stat("test.txt")
	if util.RaceDetected(beforeInfo, afterInfo) {
		fmt.Println("Race condition detected")
	}
}
