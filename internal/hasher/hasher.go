package hasher

import (
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	validatorDataDir  = ".storage-validator"
	validatorDataFile = "data"
)

type Client struct {
	dataDir  string
	dataFile string

	wd string

	dataFileExists bool
}

func New(dir string) (*Client, error) {
	// Check if source directory exists.
	e, eErr := exists(dir)
	if eErr != nil {
		return nil, fmt.Errorf("failed to check whether source directory exists: %v", eErr)
	}

	if !e {
		return nil, fmt.Errorf("source directory %s doesn't exist", dir)
	}

	// Check if storage validator data directory exists.
	dataDir := fmt.Sprintf("%s/%s", dir, validatorDataDir)
	e, eErr = exists(dataDir)
	if eErr != nil {
		return nil, fmt.Errorf("failed to check whether storage validator data directory exists: %v", eErr)
	}

	if !e {
		// Create storage validator data directory.
		mkdirErr := os.Mkdir(dataDir, 0777)
		if mkdirErr != nil {
			return nil, fmt.Errorf("failed to create storage validator data directory: %v", mkdirErr)
		}
	}

	// Check if storage validator data file exists.
	dataFile := fmt.Sprintf("%s/%s/%s", dir, validatorDataDir, validatorDataFile)
	dataFileE, dataFileEErr := exists(dataFile)
	if dataFileEErr != nil {
		return nil, fmt.Errorf("failed to check whether storage validator data file exists: %v", dataFileEErr)
	}

	return &Client{
		dataDir:  dataDir,
		dataFile: dataFile,

		wd: dir,

		dataFileExists: dataFileE,
	}, nil
}

// Init walks through directory files and generates hash file.
func (c *Client) Init() error {
	// Check if data file already exists.
	if c.dataFileExists {
		return fmt.Errorf("directory %s has already been initialized, use reset mode to reset storage validator state", c.wd)
	}

	// Open data file.
	dataF, dataFErr := os.OpenFile(c.dataFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
	if dataFErr != nil {
		return fmt.Errorf("failed to open data file: %v", dataFErr)
	}

	// Greate GZip writer.
	gzipDataFW := gzip.NewWriter(dataF)
	gzipDataFWBuf := bufio.NewWriter(gzipDataFW)

	// Close data file and buffers after the execution.
	defer dataF.Close()
	defer gzipDataFW.Close()
	defer gzipDataFWBuf.Flush()

	// Count files.
	var filesNumber int64

	walkErr := filepath.Walk(
		c.wd,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			// Trim path.
			if path[len(path)-1] == '/' {
				path = path[:len(path)-1]
			}

			// Ignore storage validator files.
			if len(path) == 0 || path == c.wd || path == c.dataDir || path == c.dataFile {
				return nil
			}

			filesNumber++

			return nil
		})
	if walkErr != nil {
		return fmt.Errorf("failed to get contents of %s: %v", c.wd, walkErr)
	}

	log.Printf("processing %d files", filesNumber)

	// List files to encrypt.
	var filesProcessed int64
	var prevPerc int

	walkErr = filepath.Walk(
		c.wd,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			// Trim path.
			if path[len(path)-1] == '/' {
				path = path[:len(path)-1]
			}

			// Ignore storage validator files.
			if len(path) == 0 || path == c.wd || path == c.dataDir || path == c.dataFile {
				return nil
			}

			// Get file checksum.
			checksum, checksumErr := fileHash(path)
			if checksumErr != nil {
				return fmt.Errorf("failed to get checksum for file %s: %v", path, checksumErr)
			}

			// Use relative path.
			relPath := path[len(c.wd)+1:]

			// Write to data file.
			_, csWErr := gzipDataFWBuf.Write([]byte(fmt.Sprintf("%s:%s\n", relPath, checksum)))
			if csWErr != nil {
				return fmt.Errorf("failed to write checksum for file %s: %v", path, csWErr)
			}

			filesProcessed++

			// Print progress.
			pPerc := int(filesProcessed * 100 / filesNumber)
			if pPerc != prevPerc {
				log.Printf("%d%%", pPerc)
				prevPerc = pPerc
			}

			return nil
		})
	if walkErr != nil {
		return fmt.Errorf("failed to get contents of %s: %v", c.wd, walkErr)
	}

	if prevPerc != 100 {
		log.Print("100%")
	}

	return nil
}

// Validate validates directory files against previously stored checksum.
func (c *Client) Validate() error {
	// Get expected files number.
	en, enErr := c.dataFileLines()
	if enErr != nil {
		return fmt.Errorf("failed to get expected files number: %v", enErr)
	}

	log.Printf("validating %d files", en)

	// Check if data file already exists.
	if !c.dataFileExists {
		return fmt.Errorf("directory %s has not been initialized, use init mode first", c.wd)
	}

	// Init gzip and file readers.
	gzipF, gzipFErr := os.Open(c.dataFile)
	if gzipFErr != nil {
		return fmt.Errorf("failed to open data file: %v", gzipFErr)
	}

	gzipReader, gzipReaderErr := gzip.NewReader(gzipF)
	if gzipReaderErr != nil {
		return fmt.Errorf("failed to init gzip reader on file %s: %v", c.dataFile, gzipReaderErr)
	}

	// Close data file and buffers after the execution.
	defer gzipReader.Close()
	defer gzipF.Close()

	// Read data file line by line.
	var lineNum int = 1
	var prevPerc int

	scanner := bufio.NewScanner(gzipReader)
	for scanner.Scan() {
		// Extract recorded filepath and checksum.
		line := scanner.Text()
		lineSl := strings.Split(line, ":")
		if len(lineSl) < 2 {
			return fmt.Errorf("corrupted data file %s, line %d: %s", c.dataFile, lineNum, line)
		}

		recordedChecksum := lineSl[len(lineSl)-1]
		recordedFilepath := strings.Join(lineSl[:len(lineSl)-1], ":")

		// Check if file exists.
		fullPath := path.Join(c.wd, recordedFilepath)

		e, eErr := exists(fullPath)
		if eErr != nil {
			return fmt.Errorf("failed to check whether file %s exists: %v", recordedFilepath, eErr)
		}

		if !e {
			return fmt.Errorf("file %s doesn't exist", recordedFilepath)
		}

		// Validate file.
		realChecksum, realChecksumErr := fileHash(fullPath)
		if realChecksumErr != nil {
			return fmt.Errorf("failed to get checksum for file %s: %v", fullPath, realChecksumErr)
		}

		if realChecksum != recordedChecksum {
			return fmt.Errorf("checksum mismatch for file %s", fullPath)
		}

		// Print progress.
		pPerc := int(int64(lineNum) * 100 / en)
		if pPerc != prevPerc {
			log.Printf("%d%%", pPerc)
			prevPerc = pPerc
		}

		// Increase line number.
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error reading data file %s: %v", c.dataFile, err)
	}

	if prevPerc != 100 {
		log.Print("100%")
	}

	return nil
}

func (c *Client) dataFileLines() (int64, error) {
	// Check if data file already exists.
	if !c.dataFileExists {
		return 0, fmt.Errorf("directory %s has not been initialized, use init mode first", c.wd)
	}

	// Init gzip and file readers.
	gzipF, gzipFErr := os.Open(c.dataFile)
	if gzipFErr != nil {
		return 0, fmt.Errorf("failed to open data file: %v", gzipFErr)
	}

	gzipReader, gzipReaderErr := gzip.NewReader(gzipF)
	if gzipReaderErr != nil {
		return 0, fmt.Errorf("failed to init gzip reader on file %s: %v", c.dataFile, gzipReaderErr)
	}

	// Close data file and buffers after the execution.
	defer gzipReader.Close()
	defer gzipF.Close()

	// Read data file line by line.
	var lineNum int64
	scanner := bufio.NewScanner(gzipReader)
	for scanner.Scan() {
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("scanner error reading data file %s: %v", c.dataFile, err)
	}

	return lineNum, nil
}

// Reset removes storage validator data directory.
func (c *Client) Reset() {
	os.RemoveAll(c.dataDir)
}

// exists returns whether the given file or directory exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// from https://stackoverflow.com/questions/15879136/how-to-calculate-sha256-file-checksum-in-go:
// It's processed by chunks of 32KB: golang.org/src/io/io.go#L380 You can change
// that by providing your own buffer using CopyBuffer.
func fileHash(filepath string) (string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %v", filepath, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to IO copy file %s contents: %v", filepath, err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
