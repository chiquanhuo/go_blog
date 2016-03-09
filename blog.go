package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/russross/blackfriday"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"text/template"
)

type Blog struct {
	Id        int
	Author    string
	Commit_id string
	Filename  string
	Message   string
	Content   string
	Post_time string
}

var db *sql.DB
var view *template.Template

func main() {
	var err error
	db, err = sql.Open("mysql", "root:123456@(127.0.0.1:3306)/Lesson")
	checkErr(err)

	defer db.Close()

	err = LoadTemplate()
	checkErr(err)
	http.HandleFunc("/hook", hookHandler)
	http.HandleFunc("/load", loadHandler)
	http.HandleFunc("/list", listHandler)

	err = http.ListenAndServe(":9090", nil)
	checkErr(err)
}

func LoadTemplate() error {
	funcs := make(template.FuncMap)

	v := template.New("view")
	v.Funcs(funcs)

	_, err := v.ParseGlob("*.html")
	if err != nil {
		return err
	}

	view = v
	return nil
}

func loadHandler(w http.ResponseWriter, req *http.Request) {
	err := LoadTemplate()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func listHandler(w http.ResponseWriter, req *http.Request) {
	rows, err := db.Query("SELECT * FROM blogs")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	bgs := []Blog{}
	for rows.Next() {
		bg := Blog{}
		err := rows.Scan(&bg.Id, &bg.Author, &bg.Commit_id, &bg.Filename, &bg.Message, &bg.Content, &bg.Post_time)
		if nil != err {
			http.Error(w, err.Error(), 500)
			return
		}
		bgs = append(bgs, bg)
	}

	err = view.ExecuteTemplate(w, "index.html", bgs)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func hookHandler(w http.ResponseWriter, req *http.Request) {
	content, _ := ioutil.ReadAll(req.Body)
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err == nil {
		var commits = data["commits"]
		var values = commits.([]interface{})
		for _, val := range values {
			var map_val = val.(map[string]interface{})
			commit_id := map_val["id"].(string)
			fmt.Println(commit_id)
			var files = getFiles()
			for _, term := range files {
				var item = term.(map[string]string)
				fmt.Println(item)
				var result = checkDB(item["file_name"])
				fmt.Println(result)
				if result == 0 {
					author := map_val["author"].(map[string]interface{})
					name := author["name"].(string)
					message := map_val["message"].(string)
					post_time := map_val["timestamp"].(string)
					filename := item["file_name"]
					content := item["content"]
					insertDB(name, commit_id, message, post_time, filename, content)
				} else {
					message := map_val["message"].(string)
					content := item["content"]
					post_time := map_val["timestamp"].(string)
					updateDB(result, commit_id, message, post_time, content)
				}
			}
			//for key, value := range map_val {
			//	fmt.Println(key, value)
			//}
		}
	}
}

func checkDB(file string) int {
	res, err := db.Query("SELECT id FROM blogs WHERE filename=? limit 1", file)
	if err != nil {
		return 0
	} else {
		for res.Next() {
			var Id int
			err := res.Scan(&Id)
			if err != nil {
				return 0
			} else {
				return Id
			}
		}
	}
	return 0
}

func getFiles() map[int]interface{} {
	files := make(map[int]interface{})
	argv := []string{"--git-dir=/home/huozhiquan/go_blog/.git", "show", "--name-only", "--oneline"}
	cmd := exec.Command("git", argv...)
	out, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
	}
	data_str := (string(out))
	data_list := strings.Split(data_str, "\n")
	data_list = data_list[1 : len(data_list)-1]
	//fmt.Println(data_list)
	web_dir := "/home/huozhiquan/go_blog/"
	i := 0
	for _, file := range data_list {
		file_data := make(map[string]string)
		file_dir := web_dir + file
		//fmt.Println(file)
		file_data["file_name"] = file
		file_data["file_dir"] = file_dir
		contents, _ := ioutil.ReadFile(file_dir)
		//fmt.Println(string(contents))
		file_data["content"] = string(contents)
		files[i] = file_data
		i += 1
	}
	return files
}

func insertDB(author string, commit_id string, message string, post_time string, filename string, content string) {
	temp_content := []byte(content)
	content = string(blackfriday.MarkdownCommon(temp_content))
	_, err := db.Query("insert into blogs(author, commit_id, filename, message, content, post_time)values(?,?,?,?,?,?)",
		author, commit_id, filename, message, content, post_time)
	if err != nil {
		fmt.Println(err)
	}
}

func updateDB(id int, commit_id string, message string, post_time string, content string) {

	temp_content := []byte(content)
	content = string(blackfriday.MarkdownCommon(temp_content))

	_, err := db.Query("update blogs set commit_id=?, message=?, content=? where id=? order by id desc limit 1",
		commit_id, message, content, id)
	if err != nil {
		fmt.Println(err)
	}
}
