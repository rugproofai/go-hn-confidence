package main

import (
	"fmt"
	"html/template"
	"math"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/termie/go-shutil"
)

const (
	ROOT_URL   string = "https://news.ycombinator.com"
	TARGET_DIR string = "./dist"
)

type NewsItem struct {
	Title        string
	Link         string
	Points       int
	Comments     int
	CommentsLink string
}

func (ni *NewsItem) Score() float64 {
	n := float64(ni.Points + ni.Comments)
	pos := float64(ni.Points)
	if n == 0 {
		return 0
	}

	z := 1.96 // we assume a 95% confidence interval
	phat := 1.0 * pos / n
	res := (phat + z*z/(2*n) - z*math.Sqrt((phat*(1-phat)+z*z/(4*n))/n)) / (1 + z*z/n)
	return res
}

type Items []*NewsItem

func (m Items) Len() int {
	return len(m)
}

func (m Items) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m Items) Less(i, j int) bool {
	return m[i].Score() < m[j].Score()
}

func Scrape() ([]*NewsItem, error) {
	doc, err := goquery.NewDocument(ROOT_URL)
	if err != nil {
		return nil, err
	}

	res := []*NewsItem{}

	doc.Find(".athing").Each(func(i int, s *goquery.Selection) {
		el := s.Find(".title a")
		// If there's more than one found, reduce it to the first element
		if el.Size() > 1 {
			el = el.Slice(0, 1)
		}

		title := el.Text()
		if err != nil {
			fmt.Printf("error grabbing html: %s\n", err)
			return
		}
		link, exists := s.Find(".title a").Attr("href")
		if !exists {
			link = ""
		}

		if strings.HasPrefix(link, "item?") {
			link = ROOT_URL + "/" + link
		}
		item := &NewsItem{Title: title, Link: link}
		res = append(res, item)
		//fmt.Printf("%v - %v\n", title, link)
	})

	doc.Find(".subtext").Each(func(i int, s *goquery.Selection) {
		pString := s.Find(".score").Text()
		cString := s.Find("a").Last().Text()
		cString = strings.TrimSpace(strings.Replace(cString, "|", "", -1))
		cLink, exists := s.Find("a").Last().Attr("href")
		if !exists {
			cLink = ""
		} else {
			cLink = ROOT_URL + "/" + cLink
		}
		points := 0
		comments := 0

		if pString != "" {
			pSt := strings.Fields(pString)[0]
			points, err = strconv.Atoi(pSt)
			if err != nil {
				points = 0
			}
		}

		if cString != "" && cString != "discuss" && cString != "hide" {
			cSt := strings.Fields(cString)[0]
			comments, err = strconv.Atoi(cSt)
			if err != nil {
				comments = 0
			}
		}

		item := res[i]
		item.Points = points
		item.Comments = comments
		item.CommentsLink = cLink
	})

	sort.Sort(sort.Reverse(Items(res)))
	return res, nil
}

func main() {
	newsItems, err := Scrape()
	if err != nil {
		fmt.Println(err)
	}

	funcMap := template.FuncMap{
		"fdate": DateFmt,
	}

	currentDir, _ := os.Getwd()
	tmplPath := path.Join(currentDir, "template.html")
	t := template.Must(template.New("template.html").Funcs(funcMap).ParseFiles(tmplPath))

	buildDir := path.Join(currentDir, TARGET_DIR)

	filepath := path.Join(buildDir, "index.html")
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0755)
	if err != nil {
		fmt.Println(err)
	}

	presenter := struct {
		Items   []*NewsItem
		LastGen time.Time
	}{newsItems, time.Now()}
	err = t.Execute(file, presenter)
	if err != nil {
		fmt.Println(err)
	}

	staticPath := path.Join(currentDir, "static")
	buildStaticPath := path.Join(buildDir, "static")

	// CopyTree demands that the destination folder not exist
	// If it does, we delete it

	_, err = os.Stat(buildStaticPath)
	if err == nil {
		err = os.RemoveAll(buildStaticPath)
		if err != nil {
			fmt.Println(err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		fmt.Println(err)
	}

	// CopyTree options:
	// Symlinks - if true, symbolic links copied, if false symlinked files copied
	// IgnoreDanglingSymlinks - supress error thrown when symlink links to missing file
	// Optional CopyFunction
	// Optional Ignore function
	options := &shutil.CopyTreeOptions{
		Symlinks:               false,
		IgnoreDanglingSymlinks: true,
		CopyFunction:           shutil.Copy,
		Ignore:                 nil,
	}
	err = shutil.CopyTree(staticPath, buildStaticPath, options)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Build complete")
}

// Helpers
func DateFmt(tt time.Time) string {
	const layout = "3:04pm, 2 January 2006"
	return tt.Format(layout)
}
