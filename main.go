package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const emailSender string = "pepitoperezprueba3@gmail.com"
const password string = "Pepitoperezprueba1@"

type user struct {
	Email     string
	Password  string
	Nickname  string
	FirstName string
	LastName  string
	Birthdate string
	Verified  bool
}

var tpl *template.Template
var logged bool
var userLogged user

type ViewData struct {
	UsersData  []user
	UserLogged user
}

var users = []user{
	{Email: "juan.reyes@icesi.edu.co", Password: "$2a$10$NFvHxcYS2nNHFVRzrmkurOS8IYg07ORm4.ZPGBnP3dIfzWFSHcEK2", Nickname: "seyerman", FirstName: "Juan", LastName: "Reyes", Birthdate: "1995-04-01", Verified: true},                //pwdSeyerman.1
	{Email: "fabio.avellaneda@icesi.edu.co", Password: "$2a$10$5mCaZJfXCqrlyQKGJ0EmZ.OuiQEEwfXH18PVva2Hy1v.ryMP.rJKi", Nickname: "favellaneda", FirstName: "Fabio", LastName: "Avellaneda", Birthdate: "1987-09-06", Verified: true}, //pwd: Favellaneda.1
}

type Info struct {
	Link     string
	Nickname string
}

func checkErr(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	//sendEmailTo("trabajosjuan2@gmail.com", "", "Juanpa")
	var err error
	logged = false
	tpl, err = template.ParseGlob("*.html")
	if err != nil {
		panic(err.Error())
	}
	data, err := ioutil.ReadFile("database.txt")
	if err == nil {
		readDB(data)
	} else {
		fmt.Println("There was an error loading the data base")
	}
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/login/", activateHandler)
	http.HandleFunc("/loginauth", loginAuthHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/loggedIn", loggedInHandler)
	http.HandleFunc("/registerauth", registerAuthHandler)
	http.ListenAndServe("localhost:8080", nil)
}

func activateHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.String()
	parts := strings.Split(id, "/")
	id = parts[2]
	for i := 3; i < len(parts); i++ {
		id += "/" + parts[i]
	}
	for i := 0; i < len(users); i++ {
		err := bcrypt.CompareHashAndPassword([]byte(id), []byte(users[i].Email))
		if err == nil {
			users[i].Verified = true
			save()
			tpl.ExecuteTemplate(w, "login.html", "The account was activated")
			return
		}
	}
	tpl.ExecuteTemplate(w, "login.html", "Something went wrong")
}

func loggedInHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("*****loggedInHandler running*****")
	if logged {
		data, err := ioutil.ReadFile("database.txt")
		if err == nil {
			readDB(data)
			vd := ViewData{UsersData: users, UserLogged: userLogged}
			tpl.ExecuteTemplate(w, "loggedIn.html", vd)
		} else {
			fmt.Println("There was an error adding the new user account.")
			return
		}
	} else {
		tpl.ExecuteTemplate(w, "login.html", "Not Logged In...")
	}
}

// loginHandler serves form for users to login with
func loginHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("*****loginHandler running*****")
	logged = false
	tpl.ExecuteTemplate(w, "login.html", nil)
}

// loginAuthHandler authenticates user login
func loginAuthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("*****loginAuthHandler running*****")
	r.ParseForm()
	email := r.FormValue("email")
	password := r.FormValue("password")
	fmt.Println("email:", email, "password:", password)
	// retrieve password from db to compare (hash) with user supplied password's hash
	signed, err := getUserByEmail(email)
	if err != nil {
		tpl.ExecuteTemplate(w, "login.html", "Email not registered!")
		return
	}
	if !signed.Verified {
		tpl.ExecuteTemplate(w, "login.html", "Email not verified. Please go to your email and active your account")
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(signed.Password), []byte(password))
	if err == nil {
		logged = true
		userLogged = signed
		loggedInHandler(w, r)
	} else {
		tpl.ExecuteTemplate(w, "login.html", "Wrong password!")
		return
	}
}

func getUserByEmail(email string) (user, error) {
	for _, a := range users {
		if a.Email == email {
			return a, nil
		}
	}
	var null user
	return null, errors.New("user not found")
}

func getUserByNickname(nick string) (user, error) {
	for _, a := range users {
		if a.Nickname == nick {
			return a, nil
		}
	}
	var null user
	return null, errors.New("user not found")
}

// registerHandler serves form for registering new users
func registerHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("*****registerHandler running*****")
	tpl.ExecuteTemplate(w, "register.html", nil)
}

// registerAuthHandler creates new user in database
func registerAuthHandler(w http.ResponseWriter, r *http.Request) {
	/*
		1. check nickname criteria
		2. check password criteria
		3. check if nickname is already exists in database
		4. create bcrypt hash from password
		5. insert nickname and password hash in database
	*/
	fmt.Println("*****registerAuthHandler running*****")
	r.ParseForm()
	nickname := r.FormValue("nickname")
	// check username for only alphaNumeric characters
	var nameAlphaNumeric = true
	for _, char := range nickname {
		// func IsLetter(r rune) bool, func IsNumber(r rune) bool
		// if !unicode.IsLetter(char) && !unicode.IsNumber(char) {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) {
			nameAlphaNumeric = false
		}
	}
	// check username pswdLength
	var nameLength bool
	if 5 <= len(nickname) && len(nickname) <= 50 {
		nameLength = true
	}
	// check password criteria
	password := r.FormValue("password")
	fmt.Println("password:", password, "\npswdLength:", len(password))
	// variables that must pass for password creation criteria
	var pswdLowercase, pswdUppercase, pswdNumber, pswdSpecial, pswdLength, pswdNoSpaces bool
	pswdNoSpaces = true
	for _, char := range password {
		switch {
		// func IsLower(r rune) bool
		case unicode.IsLower(char):
			pswdLowercase = true
		// func IsUpper(r rune) bool
		case unicode.IsUpper(char):
			pswdUppercase = true
		// func IsNumber(r rune) bool
		case unicode.IsNumber(char):
			pswdNumber = true
		// func IsPunct(r rune) bool, func IsSymbol(r rune) bool
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			pswdSpecial = true
		// func IsSpace(r rune) bool, type rune = int32
		case unicode.IsSpace(int32(char)):
			pswdNoSpaces = false
		}
	}
	if 5 < len(password) && len(password) < 60 {
		pswdLength = true
	}
	fmt.Println("pswdLowercase:", pswdLowercase, "\npswdUppercase:", pswdUppercase, "\npswdNumber:", pswdNumber, "\npswdSpecial:", pswdSpecial, "\npswdLength:", pswdLength, "\npswdNoSpaces:", pswdNoSpaces, "\nnameAlphaNumeric:", nameAlphaNumeric, "\nnameLength:", nameLength)
	if !pswdLowercase || !pswdUppercase || !pswdNumber || !pswdSpecial || !pswdLength || !pswdNoSpaces || !nameAlphaNumeric || !nameLength {
		tpl.ExecuteTemplate(w, "register.html", "please check nickname and password criteria")
		return
	}
	// check if nickname already exists for availability
	_, err := getUserByNickname(nickname)
	if err == nil {
		fmt.Println("nickname already exists, error: ", err)
		tpl.ExecuteTemplate(w, "register.html", "Nickname already taken!")
		return
	}
	password2 := r.FormValue("password2")
	if password != password2 {
		fmt.Println("Passwords don't match")
		tpl.ExecuteTemplate(w, "register.html", "Passwords don't match!")
		return
	}
	var hash []byte
	hash, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("bcrypt err:", err)
		tpl.ExecuteTemplate(w, "register.html", "There was a problem registering account.")
		return
	}

	firstname := r.FormValue("firstname")
	lastname := r.FormValue("lastname")
	birthdate := r.FormValue("birthdate")
	if strings.Contains(firstname, " ") || strings.Contains(lastname, " ") || strings.Contains(nickname, " ") || strings.Contains(birthdate, " ") {
		fmt.Println("One of the fields contains spaces")
		tpl.ExecuteTemplate(w, "register.html", "One of the fields contains spaces!")
		return
	}
	email := r.FormValue("email")
	_, err = mail.ParseAddress(email)
	if err != nil {
		fmt.Println("email err:", err)
		tpl.ExecuteTemplate(w, "register.html", "There was a problem registering account. Invalid email address.")
		return
	}

	// check if the email already exists for availability
	_, err = getUserByEmail(email)
	if err == nil {
		fmt.Println("Email already exists, error: ", err)
		tpl.ExecuteTemplate(w, "register.html", "Email already taken!")
		return
	}

	newUser := user{
		Email:     email,
		Password:  string(hash),
		Nickname:  nickname,
		FirstName: firstname,
		LastName:  lastname,
		Birthdate: birthdate,
		Verified:  false,
	}
	users = append(users, newUser)
	id, _ := bcrypt.GenerateFromPassword([]byte(email), bcrypt.DefaultCost)
	sendEmailTo(email, string(id), nickname)
	err = save()
	if err != nil {
		fmt.Println("There was an error adding the new user account.")
		tpl.ExecuteTemplate(w, "register.html", "There was an error adding the new user account.")
		return
	}
	data, err := ioutil.ReadFile("database.txt")
	if err != nil {
		fmt.Println("There was an error adding the new user account.")
		tpl.ExecuteTemplate(w, "register.html", "There was an error adding the new user account.")
		return
	}
	readDB(data)
	tpl.ExecuteTemplate(w, "login.html", "Congrats, your account has been successfully created!")
}

func readDB(data []byte) {
	dataStr := string(data)
	parts := strings.Split(dataStr, "\n")
	users = nil
	for i := 1; i < len(parts)-1; i++ {
		parts2 := strings.Split(parts[i], " ")
		var temp bool
		if parts2[6] == "true" {
			temp = true
		} else {
			temp = false
		}
		newUser := user{
			Email:     parts2[0],
			Password:  parts2[1],
			Nickname:  parts2[2],
			FirstName: parts2[3],
			LastName:  parts2[4],
			Birthdate: parts2[5],
			Verified:  temp,
		}
		users = append(users, newUser)
	}
}

func save() error {
	filename := "database.txt"
	var res string
	res = "Email Password Nickname Firstname Lastname Birthdate\n"
	for _, a := range users {
		res += a.Email + " " + a.Password + " " + a.Nickname + " " + a.FirstName + " " + a.LastName + " " + a.Birthdate + " " + strconv.FormatBool(a.Verified) + "\n"
	}
	data := []byte(res)
	return ioutil.WriteFile(filename, data, 0600)
}

func sendEmailTo(email string, id string, nickname string) {
	from := mail.Address{"Pepito Perez", emailSender}
	to := mail.Address{nickname, email}
	subject := "Enlace para activar tu cuenta"
	dest := Info{Nickname: nickname, Link: "localhost:8080/login/" + id}

	headers := make(map[string]string)
	headers["From"] = from.String()
	headers["To"] = to.String()
	headers["Subject"] = subject
	headers["Content-Type"] = `text/html; charset="UTF-8"`

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}

	t, err1 := template.ParseFiles("template.html")
	checkErr(err1)

	buf := new(bytes.Buffer)
	err1 = t.Execute(buf, dest)
	checkErr(err1)

	message += buf.String()

	servername := "smtp.gmail.com:465"
	host := "smtp.gmail.com"

	auth := smtp.PlainAuth("", emailSender, password, host)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}

	conn, err1 := tls.Dial("tcp", servername, tlsConfig)
	checkErr(err1)

	client, err1 := smtp.NewClient(conn, host)
	checkErr(err1)

	err1 = client.Auth(auth)
	checkErr(err1)

	err1 = client.Mail(from.Address)
	checkErr(err1)

	err1 = client.Rcpt(to.Address)
	checkErr(err1)

	w, err1 := client.Data()
	checkErr(err1)

	_, err1 = w.Write([]byte(message))
	checkErr(err1)

	err1 = w.Close()
	checkErr(err1)

	client.Quit()
}
