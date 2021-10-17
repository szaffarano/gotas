package repo

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/szaffarano/gotas/config"
	"github.com/szaffarano/gotas/logger"
	"github.com/szaffarano/gotas/task/auth"
)

const (
	orgsFolder  = "orgs"
	usersFolder = "users"
	txFile      = "tx.data"
	txFileTemp  = "tx.tmp.data"
)

var log *logger.Logger

func init() {
	log = logger.Log()
}

// Repository defines an API with the task server operations, orgs and users
// ABM, initialization, etc.
type Repository struct {
	baseDir string
	orgs    []auth.Organization
}

// NewRepository create a brand new repository in the given dataDir
func NewRepository(dataDir string, defaultConfig map[string]string) (*Repository, error) {
	if fileInfo, err := os.Stat(dataDir); err != nil {
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

	for k, v := range defaultConfig {
		cfg.Set(k, v)
	}

	if err := config.Save(cfg); err != nil {
		return nil, err
	}

	return &Repository{baseDir: dataDir}, nil
}

// OpenRepository loads a repository from file system.
func OpenRepository(dataDir string) (*Repository, error) {

	orgsRoot := filepath.Join(dataDir, orgsFolder)
	var orgsToAdd []string
	err := filepath.WalkDir(orgsRoot, func(_ string, d fs.DirEntry, err error) error {
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

	repo := Repository{baseDir: dataDir}
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

// NewOrg initializes a new Organization creating the underlying file system structure.
func (r *Repository) NewOrg(orgName string) (*auth.Organization, error) {
	for _, org := range r.orgs {
		if org.Name == orgName {
			return nil, fmt.Errorf("organization %q already exists", orgName)
		}
	}

	newOrgPath := filepath.Join(r.baseDir, orgsFolder, orgName)
	if err := os.Mkdir(newOrgPath, 0775); err != nil {
		return nil, fmt.Errorf("creating new org: %v", err)
	}
	if err := os.Mkdir(filepath.Join(newOrgPath, usersFolder), 0775); err != nil {
		return nil, fmt.Errorf("creating users dir under org: %v", err)
	}

	newOrg := auth.Organization{Name: orgName}
	r.orgs = append(r.orgs, newOrg)

	return &newOrg, nil
}

// GetOrg initializes an Organization reading the information from the underlying file system.
func (r *Repository) GetOrg(orgName string) (*auth.Organization, error) {
	var users []auth.User
	root := filepath.Join(r.baseDir, orgsFolder, orgName, usersFolder)

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
				users = append(users, auth.User{
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

	org := auth.Organization{Name: orgName, Users: users}
	for idx := range users {
		users[idx].Org = &org
	}
	return &org, nil
}

// AddUser adds a new userr to the given Organization.
func (r *Repository) AddUser(orgName string, userName string) (*auth.User, error) {
	org, err := r.GetOrg(orgName)
	if err != nil {
		return nil, err
	}

	for _, u := range org.Users {
		if u.Name == userName {
			return nil, fmt.Errorf("user %q already exists", userName)
		}
	}

	key := uuid.New().String()
	userPath := filepath.Join(r.baseDir, orgsFolder, org.Name, usersFolder, key)
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

	return &auth.User{
		Name: userName,
		Key:  key,
		Org:  org,
	}, nil
}

func (r *Repository) String() string {
	return r.baseDir
}
