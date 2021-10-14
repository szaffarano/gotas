package repo

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Reader reads user transactions
type Reader interface {
	Read(user User) ([]string, error)
}

// Appender appends new transactions for a given user
type Appender interface {
	Append(user User, data []string) error
}

// ReadAppender groups the basic Read and Append taskd functionality.
type ReadAppender interface {
	Reader
	Appender
}

// DefaultReadAppender is the default ReadAppender implementation on top of a
// simple fylesystem structure
type DefaultReadAppender struct {
	baseDir string
}

// NewDefaultReadAppender creates a new ReadAppender
func NewDefaultReadAppender(baseDir string) *DefaultReadAppender {
	return &DefaultReadAppender{baseDir}
}

type source string

// Read returns all the transaction information belonging to the given user.
func (ra *DefaultReadAppender) Read(user User) ([]string, error) {
	var file *os.File
	var err error
	txFile := filepath.Join(ra.baseDir, orgsFolder, user.Org.Name, usersFolder, user.Key, txFile)
	data := make([]string, 0, 50)

	if file, err = os.OpenFile(txFile, os.O_RDWR|os.O_CREATE, 0600); err != nil {
		return nil, fmt.Errorf("open tx file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}

	return data, nil
}

// Append add data at the end of the transaction user database.
func (ra *DefaultReadAppender) Append(user User, data []string) error {
	txFilePath := filepath.Join(ra.baseDir, orgsFolder, user.Org.Name, usersFolder, user.Key, txFile)
	txFileTempPath := filepath.Join(ra.baseDir, orgsFolder, user.Org.Name, usersFolder, user.Key, txFileTemp)
	var file *os.File

	if _, err := os.Stat(txFilePath); errors.Is(err, fs.ErrNotExist) {
		if file, err = os.OpenFile(txFileTempPath, os.O_RDWR|os.O_CREATE, 0600); err != nil {
			return fmt.Errorf("open tx file: %v", err)
		}
	} else {
		if err := (source(txFilePath)).copy(txFileTempPath); err != nil {
			return err
		}

		if file, err = os.OpenFile(txFileTempPath, os.O_RDWR|os.O_APPEND, 0600); err != nil {
			return fmt.Errorf("open tx file: %v", err)
		}
	}
	defer file.Close()

	for _, line := range data {
		if _, err := file.Write([]byte(line)); err != nil {
			return err
		}
	}

	// close the file before rename it
	if err := file.Close(); err != nil {
		return err
	}

	if err := os.Rename(txFileTempPath, txFilePath); err != nil {
		return err
	}

	return nil
}

func (s source) copy(dst string) error {
	src := string(s)

	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}
