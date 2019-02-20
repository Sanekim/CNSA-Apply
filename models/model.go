﻿package models

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"golang.org/x/crypto/bcrypt"
)

const (
	// SQLConnectionString : MySQL Connection String
	SQLConnectionString = "*"
	// SALT : SALT
	SALT = "*"
)

// Database Connection
var db *gorm.DB

func init() {
	var err error
	db, err = gorm.Open("mysql", SQLConnectionString)
	if err != nil {
		panic(err)
	}
}

// Apply models
type Apply struct {
	StudentNumber string    `gorm:"type:VARCHAR(6); primary_key" json:"student-number"`
	Name          string    `gorm:"type:VARCHAR(50)" json:"name"`
	Date          time.Time `gorm:"type:DATE; primary_key; unique_index" json:"date"`
	Period        string    `gorm:"type:VARCHAR(5); primary_key; unique_index" json:"period"`
	Form          string    `gorm:"type:VARCHAR(1)" json:"form"`
	Seat          string    `gorm:"type:VARCHAR(6); unique_index" json:"seat"`
}

// TableName of Apply
func (c *Apply) TableName() string {
	return "applys"
}

// User models
type User struct {
	StudentNumber string `gorm:"type:VARCHAR(6); primary_key" json:"student-number"`
	Password      string `gorm:"type:VARCHAR(100)" json:"password"`
	Name          string `gorm:"type:VARCHAR(50)" json:"name"`
	Gender        int    `gorm:"type:INT" json:"gender"`
}

// TableName of User
func (c *User) TableName() string {
	return "users"
}

// Holyday models
type Holyday struct {
	Date time.Time `gorm:"type:DATE; primary_key" json:"date"`
	Name string    `gorm:"type:VARCHAR(50)" json:"name"`
}

// TableName of Holyday
func (c *Holyday) TableName() string {
	return "holydays"
}

// Login 학생 아이디 인증(SALT)
func Login(studentNumber string, password string) bool {
	user := User{}
	err := db.Table("users").Where("student_number = ?", studentNumber).First(&user).Error

	if err != nil {
		return false
	}

	return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password+SALT)) == nil
}

// ChangePassword 비밀번호 변경
func ChangePassword(studentNumber string, password string, newPassword string) error {
	if len(newPassword) > 30 {
		// 새로운 비밀번호의 길이가 길 때
		err := errors.New("Exceed the length")
		return err
	} else if newPassword == "" {
		// 아무것도 입력하지 않았을 때
		err := errors.New("NewPassword is empty")
		return err
	}
	user := User{}
	db.Table("users").Where("student_number = ?", studentNumber).First(&user)

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password+SALT)) != nil {
		// db의 현재 비밀번호와 입력된 현재 비밀번호가 일치하지 않을 때
		return errors.New("Password is incorrect")
	}

	bytes, _ := bcrypt.GenerateFromPassword([]byte(newPassword+SALT), bcrypt.DefaultCost)
	user.Password = string(bytes)

	return db.Save(&user).Error
}

// Tlogin 교사 아이디 인증(cnsanet)
func Tlogin(loginID string, loginPW string) bool {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, _ := client.PostForm("https://school.cnsa.hs.kr/login/userLogin", url.Values{
		"loginId": {loginID},
		"loginPw": {loginPW},
	})

	if resp.Request.Method == "GET" {
		return true
	}
	return false
}

// GetTimeTableDays : 페이지에 표시할 5일을 반환
// 월~금: 금주 월~금을 반환
// 토~일: 다음주 월~금을 반환
func GetTimeTableDays() [5]time.Time {
	var days [5]time.Time
	now := time.Now()

	if now.Weekday() > 5 {
		// 주말일 경우 시작일이 다음주 월요일
		days[0] = now.AddDate(0, 0, 8-int(now.Weekday()))
	} else {
		// 평일일 경우 시작일이 금주 월요일
		days[0] = now.AddDate(0, 0, 1-int(now.Weekday()))
	}
	days[1] = days[0].AddDate(0, 0, 1)
	days[2] = days[1].AddDate(0, 0, 1)
	days[3] = days[2].AddDate(0, 0, 1)
	days[4] = days[3].AddDate(0, 0, 1)

	return days
}

// AddApply 비어있는 좌석에 신청
// 같은 사람이 같은 시간에 신청은 선택할 때 방지
// 발생 가능한 오류는 비슷한 시간대에 동일한 좌석에 신청
func AddApply(studentNumber string, day time.Time, period string, form string, seat string) error {
	user := User{}
	db.Table("users").Where("student_number = ?", studentNumber).First(&user)
	apply := Apply{
		StudentNumber: studentNumber,
		Name:          user.Name,
		Date:          day,
		Period:        period,
		Form:          form,
		Seat:          seat,
	}

	err := db.Save(&apply).Error
	if err != nil {
		err = errors.New("The seat has been applied")
	}

	return err
}
