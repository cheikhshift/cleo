package main

import (
	"fmt"
	"github.com/cheikhshift/gos/core"
	"io/ioutil"
	"path/filepath"
)

func LaunchApp(cmmand string, test Test, app App) {
	var shscript string

	if !isWindows {
		if !test.NoBuild {
			shscript = fmt.Sprintf(BuildScript, cmmand, filepath.Join(cleoWorkspace, test.ID), filepath.Join(cleoWorkspace, app.ID), serverWaitTime, filepath.Join(cleoWorkspace, test.ID))
		} else {
			shscript = fmt.Sprintf(LaunchScript, cmmand, filepath.Join(cleoWorkspace, test.ID))
		}
		bspath := filepath.Join(cleoWorkspace, fmt.Sprintf("%s.sh", test.ID))
		ioutil.WriteFile(bspath, []byte(shscript), 0777)
		core.RunCmdSmart(fmt.Sprintf("sh %s &>/dev/null", bspath))
	} else {

		if !test.NoBuild {
			shscript = fmt.Sprintf(BatchBuildScript, filepath.Join(cleoWorkspace, test.ID), filepath.Join(cleoWorkspace, app.ID), serverWaitTime,filepath.Join(dfd, "bin") ,cmmand, filepath.Join(cleoWorkspace, test.ID))
		} else {
			shscript = fmt.Sprintf(BatchLaunchScript, filepath.Join(dfd, "bin"),cmmand, filepath.Join(cleoWorkspace, test.ID))
		}
		bspath := filepath.Join(cleoWorkspace, fmt.Sprintf("%s.bat", test.ID))
		ioutil.WriteFile(bspath, []byte(shscript), 0777)
		
		core.RunCmdSmart(fmt.Sprintf("%s", bspath))
	}
}
