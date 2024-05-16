package main

import (
	// "archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	// "path/filepath"
	"strconv"
	"strings"
)

type ResponseBody struct {
	FrameCount int    `json:"frameCount"`
	Error      string `json:"error"`
	Result     []struct {
		Anilist    int         `json:"anilist"`
		Filename   string      `json:"filename"`
		Episode    interface{} `json:"episode"`
		From       float64     `json:"from"`
		To         float64     `json:"to"`
		Similarity float64     `json:"similarity"`
		Video      string      `json:"video"`
		Image      string      `json:"image"`
	} `json:"result"`
}

type Url struct {
	ImageURL string
	Info     string
}

// func extractZip(filename string) ([]Url, error) {
// 	fmt.Println("Extracting file", filename)
// 	// open zip file
// 	reader, err := zip.OpenReader(filename)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer reader.Close()

// 	// make folder to store extracted files
// 	folderName := strings.Split(filename, ".")[0]
// 	errMakeFolder := os.Mkdir(folderName, 0755)
// 	if errMakeFolder != nil {
// 		return nil, errMakeFolder
// 	}

// 	var urls []Url
// 	for _, file := range reader.File {
// 		if !strings.HasSuffix(strings.ToLower(file.Name), ".jpg") && !strings.HasSuffix(strings.ToLower(file.Name), ".png") && !strings.HasSuffix(strings.ToLower(file.Name), ".jpeg") {
// 			continue
// 		}

// 		// open file
// 		fileReader, errOpenFile := file.Open()
// 		if errOpenFile != nil {
// 			return nil, errOpenFile
// 		}

// 		// create file
// 		fileName := strings.Split(file.Name, "/")[len(strings.Split(file.Name, "/"))-1]
// 		newFile, errCreateFile := os.Create(folderName + "/" + fileName)
// 		if errCreateFile != nil {
// 			return nil, errCreateFile
// 		}

// 		// copy file
// 		_, errCopy := io.Copy(newFile, fileReader)
// 		if errCopy != nil {
// 			return nil, errCopy
// 		}

// 		// close file
// 		errCloseFile := newFile.Close()
// 		if errCloseFile != nil {
// 			return nil, errCloseFile
// 		}

// 		// close file reader
// 		errCloseFileReader := fileReader.Close()
// 		if errCloseFileReader != nil {
// 			return nil, errCloseFileReader
// 		}
// 	}

// 	fmt.Println("Successfully extracted files to folder", folderName)
// 	urls, errReadFolder := ReadFromFolder(folderName)
// 	if errReadFolder != nil {
// 		return nil, errReadFolder
// 	}
// 	return urls, nil
// }

func ReadFromFolder(folderName string) ([]Url, error) {
	fmt.Println("Reading images from folder...")
	folder, errOpenFolder := os.Open(folderName)
	if errOpenFolder != nil {
		return nil, errOpenFolder
	}
	defer folder.Close()

	files, errReadFolder := folder.Readdir(-1)
	if errReadFolder != nil {
		return nil, errReadFolder
	}

	var urls []Url
	for _, file := range files {
		if !strings.HasSuffix(strings.ToLower(file.Name()), ".jpg") && !strings.HasSuffix(strings.ToLower(file.Name()), ".png") && !strings.HasSuffix(strings.ToLower(file.Name()), ".jpeg") {
			continue
		}

		urls = append(urls, Url{
			ImageURL: folderName + "/" + file.Name(),
			Info:     "",
		})
	}

	return urls, nil
}

func CreateFormData(filename string) (*bytes.Buffer, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// create form file
	formFile, errCreateFormFile := writer.CreateFormFile("image", filename)
	if errCreateFormFile != nil {
		return nil, errCreateFormFile
	}

	// open file
	fileReader, errOpenFile := os.Open(filename)
	if errOpenFile != nil {
		return nil, errOpenFile
	}
	defer fileReader.Close()

	// copy file to form file
	_, errCopy := io.Copy(formFile, fileReader)
	if errCopy != nil {
		return nil, errCopy
	}

	// close writer
	errCloseWriter := writer.Close()
	if errCloseWriter != nil {
		return nil, errCloseWriter
	}

	return body, nil
}

func GetAnimeInfo(imageUrls []Url) []Url {
	var animeInfos []Url
	for _, url := range imageUrls {
		// generate form data with image
		body, errCreateBody := CreateFormData(url.ImageURL)
		if errCreateBody != nil {
			animeInfos = append(animeInfos, Url{
				ImageURL: url.ImageURL,
				Info:     "Error: " + errCreateBody.Error(),
			})
			continue
		}

		// hit api
		request, errRequest := http.NewRequest("POST", "https://api.trace.moe/search", body)
		if errRequest != nil {
			animeInfos = append(animeInfos, Url{
				ImageURL: url.ImageURL,
				Info:     "Error when creating request : " + errRequest.Error(),
			})
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")

		client := new(http.Client)
		response, errHttpClient := client.Do(request)
		if errHttpClient != nil {
			animeInfos = append(animeInfos, Url{
				ImageURL: url.ImageURL,
				Info:     "Error when hit api : " + errHttpClient.Error(),
			})
			continue
		}

		// get response from api
		responseBody := response.Body
		defer responseBody.Close()

		if response.StatusCode != 200 {
			reader, _ := io.ReadAll(responseBody)
			fmt.Println(string(reader))
			animeInfos = append(animeInfos, Url{
				ImageURL: url.ImageURL,
				Info:     "Error when hit api : " + strconv.Itoa(response.StatusCode),
			})
			break
		}

		res := ResponseBody{}
		decoder := json.NewDecoder(responseBody)
		errDecoder := decoder.Decode(&res)
		if errDecoder != nil {
			fmt.Println(response.StatusCode, response.Header)
			animeInfos = append(animeInfos, Url{
				ImageURL: url.ImageURL,
				Info:     "Error when decoding response : " + errDecoder.Error(),
			})
			continue
		}

		if response.StatusCode == 200 {
			animeInfos = append(animeInfos, Url{
				ImageURL: url.ImageURL,
				Info:     res.Result[0].Filename,
			})
			continue
		}
	}

	return animeInfos
}

func CreateResultFile(animeInfos []Url) (string, error) {
	file, err := os.Create("result.txt")
	if err != nil {
		return "", err
	}
	defer file.Close()

	for _, info := range animeInfos {
		file.WriteString(info.ImageURL + ": " + info.Info + "\n")
	}

	return "result.txt", nil
}

func main() {
	fmt.Println("Pastikan file zip berada di direktori yang sama dengan file ini")
	fmt.Print("Masukkan nama file zip yang ingin diextract (contoh: anime.zip) : ")

	var filename string
	fmt.Scanln(&filename)

	// extract file
	// imageUrls, errExtract := extractZip(filename)
	// if errExtract != nil {
	// 	fmt.Println(errExtract)
	// 	return
	// }

	// read from folder
	imageUrls, errReadFolder := ReadFromFolder("example_anime")
	if errReadFolder != nil {
		fmt.Println(errReadFolder)
		return
	}

	// process image urls
	imageWithInfo := GetAnimeInfo(imageUrls)

	// generate result file
	resultFileName, errCreateResultFile := CreateResultFile(imageWithInfo)
	if errCreateResultFile != nil {
		fmt.Println(errCreateResultFile)
		return
	}

	fmt.Println("Result file created with name", resultFileName)
}
