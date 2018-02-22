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
	"github.com/opentracing/opentracing-go"
	"html"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sourcegraph.com/sourcegraph/appdash"
	appdashot "sourcegraph.com/sourcegraph/appdash/opentracing"
	"sourcegraph.com/sourcegraph/appdash/traceapp"
	"strconv"
	"strings"
	"time"
)

var store = sessions.NewCookieStore([]byte("a very very very very secret key"))

var Prod = false

var TemplateFuncStore template.FuncMap
var templateCache = gosweb.NewTemplateCache()

func StoreNetfn() int {
	TemplateFuncStore = template.FuncMap{"a": gosweb.Netadd, "s": gosweb.Netsubs, "m": gosweb.Netmultiply, "d": gosweb.Netdivided, "js": gosweb.Netimportjs, "css": gosweb.Netimportcss, "sd": gosweb.NetsessionDelete, "sr": gosweb.NetsessionRemove, "sc": gosweb.NetsessionKey, "ss": gosweb.NetsessionSet, "sso": gosweb.NetsessionSetInt, "sgo": gosweb.NetsessionGetInt, "sg": gosweb.NetsessionGet, "form": gosweb.Formval, "eq": gosweb.Equalz, "neq": gosweb.Nequalz, "lte": gosweb.Netlt, "LoadWebAsset": NetLoadWebAsset, "AddApp": NetAddApp, "AddTest": NetAddTest, "Cleo": NetCleo, "DeleteAlerts": NetDeleteAlerts, "GetList": NetGetList, "GetTop": NetGetTop, "GetCard": NetGetCard, "Start": NetStart, "Cancel": NetCancel, "Nuke": NetNuke, "UpdateApp": NetUpdateApp, "UpdateTest": NetUpdateTest, "UpdateSettings": NetUpdateSettings, "DeleteApp": NetDeleteApp, "DeleteTest": NetDeleteTest, "ang": Netang, "bang": Netbang, "cang": Netcang, "server": Netserver, "bserver": Netbserver, "cserver": Netcserver, "jquery": Netjquery, "bjquery": Netbjquery, "cjquery": Netcjquery, "App": NetstructApp, "isApp": NetcastApp, "Setting": NetstructSetting, "isSetting": NetcastSetting, "EnvVar": NetstructEnvVar, "isEnvVar": NetcastEnvVar, "Test": NetstructTest, "isTest": NetcastTest, "HeapFrame": NetstructHeapFrame, "isHeapFrame": NetcastHeapFrame, "CleoSet": NetstructCleoSet, "isCleoSet": NetcastCleoSet, "Alert": NetstructAlert, "isAlert": NetcastAlert, "TopDist": NetstructTopDist, "isTopDist": NetcastTopDist}
	return 0
}

var FuncStored = StoreNetfn()

type dbflf db.O

func renderTemplate(w http.ResponseWriter, p *gosweb.Page, span opentracing.Span) {
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
				renderTemplate(w, pag, span) ///your-500-page"

			}
		}
	}()

	var sp opentracing.Span
	opName := fmt.Sprintf("Building template %s%s", p.R.URL.Path, ".tmpl")

	if true {
		carrier := opentracing.HTTPHeadersCarrier(p.R.Header)
		wireContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
		if err != nil {
			sp = opentracing.StartSpan(opName)
		} else {
			sp = opentracing.StartSpan(opName, opentracing.ChildOf(wireContext))
		}
	}
	defer sp.Finish()

	// TemplateFuncStore

	if _, ok := templateCache.Get(p.R.URL.Path); !ok || !Prod {
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
			renderTemplate(w, pag, span) // "/your-500-page"

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
func MakeHandler(fn func(http.ResponseWriter, *http.Request, opentracing.Span)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		span := opentracing.StartSpan(fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		defer span.Finish()
		carrier := opentracing.HTTPHeadersCarrier(r.Header)
		if err := span.Tracer().Inject(span.Context(), opentracing.HTTPHeaders, carrier); err != nil {
			log.Fatalf("Could not inject span context into header: %v", err)
		}

		if attmpt := apiAttempt(w, r, span); !attmpt {
			fn(w, r, span)
		}
		context.Clear(r)

	}
}

func mResponse(v interface{}) string {
	data, _ := json.Marshal(&v)
	return string(data)
}
func apiAttempt(w http.ResponseWriter, r *http.Request, span opentracing.Span) (callmet bool) {
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
func Handler(w http.ResponseWriter, r *http.Request, span opentracing.Span) {
	var p *gosweb.Page
	p, err := loadPage(r.URL.Path)
	var session *sessions.Session
	var er error
	if session, er = store.Get(r, "session-"); er != nil {
		session, _ = store.New(r, "session-")
	}

	var sp opentracing.Span
	opName := fmt.Sprintf(fmt.Sprintf("Web:/%s", r.URL.Path))

	if true {
		carrier := opentracing.HTTPHeadersCarrier(r.Header)
		wireContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
		if err != nil {
			sp = opentracing.StartSpan(opName)
		} else {
			sp = opentracing.StartSpan(opName, opentracing.ChildOf(wireContext))
		}
	}
	defer sp.Finish()

	if err != nil {
		log.Println(err.Error())

		w.WriteHeader(http.StatusNotFound)
		span.SetTag("error", true)
		span.LogEvent(fmt.Sprintf("%s request at %s, reason : %s ", r.Method, r.URL.Path, err))
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
			renderTemplate(w, pag, span) //"/your-500-page"
		}
		session = nil
		context.Clear(r)
		return
	}

	if !p.IsResource {
		w.Header().Set("Content-Type", "text/html")
		p.Session = session
		p.R = r
		renderTemplate(w, p, span) //fmt.Sprintf("web%s", r.URL.Path)
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

	//wheredefault

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

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `data,err := Asset( fmt.Sprintf("web%s", args[0].(string) ) )`
	data, err := Asset(fmt.Sprintf("web%s", args[0].(string)))
	lastLine = `if err != nil {`
	if err != nil {
		lastLine = `return err.Error()`
		return err.Error()
		lastLine = `}`
	}
	lastLine = `return string(data)`
	return string(data)
}

//
func NetAddApp(app App) (done bool) {

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `app.ID = core.NewLen(10)`
	app.ID = core.NewLen(10)
	lastLine = `Mset.Apps = append(Mset.Apps, app)`
	Mset.Apps = append(Mset.Apps, app)
	lastLine = `SaveConfig()`
	SaveConfig()
	lastLine = `done = true`
	done = true
	lastLine = `return`
	return
}

//
func NetAddTest(test Test) (done bool) {

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `test.ID = core.NewLen(10)`
	test.ID = core.NewLen(10)
	lastLine = `Mset.Tests = append(Mset.Tests, test)`
	Mset.Tests = append(Mset.Tests, test)
	lastLine = `SaveConfig()`
	SaveConfig()
	lastLine = `done = true`
	done = true
	lastLine = `return`
	return
}

//
func NetCleo() (cleo *CleoSet) {

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `cleo = Mset`
	cleo = Mset
	lastLine = `return`
	return
}

//
func NetDeleteAlerts() (done bool) {

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `Mset.Alerts = []Alert{}`
	Mset.Alerts = []Alert{}
	lastLine = `SaveConfig()`
	SaveConfig()
	lastLine = `return`
	return
}

//
func NetGetList(test Test, lookup string) (list string) {

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `if strings.Contains(lookup, "*") {`
	if strings.Contains(lookup, "*") {
		lastLine = `parts := strings.Split(lookup, ".")`
		parts := strings.Split(lookup, ".")
		lastLine = `lookup = fmt.Sprintf("%s()", parts[len(parts) - 1])`
		lookup = fmt.Sprintf("%s()", parts[len(parts)-1])
		lastLine = `}`
	}
	lastLine = `for cnt, _ := range test.HeapMinute {`
	for cnt, _ := range test.HeapMinute {
		lastLine = `cmd := fmt.Sprintf("go tool pprof --list=%s %s", lookup, filepath.Join(cleoWorkspace, Path("tests", test.ID, fmt.Sprintf("h%v", cnt ) )) )`
		cmd := fmt.Sprintf("go tool pprof --list=%s %s", lookup, filepath.Join(cleoWorkspace, Path("tests", test.ID, fmt.Sprintf("h%v", cnt))))
		lastLine = `logfull,_ := core.RunCmdSmart(cmd )`
		logfull, _ := core.RunCmdSmart(cmd)
		lastLine = `retset := strings.Split(logfull,"\n")`
		retset := strings.Split(logfull, "\n")
		lastLine = `if len(retset) > 2 {`
		if len(retset) > 2 {
			lastLine = `list = logfull`
			list = logfull
			lastLine = `break`
			break
			lastLine = `}`
		}
		lastLine = `}`
	}
	lastLine = `return`
	return
}

//
func NetGetTop(test Test) (top []TopDist) {

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `valm := make(map[string]float64)`
	valm := make(map[string]float64)
	lastLine = `for cnt, _ := range test.HeapMinute {`
	for cnt, _ := range test.HeapMinute {
		lastLine = `logfull,_ := core.RunCmdSmart(fmt.Sprintf("go tool pprof -top %s", filepath.Join(cleoWorkspace, Path("tests", test.ID, fmt.Sprintf("h%v", cnt ) )) ) )`
		logfull, _ := core.RunCmdSmart(fmt.Sprintf("go tool pprof -top %s", filepath.Join(cleoWorkspace, Path("tests", test.ID, fmt.Sprintf("h%v", cnt)))))
		lastLine = `retset := strings.Split(logfull,"\n")`
		retset := strings.Split(logfull, "\n")
		lastLine = `retset = retset[4:]`
		retset = retset[4:]
		lastLine = `for _, str := range retset {`
		for _, str := range retset {
			lastLine = `strfm := strings.Replace(strings.TrimSpace(str), "   "," ",-1 )`
			strfm := strings.Replace(strings.TrimSpace(str), "   ", " ", -1)
			lastLine = `strfm = strings.Replace(strfm, "  "," ", -1)`
			strfm = strings.Replace(strfm, "  ", " ", -1)
			lastLine = `subset := strings.Split(strfm," ")`
			subset := strings.Split(strfm, " ")
			lastLine = `if len(subset) > 5 {`
			if len(subset) > 5 {
				lastLine = `subsettwo := strings.Split(subset[len(subset) - 1], "   ")`
				subsettwo := strings.Split(subset[len(subset)-1], "   ")
				lastLine = `if strings.Contains(strfm," (inline)") {`
				if strings.Contains(strfm, " (inline)") {
					lastLine = `subsettwo = append([]string{subset[len(subset) - 3]},subsettwo...)`
					subsettwo = append([]string{subset[len(subset)-3]}, subsettwo...)
					lastLine = `} else if len(subsettwo) == 1 {`
				} else if len(subsettwo) == 1 {
					lastLine = `subsettwo = append([]string{subset[len(subset) - 2]},subsettwo...)`
					subsettwo = append([]string{subset[len(subset)-2]}, subsettwo...)
					lastLine = `}`
				}
				lastLine = `//fmt.Println(subsettwo)`
				//fmt.Println(subsettwo)
				lastLine = `_,exts := valm[subsettwo[0]]`
				_, exts := valm[subsettwo[0]]
				lastLine = `if !exts {`
				if !exts {
					lastLine = `valm[subsettwo[0]] = 0`
					valm[subsettwo[0]] = 0
					lastLine = `}`
				}
				lastLine = `f, _ := strconv.ParseFloat(strings.Replace( subset[1],"%","", -1), 64)`
				f, _ := strconv.ParseFloat(strings.Replace(subset[1], "%", "", -1), 64)
				lastLine = `valm[subsettwo[0]] += f`
				valm[subsettwo[0]] += f
				lastLine = `}`
			}
			lastLine = `}`
		}
		lastLine = `}`
	}
	lastLine = `tperc := 0.0`
	tperc := 0.0
	lastLine = `for key,val := range valm {`
	for key, val := range valm {
		lastLine = `perc := ( val/float64(len(test.HeapMinute) ) )`
		perc := (val / float64(len(test.HeapMinute)))
		lastLine = `top = append(top, TopDist{Name: key, Percent : perc })`
		top = append(top, TopDist{Name: key, Percent: perc})
		lastLine = `tperc += perc`
		tperc += perc
		lastLine = `}`
	}
	lastLine = `tperc = 100.0 - tperc`
	tperc = 100.0 - tperc
	lastLine = `top = append(top, TopDist{Name:"Other samples", Percent : tperc})`
	top = append(top, TopDist{Name: "Other samples", Percent: tperc})
	lastLine = `valm = nil`
	valm = nil
	lastLine = `return`
	return
}

//
func NetGetCard(test Test) (res string) {

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `bc, err := ioutil.ReadFile(filepath.Join(cleoWorkspace, fmt.Sprintf("%s.test", test.ID) ) )`
	bc, err := ioutil.ReadFile(filepath.Join(cleoWorkspace, fmt.Sprintf("%s.test", test.ID)))
	lastLine = `if err != nil {`
	if err != nil {
		lastLine = `res = err.Error()`
		res = err.Error()
		lastLine = `return`
		return
		lastLine = `}`
	}
	lastLine = `res = string(bc)`
	res = string(bc)
	lastLine = `return`
	return
}

//
func NetStart(test Test) (done bool) {

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `test.Working = true`
	test.Working = true
	lastLine = `test.Start = time.Now()`
	test.Start = time.Now()
	lastLine = `UpdateTest(test)`
	UpdateTest(test)
	lastLine = `SaveConfig()`
	SaveConfig()
	lastLine = `runtime.GOMAXPROCS(runtime.NumCPU() + Mset.Settings.Threads)`
	runtime.GOMAXPROCS(runtime.NumCPU() + Mset.Settings.Threads)
	lastLine = `go TestFrame(test)`
	go TestFrame(test)
	lastLine = `done = true`
	done = true
	lastLine = `return`
	return
}

//
func NetCancel(test Test) (done bool) {

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `test.Working = false`
	test.Working = false
	lastLine = `test.End = time.Now()`
	test.End = time.Now()
	lastLine = `UpdateTest(test)`
	UpdateTest(test)
	lastLine = `SaveConfig()`
	SaveConfig()
	lastLine = `done = true`
	done = true
	lastLine = `return`
	return
}

//
func NetNuke() (done bool) {

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `err := os.RemoveAll(cleoWorkspace)`
	err := os.RemoveAll(cleoWorkspace)
	lastLine = `if err != nil {`
	if err != nil {
		lastLine = `panic(err)`
		panic(err)
		lastLine = `}`
	}
	lastLine = `err = os.MkdirAll(cleoWorkspace, 0700)`
	err = os.MkdirAll(cleoWorkspace, 0700)
	lastLine = `if err != nil {`
	if err != nil {
		lastLine = `panic(err)`
		panic(err)
		lastLine = `}`
	}
	lastLine = `done = true`
	done = true
	lastLine = `Mset = &CleoSet{}`
	Mset = &CleoSet{}
	lastLine = `SaveConfig()`
	SaveConfig()
	lastLine = `return`
	return
}

//
func NetUpdateApp(app App) (done bool) {

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `UpdateEntry(app)`
	UpdateEntry(app)
	lastLine = `SaveConfig()`
	SaveConfig()
	lastLine = `done = true`
	done = true
	lastLine = `return`
	return
}

//
func NetUpdateTest(test Test) (done bool) {

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `UpdateTest(test)`
	UpdateTest(test)
	lastLine = `SaveConfig()`
	SaveConfig()
	lastLine = `done = true`
	done = true
	lastLine = `return`
	return
}

//
func NetUpdateSettings(settings Setting) (done bool) {

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `Mset.Settings = settings`
	Mset.Settings = settings
	lastLine = `SaveConfig()`
	SaveConfig()
	lastLine = `done = true`
	done = true
	lastLine = `return`
	return
}

//
func NetDeleteApp(app App) (done bool) {

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `newset := RmEntry(app)`
	newset := RmEntry(app)
	lastLine = `Mset.Apps = newset`
	Mset.Apps = newset
	lastLine = `SaveConfig()`
	SaveConfig()
	lastLine = `done = true`
	done = true
	lastLine = `return`
	return
}

//
func NetDeleteTest(test Test) (done bool) {

	lastLine := ""

	defer func() {
		if n := recover(); n != nil {
			log.Println("Pipeline failed at line :", gosweb.GetLine(".//gos.gxml", lastLine), "Of file:.//gos.gxml:", strings.TrimSpace(lastLine))
			log.Println("Reason : ", n)

		}
	}()
	lastLine = `newset := RmTest(test)`
	newset := RmTest(test)
	lastLine = `Mset.Tests = newset`
	Mset.Tests = newset
	lastLine = `SaveConfig()`
	SaveConfig()
	lastLine = `done = true`
	done = true
	lastLine = `return`
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

	if _, ok := templateCache.Get(localid); !ok || !Prod {

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

	if _, ok := templateCache.Get(localid); !ok || !Prod {

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

	if _, ok := templateCache.Get(localid); !ok || !Prod {

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

	if _, ok := templateCache.Get(localid); !ok || !Prod {

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

	if _, ok := templateCache.Get(localid); !ok || !Prod {

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

	if _, ok := templateCache.Get(localid); !ok || !Prod {

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
	store := appdash.NewMemoryStore()

	// Listen on any available TCP port locally.
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		log.Fatal(err)
	}
	collectorPort := l.Addr().(*net.TCPAddr).Port

	// Start an Appdash collection server that will listen for spans and
	// annotations and add them to the local collector (stored in-memory).
	cs := appdash.NewServer(l, appdash.NewLocalCollector(store))
	go cs.Start()

	// Print the URL at which the web UI will be running.
	appdashPort := 8700
	appdashURLStr := fmt.Sprintf("http://localhost:%d", appdashPort)
	appdashURL, err := url.Parse(appdashURLStr)
	if err != nil {
		log.Fatalf("Error parsing %s: %s", appdashURLStr, err)
	}
	color.Red("✅ Important!")
	log.Println("To see your traces, go to ", appdashURL)

	// Start the web UI in a separate goroutine.
	tapp, err := traceapp.New(nil, appdashURL)
	if err != nil {
		log.Fatal(err)
	}
	tapp.Store = store
	tapp.Queryer = store
	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", appdashPort), tapp))
	}()

	tracer := appdashot.NewTracer(appdash.NewRemoteCollector(fmt.Sprintf(":%d", collectorPort)))
	opentracing.InitGlobalTracer(tracer)

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
