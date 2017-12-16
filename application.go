package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cheikhshift/db"
	"github.com/cheikhshift/gos/core"
	gosweb "github.com/cheikhshift/gos/web"
	"github.com/elazarl/go-bindata-assetfs"
	"github.com/fatih/color"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"html"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var store = sessions.NewCookieStore([]byte("a very very very very secret key"))

var Prod = true

var TemplateFuncStore template.FuncMap
var templateCache = gosweb.NewTemplateCache()

func StoreNetfn() int {
	TemplateFuncStore = template.FuncMap{"a": gosweb.Netadd, "s": gosweb.Netsubs, "m": gosweb.Netmultiply, "d": gosweb.Netdivided, "js": gosweb.Netimportjs, "css": gosweb.Netimportcss, "sd": gosweb.NetsessionDelete, "sr": gosweb.NetsessionRemove, "sc": gosweb.NetsessionKey, "ss": gosweb.NetsessionSet, "sso": gosweb.NetsessionSetInt, "sgo": gosweb.NetsessionGetInt, "sg": gosweb.NetsessionGet, "form": gosweb.Formval, "eq": gosweb.Equalz, "neq": gosweb.Nequalz, "lte": gosweb.Netlt, "LoadWebAsset": NetLoadWebAsset, "AddApp": NetAddApp, "AddTest": NetAddTest, "Cleo": NetCleo, "DeleteAlerts": NetDeleteAlerts, "GetList": NetGetList, "GetTop": NetGetTop, "GetCard": NetGetCard, "Start": NetStart, "Cancel": NetCancel, "Nuke": NetNuke, "UpdateApp": NetUpdateApp, "UpdateTest": NetUpdateTest, "UpdateSettings": NetUpdateSettings, "DeleteApp": NetDeleteApp, "DeleteTest": NetDeleteTest, "ang": Netang, "bang": Netbang, "cang": Netcang, "server": Netserver, "bserver": Netbserver, "cserver": Netcserver, "jquery": Netjquery, "bjquery": Netbjquery, "cjquery": Netcjquery, "App": NetstructApp, "isApp": NetcastApp, "Setting": NetstructSetting, "isSetting": NetcastSetting, "EnvVar": NetstructEnvVar, "isEnvVar": NetcastEnvVar, "Test": NetstructTest, "isTest": NetcastTest, "HeapFrame": NetstructHeapFrame, "isHeapFrame": NetcastHeapFrame, "CleoSet": NetstructCleoSet, "isCleoSet": NetcastCleoSet, "Alert": NetstructAlert, "isAlert": NetcastAlert, "TopDist": NetstructTopDist, "isTopDist": NetcastTopDist}
	return 0
}

var FuncStored = StoreNetfn()

type dbflf db.O

func renderTemplate(w http.ResponseWriter, p *gosweb.Page) {
	defer func() {
		if n := recover(); n != nil {
			color.Red(fmt.Sprintf("Error loading template in path : web%s.tmpl reason : %s", p.R.URL.Path, n))

			DebugTemplate(w, p.R, fmt.Sprintf("web%s", p.R.URL.Path))
			w.WriteHeader(http.StatusInternalServerError)

			pag, err := loadPage("/your-500-page")

			if err != nil {
				log.Println(err.Error())
				return
			}

			if pag.IsResource {
				w.Write(pag.Body)
			} else {
				pag.R = p.R
				pag.Session = p.Session
				renderTemplate(w, pag) ///your-500-page"

			}
		}
	}()

	// TemplateFuncStore

	if _, ok := templateCache.Get(p.R.URL.Path); !ok && Prod {
		var tmpstr = string(p.Body)
		var localtemplate = template.New(p.R.URL.Path)

		localtemplate.Funcs(TemplateFuncStore)
		localtemplate.Parse(tmpstr)
		templateCache.Put(p.R.URL.Path, localtemplate)
	}

	outp := new(bytes.Buffer)
	err := templateCache.JGet(p.R.URL.Path).Execute(outp, p)
	if err != nil {
		log.Println(err.Error())
		DebugTemplate(w, p.R, fmt.Sprintf("web%s", p.R.URL.Path))
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "text/html")
		pag, err := loadPage("/your-500-page")

		if err != nil {
			log.Println(err.Error())
			return
		}
		pag.R = p.R
		pag.Session = p.Session

		if pag.IsResource {
			w.Write(pag.Body)
		} else {
			renderTemplate(w, pag) // "/your-500-page"

		}
		return
	}

	// p.Session.Save(p.R, w)

	var outps = outp.String()
	var outpescaped = html.UnescapeString(outps)
	outp = nil
	fmt.Fprintf(w, outpescaped)

}

// Access you .gxml's end tags with
// this http.HandlerFunc.
// Use MakeHandler(http.HandlerFunc) to serve your web
// directory from memory.
func MakeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if attmpt := apiAttempt(w, r); !attmpt {
			fn(w, r)
		}
		context.Clear(r)

	}
}

func mResponse(v interface{}) string {
	data, _ := json.Marshal(&v)
	return string(data)
}
func apiAttempt(w http.ResponseWriter, r *http.Request) (callmet bool) {
	var response string
	response = ""
	var session *sessions.Session
	var er error
	if session, er = store.Get(r, "session-"); er != nil {
		session, _ = store.New(r, "session-")
	}

	if r.Method == "RESET" {
		return true
	}

	if callmet {
		session.Save(r, w)
		session = nil
		if response != "" {
			//Unmarshal json
			//w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(response))
		}
		return
	}
	session = nil
	return
}
func SetField(obj interface{}, name string, value interface{}) error {
	structValue := reflect.ValueOf(obj).Elem()
	structFieldValue := structValue.FieldByName(name)

	if !structFieldValue.IsValid() {
		return fmt.Errorf("No such field: %s in obj", name)
	}

	if !structFieldValue.CanSet() {
		return fmt.Errorf("Cannot set %s field value", name)
	}

	structFieldType := structFieldValue.Type()
	val := reflect.ValueOf(value)
	if structFieldType != val.Type() {
		invalidTypeError := errors.New("Provided value type didn't match obj field type")
		return invalidTypeError
	}

	structFieldValue.Set(val)
	return nil
}
func DebugTemplate(w http.ResponseWriter, r *http.Request, tmpl string) {
	lastline := 0
	linestring := ""
	defer func() {
		if n := recover(); n != nil {
			log.Println()
			// log.Println(n)
			log.Println("Error on line :", lastline+1, ":"+strings.TrimSpace(linestring))
			//http.Redirect(w,r,"/your-500-page",307)
		}
	}()

	p, err := loadPage(r.URL.Path)
	filename := tmpl + ".tmpl"
	body, err := Asset(filename)
	session, er := store.Get(r, "session-")

	if er != nil {
		session, er = store.New(r, "session-")
	}
	p.Session = session
	p.R = r
	if err != nil {
		log.Print(err)

	} else {

		lines := strings.Split(string(body), "\n")
		// log.Println( lines )
		linebuffer := ""
		waitend := false
		open := 0
		for i, line := range lines {

			processd := false

			if strings.Contains(line, "{{with") || strings.Contains(line, "{{ with") || strings.Contains(line, "with}}") || strings.Contains(line, "with }}") || strings.Contains(line, "{{range") || strings.Contains(line, "{{ range") || strings.Contains(line, "range }}") || strings.Contains(line, "range}}") || strings.Contains(line, "{{if") || strings.Contains(line, "{{ if") || strings.Contains(line, "if }}") || strings.Contains(line, "if}}") || strings.Contains(line, "{{block") || strings.Contains(line, "{{ block") || strings.Contains(line, "block }}") || strings.Contains(line, "block}}") {
				linebuffer += line
				waitend = true

				endstr := ""
				processd = true
				if !(strings.Contains(line, "{{end") || strings.Contains(line, "{{ end") || strings.Contains(line, "end}}") || strings.Contains(line, "end }}")) {

					open++

				}
				for i := 0; i < open; i++ {
					endstr += "\n{{end}}"
				}
				//exec
				outp := new(bytes.Buffer)
				t := template.New("PageWrapper")
				t = t.Funcs(TemplateFuncStore)
				t, _ = t.Parse(string(body))
				lastline = i
				linestring = line
				erro := t.Execute(outp, p)
				if erro != nil {
					log.Println("Error on line :", i+1, line, erro.Error())
				}
			}

			if waitend && !processd && !(strings.Contains(line, "{{end") || strings.Contains(line, "{{ end")) {
				linebuffer += line

				endstr := ""
				for i := 0; i < open; i++ {
					endstr += "\n{{end}}"
				}
				//exec
				outp := new(bytes.Buffer)
				t := template.New("PageWrapper")
				t = t.Funcs(TemplateFuncStore)
				t, _ = t.Parse(string(body))
				lastline = i
				linestring = line
				erro := t.Execute(outp, p)
				if erro != nil {
					log.Println("Error on line :", i+1, line, erro.Error())
				}

			}

			if !waitend && !processd {
				outp := new(bytes.Buffer)
				t := template.New("PageWrapper")
				t = t.Funcs(TemplateFuncStore)
				t, _ = t.Parse(string(body))
				lastline = i
				linestring = line
				erro := t.Execute(outp, p)
				if erro != nil {
					log.Println("Error on line :", i+1, line, erro.Error())
				}
			}

			if !processd && (strings.Contains(line, "{{end") || strings.Contains(line, "{{ end")) {
				open--

				if open == 0 {
					waitend = false

				}
			}
		}

	}

}

func DebugTemplatePath(tmpl string, intrf interface{}) {
	lastline := 0
	linestring := ""
	defer func() {
		if n := recover(); n != nil {

			log.Println("Error on line :", lastline+1, ":"+strings.TrimSpace(linestring))
			log.Println(n)
			//http.Redirect(w,r,"/your-500-page",307)
		}
	}()

	filename := tmpl
	body, err := Asset(filename)

	if err != nil {
		log.Print(err)

	} else {

		lines := strings.Split(string(body), "\n")
		// log.Println( lines )
		linebuffer := ""
		waitend := false
		open := 0
		for i, line := range lines {

			processd := false

			if strings.Contains(line, "{{with") || strings.Contains(line, "{{ with") || strings.Contains(line, "with}}") || strings.Contains(line, "with }}") || strings.Contains(line, "{{range") || strings.Contains(line, "{{ range") || strings.Contains(line, "range }}") || strings.Contains(line, "range}}") || strings.Contains(line, "{{if") || strings.Contains(line, "{{ if") || strings.Contains(line, "if }}") || strings.Contains(line, "if}}") || strings.Contains(line, "{{block") || strings.Contains(line, "{{ block") || strings.Contains(line, "block }}") || strings.Contains(line, "block}}") {
				linebuffer += line
				waitend = true

				endstr := ""
				if !(strings.Contains(line, "{{end") || strings.Contains(line, "{{ end") || strings.Contains(line, "end}}") || strings.Contains(line, "end }}")) {

					open++

				}

				for i := 0; i < open; i++ {
					endstr += "\n{{end}}"
				}
				//exec

				processd = true
				outp := new(bytes.Buffer)
				t := template.New("PageWrapper")
				t = t.Funcs(TemplateFuncStore)
				t, _ = t.Parse(string([]byte(fmt.Sprintf("%s%s", linebuffer, endstr))))
				lastline = i
				linestring = line
				erro := t.Execute(outp, intrf)
				if erro != nil {
					log.Println("Error on line :", i+1, line, erro.Error())
				}
			}

			if waitend && !processd && !(strings.Contains(line, "{{end") || strings.Contains(line, "{{ end") || strings.Contains(line, "end}}") || strings.Contains(line, "end }}")) {
				linebuffer += line

				endstr := ""
				for i := 0; i < open; i++ {
					endstr += "\n{{end}}"
				}
				//exec
				outp := new(bytes.Buffer)
				t := template.New("PageWrapper")
				t = t.Funcs(TemplateFuncStore)
				t, _ = t.Parse(string([]byte(fmt.Sprintf("%s%s", linebuffer, endstr))))
				lastline = i
				linestring = line
				erro := t.Execute(outp, intrf)
				if erro != nil {
					log.Println("Error on line :", i+1, line, erro.Error())
				}

			}

			if !waitend && !processd {
				outp := new(bytes.Buffer)
				t := template.New("PageWrapper")
				t = t.Funcs(TemplateFuncStore)
				t, _ = t.Parse(string([]byte(fmt.Sprintf("%s%s", linebuffer))))
				lastline = i
				linestring = line
				erro := t.Execute(outp, intrf)
				if erro != nil {
					log.Println("Error on line :", i+1, line, erro.Error())
				}
			}

			if !processd && (strings.Contains(line, "{{end") || strings.Contains(line, "{{ end") || strings.Contains(line, "end}}") || strings.Contains(line, "end }}")) {
				open--

				if open == 0 {
					waitend = false

				}
			}
		}

	}

}
func Handler(w http.ResponseWriter, r *http.Request) {
	var p *gosweb.Page
	p, err := loadPage(r.URL.Path)
	var session *sessions.Session
	var er error
	if session, er = store.Get(r, "session-"); er != nil {
		session, _ = store.New(r, "session-")
	}

	if err != nil {
		log.Println(err.Error())

		w.WriteHeader(http.StatusNotFound)

		pag, err := loadPage("/your-404-page")

		if err != nil {
			log.Println(err.Error())
			//context.Clear(r)
			return
		}
		pag.R = r
		pag.Session = session
		if p != nil {
			p.Session = nil
			p.Body = nil
			p.R = nil
			p = nil
		}

		if pag.IsResource {
			w.Write(pag.Body)
		} else {
			renderTemplate(w, pag) //"/your-500-page"
		}
		session = nil
		context.Clear(r)
		return
	}

	if !p.IsResource {
		w.Header().Set("Content-Type", "text/html")
		p.Session = session
		p.R = r
		renderTemplate(w, p) //fmt.Sprintf("web%s", r.URL.Path)
		session.Save(r, w)
		// log.Println(w)
	} else {
		w.Header().Set("Cache-Control", "public")
		if strings.Contains(r.URL.Path, ".css") {
			w.Header().Add("Content-Type", "text/css")
		} else if strings.Contains(r.URL.Path, ".js") {
			w.Header().Add("Content-Type", "application/javascript")
		} else {
			w.Header().Add("Content-Type", http.DetectContentType(p.Body))
		}

		w.Write(p.Body)
	}

	p.Session = nil
	p.Body = nil
	p.R = nil
	p = nil
	session = nil
	context.Clear(r)
	return
}

var WebCache = gosweb.NewCache()

func loadPage(title string) (*gosweb.Page, error) {

	if lPage, ok := WebCache.Get(title); ok {
		return &lPage, nil
	}

	var nPage = gosweb.Page{}
	if roottitle := (title == "/"); roottitle {
		webbase := "web/"
		fname := fmt.Sprintf("%s%s", webbase, "index.html")
		body, err := Asset(fname)
		if err != nil {
			fname = fmt.Sprintf("%s%s", webbase, "index.tmpl")
			body, err = Asset(fname)
			if err != nil {
				return nil, err
			}
			nPage.Body = body
			WebCache.Put(title, nPage)
			body = nil
			return &nPage, nil
		}
		nPage.Body = body
		nPage.IsResource = true
		WebCache.Put(title, nPage)
		body = nil
		return &nPage, nil

	}

	filename := fmt.Sprintf("web%s.tmpl", title)

	if body, err := Asset(filename); err != nil {
		filename = fmt.Sprintf("web%s.html", title)

		if body, err = Asset(filename); err != nil {
			filename = fmt.Sprintf("web%s", title)

			if body, err = Asset(filename); err != nil {
				return nil, err
			} else {
				if strings.Contains(title, ".tmpl") {
					return nil, nil
				}
				nPage.Body = body
				nPage.IsResource = true
				WebCache.Put(title, nPage)
				body = nil
				return &nPage, nil
			}
		} else {
			nPage.Body = body
			nPage.IsResource = true
			WebCache.Put(title, nPage)
			body = nil
			return &nPage, nil
		}
	} else {
		nPage.Body = body
		WebCache.Put(title, nPage)
		body = nil
		return &nPage, nil
	}

}

var Mset *CleoSet

func init() {

}

type App struct {
	Name, Path, ID string
	FetchOntest    bool
	Envs           []EnvVar
}

func NetcastApp(args ...interface{}) *App {

	s := App{}
	mapp := args[0].(db.O)
	if _, ok := mapp["_id"]; ok {
		mapp["Id"] = mapp["_id"]
	}
	data, _ := json.Marshal(&mapp)

	err := json.Unmarshal(data, &s)
	if err != nil {
		log.Println(err.Error())
	}

	return &s
}
func NetstructApp() *App { return &App{} }

type Setting struct {
	SMTPEmail, SMTPHost, SMTPPass, Emails string
	EmailOnAlert                          bool
	SMTPPort, Threads, Connections        int
}

func NetcastSetting(args ...interface{}) *Setting {

	s := Setting{}
	mapp := args[0].(db.O)
	if _, ok := mapp["_id"]; ok {
		mapp["Id"] = mapp["_id"]
	}
	data, _ := json.Marshal(&mapp)

	err := json.Unmarshal(data, &s)
	if err != nil {
		log.Println(err.Error())
	}

	return &s
}
func NetstructSetting() *Setting { return &Setting{} }

type EnvVar struct {
	Key, Value string
}

func NetcastEnvVar(args ...interface{}) *EnvVar {

	s := EnvVar{}
	mapp := args[0].(db.O)
	if _, ok := mapp["_id"]; ok {
		mapp["Id"] = mapp["_id"]
	}
	data, _ := json.Marshal(&mapp)

	err := json.Unmarshal(data, &s)
	if err != nil {
		log.Println(err.Error())
	}

	return &s
}
func NetstructEnvVar() *EnvVar { return &EnvVar{} }

type Test struct {
	ID, TargetID                string
	Name, Data, Path, Method, H string
	NReqs                       int
	Finished, Working           bool
	Duration                    float64
	HeapMinute                  []HeapFrame
	Start, End                  time.Time
}

func NetcastTest(args ...interface{}) *Test {

	s := Test{}
	mapp := args[0].(db.O)
	if _, ok := mapp["_id"]; ok {
		mapp["Id"] = mapp["_id"]
	}
	data, _ := json.Marshal(&mapp)

	err := json.Unmarshal(data, &s)
	if err != nil {
		log.Println(err.Error())
	}

	return &s
}
func NetstructTest() *Test { return &Test{} }

type HeapFrame struct {
	Iu, Rl, Ho int
	Time       time.Time
}

func NetcastHeapFrame(args ...interface{}) *HeapFrame {

	s := HeapFrame{}
	mapp := args[0].(db.O)
	if _, ok := mapp["_id"]; ok {
		mapp["Id"] = mapp["_id"]
	}
	data, _ := json.Marshal(&mapp)

	err := json.Unmarshal(data, &s)
	if err != nil {
		log.Println(err.Error())
	}

	return &s
}
func NetstructHeapFrame() *HeapFrame { return &HeapFrame{} }

type CleoSet struct {
	Apps     []App
	Settings Setting
	Tests    []Test
	Alerts   []Alert
}

func NetcastCleoSet(args ...interface{}) *CleoSet {

	s := CleoSet{}
	mapp := args[0].(db.O)
	if _, ok := mapp["_id"]; ok {
		mapp["Id"] = mapp["_id"]
	}
	data, _ := json.Marshal(&mapp)

	err := json.Unmarshal(data, &s)
	if err != nil {
		log.Println(err.Error())
	}

	return &s
}
func NetstructCleoSet() *CleoSet { return &CleoSet{} }

type Alert struct {
	Danger  bool
	Message string
	Time    time.Time
}

func NetcastAlert(args ...interface{}) *Alert {

	s := Alert{}
	mapp := args[0].(db.O)
	if _, ok := mapp["_id"]; ok {
		mapp["Id"] = mapp["_id"]
	}
	data, _ := json.Marshal(&mapp)

	err := json.Unmarshal(data, &s)
	if err != nil {
		log.Println(err.Error())
	}

	return &s
}
func NetstructAlert() *Alert { return &Alert{} }

type TopDist struct {
	Name    string
	Percent float64
}

func NetcastTopDist(args ...interface{}) *TopDist {

	s := TopDist{}
	mapp := args[0].(db.O)
	if _, ok := mapp["_id"]; ok {
		mapp["Id"] = mapp["_id"]
	}
	data, _ := json.Marshal(&mapp)

	err := json.Unmarshal(data, &s)
	if err != nil {
		log.Println(err.Error())
	}

	return &s
}
func NetstructTopDist() *TopDist { return &TopDist{} }

//
func NetLoadWebAsset(args ...interface{}) string {

	data, err := Asset(fmt.Sprintf("web%s", args[0].(string)))
	if err != nil {
		return err.Error()
	}
	return string(data)

}

//
func NetAddApp(app App) (done bool) {

	app.ID = core.NewLen(10)
	Mset.Apps = append(Mset.Apps, app)
	SaveConfig()
	done = true
	return

}

//
func NetAddTest(test Test) (done bool) {

	test.ID = core.NewLen(10)
	Mset.Tests = append(Mset.Tests, test)
	SaveConfig()
	done = true
	return

}

//
func NetCleo() (cleo *CleoSet) {

	cleo = Mset
	return

}

//
func NetDeleteAlerts() (done bool) {

	Mset.Alerts = []Alert{}
	SaveConfig()
	return

}

//
func NetGetList(test Test, lookup string) (list string) {

	for cnt, _ := range test.HeapMinute {
		cmd := fmt.Sprintf("go tool pprof --list=%s %s", lookup, filepath.Join(cleoWorkspace, Path("tests", test.ID, fmt.Sprintf("h%v", cnt))))
		logfull, _ := core.RunCmdSmart(cmd)

		retset := strings.Split(logfull, "\n")

		if len(retset) > 2 {
			list = logfull
			break
		}
	}

	return

}

//
func NetGetTop(test Test) (top []TopDist) {

	valm := make(map[string]float64)
	for cnt, _ := range test.HeapMinute {
		logfull, _ := core.RunCmdSmart(fmt.Sprintf("go tool pprof -top %s", filepath.Join(cleoWorkspace, Path("tests", test.ID, fmt.Sprintf("h%v", cnt)))))
		retset := strings.Split(logfull, "\n")
		retset = retset[4:]

		for _, str := range retset {
			strfm := strings.Replace(strings.TrimSpace(str), "   ", " ", -1)
			strfm = strings.Replace(strfm, "  ", " ", -1)

			subset := strings.Split(strfm, " ")

			if len(subset) > 5 {

				subsettwo := strings.Split(subset[len(subset)-1], "   ")

				if strings.Contains(strfm, " (inline)") {
					subsettwo = append([]string{subset[len(subset)-3]}, subsettwo...)
				} else if len(subsettwo) == 1 {
					subsettwo = append([]string{subset[len(subset)-2]}, subsettwo...)
				}
				//fmt.Println(subsettwo)
				_, exts := valm[subsettwo[0]]
				if !exts {
					valm[subsettwo[0]] = 0
				}
				f, _ := strconv.ParseFloat(strings.Replace(subset[1], "%", "", -1), 64)

				valm[subsettwo[0]] += f
			}
		}
	}

	tperc := 0.0
	for key, val := range valm {
		perc := (val / float64(len(test.HeapMinute)))
		top = append(top, TopDist{Name: key, Percent: perc})
		tperc += perc
	}

	tperc = 100.0 - tperc
	top = append(top, TopDist{Name: "Other samples", Percent: tperc})
	valm = nil

	return

}

//
func NetGetCard(test Test) (res string) {

	bc, err := ioutil.ReadFile(filepath.Join(cleoWorkspace, fmt.Sprintf("%s.test", test.ID)))
	if err != nil {
		res = err.Error()
		return
	}
	res = string(bc)
	return

}

//
func NetStart(test Test) (done bool) {

	test.Working = true
	test.Start = time.Now()
	UpdateTest(test)
	SaveConfig()
	runtime.GOMAXPROCS(runtime.NumCPU() + Mset.Settings.Threads)
	go TestFrame(test)
	done = true
	return

}

//
func NetCancel(test Test) (done bool) {

	test.Working = false
	test.End = time.Now()
	UpdateTest(test)
	SaveConfig()

	done = true
	return

}

//
func NetNuke() (done bool) {

	err := os.RemoveAll(cleoWorkspace)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(cleoWorkspace, 0700)
	if err != nil {
		panic(err)
	}
	done = true
	Mset = &CleoSet{}
	SaveConfig()
	return

}

//
func NetUpdateApp(app App) (done bool) {

	UpdateEntry(app)
	SaveConfig()
	done = true
	return

}

//
func NetUpdateTest(test Test) (done bool) {

	UpdateTest(test)
	SaveConfig()
	done = true
	return

}

//
func NetUpdateSettings(settings Setting) (done bool) {

	Mset.Settings = settings
	SaveConfig()
	done = true
	return

}

//
func NetDeleteApp(app App) (done bool) {

	newset := RmEntry(app)
	Mset.Apps = newset
	SaveConfig()
	done = true
	return

}

//
func NetDeleteTest(test Test) (done bool) {

	newset := RmTest(test)
	Mset.Tests = newset
	SaveConfig()
	done = true
	return

}

func templateFNang(localid string, d interface{}) {
	if n := recover(); n != nil {
		color.Red(fmt.Sprintf("Error loading template in path (momentum/ang) : %s", localid))
		// log.Println(n)
		DebugTemplatePath(localid, d)
	}
}

var templateIDang = "tmpl/momentum/ang.tmpl"

func Netang(args ...interface{}) string {

	localid := templateIDang
	var d *gosweb.NoStruct
	defer templateFNang(localid, d)
	if len(args) > 0 {
		jso := args[0].(string)
		var jsonBlob = []byte(jso)
		err := json.Unmarshal(jsonBlob, d)
		if err != nil {
			return err.Error()
		}
	} else {
		d = &gosweb.NoStruct{}
	}

	output := new(bytes.Buffer)

	if _, ok := templateCache.Get(localid); !ok {

		body, er := Asset(localid)
		if er != nil {
			return ""
		}
		var localtemplate = template.New("ang")
		localtemplate.Funcs(TemplateFuncStore)
		var tmpstr = string(body)
		localtemplate.Parse(tmpstr)
		body = nil
		templateCache.Put(localid, localtemplate)
	}

	erro := templateCache.JGet(localid).Execute(output, d)
	if erro != nil {
		color.Red(fmt.Sprintf("Error processing template %s", localid))
		DebugTemplatePath(localid, d)
	}
	var outps = output.String()
	var outpescaped = html.UnescapeString(outps)
	d = nil
	output.Reset()
	output = nil
	args = nil
	return outpescaped

}
func bang(d gosweb.NoStruct) string {
	return Netbang(d)
}

//
func Netbang(d gosweb.NoStruct) string {
	localid := templateIDang
	defer templateFNang(localid, d)
	output := new(bytes.Buffer)

	if _, ok := templateCache.Get(localid); !ok {

		body, er := Asset(localid)
		if er != nil {
			return ""
		}
		var localtemplate = template.New("ang")
		localtemplate.Funcs(TemplateFuncStore)
		var tmpstr = string(body)
		localtemplate.Parse(tmpstr)
		body = nil
		templateCache.Put(localid, localtemplate)
	}

	erro := templateCache.JGet(localid).Execute(output, d)
	if erro != nil {
		log.Println(erro)
	}
	var outps = output.String()
	var outpescaped = html.UnescapeString(outps)
	d = gosweb.NoStruct{}
	output.Reset()
	output = nil
	return outpescaped
}
func Netcang(args ...interface{}) (d gosweb.NoStruct) {
	if len(args) > 0 {
		var jsonBlob = []byte(args[0].(string))
		err := json.Unmarshal(jsonBlob, &d)
		if err != nil {
			log.Println("error:", err)
			return
		}
	} else {
		d = gosweb.NoStruct{}
	}
	return
}

func cang(args ...interface{}) (d gosweb.NoStruct) {
	if len(args) > 0 {
		d = Netcang(args[0])
	} else {
		d = Netcang()
	}
	return
}

func templateFNserver(localid string, d interface{}) {
	if n := recover(); n != nil {
		color.Red(fmt.Sprintf("Error loading template in path (momentum/server) : %s", localid))
		// log.Println(n)
		DebugTemplatePath(localid, d)
	}
}

var templateIDserver = "tmpl/momentum/server.tmpl"

func Netserver(args ...interface{}) string {

	localid := templateIDserver
	var d *gosweb.NoStruct
	defer templateFNserver(localid, d)
	if len(args) > 0 {
		jso := args[0].(string)
		var jsonBlob = []byte(jso)
		err := json.Unmarshal(jsonBlob, d)
		if err != nil {
			return err.Error()
		}
	} else {
		d = &gosweb.NoStruct{}
	}

	output := new(bytes.Buffer)

	if _, ok := templateCache.Get(localid); !ok {

		body, er := Asset(localid)
		if er != nil {
			return ""
		}
		var localtemplate = template.New("server")
		localtemplate.Funcs(TemplateFuncStore)
		var tmpstr = string(body)
		localtemplate.Parse(tmpstr)
		body = nil
		templateCache.Put(localid, localtemplate)
	}

	erro := templateCache.JGet(localid).Execute(output, d)
	if erro != nil {
		color.Red(fmt.Sprintf("Error processing template %s", localid))
		DebugTemplatePath(localid, d)
	}
	var outps = output.String()
	var outpescaped = html.UnescapeString(outps)
	d = nil
	output.Reset()
	output = nil
	args = nil
	return outpescaped

}
func bserver(d gosweb.NoStruct) string {
	return Netbserver(d)
}

//
func Netbserver(d gosweb.NoStruct) string {
	localid := templateIDserver
	defer templateFNserver(localid, d)
	output := new(bytes.Buffer)

	if _, ok := templateCache.Get(localid); !ok {

		body, er := Asset(localid)
		if er != nil {
			return ""
		}
		var localtemplate = template.New("server")
		localtemplate.Funcs(TemplateFuncStore)
		var tmpstr = string(body)
		localtemplate.Parse(tmpstr)
		body = nil
		templateCache.Put(localid, localtemplate)
	}

	erro := templateCache.JGet(localid).Execute(output, d)
	if erro != nil {
		log.Println(erro)
	}
	var outps = output.String()
	var outpescaped = html.UnescapeString(outps)
	d = gosweb.NoStruct{}
	output.Reset()
	output = nil
	return outpescaped
}
func Netcserver(args ...interface{}) (d gosweb.NoStruct) {
	if len(args) > 0 {
		var jsonBlob = []byte(args[0].(string))
		err := json.Unmarshal(jsonBlob, &d)
		if err != nil {
			log.Println("error:", err)
			return
		}
	} else {
		d = gosweb.NoStruct{}
	}
	return
}

func cserver(args ...interface{}) (d gosweb.NoStruct) {
	if len(args) > 0 {
		d = Netcserver(args[0])
	} else {
		d = Netcserver()
	}
	return
}

func templateFNjquery(localid string, d interface{}) {
	if n := recover(); n != nil {
		color.Red(fmt.Sprintf("Error loading template in path (momentum/jquery) : %s", localid))
		// log.Println(n)
		DebugTemplatePath(localid, d)
	}
}

var templateIDjquery = "tmpl/momentum/jquery.tmpl"

func Netjquery(args ...interface{}) string {

	localid := templateIDjquery
	var d *gosweb.NoStruct
	defer templateFNjquery(localid, d)
	if len(args) > 0 {
		jso := args[0].(string)
		var jsonBlob = []byte(jso)
		err := json.Unmarshal(jsonBlob, d)
		if err != nil {
			return err.Error()
		}
	} else {
		d = &gosweb.NoStruct{}
	}

	output := new(bytes.Buffer)

	if _, ok := templateCache.Get(localid); !ok {

		body, er := Asset(localid)
		if er != nil {
			return ""
		}
		var localtemplate = template.New("jquery")
		localtemplate.Funcs(TemplateFuncStore)
		var tmpstr = string(body)
		localtemplate.Parse(tmpstr)
		body = nil
		templateCache.Put(localid, localtemplate)
	}

	erro := templateCache.JGet(localid).Execute(output, d)
	if erro != nil {
		color.Red(fmt.Sprintf("Error processing template %s", localid))
		DebugTemplatePath(localid, d)
	}
	var outps = output.String()
	var outpescaped = html.UnescapeString(outps)
	d = nil
	output.Reset()
	output = nil
	args = nil
	return outpescaped

}
func bjquery(d gosweb.NoStruct) string {
	return Netbjquery(d)
}

//
func Netbjquery(d gosweb.NoStruct) string {
	localid := templateIDjquery
	defer templateFNjquery(localid, d)
	output := new(bytes.Buffer)

	if _, ok := templateCache.Get(localid); !ok {

		body, er := Asset(localid)
		if er != nil {
			return ""
		}
		var localtemplate = template.New("jquery")
		localtemplate.Funcs(TemplateFuncStore)
		var tmpstr = string(body)
		localtemplate.Parse(tmpstr)
		body = nil
		templateCache.Put(localid, localtemplate)
	}

	erro := templateCache.JGet(localid).Execute(output, d)
	if erro != nil {
		log.Println(erro)
	}
	var outps = output.String()
	var outpescaped = html.UnescapeString(outps)
	d = gosweb.NoStruct{}
	output.Reset()
	output = nil
	return outpescaped
}
func Netcjquery(args ...interface{}) (d gosweb.NoStruct) {
	if len(args) > 0 {
		var jsonBlob = []byte(args[0].(string))
		err := json.Unmarshal(jsonBlob, &d)
		if err != nil {
			log.Println("error:", err)
			return
		}
	} else {
		d = gosweb.NoStruct{}
	}
	return
}

func cjquery(args ...interface{}) (d gosweb.NoStruct) {
	if len(args) > 0 {
		d = Netcjquery(args[0])
	} else {
		d = Netcjquery()
	}
	return
}

func dummy_timer() {
	dg := time.Second * 5
	log.Println(dg)
}
func main() {
	fmt.Fprintf(os.Stdout, "%v\n", os.Getpid())

	//psss go code here : func main()
	Windows := strings.Contains(runtime.GOOS, "windows")
	if Windows {
		os.Chdir(os.ExpandEnv("$USERPROFILE"))
	} else {
		os.Chdir(os.ExpandEnv("$HOME"))
	}

	if _, err := os.Stat(cleoWorkspace); os.IsNotExist(err) {
		err = os.MkdirAll(cleoWorkspace, 0700)
		if err != nil {
			panic(err)
		}
		Mset = &CleoSet{}
		SaveConfig()
	} else {
		err = Load(Path("configs", "default", "000"), &Mset)
		if err != nil {
			panic(err)
		}

	}

	//TestFrame(Mset.Tests[0])

	if Prod {
		if !Windows {
			if isMac := strings.Contains(runtime.GOOS, "arwin"); isMac {
				core.RunCmd("open http://localhost:9000/index")
			} else {
				core.RunCmd("xdg-open http://localhost:9000/index")
			}
		} else {
			core.RunCmd("cmd /C start http://localhost:9000/index")
		}
	}

	//psss go code here : func main()
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   true,
		Domain:   "",
	}

	port := ":9000"
	if envport := os.ExpandEnv("$PORT"); envport != "" {
		port = fmt.Sprintf(":%s", envport)
	}
	log.Printf("Listenning on Port %v\n", port)
	http.HandleFunc("/", MakeHandler(Handler))

	http.HandleFunc("/momentum/templates", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.FormValue("name") == "reset" || r.Method == "OPTIONS" {
			return
		} else if r.FormValue("name") == "ang" {
			w.Header().Set("Content-Type", "text/html")
			tmplRendered := Netang(r.FormValue("payload"))
			w.Write([]byte(tmplRendered))
		} else if r.FormValue("name") == "server" {
			w.Header().Set("Content-Type", "text/html")
			tmplRendered := Netserver(r.FormValue("payload"))
			w.Write([]byte(tmplRendered))
		} else if r.FormValue("name") == "jquery" {
			w.Header().Set("Content-Type", "text/html")
			tmplRendered := Netjquery(r.FormValue("payload"))
			w.Write([]byte(tmplRendered))

		}
	})

	http.HandleFunc("/funcfactory.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		w.Write([]byte(`function ang(dataOfInterface, cb){ jsrequestmomentum("/momentum/templates", {name: "ang", payload: JSON.stringify(dataOfInterface)},"POST",  cb) }
function server(dataOfInterface, cb){ jsrequestmomentum("/momentum/templates", {name: "server", payload: JSON.stringify(dataOfInterface)},"POST",  cb) }
function jquery(dataOfInterface, cb){ jsrequestmomentum("/momentum/templates", {name: "jquery", payload: JSON.stringify(dataOfInterface)},"POST",  cb) }
function AddApp(App , cb){
	var t = {}
	
	t.App = App
	jsrequestmomentum("/momentum/funcs?name=AddApp", t, "POSTJSON", cb)
}
function AddTest(Test , cb){
	var t = {}
	
	t.Test = Test
	jsrequestmomentum("/momentum/funcs?name=AddTest", t, "POSTJSON", cb)
}
function Cleo(  cb){
	var t = {}
	
	jsrequestmomentum("/momentum/funcs?name=Cleo", t, "POSTJSON", cb)
}
function DeleteAlerts(  cb){
	var t = {}
	
	jsrequestmomentum("/momentum/funcs?name=DeleteAlerts", t, "POSTJSON", cb)
}
function GetList(Test,Lookup , cb){
	var t = {}
	
	t.Test = Test
	t.Lookup = Lookup
	jsrequestmomentum("/momentum/funcs?name=GetList", t, "POSTJSON", cb)
}
function GetTop(Test , cb){
	var t = {}
	
	t.Test = Test
	jsrequestmomentum("/momentum/funcs?name=GetTop", t, "POSTJSON", cb)
}
function GetCard(Test , cb){
	var t = {}
	
	t.Test = Test
	jsrequestmomentum("/momentum/funcs?name=GetCard", t, "POSTJSON", cb)
}
function Start(Test , cb){
	var t = {}
	
	t.Test = Test
	jsrequestmomentum("/momentum/funcs?name=Start", t, "POSTJSON", cb)
}
function Cancel(Test , cb){
	var t = {}
	
	t.Test = Test
	jsrequestmomentum("/momentum/funcs?name=Cancel", t, "POSTJSON", cb)
}
function Nuke(  cb){
	var t = {}
	
	jsrequestmomentum("/momentum/funcs?name=Nuke", t, "POSTJSON", cb)
}
function UpdateApp(App , cb){
	var t = {}
	
	t.App = App
	jsrequestmomentum("/momentum/funcs?name=UpdateApp", t, "POSTJSON", cb)
}
function UpdateTest(Test , cb){
	var t = {}
	
	t.Test = Test
	jsrequestmomentum("/momentum/funcs?name=UpdateTest", t, "POSTJSON", cb)
}
function UpdateSettings(Settings , cb){
	var t = {}
	
	t.Settings = Settings
	jsrequestmomentum("/momentum/funcs?name=UpdateSettings", t, "POSTJSON", cb)
}
function DeleteApp(App , cb){
	var t = {}
	
	t.App = App
	jsrequestmomentum("/momentum/funcs?name=DeleteApp", t, "POSTJSON", cb)
}
function DeleteTest(Test , cb){
	var t = {}
	
	t.Test = Test
	jsrequestmomentum("/momentum/funcs?name=DeleteTest", t, "POSTJSON", cb)
}
`))
	})

	http.HandleFunc("/momentum/funcs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.FormValue("name") == "reset" || r.Method == "OPTIONS" {
			return
		} else if r.FormValue("name") == "AddApp" {
			w.Header().Set("Content-Type", "application/json")
			type PayloadAddApp struct {
				App App
			}
			decoder := json.NewDecoder(r.Body)
			var tmvv PayloadAddApp
			err := decoder.Decode(&tmvv)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
				return
			}
			resp := db.O{}
			respdone0 := NetAddApp(tmvv.App)

			resp["done"] = respdone0
			w.Write([]byte(mResponse(resp)))
		} else if r.FormValue("name") == "AddTest" {
			w.Header().Set("Content-Type", "application/json")
			type PayloadAddTest struct {
				Test Test
			}
			decoder := json.NewDecoder(r.Body)
			var tmvv PayloadAddTest
			err := decoder.Decode(&tmvv)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
				return
			}
			resp := db.O{}
			respdone0 := NetAddTest(tmvv.Test)

			resp["done"] = respdone0
			w.Write([]byte(mResponse(resp)))
		} else if r.FormValue("name") == "Cleo" {
			w.Header().Set("Content-Type", "application/json")
			type PayloadCleo struct {
			}
			decoder := json.NewDecoder(r.Body)
			var tmvv PayloadCleo
			err := decoder.Decode(&tmvv)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
				return
			}
			resp := db.O{}
			respcleo0 := NetCleo()

			resp["cleo"] = respcleo0
			w.Write([]byte(mResponse(resp)))
		} else if r.FormValue("name") == "DeleteAlerts" {
			w.Header().Set("Content-Type", "application/json")
			type PayloadDeleteAlerts struct {
			}
			decoder := json.NewDecoder(r.Body)
			var tmvv PayloadDeleteAlerts
			err := decoder.Decode(&tmvv)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
				return
			}
			resp := db.O{}
			respdone0 := NetDeleteAlerts()

			resp["done"] = respdone0
			w.Write([]byte(mResponse(resp)))
		} else if r.FormValue("name") == "GetList" {
			w.Header().Set("Content-Type", "application/json")
			type PayloadGetList struct {
				Test   Test
				Lookup string
			}
			decoder := json.NewDecoder(r.Body)
			var tmvv PayloadGetList
			err := decoder.Decode(&tmvv)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
				return
			}
			resp := db.O{}
			resplist0 := NetGetList(tmvv.Test, tmvv.Lookup)

			resp["list"] = resplist0
			w.Write([]byte(mResponse(resp)))
		} else if r.FormValue("name") == "GetTop" {
			w.Header().Set("Content-Type", "application/json")
			type PayloadGetTop struct {
				Test Test
			}
			decoder := json.NewDecoder(r.Body)
			var tmvv PayloadGetTop
			err := decoder.Decode(&tmvv)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
				return
			}
			resp := db.O{}
			resptop0 := NetGetTop(tmvv.Test)

			resp["top"] = resptop0
			w.Write([]byte(mResponse(resp)))
		} else if r.FormValue("name") == "GetCard" {
			w.Header().Set("Content-Type", "application/json")
			type PayloadGetCard struct {
				Test Test
			}
			decoder := json.NewDecoder(r.Body)
			var tmvv PayloadGetCard
			err := decoder.Decode(&tmvv)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
				return
			}
			resp := db.O{}
			respres0 := NetGetCard(tmvv.Test)

			resp["res"] = respres0
			w.Write([]byte(mResponse(resp)))
		} else if r.FormValue("name") == "Start" {
			w.Header().Set("Content-Type", "application/json")
			type PayloadStart struct {
				Test Test
			}
			decoder := json.NewDecoder(r.Body)
			var tmvv PayloadStart
			err := decoder.Decode(&tmvv)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
				return
			}
			resp := db.O{}
			respdone0 := NetStart(tmvv.Test)

			resp["done"] = respdone0
			w.Write([]byte(mResponse(resp)))
		} else if r.FormValue("name") == "Cancel" {
			w.Header().Set("Content-Type", "application/json")
			type PayloadCancel struct {
				Test Test
			}
			decoder := json.NewDecoder(r.Body)
			var tmvv PayloadCancel
			err := decoder.Decode(&tmvv)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
				return
			}
			resp := db.O{}
			respdone0 := NetCancel(tmvv.Test)

			resp["done"] = respdone0
			w.Write([]byte(mResponse(resp)))
		} else if r.FormValue("name") == "Nuke" {
			w.Header().Set("Content-Type", "application/json")
			type PayloadNuke struct {
			}
			decoder := json.NewDecoder(r.Body)
			var tmvv PayloadNuke
			err := decoder.Decode(&tmvv)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
				return
			}
			resp := db.O{}
			respdone0 := NetNuke()

			resp["done"] = respdone0
			w.Write([]byte(mResponse(resp)))
		} else if r.FormValue("name") == "UpdateApp" {
			w.Header().Set("Content-Type", "application/json")
			type PayloadUpdateApp struct {
				App App
			}
			decoder := json.NewDecoder(r.Body)
			var tmvv PayloadUpdateApp
			err := decoder.Decode(&tmvv)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
				return
			}
			resp := db.O{}
			respdone0 := NetUpdateApp(tmvv.App)

			resp["done"] = respdone0
			w.Write([]byte(mResponse(resp)))
		} else if r.FormValue("name") == "UpdateTest" {
			w.Header().Set("Content-Type", "application/json")
			type PayloadUpdateTest struct {
				Test Test
			}
			decoder := json.NewDecoder(r.Body)
			var tmvv PayloadUpdateTest
			err := decoder.Decode(&tmvv)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
				return
			}
			resp := db.O{}
			respdone0 := NetUpdateTest(tmvv.Test)

			resp["done"] = respdone0
			w.Write([]byte(mResponse(resp)))
		} else if r.FormValue("name") == "UpdateSettings" {
			w.Header().Set("Content-Type", "application/json")
			type PayloadUpdateSettings struct {
				Settings Setting
			}
			decoder := json.NewDecoder(r.Body)
			var tmvv PayloadUpdateSettings
			err := decoder.Decode(&tmvv)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
				return
			}
			resp := db.O{}
			respdone0 := NetUpdateSettings(tmvv.Settings)

			resp["done"] = respdone0
			w.Write([]byte(mResponse(resp)))
		} else if r.FormValue("name") == "DeleteApp" {
			w.Header().Set("Content-Type", "application/json")
			type PayloadDeleteApp struct {
				App App
			}
			decoder := json.NewDecoder(r.Body)
			var tmvv PayloadDeleteApp
			err := decoder.Decode(&tmvv)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
				return
			}
			resp := db.O{}
			respdone0 := NetDeleteApp(tmvv.App)

			resp["done"] = respdone0
			w.Write([]byte(mResponse(resp)))
		} else if r.FormValue("name") == "DeleteTest" {
			w.Header().Set("Content-Type", "application/json")
			type PayloadDeleteTest struct {
				Test Test
			}
			decoder := json.NewDecoder(r.Body)
			var tmvv PayloadDeleteTest
			err := decoder.Decode(&tmvv)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())))
				return
			}
			resp := db.O{}
			respdone0 := NetDeleteTest(tmvv.Test)

			resp["done"] = respdone0
			w.Write([]byte(mResponse(resp)))

		}
	})
	//+++extendgxmlmain+++
	http.Handle("/dist/", http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: "web"}))

	errgos := http.ListenAndServe(port, nil)
	if errgos != nil {
		log.Fatal(errgos)
	}

}

//+++extendgxmlroot+++
