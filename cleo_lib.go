package main

import (
	"encoding/json"
	"fmt"
	"github.com/cheikhshift/form"
	"github.com/cheikhshift/gos/core"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var TestCount int

func Save(name string, v interface{}) error {

	str := form.Encrypt(Key, mResponse(v))
	pathoffile := filepath.Join(cleoWorkspace, name)
	strbytes := []byte(str)
	err := ioutil.WriteFile(pathoffile, strbytes, 0700)
	strbytes = nil
	return err
}
func SaveConfig() {
	Save(Path("configs", "default", "000"), Mset)
}

func RmEntry(app App) []App {
	newset := []App{}
	for i := 0; i < len(Mset.Apps); i++ {
		if Mset.Apps[i].ID != app.ID {
			newset = append(newset, Mset.Apps[i])
		}
	}
	return newset
}

func UpdateEntry(app App) {
	for i := 0; i < len(Mset.Apps); i++ {
		if Mset.Apps[i].ID == app.ID {
			Mset.Apps[i] = app
		}
	}
}

func GetApp(id string) App {
	for i := 0; i < len(Mset.Apps); i++ {
		if Mset.Apps[i].ID == id {
			return Mset.Apps[i]
		}
	}

	return App{}
}

func RmTest(test Test) []Test {
	newset := []Test{}
	for i := 0; i < len(Mset.Tests); i++ {
		if Mset.Tests[i].ID != test.ID {
			newset = append(newset, Mset.Tests[i])
		}
	}
	return newset
}

func UpdateTest(test Test) {
	for i := 0; i < len(Mset.Tests); i++ {
		if Mset.Tests[i].ID == test.ID {
			Mset.Tests[i] = test
		}
	}
}

func Path(module, id, name string) string {
	pathoffolder := filepath.Join(cleoWorkspace, module, id)
	err := os.MkdirAll(pathoffolder, 0700)
	if err != nil {
		panic(err)
	}
	pathoffile := filepath.Join(module, id, name)
	return pathoffile
}

func AlertSys(danger bool, text string, Time time.Time) {
	Mset.Alerts = append(Mset.Alerts, Alert{danger, text, Time})
	SaveConfig()
}

func TestFrame(test Test) {

	var killCommand string
	app := GetApp(test.TargetID)

	if !test.NoBuild {
		// attempt to
		log.Println("Attempting to kill any previous processes.")

		if !isWindows {
			killCommand = fmt.Sprintf("killall -3 %s-cleo", test.ID)
		} else {
			killCommand = fmt.Sprintf("taskkill /f /im %s-cleo.exe", test.ID)
		}
		core.RunCmd(killCommand)

		pathtemp := filepath.Join(dfd, "src", app.Path)
		pkgpath := strings.Split(pathtemp, "/")
		pkgpath = append(pkgpath, Utilname)
		if app.FetchOntest {
			core.RunCmd(fmt.Sprintf("go get -u %s", app.Path))
		}
		os.Remove(filepath.Join(cleoWorkspace, fmt.Sprintf("%s-cleo", test.ID)))
		os.Remove(filepath.Join(cleoWorkspace, fmt.Sprintf("%s.test", test.ID)))

		var pathToUtil string

		if !isWindows {
			pathToUtil = fmt.Sprintf("/%s", filepath.Join(pkgpath...))
		} else {
			pathToUtil = fmt.Sprintf("\\%s", filepath.Join(pkgpath...))
		}

		defer os.Remove(pathToUtil)

		err := ioutil.WriteFile(pathToUtil, CleoUtil, 0700)
		if err != nil {
			test.Working = false
			test.Finished = true
			test.End = time.Now()
			AlertSys(true, "Error during setup of application.", time.Now())
			return
		}

		logd, err := core.RunCmdSmart(fmt.Sprintf("go build -o %s-cleo %s", filepath.Join(cleoWorkspace, test.ID), app.Path))

		if err != nil {
			test.Working = false
			test.Finished = true
			test.End = time.Now()
			AlertSys(true, fmt.Sprintf("Error installing web application. Log : %s", logd), time.Now())
			return
		}

	}
	if Mset.Settings.Connections == 0 {
		Mset.Settings.Connections = 100
	}
	if Mset.Settings.Threads == 0 {
		Mset.Settings.Threads = 2
	}

	TestCount++
	var addr, cmmand string

	if !test.NoBuild {
		port := fmt.Sprintf("%v", TestCount+45000)
		err := os.Setenv("PORT", port)
		if app.Envs != nil {
			for i := 0; i < len(app.Envs); i++ {
				os.Setenv(app.Envs[i].Key, app.Envs[i].Value)
			}
		}
		if err != nil {
			test.Working = false
			test.Finished = true
			test.End = time.Now()
			AlertSys(true, "Error setting port", time.Now())
			return
		}
		addr = fmt.Sprintf("%s:%s", HostAddress, port)
		cmmand = fmt.Sprintf(`go-wrk -c=%v -m="%s" -b="%s" -n=%v -H="%s" -t=%v %s%s`, Mset.Settings.Connections, test.Method, test.Data, test.NReqs, test.H, Mset.Settings.Threads, addr, test.Path)
	} else {
		addr = fmt.Sprintf("%s:%s", test.CustomAddress, test.PortNumber)
		cmmand = fmt.Sprintf(`go-wrk -c=%v -m="%s" -b="%s" -n=%v -H="%s" -t=%v %s:%s%s`, Mset.Settings.Connections, test.Method, test.Data, test.NReqs, test.H, Mset.Settings.Threads, addr, test.Path)
	}

	go func() {

		HeapCount := 0
		test.HeapMinute = []HeapFrame{}

		for {

			fi, err := os.Stat(filepath.Join(cleoWorkspace, fmt.Sprintf("%s.test", test.ID)))

			if err == nil {
				// Could not obtain stat, handle error

				if fi.Size() > 100 {

					if !test.NoBuild {
						core.RunCmd(killCommand)
					}

					test.Working = false
					test.Finished = true
					test.End = time.Now()
					TestCount--
					HeapCount--
					AlertSys(false, fmt.Sprintf("Test %s complete.", test.Name), time.Now())
					break
				} else {

					client := &http.Client{Transport: tr}
					resp, err := client.Get(fmt.Sprintf("%s%s", addr, HeapUrl))

					if err == nil {

						body, _ := ioutil.ReadAll(resp.Body)
						fname := filepath.Join(cleoWorkspace, Path("tests", test.ID, fmt.Sprintf("h%v", HeapCount)))

						err = ioutil.WriteFile(fname, body, 0700)
						if err != nil {
							test.Working = false
							test.Finished = true
							AlertSys(true, fmt.Sprintf("%s Error: %s", test.Name, err.Error()), time.Now())
							break
						}

						//if test.CPU {
						body = nil
						client = &http.Client{Transport: tr}
						resp, err = client.Get(fmt.Sprintf("%s%s", addr, ProfileUrl))

						if err != nil {
							test.Working = false
							test.Finished = true
							AlertSys(true, fmt.Sprintf("%s Error: %s", test.Name, err.Error()), time.Now())
							break
						}

						body, _ = ioutil.ReadAll(resp.Body)
						fname = filepath.Join(cleoWorkspace, Path("tests", test.ID, fmt.Sprintf("p%v", HeapCount)))
						err = ioutil.WriteFile(fname, body, 0700)
						if err != nil {
							test.Working = false
							test.Finished = true
							AlertSys(true, fmt.Sprintf("%s Error: %s", test.Name, err.Error()), time.Now())
							break
						}
						//}

						body = nil
						client = &http.Client{Transport: tr}
						resp, err = client.Get(fmt.Sprintf("%s%s%s", addr, HeapUrl, DebugParam))

						if err == nil {

							body, _ = ioutil.ReadAll(resp.Body)
							strbody := string(body)
							parts := strings.Split(strbody, "# runtime.MemStats")
							partsKeys := strings.Split(parts[1], "#")
							body = nil
							hframe := HeapFrame{Time: time.Now()}
							for i := 0; i < len(partsKeys); i++ {
								cl := partsKeys[i]

								if strings.Contains(cl, "HeapInuse") {
									lineseg := strings.Split(cl, "=")
									lineseg[1] = strings.TrimSpace(lineseg[1])
									o, _ := strconv.Atoi(lineseg[1])
									hframe.Iu = o
								} else if strings.Contains(cl, "HeapReleased") {
									lineseg := strings.Split(cl, "=")
									lineseg[1] = strings.TrimSpace(lineseg[1])
									o, _ := strconv.Atoi(lineseg[1])
									hframe.Rl = o
								} else if strings.Contains(cl, "HeapObjects") {
									lineseg := strings.Split(cl, "=")
									lineseg[1] = strings.TrimSpace(lineseg[1])
									o, _ := strconv.Atoi(lineseg[1])
									hframe.Ho = o
								}
							}

							test.HeapMinute = append(test.HeapMinute, hframe)
							body = nil
							fmt.Println("Latest frame :", hframe)
							time.Sleep(time.Second * 2)
							HeapCount++
						}
					}
				}
			}
		}
		UpdateTest(test)
		SaveConfig()
	}()

	LaunchApp(cmmand, test, app)

}
func EscapeRegexp(lookup string) string {
	escaped := strings.Replace(lookup, "(", "\\(", -1)
	escaped = strings.Replace(escaped, ")", "\\)", -1)
	escaped = strings.Replace(escaped, "/", "\\/", -1)
	escaped = strings.Replace(escaped, ".", "\\.", -1)
	escaped = strings.Replace(escaped, "*", "\\*", -1)
	return fmt.Sprintf("^(%s)", escaped)
}

func Load(name string, targ interface{}) error {

	pathoffile := filepath.Join(cleoWorkspace, name)

	data, err := ioutil.ReadFile(pathoffile)
	if err != nil {
		return err
	}
	strdata := string(data)
	s := form.Decrypt(Key, strdata)
	data = nil
	b := []byte(s)
	err = json.Unmarshal(b, targ)
	b = nil
	return err

}
