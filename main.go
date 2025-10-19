package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/PuerkitoBio/goquery"

	homedir "github.com/mitchellh/go-homedir"
)

const APP_NAME = "procon-gardener"

type AtCoderSubmission struct {
	ID          int
	EpochSecond int64
	ProblemID   string
	ContestID   string
	UserID      string
	Language    string
	Result      string
}

type Service struct {
	RepositoryPath string `json:"repository_path"`
	UserID         string `json:"user_id"`
	UserEmail      string `json:"user_email"`
}
type Config struct {
	Atcoder Service `json:"atcoder"`
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

func loadConfig() (*Config, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}
	configFile := filepath.Join(home, "."+APP_NAME, "config.json")
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	var config Config
	if err = json.Unmarshal(bytes, &config); err != nil {
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

	jsonBytes, err := json.MarshalIndent(submission, "", "\t")
	if err != nil {
		log.Println(err)
		return nil
	}
	fileJSON, err := os.Create(filepath.Join(path, "submission.json"))
	if err != nil {
		log.Println(err)
		return nil
	}
	defer fileJSON.Close()
	fileJSON.WriteString(string(jsonBytes))

	return nil
}

func archiveCmd() {
	config, err := loadConfig()
	if err != nil {
		log.Println(err)
		return
	}

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

				result := tds.Eq(6).Find("span").Text()
				if result != "AC" {
					return
				}

				submissionLink, exists := tds.Eq(9).Find("a.submission-details-link").Attr("href")
				if !exists {
					return
				}
				idStr := strings.Split(submissionLink, "/")[4]
				id, _ := strconv.Atoi(idStr)

				problemLink, exists := tds.Eq(1).Find("a").Attr("href")
				if !exists {
					return
				}
				problemID := strings.Split(problemLink, "/")[4]

				language := tds.Eq(3).Find("a").Text()
				timeStr := tds.Eq(0).Find("time").Text()
				epoch := time.Now().Unix()
				if t, err := time.Parse("2006-01-02 15:04:05", timeStr); err == nil {
					epoch = t.Unix()
				}

				ss = append(ss, AtCoderSubmission{
					ID:          id,
					ProblemID:   problemID,
					ContestID:   contestID,
					UserID:      config.Atcoder.UserID,
					Language:    language,
					Result:      result,
					EpochSecond: epoch,
				})
			})

			page++
			time.Sleep(500 * time.Millisecond)
		}
	}

	// 差分チェック
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

	// アーカイブ
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

		fileName := "Main.txt"
		archiveDirPath := filepath.Join(config.Atcoder.RepositoryPath, "atcoder.jp", s.ContestID, s.ProblemID)
		if err = archiveFile(code, fileName, archiveDirPath, s); err != nil {
			log.Println("Fail to archive code at", filepath.Join(archiveDirPath, fileName))
			continue
		}

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
		time.Sleep(1500 * time.Millisecond)
	}

	log.Println("Archive completed.")
}