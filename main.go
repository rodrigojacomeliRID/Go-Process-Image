package main

import (
	"archive/zip"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nfnt/resize"
)

func createDirIfNotExist(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, os.ModePerm)
	}
}

func overlayImages(artPath, footerPath, outputPath string, wg *sync.WaitGroup, sem chan struct{}) {
	log.Println("Procenssando: ", artPath, footerPath)

	defer wg.Done()
	sem <- struct{}{}

	defer func() {
		<-sem
	}()

	artFile, err := os.Open(artPath)
	if err != nil {
		fmt.Println("Error opening art image:", err)
		return
	}
	defer artFile.Close()

	footerFile, err := os.Open(footerPath)
	if err != nil {
		fmt.Println("Error opening footer image:", err)
		return
	}
	defer footerFile.Close()

	artImg, err := jpeg.Decode(artFile)
	if err != nil {
		fmt.Println("Error decoding art image:", err)
		return
	}

	footerImg, err := png.Decode(footerFile)
	if err != nil {
		fmt.Println("Error decoding footer image:", err)
		return
	}

	offset := image.Pt(120, 1220)
	resizedFooter := resize.Resize(550, 120, footerImg, resize.Lanczos3)
	bounds := artImg.Bounds()
	mergedImg := image.NewRGBA(bounds)
	draw.Draw(mergedImg, bounds, artImg, image.Point{}, draw.Src)
	draw.Draw(mergedImg, resizedFooter.Bounds().Add(offset), resizedFooter, image.Point{}, draw.Over)

	outputFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Println("Error creating output image:", err)
		return
	}
	defer outputFile.Close()

	err = jpeg.Encode(outputFile, mergedImg, nil)
	if err != nil {
		fmt.Println("Error encoding output image:", err)
	}
}

func compressFiles(inputDir, outputFilePath string) {
	zipFile, err := os.Create(outputFilePath)
	if err != nil {
		fmt.Println("Error creating ZIP file:", err)
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	files, err := ioutil.ReadDir(inputDir)
	if err != nil {
		fmt.Println("Error reading input directory:", err)
		return
	}

	for _, file := range files {
		filePath := filepath.Join(inputDir, file.Name())
		fileToZip, err := os.Open(filePath)
		if err != nil {
			fmt.Println("Error opening file to ZIP:", err)
			return
		}
		defer fileToZip.Close()

		zipEntryWriter, err := zipWriter.Create(file.Name())
		if err != nil {
			fmt.Println("Error creating ZIP entry:", err)
			return
		}

		_, err = io.Copy(zipEntryWriter, fileToZip)
		if err != nil {
			fmt.Println("Error writing to ZIP entry:", err)
			return
		}
	}
}

func processImages() {
	artDir := "./ARTES"
	footerBaseDir := "./RODAPE"
	outputDir := "./ARTES_PRONTAS"
	tempDir := "./TEMP"

	createDirIfNotExist(outputDir)
	createDirIfNotExist(tempDir)

	var wg sync.WaitGroup

	arts, err := ioutil.ReadDir(artDir)
	if err != nil {
		fmt.Println("Error reading art directory:", err)
		return
	}

	footerDirs, err := ioutil.ReadDir(footerBaseDir)
	if err != nil {
		fmt.Println("Error reading footer base directory:", err)
		return
	}

	sem := make(chan struct{}, 10) // Limitar a 10 goroutines simultâneas

	for _, footerDir := range footerDirs {
		if footerDir.IsDir() {
			footerDirPath := filepath.Join(footerBaseDir, footerDir.Name())
			footers, err := ioutil.ReadDir(footerDirPath)
			if err != nil {
				fmt.Println("Error reading footer directory:", err)
				continue
			}

			for _, footer := range footers {
				if !footer.IsDir() && filepath.Ext(footer.Name()) == ".png" {
					footerPath := filepath.Join(footerDirPath, footer.Name())
					tempSubDir := filepath.Join(tempDir, footer.Name())
					createDirIfNotExist(tempSubDir)

					for _, art := range arts {
						if !art.IsDir() && filepath.Ext(art.Name()) == ".jpg" {
							artPath := filepath.Join(artDir, art.Name())
							outputFilePath := filepath.Join(tempSubDir, art.Name())
							wg.Add(1)
							go overlayImages(artPath, footerPath, outputFilePath, &wg, sem)
						}
					}

					wg.Wait()

					// Compress the temporary folder
					compressedFileName := fmt.Sprintf("%s.zip", footer.Name())
					compressedFilePath := filepath.Join(outputDir, compressedFileName)
					compressFiles(tempSubDir, compressedFilePath)

					// Clean up temporary directory
					os.RemoveAll(tempSubDir)
				}
			}
		}
	}

	fmt.Println("Image processing complete.")
}

func main() {

	createDirIfNotExist("./ARTES")
	createDirIfNotExist("./RODAPE")
	createDirIfNotExist("./ARTES_PRONTAS")

	fmt.Println("Pressione Enter para iniciar o processamento...")
	fmt.Scanln() // Espera a entrada do usuário

	startTime := time.Now() // Captura o tempo de início

	log.Printf("### %v INICIANDO PROCESSAMENTO ... ", startTime)

	processImages()

	// Calcula a duração desde o início até agora
	elapsedTime := time.Since(startTime)

	fmt.Printf("### PROCESSAMENTO FINALIZADO COM SUCESSO!!! \n Tempo decorrido: %v\n", elapsedTime)

	fmt.Println("Pressione Enter para sair...")
	fmt.Scanln() // Espera a entrada do usuário antes de fechar
}
