package repo

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/google/uuid"
	"github.com/szaffarano/gotas/pkg/config"
)

const (
	Confirmation = "confirmation"
	Extensions   = "extensions"
	IpLog        = "ip.log"
	Log          = "log"
	PidFile      = "pid.file"
	QueueSize    = "queue.size"
	RequestLimit = "request.limit"
	Root         = "root"
	BindAddress  = "server"
	Trust        = "trust"
	Verbose      = "verbose"
	ClientCert   = "client.cert"
	ClientKey    = "client.key"
	ServerKey    = "server.key"
	ServerCert   = "server.cert"
	ServerCrl    = "server.crl"
	CaCert       = "ca.cert"
)

const (
	orgsFolder  = "orgs"
	usersFolder = "users"
	txFile      = "tx.data"
	txFileTemp  = "tx.tmp.data"
)

// Repository defines an API with the task server operations, orgs and users ABM, initialization, etc.
type Repository struct {
	Config config.Config
	orgs   []Organization
}

type Organization struct {
	Name  string
	Users []User
}

type User struct {
	Name string
	Key  string
	Org  *Organization
}

type AuthenticationError struct {
	Code string
	Msg  string
}

func (e AuthenticationError) Error() string {
	return e.Msg
}

func (r *Repository) NewOrg(orgName string) (*Organization, error) {
	for _, org := range r.orgs {
		if org.Name == orgName {
			return nil, fmt.Errorf("Organization %q already exists", orgName)
		}
	}

	newOrgPath := filepath.Join(r.Config.Get(Root), orgsFolder, orgName)
	if err := os.Mkdir(newOrgPath, 0775); err != nil {
		return nil, fmt.Errorf("creating new org: %v", err)
	}
	if err := os.Mkdir(filepath.Join(newOrgPath, usersFolder), 0775); err != nil {
		return nil, fmt.Errorf("creating users dir under org: %v", err)
	}

	newOrg := Organization{Name: orgName}
	r.orgs = append(r.orgs, newOrg)

	return &newOrg, nil
}

func (r *Repository) GetOrg(orgName string) (*Organization, error) {
	var users []User
	root := filepath.Join(r.Config.Get(Root), orgsFolder, orgName, usersFolder)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if d.Name() == usersFolder {
				return nil
			}
			userConfigPath := filepath.Join(path, "config")
			if userConfig, err := config.Load(userConfigPath); err == nil {
				users = append(users, User{
					Key:  d.Name(),
					Name: userConfig.Get("user"),
				})
			} else {
				log.Warnf("Ignoring user %q: %v", d.Name(), err)
				return fs.SkipDir
			}
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getting users: %v", err)
	}

	org := Organization{Name: orgName, Users: users}
	for idx := range users {
		users[idx].Org = &org
	}
	return &org, nil
}

func (r *Repository) AddUser(orgName string, userName string) (*User, error) {
	org, err := r.GetOrg(orgName)
	if err != nil {
		return nil, err
	}

	for _, u := range org.Users {
		if u.Name == userName {
			return nil, fmt.Errorf("User %q already exists", userName)
		}
	}

	key := uuid.New().String()
	userPath := filepath.Join(r.Config.Get(Root), orgsFolder, org.Name, usersFolder, key)
	if err := os.Mkdir(userPath, 0755); err != nil {
		return nil, fmt.Errorf("creating user home: %v", err)
	}

	cfg, err := config.New(filepath.Join(userPath, "config"))
	if err != nil {
		return nil, fmt.Errorf("creating user config: %v", err)
	}
	cfg.Set("user", userName)
	if err := config.Save(cfg); err != nil {
		return nil, fmt.Errorf("saving user config: %v", err)
	}

	return &User{
		Name: userName,
		Key:  key,
		Org:  org,
	}, nil
}

// NewRepository create a brand new repository in the given dataDir
func NewRepository(dataDir string) (*Repository, error) {
	if fileInfo, err := os.Stat(dataDir); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%v: does not exist", dataDir)
	} else if errors.Is(err, fs.ErrPermission) {
		return nil, fmt.Errorf("%v: permission denied", dataDir)
	} else if err != nil {
		return nil, fmt.Errorf("read dir info %v: %v", dataDir, err)
	} else if !fileInfo.IsDir() {
		return nil, fmt.Errorf("%v: directory expected", dataDir)
	} else if dataDir, err = filepath.Abs(dataDir); err != nil {
		return nil, fmt.Errorf("calculate dir absolute path %v: %v", dataDir, err)
	} else if files, err := ioutil.ReadDir(dataDir); err != nil {
		return nil, fmt.Errorf("list dir %v: %v", dataDir, err)
	} else if len(files) > 0 {
		return nil, fmt.Errorf("%s: not empty", dataDir)
	}

	orgPath := filepath.Join(dataDir, orgsFolder)
	if err := os.Mkdir(orgPath, 0755); err != nil {
		return nil, fmt.Errorf("create initial structure %v: %v", orgPath, err)
	}

	configFilePath := filepath.Join(dataDir, "config")
	cfg, err := config.New(configFilePath)
	if err != nil {
		return nil, err
	}

	// set default values
	cfg.SetBool(Confirmation, true)
	cfg.Set(Log, filepath.Join(os.TempDir(), "taskd.log"))
	cfg.Set(PidFile, filepath.Join(os.TempDir(), "taskd.pid"))
	cfg.SetInt(QueueSize, 10)
	cfg.SetInt(RequestLimit, 1048576)
	cfg.Set(Root, dataDir)
	cfg.Set(Trust, "strict")
	cfg.SetBool(Verbose, true)

	if err := config.Save(cfg); err != nil {
		return nil, err
	}

	return &Repository{Config: cfg}, nil
}

func OpenRepository(dataDir string) (*Repository, error) {
	configFilePath := filepath.Join(dataDir, "config")
	cfg, err := config.Load(configFilePath)
	if err != nil {
		return nil, err
	}

	orgsRoot := filepath.Join(dataDir, orgsFolder)
	var orgsToAdd []string
	err = filepath.WalkDir(orgsRoot, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if d.Name() == orgsFolder {
				return nil
			}
			orgsToAdd = append(orgsToAdd, d.Name())
			return fs.SkipDir
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("opening repository: %v", err)
	}

	repo := Repository{Config: cfg}
	for _, orgName := range orgsToAdd {
		org, err := repo.GetOrg(orgName)
		if err != nil {
			log.Warnf("Ignoring organization %q:  %v", orgName, err)
			continue
		}
		repo.orgs = append(repo.orgs, *org)
	}

	return &repo, nil
}

func (r *Repository) Authenticate(orgName, userName, key string) (User, error) {
	org, err := r.GetOrg(orgName)
	if err != nil {
		return User{}, AuthenticationError{"400", "Invalid org"}
	}

	for _, u := range org.Users {
		if u.Key == key && u.Name == userName {
			return u, nil
		}
	}

	return User{}, AuthenticationError{"401", "Invalid username or key"}
}

func (r *Repository) GetData(user User) ([]string, error) {
	txFile := filepath.Join(r.Config.Get(Root), orgsFolder, user.Org.Name, usersFolder, user.Key, txFile)
	var file *os.File
	var err error
	data := make([]string, 0, 50)

	if file, err = os.OpenFile(txFile, os.O_RDWR|os.O_CREATE, 0600); err != nil {
		return nil, fmt.Errorf("open tx file: %v", err)
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		data = append(data, scanner.Text())
	}

	return data, nil
}

func (r *Repository) AppendData(user User, data []string) error {
	txFilePath := filepath.Join(r.Config.Get(Root), orgsFolder, user.Org.Name, usersFolder, user.Key, txFile)
	txFileTempPath := filepath.Join(r.Config.Get(Root), orgsFolder, user.Org.Name, usersFolder, user.Key, txFileTemp)
	var file *os.File

	if _, err := os.Stat(txFilePath); errors.Is(err, fs.ErrNotExist) {
		if file, err = os.OpenFile(txFileTempPath, os.O_RDWR|os.O_CREATE, 0600); err != nil {
			return fmt.Errorf("open tx file: %v", err)
		}
	} else {
		copy(txFilePath, txFileTempPath)
		if file, err = os.OpenFile(txFileTempPath, os.O_RDWR|os.O_APPEND, 0600); err != nil {
			return fmt.Errorf("open tx file: %v", err)
		}
	}
	defer func() {
		file.Close()
	}()

	for _, line := range data {
		if _, err := file.Write([]byte(line)); err != nil {
			return err
		}
	}

	if err := file.Close(); err != nil {
		return err
	}

	if err := os.Rename(txFileTempPath, txFilePath); err != nil {
		return err
	}

	return nil
}

func copy(src, dst string) error {
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
