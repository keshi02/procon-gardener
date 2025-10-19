package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/PuerkitoBio/goquery"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/skratchdot/open-golang/open"
	"github.com/thoas/go-funk"
	cli "github.com/urfave/cli/v2"
)

const APP_NAME = "procon-gardener"
const ATCODER_API_SUBMISSION_URL = "https://kenkoooo.com/atcoder/atcoder-api/v3/user/submissions?user=%s&from_second=0"

type AtCoderSubmission struct {
	ID            int     `json:"id"`
	EpochSecond   int64   `json:"epoch_second"`
	ProblemID     string  `json:"problem_id"`
	ContestID     string  `json:"contest_id"`
	UserID        string  `json:"user_id"`
	Language      string  `json:"language"`
	Point         float64 `json:"point"`
	Length        int     `json:"length"`
	Result        string  `json:"result"`
	ExecutionTime int     `json:"execution_time"`
}

func isDirExist(path string) bool {
	info, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}
func isFileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

type Service struct {
	RepositoryPath string `json:"repository_path"`
	UserID         string `json:"user_id"`
	UserEmail      string `json:"user_email"`
}
type Config struct {
	Atcoder Service `json:"atcoder"`
}

func languageToFileName(language string) string {
	//e.g C++14 (GCC 5.4.1)
	//C++14
	language = strings.Split(language, "(")[0]
	//remove extra last whitespace
	language = language[:len(language)-1]
	if strings.HasPrefix(language, "C++") {
		return "Main.cpp"
	}
	if strings.HasPrefix(language, "Bash") {
		return "Main.sh"
	}

	//C (GCC 5.4.1)
	//C (Clang 3.8.0)
	if language == "C" {
		return "Main.c"
	}

	if language == "C#" {
		return "Main.cs"
	}

	if language == "Clojure" {
		return "Main.clj"
	}

	if strings.HasPrefix(language, "Common Lisp") {
		return "Main.lisp"
	}

	//D (DMD64 v2.070.1)
	if language == "D" {
		return "Main.d"
	}

	if language == "Fortran" {
		return "Main.f08"
	}

	if language == "Go" {
		return "Main.go"
	}

	if language == "Haskell" {
		return "Main.hs"
	}

	if language == "JavaScript" {
		return "Main.js"
	}
	if language == "Java" {
		return "Main.java"
	}
	if language == "OCaml" {
		return "Main.ml"
	}

	if language == "Pascal" {
		return "Main.pas"
	}

	if language == "Perl" {
		return "Main.pl"
	}

	if language == "PHP" {
		return "Main.php"
	}

	if strings.HasPrefix(language, "Python") {
		return "Main.py"
	}

	if language == "Ruby" {
		return "Main.rb"
	}

	if language == "Scala" {
		return "Main.scala"
	}

	if language == "Scheme" {
		return "Main.scm"
	}

	if language == "Main.txt" {
		return "Main.txt"
	}

	if language == "Visual Basic" {
		return "Main.vb"
	}

	if language == "Objective-C" {
		return "Main.m"
	}

	if language == "Swift" {
		return "Main.swift"
	}

	if language == "Rust" {
		return "Main.rs"
	}

	if language == "Sed" {
		return "Main.sed"
	}

	if language == "Awk" {
		return "Main.awk"
	}

	if language == "Brainfuck" {
		return "Main.bf"
	}

	if language == "Standard ML" {
		return "Main.sml"
	}

	if strings.HasPrefix(language, "PyPy") {
		return "Main.py"
	}

	if language == "Crystal" {
		return "Main.cr"
	}

	if language == "F#" {
		return "Main.fs"
	}

	if language == "Unlambda" {
		return "Main.unl"
	}

	if language == "Lua" {
		return "Main.lua"
	}

	if language == "LuaJIT" {
		return "Main.lua"
	}

	if language == "MoonScript" {
		return "Main.moon"
	}

	if language == "Ceylon" {
		return "Main.ceylon"
	}

	if language == "Julia" {
		return "Main.jl"
	}

	if language == "Octave" {
		return "Main.m"
	}

	if language == "Nim" {
		return "Main.nim"
	}

	if language == "TypeScript" {
		return "Main.ts"
	}

	if language == "Perl6" {
		return "Main.p6"
	}

	if language == "Kotlin" {
		return "Main.kt"
	}

	if language == "COBOL" {
		return "Main.cob"
	}

	log.Printf("Unknown ... %s", language)
	return "Main.txt"
}

func initCmd(strict bool) {

	log.Println("Initialize your config...")
	home, err := homedir.Dir()
	if err != nil {
		log.Println(err)
		return
	}
	configDir := filepath.Join(home, "."+APP_NAME)
	if !isDirExist(configDir) {
		err = os.MkdirAll(configDir, 0700)
		if err != nil {
			log.Println(err)
			return
		}
	}

	configFile := filepath.Join(configDir, "config.json")
	if strict || !isFileExist(configFile) {
		//initial config
		atcoder := Service{RepositoryPath: "", UserID: ""}

		config := Config{Atcoder: atcoder}

		jsonBytes, err := json.MarshalIndent(config, "", "\t")
		if err != nil {
			log.Println(err)
			return
		}
		json := string(jsonBytes)
		file, err := os.Create(filepath.Join(configDir, "config.json"))
		if err != nil {
			log.Println(err)
			return
		}
		defer file.Close()
		file.WriteString(json)
	}
	log.Println("Initialized your config at ", configFile)
}

func loadConfig() (*Config, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}
	configDir := filepath.Join(home, "."+APP_NAME)
	configFile := filepath.Join(configDir, "config.json")
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	var config Config
	if err = json.Unmarshal(bytes, &config); err != nil {
		log.Println(err)
		return nil, err
	}
	return &config, nil
}

func archiveFile(code, fileName, path string, submission AtCoderSubmission) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	filePath := filepath.Join(path, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	file.WriteString(code)

	{
		//save submission json file
		jsonBytes, err := json.MarshalIndent(submission, "", "\t")
		if err != nil {
			log.Println(err)
		}
		json := string(jsonBytes)
		file, err := os.Create(filepath.Join(path, "submission.json"))
		if err != nil {
			log.Println(err)
		}
		defer file.Close()
		file.WriteString(json)
	}
	return nil
}

func archiveCmd() {
	config, err := loadConfig()
	if err != nil {
		log.Println(err)
		return
	}

	// 対象コンテストをリストに追加
	contests := []string{"abc040"} // 必要に応じて追加

	ss := []AtCoderSubmission{}

	for _, contestID := range contests {
		page := 1
		for {
			submissionPage := fmt.Sprintf(
				"https://atcoder.jp/contests/%s/submissions?f.User=%s&page=%d",
				contestID,
				config.Atcoder.UserID,
				page,
			)

			doc, err := goquery.NewDocument(submissionPage)
			if err != nil {
				log.Println(err)
				break
			}

			rows := doc.Find("table tbody tr")
			if rows.Length() == 0 {
				break
			}

			rows.Each(func(i int, tr *goquery.Selection) {
				tds := tr.Find("td")
				if tds.Length() < 10 {
					return
				}

				// AC判定
				result := tds.Eq(6).Find("span").Text()
				if result != "AC" {
					return
				}

				// 提出ID
				submissionLink, exists := tds.Eq(9).Find("a.submission-details-link").Attr("href")
				if !exists {
					return
				}
				idStr := strings.Split(submissionLink, "/")[4]
				id, _ := strconv.Atoi(idStr)

				// 問題ID
				problemLink, exists := tds.Eq(1).Find("a").Attr("href")
				if !exists {
					return
				}
				problemID := strings.Split(problemLink, "/")[4]

				// 言語
				language := tds.Eq(3).Find("a").Text()

				// 提出時刻
				timeStr := tds.Eq(0).Find("time").Text()
				epoch := time.Now().Unix()
				if t, err := time.Parse("2006-01-02 15:04:05", timeStr); err == nil {
					epoch = t.Unix()
				}

				s := AtCoderSubmission{
					ID:          id,
					ProblemID:   problemID,
					ContestID:   contestID,
					UserID:      config.Atcoder.UserID,
					Language:    language,
					Result:      result,
					EpochSecond: epoch,
				}
				ss = append(ss, s)
			})

			page++
			time.Sleep(500 * time.Millisecond) // サーバ負荷軽減
		}
	}

	// 既にアーカイブ済みを除外
	archivedKeys := map[string]struct{}{}
	filepath.Walk(config.Atcoder.RepositoryPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(path, "submission.json") {
			bytes, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			var submission AtCoderSubmission
			if err = json.Unmarshal(bytes, &submission); err != nil {
				return err
			}
			key := submission.ContestID + "_" + submission.ProblemID
			archivedKeys[key] = struct{}{}
		}
		return nil
	})

	ssFiltered := []AtCoderSubmission{}
	for _, s := range ss {
		key := s.ContestID + "_" + s.ProblemID
		if _, ok := archivedKeys[key]; !ok {
			ssFiltered = append(ssFiltered, s)
		}
	}

	// アーカイブ処理
	startTime := time.Now()
	log.Printf("Archiving %d code...", len(ssFiltered))
	for _, s := range ssFiltered {
		url := fmt.Sprintf("https://atcoder.jp/contests/%s/submissions/%d", s.ContestID, s.ID)
		resp, err := http.Get(url)
		if err != nil {
			log.Println(err)
			continue
		}
		defer resp.Body.Close()
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.Println(err)
			continue
		}

		code := doc.Find("#submission-code").Text()
		if code == "" {
			log.Println("Empty code:", url)
			continue
		}

		fileName := languageToFileName(s.Language)
		archiveDirPath := filepath.Join(config.Atcoder.RepositoryPath, "atcoder.jp", s.ContestID, s.ProblemID)
		if err = archiveFile(code, fileName, archiveDirPath, s); err != nil {
			log.Println("Fail to archive code at", filepath.Join(archiveDirPath, fileName))
			continue
		}
		log.Println("Archived:", filepath.Join(archiveDirPath, fileName))

		// Git追加・コミット
		if !isDirExist(filepath.Join(config.Atcoder.RepositoryPath, ".git")) {
			continue
		}
		r, err := git.PlainOpen(config.Atcoder.RepositoryPath)
		if err != nil {
			log.Println(err)
			continue
		}
		w, err := r.Worktree()
		if err != nil {
			log.Println(err)
			continue
		}

		_, err = w.Add(filepath.Join("atcoder.jp", s.ContestID, s.ProblemID, fileName))
		if err != nil {
			log.Println(err)
			continue
		}
		_, err = w.Add(filepath.Join("atcoder.jp", s.ContestID, s.ProblemID, "submission.json"))
		if err != nil {
			log.Println(err)
			continue
		}

		message := fmt.Sprintf("[AC] %s %s", s.ContestID, s.ProblemID)
		_, err = w.Commit(message, &git.CommitOptions{
			Author: &object.Signature{
				Name:  s.UserID,
				Email: config.Atcoder.UserEmail,
				When:  time.Unix(s.EpochSecond, 0),
			},
		})
		if err != nil {
			log.Println(err)
			continue
		}

		time.Sleep(1500 * time.Millisecond) // サーバ負荷軽減
	}

	log.Println("Archive completed.")
}
func validateConfig(config Config) bool {
	//TODO check path
	return false
}
func editCmd() {

	home, err := homedir.Dir()
	if err != nil {
		log.Println(err)
		return
	}
	configFile := filepath.Join(home, "."+APP_NAME, "config.json")
	//Config file not found, force to run an init cmd
	if !isFileExist(configFile) {
		initCmd(true)
	}

	editor := os.Getenv("EDITOR")
	if editor != "" {
		c := exec.Command(editor, configFile)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Run()
	} else {
		open.Run(configFile)
	}

}

func main() {

	app := cli.App{Name: "procon-gardener", Usage: "archive your AC submissions",
		Commands: []*cli.Command{
			{
				Name:    "archive",
				Aliases: []string{"a"},
				Usage:   "archive your AC submissions",
				Action: func(c *cli.Context) error {
					archiveCmd()
					return nil
				},
			},
			{
				Name:    "init",
				Aliases: []string{"i"},
				Usage:   "initialize your config",
				Action: func(c *cli.Context) error {
					initCmd(true)
					return nil
				},
			},
			{
				Name:    "edit",
				Aliases: []string{"e"},
				Usage:   "edit your config file",
				Action: func(c *cli.Context) error {
					editCmd()
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
