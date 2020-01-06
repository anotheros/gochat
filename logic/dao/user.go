/**
 * Created by lock
 * Date: 2019-09-22
 * Time: 22:53
 */
package dao

import (
	"github.com/pkg/errors"
	"gochat/db"
	"time"
)

var dbIns = db.GetDb("gochat")

type User struct {
	Id         int `gorm:"primary_key"`
	UserName   string
	Password   string
	CreateTime time.Time
	db.DbGoChat
}

func (u *User) TableName() string {
	return "user"
}

func (u *User) Add() (userId int, err error) {
	if u.UserName == "" || u.Password == "" {
		return 0, errors.New("user_name or password empty!")
	}
	oUser := u.CheckHaveUserName(u.UserName)
	if oUser.Id > 0 {
		return oUser.Id, nil
	}
	u.CreateTime = time.Now()
	if err = dbIns.Table(u.TableName()).Create(&u).Error; err != nil {
		return 0, err
	}
	return u.Id, nil
}

func (u *User) CheckHaveUserName(userName string) (data User) {
	dbIns.Table(u.TableName()).Where("user_name=?", userName).First(&data)
	return
}

func (u *User) GetUserNameByUserId(userId int) (userName string) {
	if userId == 5 {
		return "laozhang5"
	}

	if userId == 6 {
		return "zhang6"
	}

	var data User
	dbIns.Table(u.TableName()).Where("id=?", userId).First(&data)
	return data.UserName
}
