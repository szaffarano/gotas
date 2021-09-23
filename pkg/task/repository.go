package task

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
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

// Repository defines an API with the task server operations, orgs and users ABM, initialization, etc.
type Repository struct {
	Config config.Config
}

// NewRepository create a brand new repository in the given dataDir
func NewRepository(dataDir string) (*Repository, error) {
	if fileInfo, err := os.Stat(dataDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s: does not exist", dataDir)
	} else if os.IsPermission(err) {
		return nil, fmt.Errorf("%s: permission denied", dataDir)
	} else if err != nil {
		return nil, errors.Wrap(err, "unknown error")
	} else if !fileInfo.IsDir() {
		return nil, fmt.Errorf("%s: is not a directory", dataDir)
	} else if dataDir, err = filepath.Abs(dataDir); err != nil {
		return nil, errors.Wrap(err, "error while calculate absolute path")
	} else if files, err := ioutil.ReadDir(dataDir); err != nil {
		return nil, errors.Wrap(err, "error reding data dir")
	} else if len(files) > 0 {
		return nil, fmt.Errorf("%s: not empty", dataDir)
	}

	if err := os.Mkdir(filepath.Join(dataDir, "orgs"), 0755); err != nil {
		return nil, errors.Wrap(err, "error creating initial directory tree")
	}

	cfg, err := config.New(filepath.Join(dataDir, "config"))
	if err != nil {
		return nil, errors.Wrap(err, "error creating configuration file")
	}

	// Default values

	cfg.SetBool(Confirmation, true)
	cfg.Set(Log, filepath.Join(os.TempDir(), "taskd.log"))
	cfg.Set(PidFile, filepath.Join(os.TempDir(), "taskd.pid"))
	cfg.SetInt(QueueSize, 10)
	cfg.SetInt(RequestLimit, 1048576)
	cfg.Set(Root, dataDir)
	cfg.Set(Trust, "strict")
	cfg.SetBool(Verbose, true)

	if err := config.Save(cfg); err != nil {
		return nil, errors.Wrap(err, "error saving configuration file")
	}

	return &Repository{Config: cfg}, nil
}

func OpenRepository(conf string) (*Repository, error) {
	return nil, errors.New("not implemented")
}
