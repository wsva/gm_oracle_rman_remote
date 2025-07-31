package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	mlib "github.com/wsva/monitor_lib_go"
	ml_detail "github.com/wsva/monitor_lib_go/detail"
)

type BackupRMAN struct {
	StartTime  string
	EndTime    string
	DeviceType string
	Size       string
	Status     string
}

func (b *BackupRMAN) String() string {
	jsonBytes, _ := json.Marshal(b)
	return string(jsonBytes)
}

func main() {
	if err := initGlobals(); err != nil {
		fmt.Println(err)
		return
	}

	wg := &sync.WaitGroup{}
	for _, v := range targetList {
		go checkRMANBackup(v, wg)
		wg.Add(1)
	}
	wg.Wait()

	jsonBytes, _ := json.Marshal(resultsRuntime)
	fmt.Println(mlib.MessageTypeMRList + string(jsonBytes))
}

func checkRMANBackup(t TargetOracle, wg *sync.WaitGroup) {
	defer wg.Done()

	if !t.Enable {
		return
	}

	var md ml_detail.MDCommon
	defer func() {
		jsonString, err := md.JSONString()
		resultsRuntimeLock.Lock()
		if err != nil {
			resultsRuntime = append(resultsRuntime,
				mlib.GetMR(t.Name, t.Address, mlib.MTypeRMANBackup, "", err.Error()))
		} else {
			resultsRuntime = append(resultsRuntime,
				mlib.GetMR(t.Name, t.Address, mlib.MTypeRMANBackup, jsonString, ""))
		}
		resultsRuntimeLock.Unlock()
	}()

	sqltext := `
SELECT TO_CHAR(A.START_TIME, 'yyyy-mm-dd hh24:mi:ss'),
	TO_CHAR(A.END_TIME, 'yyyy-mm-dd hh24:mi:ss'),
	A.OUTPUT_DEVICE_TYPE,
	ROUND(A.OUTPUT_BYTES / 1024 / 1024 / 1024, 2) || 'GB',
	A.STATUS
FROM V$RMAN_BACKUP_JOB_DETAILS A
WHERE A.START_TIME >= SYSDATE - 1`
	rows, err := t.DB.Query(sqltext)
	if err != nil {
		md.Detail = err.Error()
		return
	}
	defer t.DB.Close()

	var result []BackupRMAN
	for rows.Next() {
		var f1, f2, f3, f4, f5 sql.NullString
		err = rows.Scan(&f1, &f2, &f3, &f4, &f5)
		if err != nil {
			md.Detail = err.Error()
			return
		}
		result = append(result, BackupRMAN{
			StartTime:  f1.String,
			EndTime:    f2.String,
			DeviceType: f3.String,
			Size:       f4.String,
			Status:     f5.String,
		})
	}
	err = rows.Close()
	if err != nil {
		md.Detail = err.Error()
		return
	}

	if len(result) == 0 {
		md.Detail = "no backup found"
		return
	}

	for _, v := range result {
		md.InfoList = append(md.InfoList, v.String())
		if v.Status != "COMPLETED" {
			md.Detail += fmt.Sprintf("%v:%v;", v.DeviceType, v.Status)
		}
	}
	if md.Detail == "" {
		md.Detail = "ok"
	}
}
