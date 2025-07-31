package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"

	wl_db "github.com/wsva/lib_go_db"
	mlib "github.com/wsva/monitor_lib_go"

	"github.com/tidwall/pretty"
)

type TargetOracle struct {
	Name    string   `json:"Name"`
	Enable  bool     `json:"Enable"`
	Address string   `json:"Address"`
	DB      wl_db.DB `json:"DB"`
}

// file comment
var (
	MainConfigFile = "gm_oracle_rman_remote_targets.json"
)

const (
	AESKey = "1"
	AESIV  = "2"
)

var targetList []TargetOracle

var resultsRuntime []mlib.MR
var resultsRuntimeLock sync.Mutex

func initGlobals() error {
	basepath, err := os.Executable()
	if err != nil {
		return err
	}
	MainConfigFile = path.Join(filepath.Dir(basepath), MainConfigFile)

	contentBytes, err := ioutil.ReadFile(MainConfigFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(contentBytes, &targetList)
	if err != nil {
		return err
	}

	err = encryptMainConfigFile()
	if err != nil {
		return err
	}

	err = decryptMainConfig()
	if err != nil {
		return err
	}

	return nil
}

func decryptMainConfig() error {
	for k := range targetList {
		err := targetList[k].DB.Decrypt(AESKey, AESIV)
		if err != nil {
			return err
		}
	}
	return nil
}

func encryptMainConfigFile() error {
	newTargetList := make([]TargetOracle, len(targetList))
	copy(newTargetList, targetList)
	needEncrypt := false
	for k := range newTargetList {
		if newTargetList[k].DB.NeedEncrypt() {
			needEncrypt = true
			err := newTargetList[k].DB.Encrypt(AESKey, AESIV)
			if err != nil {
				return err
			}
		}
	}
	if !needEncrypt {
		return nil
	}
	jsonBytes, err := json.Marshal(newTargetList)
	if err != nil {
		return err
	}
	err = os.WriteFile(MainConfigFile, pretty.Pretty(jsonBytes), 0666)
	if err != nil {
		return err
	}
	return nil
}
