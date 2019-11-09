package main

import (
	"database/sql"
	"errors"
	_ "github.com/go-sql-driver/mysql"
	"github.com/tealeg/xlsx"
	"os"
	"strings"
	"time"
)

var (
	//使用正确的数据库参数
	username = "xxxx"
	password = "xxxxxxxx"
	host = "xxxxxxxx"
	dbname = "xxxxxxxx"

	excelName = `\PendingChangeEmails.xlsx`
	basePath = `\uee_`

)


func main()  {

	//查询当前目录, 初始化数据
	str, _ := os.Getwd()
	excelName = str + excelName
	basePath = str + basePath

	//读取excel中的老邮箱集合与新邮箱集合
	olds, news := readExcel(excelName)

	// 查询要成的新邮箱中是否已被注册
	existsEmails := getExistsEmail(concatOldEmail(news))
	show("exist emails",existsEmails)

	//删除要成的新邮箱中已注册过的邮箱
	oldsFix, newsFix := fixOldsNews(olds, news, existsEmails)

	//查询要改的旧邮箱对应的userID
	users := getUsers(concatOldEmail(oldsFix))

	//生成脚本文件
	createSqlFile(oldsFix,newsFix, users)

}


/*生成脚本文件*/
func createSqlFile(olds []string,news []string, users []User) {
	var newFileName = basePath + time.Now().Format("20060102_150405") + ".sql"

	contents := getContent(olds, news, users)
	println("success!! file name is : ", newFileName)

	os.Create(newFileName)
	//打开FileName文件，如果不存在就创建新文件，打开的权限是可读可写，权限是644。这种打开方式相对下面的打开方式权限会更大一些。
	file, e := os.OpenFile(newFileName, os.O_APPEND,0644)
	checkError(e)

	for _, content := range contents {
		file.WriteString(content)
	}
	defer file.Close()

}

/*获取内容*/
func getContent(olds []string, news []string, users []User) ([]string){

	var result []string

	for i, old := range olds {
		id,err := getUserIdByEmail(old, users)
		checkError(err)

		new := news[i]
		var sql = "update yamibuy_master.xysc_users set email  = '"+new+"',edit_dtm=unix_timestamp() where email = '"+old+"' and user_id = "+id+";\n"

		result = append(result, sql)
	}
	return result

}
func getUserIdByEmail(email string, users []User) (string , error){
	for _, user := range users {
		if user.Email == email {
			return user.UserId, nil
		}
	}
	return "", errors.New("don't find userId")
}



func fixOldsNews(olds []string, news []string, existsEmails []string) ([]string, []string){

	for _, target := range existsEmails {
		var index = 0
		for i, newValue := range news {
			if target == newValue {
				// 删除news中第index个元素
				news = append(news[:i], news[i+1:]...)
				index = i
			}
		}
		olds = append(olds[:index], olds[index+1:]...)
	}
	return olds, news
}


func show(title string,existsEmails []string) {
	length := len(existsEmails)
	println("-----------",title,"(",length,")-----------")
	for _, value := range existsEmails {
		println(value)
	}
	println("-----------",title," End-----------")
}

func getExistsEmail(emailsStr string) []string {
	db, err := sql.Open("mysql", username+":"+password+"@tcp("+host+")/"+dbname+"?charset=utf8")
	checkError(err)

	var sql = "select email from yamibuy_master.xysc_users where email in ("+emailsStr+")"
	println("query exists emails sql: ", sql)
	stmt, err := db.Prepare(sql)
	checkError(err)
	rows, err := stmt.Query()

	var results []string
	for rows.Next() {
		var email string
		err := rows.Scan(&email)
		checkError(err)

		results = append(results, email)
	}
	defer db.Close()
	defer stmt.Close()
	defer rows.Close()
	return results
}

func readExcel(fileName string) ([]string, []string){
	file, err := xlsx.OpenFile(fileName)
	checkError(err)

	var (oldEmails []string
		 newEmails []string
	)
	for _, sheet := range file.Sheets {
		for _, row := range sheet.Rows {
			cells := row.Cells
			oldEmails = append(oldEmails, cells[0].String())
			newEmails = append(newEmails, cells[1].String())
		}
	}
	return oldEmails, newEmails
}

func concatOldEmail(oldEmail []string) string {
	var result = ""
	for _, value := range oldEmail {
		result += "'" + strings.Trim(value," ") + "',"
	}
	// 截掉最后一位
	return result[0:len(result)-1]
}





func checkError(e error) {
	if(e != nil) {
		println("error occured : ", e)
	}
}


type User struct {
	UserId string
	Email string
}



func getUsers(emailsStr string) []User {
	db, err := sql.Open("mysql", username+":"+password+"@tcp("+host+")/"+dbname+"?charset=utf8")
	checkError(err)
	var sql = "select user_id, email from yamibuy_master.xysc_users where email in ("+emailsStr+")"
	println("query users sql: ", sql)
	stmt, err := db.Prepare(sql)
	checkError(err)
	rows, err := stmt.Query()

	var results []User
	for rows.Next() {
		var userId string
		var email string
		err := rows.Scan(&userId,&email)
		checkError(err)

		var user User

		user.UserId = userId
		user.Email = email

		results = append(results, user)
	}
	defer db.Close()
	defer stmt.Close()
	defer rows.Close()
	return results
}